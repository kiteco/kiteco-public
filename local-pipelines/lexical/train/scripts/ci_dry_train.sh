#!/bin/bash
set -e

#####################################################
#
# Dry run of lexical model training
# This script execute a really short training of the lexical model
# It's executed on Travis after any changes of a python file touching the model and training process
#
# Files required to run this test can be updated with the script build_data_for_dry_train.sh
#
####################################################


# This script is intended to be executed from kiteco root directory (as it is done in Travis)
cd local-pipelines/lexical/train
TRAIN_DIR=`pwd`

mkdir -p bin_dry_run
go build -o bin_dry_run/configgen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/configgen
go build -o bin_dry_run/searchconfiggen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/searchconfiggen

python3 -m venv venv_dry_run
source venv_dry_run/bin/activate
pip install --upgrade pip
pip install tensorflow==1.15.2
pip install scipy
cd ../../../kite-python/kite_ml/
pip install -e .
cd $TRAIN_DIR

echo "running train dry run"
rm -rf out_dry_run

CONFIG="LANG=javascript VOCAB_SIZE=500 NUM_LAYERS=1 CONTEXT_SIZE=64 EMBEDDING_SIZE=64 BATCH=2 STEPS=10 NUM_HEADS=4 MODEL_TYPE=lexical"
MAKEFILE_OVERRIDE="DOCKER_MNT=. TENSORBOARD_DIR=/tmp_dry_run/tensorboard OUT_DIR=out_dry_run TMP_DIR=tmp_dry_run DATA_DIR=data_dry_run GLOBAL_DATA_DIR=data_dry_run DOCKER_CMD_PREFIX=time HOROVOD_CMD_PREFIX=time DIRS_TO_CREATE=tmp_dry_run"

PATH=$PATH:`pwd`/bin_dry_run

tar -xvzf testdata/dry_train_data.tar.gz
./bin_dry_run/searchconfiggen -lang=javascript -out=out_dry_run/searchconfig.json
make -f Makefile.docker $CONFIG $MAKEFILE_OVERRIDE train_on_cluster
make -f Makefile.docker $CONFIG $MAKEFILE_OVERRIDE generate_local_model
make -f Makefile.docker $CONFIG $MAKEFILE_OVERRIDE generate_tfserving_model

rm -rf data_dry_run
rm -rf out_dry_run
rm -rf tmp_dry_run
rm -rf venv_dry_run
rm -rf bin_dry_run
