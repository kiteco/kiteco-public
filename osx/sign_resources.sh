#!/usr/bin/env bash

CODE_SIGN_IDENTITY="Developer ID Application"

# TODO dedupe
codesign --force --deep -o runtime --timestamp --sign "$CODE_SIGN_IDENTITY" ./Sparkle.framework/Versions/A/Resources/Autoupdate.app
codesign --force --deep -o runtime --timestamp --sign "$CODE_SIGN_IDENTITY" ./Sparkle.framework
codesign --force --deep -o runtime --timestamp --sign "$CODE_SIGN_IDENTITY" ./KiteHelper/Sparkle.framework/Versions/A/Resources/Autoupdate.app
codesign --force --deep -o runtime --timestamp --sign "$CODE_SIGN_IDENTITY" ./KiteHelper/Sparkle.framework
