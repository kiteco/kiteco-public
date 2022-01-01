#!/bin/bash

set -e

VER=$(go version | cut -d" " -f3)
if [ "$VER" != "go1.15.3" ]; then
    echo "Please install Go 1.15.3"
    exit 1
fi

export KITECO="${KITECO:-$(go env GOPATH)/src/github.com/kiteco/kiteco}"

export GOPRIVATE=github.com/kiteco/*

# See notes at https://golang.org/cmd/cgo/#hdr-Using_cgo_with_the_go_command
export CGO_CFLAGS_ALLOW=".+"
export CGO_LDFLAGS_ALLOW=".+"

rm -rf $KITECO/osx/libkited
mkdir -p $KITECO/osx/libkited
cd $KITECO/osx/libkited
go build \
	-buildmode=c-archive \
	-ldflags "-L $KITECO/osx/tensorflow/lib" \
	-gcflags "-I $KITECO/osx/tensorflow/include" \
	-ldflags "-X github.com/kiteco/kiteco/kite-go/client/internal/clientapp.gitCommit=$(git rev-parse --short HEAD)" \
	github.com/kiteco/kiteco/kite-go/client/libkited
