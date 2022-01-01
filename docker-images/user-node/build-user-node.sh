#!/bin/bash

SELF_DIR=$(dirname $BASH_SOURCE)

# If under linux then build natively
if go version | grep -q linux; then
	go build github.com/kiteco/kiteco/kite-go/cmds/user-node
	exit
fi

echo "Creating kiteco.tar.gz..."
rm -f /tmp/kiteco.tar.gz
tar czf /tmp/kiteco.tar.gz \
	-C $GOPATH/src \
	github.com/kiteco/kiteco/kite-go \
	github.com/kiteco/kiteco/kite-golib \
	github.com/kiteco/kiteco/vendor

echo "Creating container..."
CONTAINER=$(docker create -e GO15VENDOREXPERIMENT=1 golang:1.6 go build -o /user-node github.com/kiteco/kiteco/kite-go/cmds/user-node)
echo "Created $CONTAINER"

echo "Copying kiteco source into container..."
cat /tmp/kiteco.tar.gz | docker cp - $CONTAINER:/go/src/

echo "Starting container..."
docker start --attach $CONTAINER || exit 1

echo "Retrieving binary..."
docker cp $CONTAINER:/user-node user-node || exit 1

echo "Cleaning up..."
docker rm -f $CONTAINER || exit 1
rm -f /tmp/kiteco.tar.gz
