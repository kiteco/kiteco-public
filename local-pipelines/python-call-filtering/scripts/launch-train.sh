#!/usr/bin/env bash

set -e

if [[ `uname` != "Linux" ]]; then
   echo "launch-train.sh needs to be started from Linux"
   exit 1
fi

RUNDB_ROOT=$1

if [[ -z $RUNDB_ROOT ]]; then
    echo "usage: launch-train-shard.sh RUNDB_ROOT"
    exit 1
fi

echo "Params:"
echo "- RUNDB_ROOT: $RUNDB_ROOT"
echo
echo "installing rundb"
go install github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb

INSTANCE_COUNT=1
INSTANCE_TYPE=Standard_D4_v2
CLUSTER_PREFIX=weekly-call-filtering

TMP_DIR=`mktemp -d`

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

BUNDLE_FILE=$TMP_DIR/call-filtering-bundle.tar.gz
echo "creating datagen bundle at $BUNDLE_FILE"

cat << EOF > $TMP_DIR/call-filtering-run.sh
sudo apt-get install -y make
cd /var/kite/bundle/kiteco/local-pipelines/python-call-filtering

rundb get-artifact $RUNDB_ROOT shards.json shards.json

make -f Makefile.cluster SHARDS_FILE=shards.json RUNDB=$RUNDBD_ROOT/call-filtering traindata
make -f Makefile.cluster SHARDS_FILE=shards.json train
make -f Makefile.cluster SHARDS_FILE=shards.json gtdata
make -f Makefile.cluster SHARDS_FILE=shards.json threshold

rundb add-artifact $RUNDB_ROOT out call-filtering --recursive
echo "call-filtering done" >> DONE
rundb add-artifact $RUNDB_ROOT DONE call-filtering/DONE
EOF

azure-cluster bundle $BUNDLE_FILE $TMP_DIR/call-filtering-run.sh --kite-ml \
    --go-binary github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb \
    --go-binary github.com/kiteco/kiteco/local-pipelines/python-call-filtering/python-call-prob/traindata \
    --go-binary github.com/kiteco/kiteco/local-pipelines/python-call-filtering/gtdata \
    --go-binary github.com/kiteco/kiteco/local-pipelines/python-call-filtering/threshold \
    --kiteco-path local-pipelines/python-call-filtering/python-call-prob/train/train.py \
    --kiteco-path local-pipelines/python-call-filtering/Makefile.cluster

CLUSTER=`azure-cluster start $CLUSTER_PREFIX $INSTANCE_COUNT --instance_type $INSTANCE_TYPE`
azure-cluster ready $CLUSTER
azure-cluster deploy $BUNDLE_FILE $CLUSTER

rundb wait-artifact $RUNDB_ROOT call-filtering/DONE

#azure-cluster stop $CLUSTER

rm -rf $TMP_DIR