#!/usr/bin/env bash
VER=$(go version | cut -d" " -f3)
if [ "$VER" != "go1.15.3" ]; then
    echo "Please install Go 1.15.3"
    exit 1
fi
go $1 -tags standalone github.com/kiteco/kiteco/kite-go/client/cmds/kited
