#!/bin/bash
set -e

if [[ -z "$1" ]]; then
  echo "no package directory given. using cwd"
else
  cd $1
fi

# remove existing assets to ensure a fresh install
rm -rf node_modules/
rm -f package-lock.json

# the below expects solness env variables
npm-login-noninteractive
npm install

# npm install will change package-lock.json -
# we don't care about that as it is not used in npm publish
git checkout -- package-lock.json
npm version minor
npm publish

