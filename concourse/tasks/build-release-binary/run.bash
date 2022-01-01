#!/usr/bin/env bash

set -e

export GOPATH=$PWD/gopath
export BUILD_DIR=$PWD
export GO111MODULE="on"

KITECO=$GOPATH/src/github.com/kiteco/kiteco
cd $KITECO

go build -o $BUILD_DIR/release_bin/release github.com/kiteco/kiteco/kite-go/cmds/release
