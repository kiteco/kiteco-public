#!/bin/bash

set -e
KITECO=${KITECO:-$(go env GOPATH)/src/github.com/kiteco/kiteco}
ENVIRONMENT="${REACT_APP_ENV:-development}"

cd $KITECO/sidebar
rm -rf dist
npm install
REACT_APP_ENV=$ENVIRONMENT npm run pack:win
rm -rf $KITECO/windows/win-unpacked
cp -r $KITECO/sidebar/dist/win-unpacked $KITECO/windows/
rm -rf $KITECO/sidebar/dist
