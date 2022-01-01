#!/usr/bin/env bash

set -e

CLUSTERS_FILE=$1
CLUSTER_PREFIX=$2

if [[ -z $CLUSTERS_FILE ]] || [[ -z $CLUSTER_PREFIX ]]; then
    echo "usage: launch-train.sh CLUSTERS_FILE CLUSTER_PREFIX"
    exit 1
fi

if [[ `uname` != "Linux" ]]; then
   echo "launch-train.sh needs to be started from Linux"
   exit 1
fi

if [[ -e $CLUSTERS_FILE ]]; then
    echo "$CLUSTERS_FILE already exists; do you have a cluster running?"
    exit 1
fi

DATAGEN_INSTANCE_TYPE=Standard_D4_v2
DATAGEN_INSTANCE_COUNT=8

TRAIN_INSTANCE_TYPE=Standard_NV6

TMP_DIR=`mktemp -d`

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

DATAGEN_BUNDLE_FILE=$TMP_DIR/datagen-bundle.tar.gz
echo "creating datagen bundle at $DATAGEN_BUNDLE_FILE"
cat << 'EOF' > $TMP_DIR/datagen-run.sh
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
make datagen_on_cluster
EOF

azure-cluster bundle $DATAGEN_BUNDLE_FILE $TMP_DIR/datagen-run.sh --kite-ml \
    --go-binary github.com/kiteco/kiteco/kite-go/lang/python/cmds/graph-data-server \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/datagen/get_data.py \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/Makefile \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/out/metainfo.json

echo "starting datagen cluster"
DATAGEN_CLUSTER=`azure-cluster start $CLUSTER_PREFIX-datagen $DATAGEN_INSTANCE_COUNT --instance_type $DATAGEN_INSTANCE_TYPE`

echo "waiting for datagen cluster to be ready"
azure-cluster ready $DATAGEN_CLUSTER

echo "deploying datagen bundle"
azure-cluster deploy $DATAGEN_BUNDLE_FILE $DATAGEN_CLUSTER

DATAGEN_HOSTS=`azure-cluster ips $DATAGEN_CLUSTER | tr '\n' ' '`
echo "datagen hosts: $DATAGEN_HOSTS"

TRAIN_BUNDLE_FILE=$TMP_DIR/train-bundle.tar.gz
echo "creating train bundle at $TRAIN_BUNDLE_FILE"

echo "export DATAGEN_HOSTS=\"$DATAGEN_HOSTS\"" > $TMP_DIR/train-run.sh
cat << 'EOF' >> $TMP_DIR/train-run.sh
sudo apt-get install -y make

pip uninstall tensorflow
pip install tensorflow-gpu==1.8.0
pip install tensorboard==1.8.0

echo "running sync-data in background; logs go to /var/kite/log/sync-data.log"
cd /var/kite/bundle/kiteco/local-pipelines/python-ggnn-expr-completion
make sync_on_cluster >> /var/kite/log/sync-data.log 2>&1 &

echo "running train script; logs go to /var/kite/log/train.log"
make train_on_cluster >> /var/kite/log/train.log 2>&1
EOF

azure-cluster bundle $TRAIN_BUNDLE_FILE $TMP_DIR/train-run.sh --kite-ml --cuda \
    --go-binary github.com/kiteco/kiteco/kite-go/traindata/cmds/sync-data \
    --go-binary github.com/kiteco/kiteco/kite-go/cmds/kfsput \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/train/train.py \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/codebook/compress_embeddings.py \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/Makefile \
    --kiteco-path local-pipelines/python-ggnn-expr-completion/out/metainfo.json

echo "starting train cluster"
TRAIN_CLUSTER=`azure-cluster start $CLUSTER_PREFIX-train 1 --instance_type $TRAIN_INSTANCE_TYPE`

echo "waiting for train cluster to be ready"
azure-cluster ready $TRAIN_CLUSTER

echo "deploying train bundle"
azure-cluster deploy $TRAIN_BUNDLE_FILE $TRAIN_CLUSTER



echo $DATAGEN_CLUSTER >> $CLUSTERS_FILE
echo $TRAIN_CLUSTER >> $CLUSTERS_FILE

rm -rf $TMP_DIR

echo "done!"
echo "datagen cluster name: $DATAGEN_CLUSTER"
echo "datagen IPs:"
echo `azure-cluster ips $DATAGEN_CLUSTER`
echo "train cluster name: $TRAIN_CLUSTER"
TRAIN_HOST=`azure-cluster ips $TRAIN_CLUSTER`
echo "train IP: $TRAIN_HOST"
echo "train IP: $TRAIN_HOST"
echo "to run tensorboard:"
echo "  1) ssh $TRAIN_HOST"
echo "  2) source /var/kite/bundle/env.sh"
echo "  3) source /var/kite/bundle/kiteco/kite-python/kite_ml/venv/bin/activate"
echo "  4) tensorboard --logdir=/var/kite/bundle/kiteco/local-pipelines/python-ggnn-expr-completion/tensorboard"
echo "In short:" 
echo "  ssh $TRAIN_HOST && source /var/kite/bundle/env.sh && source /var/kite/bundle/kiteco/kite-python/kite_ml/venv/bin/activate && tensorboard --logdir=/var/kite/bundle/kiteco/local-pipelines/python-ggnn-expr-completion/tensorboard"
