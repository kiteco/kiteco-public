#!/bin/bash

set -e

echo $RELEASE_DB_URI
if [ -z "$RELEASE_DB_URI" ]; then
  echo "Release DB URI not configured. Aborting."
  exit 1
fi
export RELEASE_DB_DRIVER='postgres'


tag=$(cd kiteco && git describe --tags --exact-match)
if [[ $tag =~ ^v[0-9]{8}\.[0-9]+$ ]]; then
    true
else
    echo "Error preparing release: invalid tag format ($tag). Aborting."
    exit 1
fi
version=$(echo $tag | cut -c2-)
IFS=. read version_date version_patch <<< "$version"

now_ts=$(date +%s)
if [[ "$OSTYPE" == "darwin"* ]]; then
  # this is necessary because we dispatch to this script on macOS for the macOS build
  version_date_ts=$(date -j -f "%Y%m%d" $version_date +%s)
else
  version_date_ts=$(date -d $version_date +%s)
fi

if [ $version_date_ts -gt $now_ts ]; then
    echo "Error preparing release: cannot release a version from the future. Aborting."
    exit 1
fi

case $platform in
mac)
    version=0.$version
    ;;
windows)
    version=$(date -d $version_date +1.%Y.%-m%d.$version_patch)
    ;;
linux)
    version=2.$version
    ;;
*)
    echo "Error preparing release: invalid platform ($platform). Aborting."
    exit 1
    ;;
esac

# The "previous" version as used to generate delta updates is the latest release on production
metadata=$(RELEASE_DB_ENV="prod" _PLATFORM=$platform ./release_bin/release latest)
echo $metadata
eval $metadata
prev=$_VERSION
commit=$(cd kiteco && git rev-parse HEAD)

echo "$version" > version/version
echo "$prev" > version/prev
echo "$commit" > version/commit

echo "Building $platform $version" | tee slack/message
