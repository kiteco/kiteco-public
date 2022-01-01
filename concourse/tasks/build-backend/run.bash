#!/usr/bin/env bash

set -e

export GOPATH="$PWD/gopath"
export GO111MODULE="on"
BUILD_DIR="$PWD/build"
KITECO=$GOPATH/src/github.com/kiteco/kiteco

cd $KITECO

# copied from prepare-release: TODO(naman) factor this out
tag=$(git describe --tags --exact-match)
if [[ $tag =~ ^v[0-9]{8}\.[0-9]+$ ]]; then
    true
else
    echo "Error preparing release: invalid tag format ($tag). Aborting."
    exit 1
fi
VERSION=$tag
COMMIT=$(git rev-parse HEAD)


echo "Building backend..."
echo "VERSION=$VERSION"
echo "COMMIT=$COMMIT"


CGO_LDFLAGS_ALLOW=".*" go build -o $BUILD_DIR/user-node github.com/kiteco/kiteco/kite-go/cmds/user-node
CGO_LDFLAGS_ALLOW=".*" go build -o $BUILD_DIR/user-mux github.com/kiteco/kiteco/kite-go/cmds/user-mux
CGO_LDFLAGS_ALLOW=".*" go build -o $BUILD_DIR/release github.com/kiteco/kiteco/kite-go/cmds/release

pushd $KITECO/kite-server
make kite-server.tgz DIR=kite-server TOKEN=placeholder
popd
cp $KITECO/kite-server/kite-server.tgz $BUILD_DIR

echo "VERSION=$VERSION" >> $BUILD_DIR/META
echo "COMMIT=$COMMIT" >> $BUILD_DIR/META
