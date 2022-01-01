#!/bin/bash

# Setup golang
UNAME=`uname`
VERSION=1.15.3
if [[ "$UNAME" == "Darwin" ]]; then
	OS="darwin-amd64"
else
	OS="linux-amd64"
fi

if [[ -d /usr/local/go ]] ; then
    sudo rm -rf /usr/local/go
    rm -rf $GOPATH/pkg/*
fi

DLHOST=https://storage.googleapis.com/golang
FILENAME=go$VERSION.$OS.tar.gz

if [[ "$UNAME" == "Darwin" ]]; then
    if [[ ! -d /usr/local ]]; then
        sudo mkdir -p /usr/local
        sudo chown -R $USER:staff /usr/local
    fi
fi

echo "Downloading $FILENAME"
curl $DLHOST/$FILENAME -o $FILENAME
sudo tar -C /usr/local/ -xzf $FILENAME
rm -f $FILENAME
