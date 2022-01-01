#!/bin/bash

set -ev # exit if any command fails, verbose

GOPATH=$HOME/go
KITECO=$GOPATH/src/github.com/kiteco/kiteco

# parse args
while [[ $# > 1 ]]
do
    key="$1"
    case $key in
        --version)
        VERSION=$2
        shift
        ;;
        *)
        # unknown option
        echo "Unknown option:" $key
        exit 1
        ;;
    esac
done

if [[ -z "$VERSION" ]]; then
    echo "VERSION is not set, set it using --version. exiting."
    exit 1
fi

cd $KITECO

echo "Building test release with version $VERSION..."
./scripts/stage_new_build.sh --ignore-git --no-upload --testing --version $VERSION
