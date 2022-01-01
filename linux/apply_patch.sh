#!/usr/bin/env bash
set -e

PATCH_FILE="$1"
[ -n "$PATCH_FILE" ] || { echo >&2 "patch file parameter not passed. Usage: $0 <patch file> <previous version> <version> <patch dir>"; exit 1; }

PREVIOUS_VERSION="$2"
[ -n "$PREVIOUS_VERSION" ] || { echo >&2 "previous version parameter not passed. Usage: $0 <patch file> <previous version> <version> <patch dir>"; exit 1; }
PREVIOUS_VERSION="kite-v$PREVIOUS_VERSION"

VERSION="$3"
[ -n "$VERSION" ] || { echo >&2 "version parameter not passed. Usage: $0 <patch file> <previous version> <version> <patch dir>"; exit 1; }
VERSION="kite-v$VERSION"

PATCH_DIR="$4"
[ -n "$PATCH_DIR" ] || { echo >&2 "patch dir parameter not passed. Usage: $0 <patch file> <previous version> <version> <patch dir>"; exit 1; }

PREV_TAR=$PREVIOUS_VERSION.tar
NEXT_TAR=$VERSION.tar

cleanup () {
    EXIT_CODE=$?

    echo "Cleaning up..."
    cd ..
    rm -rf $PATCH_DIR

    if [[ $EXIT_CODE -ne 0 ]]
    then
        # clean up failed patched version directory
        rm -rf "$VERSION"
    fi

    exit $EXIT_CODE
}

trap cleanup EXIT

echo "Creating tar of previous version directory..."
# these flags are necessary to ensure the tar is the same when we create the patch and when
# we apply it
./tar archive ../$PREVIOUS_VERSION > $PREV_TAR

echo "Applying patch update... "
./bspatch $PREV_TAR $NEXT_TAR $PATCH_FILE

echo "Extracting patched update..."
mkdir ../$VERSION
./tar extract ../$VERSION < $NEXT_TAR
