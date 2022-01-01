#!/bin/bash

set -e
export KITECO="${KITECO:-$(go env GOPATH)/src/github.com/kiteco/kiteco}"
export ENVIRONMENT="${REACT_APP_ENV:-development}"

cd $KITECO/sidebar
rm -rf dist
npm install
REACT_APP_ENV=$ENVIRONMENT npm run pack:linux
rm -rf $KITECO/linux/linux-unpacked
cp -r $KITECO/sidebar/dist/linux-unpacked $KITECO/linux/
rm -rf $KITECO/sidebar/dist
