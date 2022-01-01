#!/usr/bin/env bash

set -e

CLUSTERS_FILE=$1

if [[ -z $CLUSTERS_FILE ]]; then
    echo "usage: stop-cluster.sh CLUSTERS_FILE"
    exit 1
fi

if [[ ! -e $CLUSTERS_FILE ]]; then
    echo "$CLUSTERS_FILE doesn't exist"
    exit 1
fi

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

while read p; do
  azure-cluster stop "$p"
done < $CLUSTERS_FILE

echo "removing $CLUSTERS_FILE"
rm $CLUSTERS_FILE
