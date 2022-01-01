#!/usr/bin/env bash
set -e

BUILD_DIR="$PWD/build"
apt-get update && apt-get install -y git
cd kiteco/devops/puppet && bolt puppetfile install && tar -zcvf $BUILD_DIR/puppet.tar.gz .
