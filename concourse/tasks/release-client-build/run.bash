#!/usr/bin/env bash

set -e

export GOPATH=$PWD/gopath

[ -n $RELEASE_DB_URI ] || { echo "Release DB URI not configured. Aborting."; exit 1; }
[ -n $PERCENTAGE ] || { echo "Percentage not configured. Aborting."; exit 1; }
export RELEASE_DB_DRIVER='postgres'

source meta/META

(
    set -ax
    _PLATFORM=$PLATFORM
    _VERSION=$VERSION
    _GIT_HASH=$COMMIT
    _CANARY_PERCENTAGE=$PERCENTAGE
    _SIGNATURE=$SIGNATURE
    ./release_bin/release add
)

(
    set -a

    _NUM_DELTAS=${#DELTA_FROM[@]}
    i=0
    for FROM_VERSION in "${DELTA_FROM[@]}"
    do
        source meta/deltaFrom/$FROM_VERSION/META

        set -x
        declare _PLATFORM_DELTA_$i=$PLATFORM
        declare _FROM_VERSION_DELTA_$i=$FROM_VERSION
        declare _VERSION_DELTA_$i=$TO_VERSION
        declare _SIGNATURE_DELTA_$i=$SIGNATURE
        set +x

        let i=i+1
    done

    set -x
    ./release_bin/release addDeltas
)
