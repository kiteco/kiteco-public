#!/usr/bin/env bash

# This script opens xcode for the osx project and then closes it.
# this is done to inform xcode about our new project path so it can build later

# if you don't run this script, attempting to build the project with xcodebuild will
# result in no error and an indefinite hang by the xcodebuild process


KITECO="${KITECO:-$HOME/kiteco}"

cd "${KITECO}/osx"

# note the -a option only works on osx
open -a Xcode Kite.xcodeproj

# wait time for xcode to open and do it's thing
sleep 10

# ideally xcode done doing its thing by now
killall Xcode

# wait time for cleanup
sleep 5

echo "xcode initialized for new folder"
