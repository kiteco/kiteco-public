#!/bin/bash
set -e

# this script is intended to be run from local-pipelines/lexical/train/scripts

cd ..
# Needs to create manually the folders (error when passing multiple values for DIRS_TO_CREATE
mkdir out_dry_run
mkdir data_dry_run
mkdir -p tmp_dry_run

go build -o tmp_dry_run/rundb github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb
go build -o tmp_dry_run/datagen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/datagen

CONFIG="LANG=javascript VOCAB_SIZE=500 NUM_LAYERS=1 CONTEXT_SIZE=512 BATCH=5 STEPS=30"
MAKEFILE_OVERRIDE="OUT_DIR=out_dry_run TMP_DIR=tmp_dry_run DATA_DIR=data_dry_run GLOBAL_DATA_DIR=data_dry_run DIRS_TO_CREATE=tmp_dry_run"
PATH=$PATH:`pwd`/tmp_dry_run

make -f Makefile.docker $CONFIG $MAKEFILE_OVERRIDE fetch_vocab
make -f Makefile.docker $CONFIG $MAKEFILE_OVERRIDE datagen_on_cluster
tar -cvzf testdata/dry_train_data.tar.gz data_dry_run/train/* data_dry_run/validate/* out_dry_run/ident-vocab-entries.bpe


rm -rf tmp_dry_run
rm -rf data_dry_run
rm -rf out_dry_run