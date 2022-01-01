#!/usr/bin/env bash

set -e

if [[ `uname` != "Linux" ]]; then
   echo "launch-pipeline.sh needs to be started from Linux"
   exit 1
fi

if [[ -z $1 ]]; then
  echo "usage: launch-pipeline.sh PIPELINE_CMD_PACKAGE [PIPELINE_ARGS]"
  exit 1
fi

PIPE_PKG=$1
PIPE_ARGS="${@:2}"

if [[ -z "${INSTANCE_TYPE}" ]]; then
    INSTANCE_TYPE=standard_d8s_v3
fi

PIPE_CMD=`basename $PIPE_PKG`
CLUSTER_PREFIX=$PIPE_CMD
STOP_AFTER_DONE=0

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

TMP_DIR=`mktemp -d`
BUNDLE_FILE=$TMP_DIR/bundle.tar.gz
echo "creating bundle at $BUNDLE_FILE"
echo "$PIPE_NAME $PIPE_ARGS" > $TMP_DIR/run.sh
azure-cluster bundle $BUNDLE_FILE $TMP_DIR/run.sh --go-binary $PIPE_PKG

echo "starting cluster: $CLUSTER_NAME"
# the instance will be automatically stopped when the pipeline successfully runs
CLUSTER_NAME=`azure-cluster start $CLUSTER_PREFIX 1 --instance_type $INSTANCE_TYPE`

echo "waiting for cluster to be ready"
azure-cluster ready $CLUSTER_NAME

echo "Deploying bundle"
if [[ "$STOP_AFTER_DONE" == 1 ]]; then
  CLEANUP="--cleanup $CLUSTER_NAME"
fi
azure-cluster deploy $BUNDLE_FILE $CLUSTER_NAME $CLEANUP

echo "done!"
rm -rf $TMP_DIR

echo "cluster name: $CLUSTER_NAME"
echo "IP address of instance:"
echo `azure-cluster ips $CLUSTER_NAME`