#!/usr/bin/env bash
set -e
export KITECO="${KITECO:-$(go env GOPATH)/src/github.com/kiteco/kiteco}"
export ENVIRONMENT=development

cd $KITECO

if [[ $CONFIGURATION == "Release" ]]; then
    ENVIRONMENT=production
fi

if [[ $1 == "force" || $CONFIGURATION == "Release" ]]; then
    rm -rf dist
    cd $KITECO/sidebar
    npm install
    echo "ENV: $ENVIRONMENT"
    REACT_APP_ENV=$ENVIRONMENT yarn run pack
    exit 0
fi

echo "Checking for electon/Kite.app..."
if [ ! -d "sidebar/dist/mac/Kite.app" ]; then
    echo "... not found. Please build the sidebar application."
    exit 1
fi

echo "... found!"
