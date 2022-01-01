#!/usr/bin/env bash

set -e

if [[ `uname` != "Linux" ]]; then
   echo "launch-dist-pipeline.sh needs to be started from Linux"
   exit 1
fi

if [[ "$#" -lt 2 ]]; then
  echo "usage: launch-dist-pipeline.sh PIPELINE_CMD_PACKAGE INSTANCE_COUNT [PIPELINE_ARGS]"
  exit 1
fi

PIPE_PKG=$1
INSTANCE_COUNT=$2
PIPE_ARGS="${@:3}"

if [[ -z "${INSTANCE_TYPE}" ]]; then
    INSTANCE_TYPE=standard_d8_v3
fi
STOP_AFTER_DONE=1
PORT=3111

PIPE_NAME=`basename $PIPE_PKG`
TMP_DIR=`mktemp -d`

echo "launching $INSTANCE_COUNT instances of type: $INSTANCE_TYPE"

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

SHARD_BUNDLE_FILE=$TMP_DIR/shard-bundle.tar.gz
echo "creating shard bundle at $SHARD_BUNDLE_FILE"
echo "$PIPE_NAME --role shard --port $PORT $PIPE_ARGS" > $TMP_DIR/shard-run.sh
azure-cluster bundle $SHARD_BUNDLE_FILE $TMP_DIR/shard-run.sh --go-binary $PIPE_PKG

echo "starting shard cluster"
SHARD_CLUSTER=`azure-cluster start shard-$PIPE_NAME $INSTANCE_COUNT  --instance_type $INSTANCE_TYPE`

SHARD_ENDPOINTS=`azure-cluster ips $SHARD_CLUSTER | tr '\n' ',' | sed "s/,/:$PORT /g"`
echo "shard endpoints: $SHARD_ENDPOINTS"

COORD_BUNDLE_FILE=$TMP_DIR/coordinator-bundle.tar.gz
echo "creating coordinator bundle at $COORD_BUNDLE_FILE"
echo "$PIPE_NAME --role coordinator --endpoints $SHARD_ENDPOINTS $PIPE_ARGS" > $TMP_DIR/coordinator-run.sh
azure-cluster bundle $COORD_BUNDLE_FILE $TMP_DIR/coordinator-run.sh --go-binary $PIPE_PKG

echo "starting coordinator cluster"
COORD_CLUSTER=`azure-cluster start coord-$PIPE_NAME 1 --instance_type $INSTANCE_TYPE`

echo "waiting for shard cluster to be ready"
azure-cluster ready $SHARD_CLUSTER

echo "deploying shard bundle"
azure-cluster deploy $SHARD_BUNDLE_FILE $SHARD_CLUSTER

echo "waiting for coordinator cluster to be ready"
azure-cluster ready $COORD_CLUSTER

echo "deploying coordinator bundle $COORD_BUNDLE_FILE"
if [[ "$STOP_AFTER_DONE" == 1 ]]; then
  CLEANUP="--cleanup $SHARD_CLUSTER --cleanup $COORD_CLUSTER"
fi
azure-cluster deploy $COORD_BUNDLE_FILE $COORD_CLUSTER $CLEANUP

echo "done!"
rm -rf $TMP_DIR

echo "coordinator cluster name: $COORD_CLUSTER"
echo "coordinator IP:"
echo `azure-cluster ips $COORD_CLUSTER`

echo "shard cluster name: $SHARD_CLUSTER"
echo "shard IPs:"
echo `azure-cluster ips $SHARD_CLUSTER`
