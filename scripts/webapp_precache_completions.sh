#!/bin/bash
set -e

KITECO="${KITECO:-$GOPATH/src/github.com/kiteco/kiteco}"
OUTDIR=$KITECO/webapp/src/assets/data/precaching

cd $KITECO/kite-go/cmds/completions-precacher
go generate
go build
mkdir -p $OUTDIR
./completions-precacher -out=$OUTDIR -name=completions.json