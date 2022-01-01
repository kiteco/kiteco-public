#!/usr/bin/env bash
set -e

BUILD_DIR="$PWD/build"
KITECO=kiteco

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

echo "Uploading Puppet build..."
aws s3 cp $BUILD_DIR/puppet.tar.gz s3://kite-deploys/$VERSION/puppet.tar.gz
