# Default to Go 1.16
ARG GO_VERSION=1.16
FROM golang:${GO_VERSION}-alpine as build

# Necessary to run 'go get' and to compile the linked binary
RUN apk add git musl-dev

ADD . /go/src/github.com/dutchcoders/transfer.sh

WORKDIR /go/src/github.com/dutchcoders/transfer.sh

ENV GO111MODULE=on

# build & install server
RUN CGO_ENABLED=0 go build -tags netgo -ldflags "-X github.com/dutchcoders/transfer.sh/cmd.Version=$(git describe --tags) -a -s -w -extldflags '-static'" -o /go/bin/transfersh

FROM alpine:3 AS final
LABEL maintainer="Andrea Spacca <andrea.spacca@gmail.com>"

# gawk and GNU sed are required for the ansi2html.sh script.
RUN apk add --no-cache gawk sed

ENV USER=transfer
ENV UID=1000
ENV GID=1000

RUN addgroup -g ${GID} -S ${USER} && \
    adduser -u ${UID} -S ${USER} -G ${USER}

COPY --from=build  /go/bin/transfersh /go/bin/transfersh
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
VOLUME [ "/app" ]

EXPOSE 8080

USER transfer

ENTRYPOINT ["/go/bin/transfersh"]

