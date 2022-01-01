FROM golang:1.15.3-buster as builder

ENV GOPRIVATE=github.com/kiteco/*

WORKDIR /go/src/github.com/kiteco/kiteco

COPY go.mod .
COPY go.sum .
COPY kite-go ./kite-go
COPY kite-golib ./kite-golib
COPY services/release.kite.com ./services/release.kite.com

WORKDIR /go/src/github.com/kiteco/kiteco/services/release.kite.com
RUN go build .

FROM ubuntu:20.04
WORKDIR /root/
ENV RELEASE_DB_DRIVER=postgres
COPY --from=0 /go/src/github.com/kiteco/kiteco/services/release.kite.com/release.kite.com .
ENTRYPOINT ["./release.kite.com"]

