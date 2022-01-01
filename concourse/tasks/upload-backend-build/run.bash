#!/usr/bin/env bash

set -e

source build/META

[ -n $VERSION ] || { echo "Version not specified in metadata. Aborting."; exit 1; }

aws s3 sync build/ s3://kite-deploys/$VERSION/
