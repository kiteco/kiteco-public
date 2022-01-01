#!/usr/bin/env bash
set -e
go build github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/cmds/build-datadeps-filemap
./build-datadeps-filemap -verify
rm -f build-datadeps-filemap
