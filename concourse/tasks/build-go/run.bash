#!/usr/bin/env bash

set -e

export GOPATH="$PWD/gopath"
BUILD_DIR="$PWD/build"
KITECO=$GOPATH/src/github.com/kiteco/kiteco


# copied from prepare-release: TODO(naman) factor this out
tag=$(cd $KITECO && git describe --tags --exact-match)
if [[ $tag =~ ^v[0-9]{8}\.[0-9]+$ ]]; then
    true
else
    echo "Error preparing release: invalid tag format ($tag). Aborting."
    exit 1
fi
VERSION=$tag
COMMIT=$(cd $KITECO && git rev-parse HEAD)

echo "Building backend..."
echo "VERSION=$VERSION"
echo "COMMIT=$COMMIT"

CGO_LDFLAGS_ALLOW=".*" go build -o "build/$2-$VERSION" $1

echo "VERSION=$VERSION" >> build/META
echo "COMMIT=$COMMIT" >> build/META
