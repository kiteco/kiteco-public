#!/usr/bin/env bash

set -e

echo "Building Lambda functions..."

BUILD_DIR="$PWD/build/"

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

cd kiteco/lambda-functions/telemetry-loader && make BUILD_DIR=$BUILD_DIR package
