#!/usr/bin/env bash

set -e

NAME=$1

if [[ -z $NAME ]]; then
    echo "usage: launch-vocabgen-corpus.sh NAME"
    exit 1
fi

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

echo "installing rundb"
go install github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb

ROOT=`rundb create s3://kite-data/run-db summarize-vocabgen-$NAME`
echo "using rundb root of $ROOT"

TMP_DIR=`mktemp -d`

CLUSTER_PREFIX=summarize-vocabgen-$NAME
INSTANCE_COUNT=1
INSTANCE_TYPE=Standard_E16_V3

BUNDLE_FILE=${TMP_DIR}/bundle.tar.gz
echo "creating bundle at $BUNDLE_FILE"

echo "export ROOT=\"$ROOT\"" > $TMP_DIR/bundle-run.sh
cat << 'EOF' >> ${TMP_DIR}/bundle-run.sh
sudo apt-get install -y make

source /var/kite/bundle/env.sh

echo "running train script; logs go to /var/kite/log/"
cd /var/kite/bundle/kiteco/local-pipelines/summarize

export KITE_USE_AZURE_MIRROR=0

mkdir -p logs
make -f Makefile.vocabgen wordcount &> logs/wordcount.log
make -f Makefile.vocabgen vocabgen &> logs/vocabgen.log
make -f Makefile.vocabgen RUNDB=$ROOT upload &> upload.log
EOF

azure-cluster bundle ${BUNDLE_FILE} ${TMP_DIR}/bundle-run.sh \
    --go-binary github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb \
    --go-binary github.com/kiteco/kiteco/kite-go/summarize/cmds/vocab \
    --kiteco-path local-pipelines/summarize/Makefile.vocabgen

CLUSTER=`azure-cluster start ${CLUSTER_PREFIX} ${INSTANCE_COUNT} --instance_type ${INSTANCE_TYPE}`
azure-cluster ready ${CLUSTER}
azure-cluster deploy ${BUNDLE_FILE} ${CLUSTER}

echo "started!"

echo "cluster name: $CLUSTER"
CLUSTER_HOST=`azure-cluster ips ${CLUSTER}`
echo "cluster IP: $CLUSTER_HOST"