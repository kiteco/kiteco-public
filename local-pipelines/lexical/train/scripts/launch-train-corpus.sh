#!/usr/bin/env bash

set -e

PREFIX=$1
RESUME_FROM=$2
RESUME_STEPS=$3

if [[ -z $PREFIX ]]; then
    echo "usage: launch-train-corpus.sh PREFIX <RESUME_FROM> <RESUME_STEPS>"
    exit 1
fi

if [[ -z $RESUME_STEPS ]]; then
    RESUME_STEPS=0
else
    echo "using resume steps $RESUME_STEPS"
fi

echo "installing azure-cluster"
go install github.com/kiteco/kiteco/kite-go/cmds/azure-cluster

echo "installing rundb"
go install github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb

ROOT=`rundb create s3://kite-data/run-db lexical-model-experiments`
echo "using rundb root of $ROOT"

TMP_DIR=`mktemp -d`

CLUSTER_PREFIX=lexical-$PREFIX
INSTANCE_COUNT=1
INSTANCE_TYPE=Standard_NV6

BUNDLE_FILE=${TMP_DIR}/bundle.tar.gz
echo "creating bundle at $BUNDLE_FILE"

echo "export ROOT=\"$ROOT\"" > $TMP_DIR/bundle-run.sh
echo "export RESUME_FROM=\"$RESUME_FROM\"" >> $TMP_DIR/bundle-run.sh
echo "export RESUME_STEPS=$RESUME_STEPS" >> $TMP_DIR/bundle-run.sh
cat << 'EOF' >> ${TMP_DIR}/bundle-run.sh
# sudo apt-get install -y make
pip install slack-webhook-cli

echo "running train script; logs go to /var/kite/log/"
cd /var/kite/bundle/kiteco/local-pipelines/lexical/train

# Slack webhook pointing to PipelineBot on #lexical-completions-ml channel
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/XXXXXXX/XXXXXXX/XXXXXXX
export KITE_USE_AZURE_MIRROR=0

source /var/kite/bundle/env.sh

mkdir -p logs
# tensorboard --logdir=out/tensorboard &> tensorboard.log &

## Use this for multiple runs
CONFIG1="LANG=text__python-go-javascript-jsx-vue-css-html-less-typescript-tsx-java-scala-kotlin-c-cpp-objectivec-csharp-php-ruby-bash VOCAB_SIZE=20000 NUM_HEADS=12 NUM_LAYERS=12 EMBEDDING_SIZE=720 CONTEXT_SIZE=512 BATCH=40 STEPS=50000 MODEL_TYPE=lexical"

## If RESUME_FROM is set, invoke a slightly different set of make commands
## TODO(tarak): Try to consolidate this later?
if [ ! -z $RESUME_FROM ]; then
    for i in `seq 1 1`; do
        x="CONFIG$i"
        rm -rf logs/*
        echo "${!x}" > logs/config
        make -f Makefile.docker ${!x} RUNDB=$ROOT MSG="starting training run; resuming from $RESUME_FROM" slack
        make -f Makefile.docker ${!x} clean
        make -f Makefile.docker ${!x} RESUME_FROM=$RESUME_FROM RESUME_STEPS=$RESUME_STEPS configure_resume &> logs/configure_resume.log
        make -f Makefile.docker ${!x} resume_datagen_on_cluster &> logs/resume_datagen.log &
        make -f Makefile.docker ${!x} resume_train_on_cluster &> logs/resume_train.log
        make -f Makefile.docker ${!x} generate_local_model &> logs/generate_local_model.log
        make -f Makefile.docker ${!x} searchconfiggen_on_cluster &> logs/searchconfiggen.log
        make -f Makefile.docker ${!x} test_on_cluster &> logs/test.log
        make -f Makefile.docker ${!x} RUNDB=$ROOT upload &> upload-$i.log
        make -f Makefile.docker ${!x} RUNDB=$ROOT MSG="completed training run" slack
        killall datagen
    done;
    exit 0
fi


for i in `seq 1 1`; do
    x="CONFIG$i"
    rm -rf logs/*
    echo "${!x}" > logs/config
    make -f Makefile.docker ${!x} RUNDB=$ROOT MSG="starting training run" slack
    make -f Makefile.docker ${!x} clean
    make -f Makefile.docker ${!x} fetch_vocab &> logs/fetch_vocab.log
    make -f Makefile.docker ${!x} datagen_on_cluster &> logs/datagen.log &
    make -f Makefile.docker ${!x} train_on_cluster &> logs/train.log
    make -f Makefile.docker ${!x} generate_local_model &> logs/generate_local_model.log
    make -f Makefile.docker ${!x} searchconfiggen_on_cluster &> logs/searchconfiggen.log
    make -f Makefile.docker ${!x} test_on_cluster &> logs/test.log
    make -f Makefile.docker ${!x} RUNDB=$ROOT upload &> upload-$i.log
    make -f Makefile.docker ${!x} RUNDB=$ROOT MSG="completed training run" slack
    killall datagen
done;
EOF

azure-cluster bundle ${BUNDLE_FILE} ${TMP_DIR}/bundle-run.sh --kite-ml --docker-ml \
    --go-binary github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/datagen \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/resume-training \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/configgen \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/searchconfiggen \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/minp \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/calibration \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/calibrate-temperature-scaling/traindata_temperature_scaling \
    --go-binary github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/performance \
    --kiteco-path kite-golib \
    --kiteco-path kite-go \
    --kiteco-path sidebar \
    --kiteco-path kite-answers \
    --kiteco-path kite-python \
    --kiteco-path local-pipelines/lexical/train/model \
    --kiteco-path local-pipelines/lexical/train/train.py \
    --kiteco-path local-pipelines/lexical/train/local_model.py \
    --kiteco-path local-pipelines/lexical/train/tfserve_model.py \
    --kiteco-path local-pipelines/lexical/train/tfserve_warmup_assets.py \
    --kiteco-path local-pipelines/lexical/train/model_from_checkpoint.py \
    --kiteco-path local-pipelines/lexical/train/cmds/calibrate-temperature-scaling/train/train_temperature_scaling.py \
    --kiteco-path local-pipelines/lexical/train/Makefile.docker \
    --kiteco-path local-pipelines/lexical/train/scripts/datagen.sh

# CLUSTER=`azure-cluster start ${CLUSTER_PREFIX} ${INSTANCE_COUNT} --instance_type ${INSTANCE_TYPE} --bionic`
# azure-cluster ready ${CLUSTER}
# azure-cluster deploy ${BUNDLE_FILE} ${CLUSTER}

# echo "started!"

# echo "cluster name: $CLUSTER"
# CLUSTER_HOST=`azure-cluster ips ${CLUSTER}`
# echo "cluster IP: $CLUSTER_HOST"
# echo "see tensorboard at http://$CLUSTER_HOST:6006/ when it starts up"
