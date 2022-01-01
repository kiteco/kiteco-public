#!/bin/bash
set -e

#####################################################
#
# Dry run of lexical model searchconfiggen
# This script gets the model from datadeps and then generate a searchconfig for it
# Then it compares the result with the searchconfig in datadeps
#
####################################################


# This script is intended to be executed from kiteco root directory (as it is done in Travis)
KITECO_ROOT=`pwd`



cd local-pipelines/lexical/train
TRAIN_DIR=`pwd`

rm -rf data_dry_run/
rm -rf out_dry_run/
rm -rf tmp_dry_run/
rm -rf venv_dry_run/
rm -rf bin_dry_run/

mkdir -p bin_dry_run
go build -o bin_dry_run/searchconfiggen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/searchconfiggen
go build -o bin_dry_run/calibration github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/calibration
go build -o bin_dry_run/traindata_temperature_scaling github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/calibrate-temperature-scaling/traindata_temperature_scaling
go build -o bin_dry_run/minp github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/minp
go build -o bin_dry_run/datadeps-extractor github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/datadeps-extractor
go build -o bin_dry_run/configgen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/configgen
go build -o bin_dry_run/searchconfigcomparator github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/searchconfigcomparator

mkdir -p tmp_dry_run
python3 -m venv venv_dry_run >> tmp_dry_run/log_python.txt
source venv_dry_run/bin/activate
pip install --upgrade pip >> tmp_dry_run/log_python.txt
pip install --upgrade setuptools >> tmp_dry_run/log_python.txt
pip install tensorflow==1.15.2 >> tmp_dry_run/log_python.txt
pip install scipy >> tmp_dry_run/log_python.txt
cd ../../../kite-python/kite_ml/
pip install -e . >> $TRAIN_DIR/tmp_dry_run/log_python.txt

cd $TRAIN_DIR

OLD_S3_CACHE=$KITE_S3CACHE
mkdir -p tmp_dry_run
mkdir -p tmp_dry_run/s3_cache
export KITE_S3CACHE=`pwd`/tmp_dry_run/s3_cache


echo "running searchconfiggen and compare result to baseline"

# The vocab size needs to be the same than the one used for the training 
CONFIG="LANG=javascript VOCAB_SIZE=20000 NUM_LAYERS=1 CONTEXT_SIZE=512 BATCH=5 STEPS=30"
MAKEFILE_OVERRIDE="KITECO_ROOT=$KITECO_ROOT DOCKER_MNT=. TENSORBOARD_DIR=/tmp_dry_run/tensorboard OUT_DIR=out_dry_run TMP_DIR=tmp_dry_run DATA_DIR=data_dry_run GLOBAL_DATA_DIR=data_dry_run DOCKER_CMD_PREFIX=time DIRS_TO_CREATE=tmp_dry_run"

PATH=$PATH:`pwd`/bin_dry_run
mkdir -p out_dry_run
datadeps-extractor --outputpath=out_dry_run >> tmp_dry_run/log_searchconfiggen.txt 2>&1
make -f Makefile.docker $CONFIG $MAKEFILE_OVERRIDE searchconfiggen_on_cluster >> tmp_dry_run/log_searchconfiggen.txt 2>&1

searchconfigcomparator --outputpath=out_dry_run

KITE_S3CACHE=$OLD_S3_CACHE
rm -rf data_dry_run/
rm -rf out_dry_run/
rm -rf tmp_dry_run/
rm -rf venv_dry_run/
rm -rf bin_dry_run/
