#!/bin/bash

VERSION=0.0.3
PUPPET=$(which puppet)
GEM=/usr/bin/gem
MODULE_DIR=$HOME/.puppet/modules

# Install puppet if it hasn't been installed
if [ ! -n "$PUPPET" ]; then
    echo "== Installing puppet...."
    # Use system ruby (via /usr/bin/gem - NOT HOMEBREW), and disable rdoc, and ri
    sudo $GEM install --no-rdoc --no-ri puppet &> /dev/null
    if [ $? != 0 ]; then
        echo "FAILED"
        exit 1
    fi
    PUPPET=$(which puppet)
    echo "OK"
fi

echo "== Building kite-kite-$VERSION..."
$PUPPET module build
if [ $? != 0 ]; then
    exit 1
fi

echo "== Installing kite-kite-$VERSION.."
rm -rf $MODULE_DIR
$PUPPET module install --modulepath=$MODULE_DIR pkg/kite-kite-$VERSION.tar.gz
if [ $? != 0 ]; then
    exit 1
fi

# Cleanup
rm -rf pkg
