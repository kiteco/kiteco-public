#!/usr/bin/env bash

set -e # exit if any command fails

go build github.com/kiteco/kiteco/kite-go/cmds/release

export RELEASE_DB_DRIVER='postgres'
export RELEASE_DB_URI='postgres://XXXXXXX:XXXXXXX@10.201.150.4/release'


set -a  # export all this stuff
_PLATFORM=$PLATFORM
eval $(_PLATFORM=$PLATFORM ./release latest)
_CANARY_PERCENTAGE=$CANARY_PERCENTAGE
set +a

set +e
# add release to prod
(
    eval $(ssh release2-vpn.kite.com 'bash --login -c "env | grep ^RELEASE_DB_DRIVER="' | sed 's/.*/export &/')
    eval $(ssh release2-vpn.kite.com 'bash --login -c "env | grep ^RELEASE_DB_URI="' | sed 's/.*/export &/')
    ./release add
)
set -e

set -a
eval $(./release latestDeltas)
set +a

(
    eval $(ssh release2-vpn.kite.com 'bash --login -c "env | grep ^RELEASE_DB_DRIVER="' | sed 's/.*/export &/')
    eval $(ssh release2-vpn.kite.com 'bash --login -c "env | grep ^RELEASE_DB_URI="' | sed 's/.*/export &/')
    ./release addDeltas
)
