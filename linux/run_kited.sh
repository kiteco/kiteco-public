#!/bin/bash

set -e

GITCOMMIT=$(git rev-parse HEAD)
export KITECO="${KITECO:-$(go env GOPATH)/src/github.com/kiteco/kiteco}"
KITECO_LINUX="$KITECO/linux"

cd "$KITECO_LINUX"

echo "Checking for the copilot (kite) ..."
if [ ! -f "linux-unpacked/kite" ]; then
    echo "Please run 'kite-go/linux/build_electron.sh' to build the copilot"
    exit 1
fi

echo "Found the copilot, building kited (${GITCOMMIT}) ..."

rm -f kited

# at runtime our locally build kited binary checks tensorflow/lib for shared libraries
# the deployed kited binary is only checking . before $LD_LIBRARY_PATH (see cmds/kited/main.go)
go build \
    -ldflags "-X github.com/kiteco/kiteco/kite-go/client/internal/clientapp.gitCommit=${GITCOMMIT}" \
    -ldflags "-r tensorflow/lib" \
    github.com/kiteco/kiteco/kite-go/client/cmds/kited

./kited
