FROM golang:1.12
ARG skipTests=true

ADD . /go/src/github.com/smecsia/go-utils

WORKDIR /go/src/github.com/smecsia/go-utils

RUN SKIP_TESTS=${skipTests} make build