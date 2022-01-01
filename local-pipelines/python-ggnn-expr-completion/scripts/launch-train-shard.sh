#!/usr/bin/env bash

set -e

if [[ `uname` != "Linux" ]]; then
   echo "launch-train-shard.sh needs to be started from Linux"
   exit 1
fi

RUNDB_ROOT=$1
PACKAGELIST_FILE=$2
SHARD_NAME=$3

if [[ -z $RUNDB_ROOT ]] || [[ -z $PACKAGELIST_FILE ]] || [[ -z $SHARD_NAME ]]; then
    echo "usage: launch-train-shard.sh RUNDB_ROOT PACKAGELIST_FILE SHARD_NAME"
    exit 1
fi

echo "Params:"
echo "- RUNDB_ROOT: $RUNDB_ROOT"
echo "- PACKAGELIST_FILE: $PACKAGELIST_FILE"
echo "- SHARD_NAME: $SHARD_NAME"
echo
echo "installing rundb"
go install github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb

EXPR_RUNDB_ROOT=$RUNDB_ROOT
echo "using $EXPR_RUNDB_ROOT"

CLUSTER_PREFIX=weekly-$SHARD_NAME

METAINFO_INSTANCE_TYPE=Standard_D4_v2
METAINFO_INSTANCE_COUNT=1

DATAGEN_INSTANCE_TYPE=Standard_D4_v2
DATAGEN_INSTANCE_COUNT=10

TRAIN_INSTANCE_TYPE=Standard_NV6
TRAIN_INSTANCE_COUNT=1

TMP_DIR=`mktemp -d`

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

METAINFO_BUNDLE_FILE=$TMP_DIR/metainfo-bundle.tar.gz
echo "creating metainfo bundle at $METAINFO_BUNDLE_FILE"
cat << EOF > $TMP_DIR/metainfo-run.sh
sudo apt-get install -y make

echo "running graph-data server in background"
echo "graph-data-server logs go to /var/kite/log/graph-data-server.log"
graph-data-server >> /var/kite/log/graph-data-server.log 2>&1 &

echo "waiting for graph server to start..."
while true; do
    if curl http://localhost:3039/some_bogus_page 2>&1 | grep -q '404 page not found'
    then
        break
    fi
    sleep 10
done

cd /var/kite/bundle/kiteco/local-pipelines/python-ggnn-expr-completion
make -f Makefile.cluster PACKAGES=$PACKAGELIST_FILE metainfo_on_cluster
rundb add-artifact $EXPR_RUNDB_ROOT out/metainfo.json $SHARD_NAME/metainfo.json
rundb add-artifact $EXPR_RUNDB_ROOT out/metainfo-inference.json $SHARD_NAME/metainfo-inference.json
rundb add-artifact $EXPR_RUNDB_ROOT $PACKAGELIST_FILE $SHARD_NAME/packagelist.txt
EOF

azure-cluster bundle $METAINFO_BUNDLE_FILE $TMP_DIR/metainfo-run.sh \
    --go-binary github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb \
    --go-binary github.com/kiteco/kiteco/kite-go/lang/python/cmds/graph-data-server \
	--go-binary github.com/kiteco/kiteco/local-pipelines/python-ggnn-expr-completion/traindata \
	--go-binary github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr/cmds/convert \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/Makefile.cluster \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/packagelist.txt \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/packagelist-cluster1.txt \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/packagelist-cluster2.txt

METAINFO_CLUSTER=`azure-cluster start $CLUSTER_PREFIX-metainfo $METAINFO_INSTANCE_COUNT --instance_type $METAINFO_INSTANCE_TYPE`
azure-cluster ready $METAINFO_CLUSTER
azure-cluster deploy $METAINFO_BUNDLE_FILE $METAINFO_CLUSTER

echo "waiting for metainfo.json to be completed..."
rundb wait-artifact $EXPR_RUNDB_ROOT $SHARD_NAME/metainfo.json
rundb wait-artifact $EXPR_RUNDB_ROOT $SHARD_NAME/packagelist.txt

echo "shutting down metainfo cluster $METAINFO_CLUSTER"
azure-cluster stop $METAINFO_CLUSTER

DATAGEN_BUNDLE_FILE=$TMP_DIR/datagen-bundle.tar.gz
echo "creating datagen bundle at $DATAGEN_BUNDLE_FILE"
cat << EOF > $TMP_DIR/datagen-run.sh
sudo apt-get install -y make

echo "running graph-data server in background"
echo "graph-data-server logs go to /var/kite/log/graph-data-server.log"
graph-data-server >> /var/kite/log/graph-data-server.log 2>&1 &

echo "waiting for graph server to start..."
while true; do
    if curl http://localhost:3039/some_bogus_page 2>&1 | grep -q '404 page not found'
    then
        break
    fi
    sleep 10
done

cd /var/kite/bundle/kiteco/local-pipelines/python-ggnn-expr-completion

echo "downloading metainfo.json"
rundb get-artifact $EXPR_RUNDB_ROOT $SHARD_NAME/metainfo.json out/metainfo.json

make -f Makefile.cluster datagen_on_cluster
EOF

azure-cluster bundle $DATAGEN_BUNDLE_FILE $TMP_DIR/datagen-run.sh --kite-ml \
    --go-binary github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb \
    --go-binary github.com/kiteco/kiteco/kite-go/lang/python/cmds/graph-data-server \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/datagen/get_data.py \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/Makefile.cluster

DATAGEN_CLUSTER=`azure-cluster start $CLUSTER_PREFIX-datagen $DATAGEN_INSTANCE_COUNT --instance_type $DATAGEN_INSTANCE_TYPE`
azure-cluster ready $DATAGEN_CLUSTER
azure-cluster deploy $DATAGEN_BUNDLE_FILE $DATAGEN_CLUSTER

DATAGEN_HOSTS=`azure-cluster ips $DATAGEN_CLUSTER | tr '\n' ' '`

TRAIN_BUNDLE_FILE=$TMP_DIR/train-bundle.tar.gz
echo "creating train bundle at $TRAIN_BUNDLE_FILE"

echo "export DATAGEN_HOSTS=\"$DATAGEN_HOSTS\"" > $TMP_DIR/train-run.sh
cat << EOF >> $TMP_DIR/train-run.sh
sudo apt-get install -y make

pip uninstall tensorflow
pip install tensorflow-gpu==1.8.0
pip install tensorboard==1.8.0

cd /var/kite/bundle/kiteco/local-pipelines/python-ggnn-expr-completion

echo "downloading metainfo.json"
rundb get-artifact $EXPR_RUNDB_ROOT $SHARD_NAME/metainfo.json out/metainfo.json
rundb get-artifact $EXPR_RUNDB_ROOT $SHARD_NAME/metainfo-inference.json out/metainfo-inference.json

echo "running sync-data in background; logs go to /var/kite/log/sync-data.log"
make -f Makefile.cluster sync_on_cluster >> /var/kite/log/sync-data.log 2>&1 &

echo "running train script; logs go to /var/kite/log/train.log"
make -f Makefile.cluster train_on_cluster >> /var/kite/log/train.log 2>&1
make -f Makefile.cluster RUNDB=$EXPR_RUNDB_ROOT/$SHARD_NAME validate_attribute >> /var/kite/log/validate-attribute.log 2>&1
make -f Makefile.cluster RUNDB=$EXPR_RUNDB_ROOT/$SHARD_NAME validate_calls >> /var/kite/log/validate-calls.log 2>&1

rundb add-artifact $EXPR_RUNDB_ROOT out $SHARD_NAME --recursive
echo "$SHARD_NAME done" >> DONE
rundb add-artifact $EXPR_RUNDB_ROOT DONE $SHARD_NAME/DONE
EOF

azure-cluster bundle $TRAIN_BUNDLE_FILE $TMP_DIR/train-run.sh --kite-ml --cuda \
    --go-binary github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb \
    --go-binary github.com/kiteco/kiteco/kite-go/traindata/cmds/sync-data \
    --go-binary github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/ggnn-attribute-completions \
	--go-binary github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/ggnn-call-completions-validate \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/train/train.py \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/codebook/compress_embeddings.py \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/attr-fallback-hardcoded.txt \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/Makefile.cluster \

TRAIN_CLUSTER=`azure-cluster start $CLUSTER_PREFIX-train $TRAIN_INSTANCE_COUNT --instance_type $TRAIN_INSTANCE_TYPE`
azure-cluster ready $TRAIN_CLUSTER
azure-cluster deploy $TRAIN_BUNDLE_FILE $TRAIN_CLUSTER

rm -rf $TMP_DIR

echo "started!"
echo
echo "=== TO RUN TENSORBOARD ==="
echo "  1) ssh $TRAIN_HOST"
echo "  2) source /var/kite/bundle/env.sh"
echo "  3) source /var/kite/bundle/kiteco/kite-python/kite_ml/venv/bin/activate"
echo "  4) tensorboard --logdir=/var/kite/bundle/kiteco/local-pipelines/python-ggnn-expr-completion/tensorboard"
echo "  5) Enjoy!"
echo
echo "waiting on expr pipeline to complete..."
rundb wait-artifact $EXPR_RUNDB_ROOT $SHARD_NAME/DONE

echo "stopping datagen cluster $DATAGEN_CLUSTER"
azure-cluster stop $DATAGEN_CLUSTER

echo "stopping train cluster $TRAIN_CLUSTER"
azure-cluster stop $TRAIN_CLUSTER