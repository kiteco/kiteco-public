#!/usr/bin/env bash

set -e

export GOPATH="$PWD/gopath"
export GO111MODULE="on"
export GOPRIVATE=github.com/kiteco/*
BUILD_DIR="$PWD/build"

KITECO=$GOPATH/src/github.com/kiteco/kiteco

VERSION=$(cat version/version)
COMMIT=$(cat version/commit)
PREVVERSION=$(cat version/prev)

echo "Building Linux client..."
echo "VERSION=$VERSION"
echo "COMMIT=$COMMIT"
echo "PREVVERSION=$PREVVERSION"
echo

# TODO make this work without the cd. cd $KITECO isn't enough either...
cd $KITECO/linux && ./docker/build_installer.sh "$VERSION" "$BUILD_DIR"
cd $KITECO/linux && ./docker/build_update_package.sh "$VERSION" "$BUILD_DIR" "$PRIVATE_KEY" "$PREVVERSION"

echo
echo

mv "$BUILD_DIR/version-$VERSION.json" "$BUILD_DIR/version.json"
mv "$BUILD_DIR/kite-installer-$VERSION" "$BUILD_DIR/kite-installer"
mv "$BUILD_DIR/kite-installer-$VERSION.sh" "$BUILD_DIR/kite-installer.sh"
mv "$BUILD_DIR/kite-updater-$VERSION.sh" "$BUILD_DIR/kite-updater.sh"
echo "PLATFORM=linux" >> "$BUILD_DIR/META"
echo "VERSION=$VERSION" >> "$BUILD_DIR/META"
echo "COMMIT=$COMMIT" >> "$BUILD_DIR/META"
echo "SIGNATURE=" >> "$BUILD_DIR/META"
echo "build/META:"
cat $BUILD_DIR/META && echo

if [ -n "$PREVVERSION" ]; then
    mkdir -p $BUILD_DIR/deltaFrom/$PREVVERSION
    mv $BUILD_DIR/kite-patch-updater$PREVVERSION-$VERSION.sh $BUILD_DIR/deltaFrom/$PREVVERSION/kite-updater.sh
    mv "$BUILD_DIR/version-$PREVVERSION-$VERSION.json" $BUILD_DIR/deltaFrom/$PREVVERSION/version.json
    echo "DELTA_FROM[0]=$PREVVERSION" >> $BUILD_DIR/META
    echo "PLATFORM=linux" >> $BUILD_DIR/deltaFrom/$PREVVERSION/META
    echo "FROM_VERSION=$PREVVERSION" >> $BUILD_DIR/deltaFrom/$PREVVERSION/META
    echo "TO_VERSION=$VERSION" >> $BUILD_DIR/deltaFrom/$PREVVERSION/META
    echo "SIGNATURE=" >> $BUILD_DIR/deltaFrom/$PREVVERSION/META
    echo "build/deltaFrom/$PREVVERSION/META:"
    cat $BUILD_DIR/deltaFrom/$PREVVERSION/META && echo
fi
