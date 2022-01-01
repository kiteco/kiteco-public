#!/usr/bin/env bash

set -e

source build/META

[ -n $PLATFORM ] || { echo "Platform not specified in metadata. Aborting."; exit 1; }
[ -n $VERSION ] || { echo "Version not specified in metadata. Aborting."; exit 1; }

aws s3 sync build/ s3://kite-downloads/$PLATFORM/$VERSION/ \
    --grants read=uri=http://acs.amazonaws.com/groups/global/AllUsers \
    --cache-control max-age=604800
