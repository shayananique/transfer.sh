/*
The MIT License (MIT)

Copyright (c) 2014-2017 DutchCoders [https://github.com/dutchcoders/]
Copyright (c) 2018-2020 Andrea Spacca.
Copyright (c) 2020- Andrea Spacca and Stefan Benten.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package server

import (
	"errors"
	"fmt"
	clamd "github.com/dutchcoders/go-clamd"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const clamavScanStatusOK = "OK"

func (s *Server) scanHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	filename := sanitize(vars["filename"])

	contentLength := r.ContentLength
	contentType := r.Header.Get("Content-Type")

	s.logger.Printf("Scanning %s %d %s", filename, contentLength, contentType)

	file, err := ioutil.TempFile(s.tempPath, "clamav-")
	defer s.cleanTmpFile(file)
	if err != nil {
		s.logger.Printf("%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(file, r.Body)
	if err != nil {
		s.logger.Printf("%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status, err := s.performScan(file.Name())
	if err != nil {
		s.logger.Printf("%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("%v\n", status)))
}

func (s *Server) performScan(path string) (string, error) {
	c := clamd.NewClamd(s.ClamAVDaemonHost)

	abort := make(chan bool)
	response := make(chan chan *clamd.ScanResult)
	err := make(chan error)
	go func(response chan chan *clamd.ScanResult, err chan error) {
		scanResponse, scanErr := c.ScanFile(path)
		if scanErr != nil {
			err <- scanErr
			return
		}

		response <- scanResponse
	}(response, err)

	select {
	case r := <-response:
		st := <-r
		return st.Status, nil
	case <-time.After(time.Second * 60):
		abort <- true
	}

	close(abort)

	return "", errors.New("clamav scan timeout")
}
