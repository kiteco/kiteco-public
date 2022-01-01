#!/bin/bash

set -e
export KITECO="${KITECO:-$GOPATH/src/github.com/kiteco/kiteco}"

rm -rf $KITECO/osx/kitelsp
mkdir -p $KITECO/osx/kitelsp
cd $KITECO/osx/kitelsp
go build \
	github.com/kiteco/kiteco/kite-go/lsp/cmds/kite-lsp
