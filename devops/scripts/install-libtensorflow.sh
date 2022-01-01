#!/bin/bash

# Sets up tensorflow c libraries and go bindings,
# assumes go is already installed.
# Based on: https://www.tensorflow.org/versions/master/install/install_go.

DLHOST=https://storage.googleapis.com/tensorflow/libtensorflow


UNAME=`uname`
VERSION=1.9.2
if [[ "$UNAME" == "Darwin" ]]; then
	OS="darwin"
else
	OS="linux"
fi

FILENAME=libtensorflow-cpu-$OS-x86_64-1.8.0.tar.gz

echo "Downloading $FILENAME"
curl $DLHOST/$FILENAME -o $FILENAME

echo "Installing c libraries, this may take awhile"
sudo tar -C /usr/local/ -xzf $FILENAME
rm -f $FILENAME

# configure linker

echo "Configuring linker"
if [[ "$UNAME" == "Darwin" ]]; then
	sudo update_dyld_shared_cache
else
	sudo ldconfig
fi

echo "Downloading go bindings, this may take awhile"
go get github.com/kiteco/tensorflow/tensorflow/go

echo "Testing go bindings"
go test github.com/kiteco/tensorflow/tensorflow/go
