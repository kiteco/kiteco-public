#!/usr/bin/env bash

KITECO="$HOME/kiteco"

cd $KITECO/osx
rm -rf build/
mkdir -p build
xcodebuild -scheme Kite -configuration Debug -derivedDataPath build
cd build/Build/Products/Debug/
zip -r Kite.zip Kite.app
python -m SimpleHTTPServer
