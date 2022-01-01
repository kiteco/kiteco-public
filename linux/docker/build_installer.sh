#!/usr/bin/env bash
# This script is supposed to be run in an Ubuntu docker container

set -e

VERSION="$1"
[ -n "$VERSION" ] || { echo >&2 "version not passed. Aborting."; exit 1; }

BUILD_DIR="$2"
[ -n "$BUILD_DIR" ] || { echo >&2 "build dir not passed. Aborting."; exit 1; }

PAR=$(dirname "$(readlink -f "$0")")
cd "$PAR"

echo "Building kite-installer binary..."
go build \
    -o "$BUILD_DIR/kite-installer-$VERSION" \
    -ldflags "-X main.version=${VERSION}" \
    github.com/kiteco/kiteco/linux/cmds/kite-installer

echo "Creating kite-installer.sh wrapper script..."
cp "./kite-installer.sh" "$BUILD_DIR/kite-installer-$VERSION.sh"
