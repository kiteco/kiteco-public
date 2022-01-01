# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/

MODEL_NAME := node_depth_300_sum_rule_context_with_context_tokens_new_name_decoder_all_packages_less_tokens_expansion_graph_requests
DATASET_NAME ?= $(MODEL_NAME)
RUNDB := s3://kite-data/run-db

TENSORBOARD_DIR := ./tensorboard
OUT_DIR := ./out
OUT_SERVE_DIR := ./out/serve
TMP_DIR := ./tmp
BIN_DIR := ./bin
SAMPLES_DIR ?= /data/samples
DATA_DIR := $(SAMPLES_DIR)/$(DATASET_NAME)
CLUSTERS_FILE := clusters-$(MODEL_NAME).txt

DIRS_TO_CREATE := $(OUT_DIR) $(OUT_SERVE_DIR) $(TMP_DIR) $(BIN_DIR) $(TENSORBOARD_DIR) $(SAMPLES_DIR) $(DATA_DIR)

KFSPUT_BIN := $(BIN_DIR)/kfsput
SYNC_TRAINDATA_BIN := $(BIN_DIR)/sync

BUNDLE_NAME := bundle

UNCURATED_EXAMPLES_DIR := uncurated-examples
CURATED_EXAMPLES_DIR := curated-examples
TENSORBOARD_PATH := $(TENSORBOARD_DIR)/$(MODEL_NAME)
TENSORBOARD_PATH_CONTINUE := $(TENSORBOARD_PATH)_continue
SAVED_MODEL_PATH := saved_model
COMPRESSED_FROZEN_MODEL_PATH := expr_model.frozen.pb
FROZEN_MODEL_PATH := expr_model.uncompressed.frozen.pb
ATTR_FALLBACK_PATH := attr-fallback.txt
CKPT_PATH := ckpt

META_INFO := metainfo.json
META_INFO_FINAL := metainfo-inference.json
KITE_ML_PATH := ../../kite-python/kite_ml/

UPLOAD_PATH := s3://kite-data/python-ggnn-expr-completion

PACKAGES := ./packagelist.txt
BATCH := 20
MAX_SAMPLES := 40
ATTR_PROPORTION := 15
ATTR_BASE_PROPORTION := 5
CALL_PROPORTION := 15
ARG_TYPE_PROPORTION := 15
KWARG_NAME_PROPORTION := 15
ARG_PLACEHOLDER_PROPORTION := 45
SAMPLES_PER_FILE := 500



TRAINDATA_CLUSTER_0 := \
	ml-training-westus2-dev-5 \
	ml-training-westus2-dev-6 \
	ml-training-westus2-dev-7 \
	ml-training-westus2-dev-8 \
	ml-training-westus2-dev-9 \
	ml-training-westus2-dev-10 \
	ml-training-westus2-dev-11 \
	ml-training-westus2-dev-12 \
	ml-training-westus2-dev-13

TRAINDATA_HOSTS ?= $(TRAINDATA_CLUSTER_0)

NUM_VALIDATE_SAMPLES := 1000


ifeq ($(REMOTE),1)
	SERVER ?= http://ml-train.kite.com:3039
else
	SERVER ?= http://localhost:3039
endif

ifeq ($(DRY_RUN),1)
	STEPS := 10
	CODEBOOK_MAX_EPOCHS := 10
else
	STEPS := 1000000
	CODEBOOK_MAX_EPOCHS := 1500
endif

SYNCER_PORT := :3087

ifeq ($(SYNC_DATA),1)
    SYNC_DATA_FLAG := --use_synced_data=1
	SYNCER_ENDPOINT_FLAG := --syncer_endpoint=http://localhost$(SYNCER_PORT)
else
    SYNC_DATA_FLAG :=
	SYNCER_ENDPOINT_FLAG :=
endif

ifeq ($(DATA_GEN_LOCAL), 1)
	DATA_GEN_LOCAL_FLAG := --data_gen_local=1
else
	DATA_GEN_LOCAL_FLAG :=
endif


SCORES_ENDPOINT := $(SERVER)/symbol/scores

$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: validate_attribute

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)
	rm -rf $(TENSORBOARD_PATH)
	rm -rf $(BUNDLE_NAME)
	rm -rf $(BUNDLE_NAME).tar.gz


traindata: $(OUT_DIR)/$(META_INFO)

clean_remote:
	python datagen/clean_remote.py \
		--dir=$(DATA_DIR) \
		--hosts $(TRAINDATA_HOSTS)

clean_remote_all:
	python datagen/clean_remote.py \
		--dir=$(SAMPLES_DIR) \
		--hosts $(TRAINDATA_HOSTS)

CLUSTER_PREFIX := expr

cluster_train: $(OUT_DIR)/$(META_INFO)
	bash ./scripts/launch-train.sh $(TMP_DIR)/$(CLUSTERS_FILE) $(CLUSTER_PREFIX)
	mv $(TMP_DIR)/$(CLUSTERS_FILE) $(OUT_DIR)/$(CLUSTERS_FILE)

stop_cluster:
	bash ./scripts/stop-cluster.sh $(OUT_DIR)/$(CLUSTERS_FILE)

# this is invoked on the cluster instance, see scripts/launch-train.sh
datagen_on_cluster:
	python datagen/get_data.py \
		--endpoint=http://localhost:3039 \
		--out_dir=$(DATA_DIR) \
		--random_seed=`python -c "print(100 + $(INSTANCE_ID))"` \
		--samples=`python -c "import math; print(int(math.ceil(1.1 * $(STEPS) / $(INSTANCE_COUNT))))"` \
		--samples_per_file=$(SAMPLES_PER_FILE) \
		--max_samples=$(MAX_SAMPLES) \
		--batch=$(BATCH) \
		--meta_info=$(OUT_DIR)/$(META_INFO) \
		--attr_proportion=$(ATTR_PROPORTION) \
		--attr_base_proportion=$(ATTR_BASE_PROPORTION) \
		--call_proportion=$(CALL_PROPORTION) \
		--arg_type_proportion=$(ARG_TYPE_PROPORTION) \
		--kwarg_name_proportion=$(KWARG_NAME_PROPORTION) \
		--arg_placeholder_proportion=$(ARG_PLACEHOLDER_PROPORTION)

# this is invoked on the cluster instance, see scripts/launch-train.sh
train_on_cluster:
	python train/train.py \
		--use_synced_data=1 \
		--syncer_endpoint=http://localhost$(SYNCER_PORT) \
		--in_dir=$(DATA_DIR) \
		--meta_info=$(OUT_DIR)/$(META_INFO) \
		--out_dir=$(TMP_DIR)/$(SAVED_MODEL_PATH) \
		--frozen_model=$(TMP_DIR)/$(FROZEN_MODEL_PATH) \
		--checkpoint_path=$(TMP_DIR)/$(CKPT_PATH) \
		--tensorboard=$(TENSORBOARD_PATH) \
		--steps=$(STEPS) \
		--batch=$(BATCH)
	mv $(TMP_DIR)/$(SAVED_MODEL_PATH) $(OUT_DIR)
	mv $(TMP_DIR)/$(FROZEN_MODEL_PATH) $(OUT_DIR)/$(FROZEN_MODEL_PATH)
	cp -rv $(TENSORBOARD_DIR) $(OUT_DIR)
	python codebook/compress_embeddings.py \
    		--meta_info=$(OUT_DIR)/$(META_INFO) \
    		--in_saved_model=$(OUT_DIR)/$(SAVED_MODEL_PATH) \
    		--out_frozen_model=$(TMP_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH) \
    		--models_path=$(TMP_DIR)/codebook_models \
			--max_epochs=$(CODEBOOK_MAX_EPOCHS) \
    		--train=1
	mv $(TMP_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH) $(OUT_SERVE_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH)
	kfsput $(OUT_DIR) $(UPLOAD_PATH)/%t

# this is invoked on the cluster instance, see scripts/launch-train.sh
sync_on_cluster:
	rm -rf $(DATA_DIR)
	mkdir -p $(DATA_DIR)
	sync-data --remotedir=$(DATA_DIR) \
		 --localdir=$(DATA_DIR) \
		 --port $(SYNCER_PORT) \
		 --hosts $(DATAGEN_HOSTS)

datagen_parallel: $(OUT_DIR)/$(META_INFO)
	python datagen/deploy.py \
		--bundle=$(BUNDLE_NAME) \
		--steps=$(STEPS) \
		--batch=$(BATCH) \
		--meta_info=$(OUT_DIR)/$(META_INFO) \
		--max_samples=$(MAX_SAMPLES) \
		--attr_proportion=$(ATTR_PROPORTION) \
		--attr_base_proportion=$(ATTR_BASE_PROPORTION) \
		--call_proportion=$(CALL_PROPORTION) \
		--arg_type_proportion=$(ARG_TYPE_PROPORTION) \
		--kwarg_name_proportion=$(KWARG_NAME_PROPORTION) \
		--arg_placeholder_proportion=$(ARG_PLACEHOLDER_PROPORTION) \
		--samples_per_file=$(SAMPLES_PER_FILE) \
		--kite_ml_path=$(KITE_ML_PATH) \
		--out_dir=$(DATA_DIR) \
		--hosts $(TRAINDATA_HOSTS)

datagen_local: $(OUT_DIR)/$(META_INFO)
	PYTHONPATH=$(KITE_ML_PATH) python datagen/get_data.py \
		--endpoint=$(SERVER) \
		--random_seed=1 \
		--meta_info=$(OUT_DIR)/$(META_INFO) \
		--batch=$(BATCH) \
		--samples=$(STEPS) \
		--max_samples=$(MAX_SAMPLES) \
		--attr_proportion=$(ATTR_PROPORTION) \
		--attr_base_proportion=$(ATTR_BASE_PROPORTION) \
		--call_proportion=$(CALL_PROPORTION) \
		--arg_type_proportion=$(ARG_TYPE_PROPORTION) \
		--kwarg_name_proportion=$(KWARG_NAME_PROPORTION) \
		--arg_placeholder_proportion=$(ARG_PLACEHOLDER_PROPORTION) \
		--out_dir=$(DATA_DIR) \
		--samples_per_file=$(SAMPLES_PER_FILE)


sync: $(SYNC_TRAINDATA_BIN)
	rm -rf $(DATA_DIR)
	mkdir -p $(DATA_DIR)
	$(SYNC_TRAINDATA_BIN) \
		--remotedir=$(DATA_DIR) \
		--localdir=$(DATA_DIR) \
		--port $(SYNCER_PORT) \
		--hosts $(TRAINDATA_HOSTS)


$(SYNC_TRAINDATA_BIN):
	go build -o $(SYNC_TRAINDATA_BIN) github.com/kiteco/kiteco/kite-go/traindata/cmds/sync-data


train: $(OUT_DIR)/$(FROZEN_MODEL_PATH)

$(OUT_DIR)/$(META_INFO):
	@echo "===== Making training info ..."
	go run github.com/kiteco/kiteco/local-pipelines/python-ggnn-expr-completion/traindata \
		--packages=$(PACKAGES) --out=$(TMP_DIR)/$(META_INFO) --endpoint=$(SCORES_ENDPOINT)
	mv $(TMP_DIR)/$(META_INFO) $(OUT_DIR)/$(META_INFO)

compress: $(OUT_SERVE_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH)

$(OUT_SERVE_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH): $(OUT_DIR)/$(FROZEN_MODEL_PATH)
	python codebook/compress_embeddings.py \
		--meta_info=$(OUT_DIR)/$(META_INFO) \
		--in_saved_model=$(OUT_DIR)/$(SAVED_MODEL_PATH) \
		--out_frozen_model=$(TMP_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH) \
		--models_path=$(TMP_DIR)/codebook_models \
		--max_epochs=$(CODEBOOK_MAX_EPOCHS) \
		--train=1
	mv $(TMP_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH) $(OUT_SERVE_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH)

$(OUT_DIR)/$(FROZEN_MODEL_PATH): $(OUT_DIR)/$(META_INFO)
	@echo "===== Training model ..."
	PYTHONPATH=$(KITE_ML_PATH) python train/train.py \
		$(SYNC_DATA_FLAG) \
		$(DATA_GEN_LOCAL_FLAG) \
		$(SYNCER_ENDPOINT_FLAG) \
		--in_dir=$(DATA_DIR) \
		--meta_info=$(OUT_DIR)/$(META_INFO) \
		--out_dir=$(TMP_DIR)/$(SAVED_MODEL_PATH) \
		--frozen_model=$(TMP_DIR)/$(FROZEN_MODEL_PATH) \
		--load_model= \
		--checkpoint_path=$(TMP_DIR)/$(CKPT_PATH) \
		--tensorboard=$(TENSORBOARD_PATH) \
		--endpoint=$(SERVER) \
		--steps=$(STEPS) \
		--batch=$(BATCH) \
		--max_samples=$(MAX_SAMPLES) \
		--attr_proportion=$(ATTR_PROPORTION) \
		--attr_base_proportion=$(ATTR_BASE_PROPORTION) \
		--call_proportion=$(CALL_PROPORTION) \
		--arg_type_proportion=$(ARG_TYPE_PROPORTION) \
		--kwarg_name_proportion=$(KWARG_NAME_PROPORTION) \
		--arg_placeholder_proportion=$(ARG_PLACEHOLDER_PROPORTION)
	cp -rv $(TENSORBOARD_DIR) $(OUT_DIR)
	mv $(TMP_DIR)/$(SAVED_MODEL_PATH) $(OUT_DIR)
	mv $(TMP_DIR)/$(FROZEN_MODEL_PATH) $(OUT_DIR)/$(FROZEN_MODEL_PATH)

continue:
	rm -rf $(OUT_DIR)/$(SAVED_MODEL_PATH)
	rm $(OUT_DIR)/$(FROZEN_MODEL_PATH)
	@echo "===== Continue training model ..."
	PYTHONPATH=$(KITE_ML_PATH) python train/train.py \
		$(SYNC_DATA_FLAG) \
		$(DATA_GEN_LOCAL_FLAG) \
		$(SYNCER_ENDPOINT_FLAG) \
		--in_dir=$(DATA_DIR) \
		--meta_info=$(OUT_DIR)/$(META_INFO) \
		--out_dir=$(TMP_DIR)/$(SAVED_MODEL_PATH) \
		--frozen_model=$(TMP_DIR)/$(FROZEN_MODEL_PATH) \
		--load_model=$(TMP_DIR)/$(CKPT_PATH) \
		--checkpoint_path=$(TMP_DIR)/$(CKPT_PATH) \
		--tensorboard=$(TENSORBOARD_PATH_CONTINUE) \
		--endpoint=$(SERVER) \
		--steps=$(STEPS) \
		--batch=$(BATCH) \
		--max_samples=$(MAX_SAMPLES) \
		--attr_proportion=$(ATTR_PROPORTION) \
		--attr_base_proportion=$(ATTR_BASE_PROPORTION) \
		--call_proportion=$(CALL_PROPORTION) \
		--arg_type_proportion=$(ARG_TYPE_PROPORTION) \
		--kwarg_name_proportion=$(KWARG_NAME_PROPORTION) \
		--arg_placeholder_proportion=$(ARG_PLACEHOLDER_PROPORTION)
	cp -rv $(TENSORBOARD_DIR) $(OUT_DIR)
	mv $(TMP_DIR)/$(SAVED_MODEL_PATH) $(OUT_DIR)
	mv $(TMP_DIR)/$(FROZEN_MODEL_PATH) $(OUT_DIR)/$(FROZEN_MODEL_PATH)

convert: $(OUT_SERVE_DIR)/$(META_INFO_FINAL)

$(OUT_SERVE_DIR)/$(META_INFO_FINAL): $(OUT_DIR)/$(META_INFO)
	go run github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr/cmds/convert \
		--in $(OUT_DIR)/$(META_INFO) --out $(TMP_DIR)/$(META_INFO_FINAL)
	mv $(TMP_DIR)/$(META_INFO_FINAL) $(OUT_SERVE_DIR)/$(META_INFO_FINAL)

validate_attribute: $(OUT_SERVE_DIR)/$(ATTR_FALLBACK_PATH)

$(OUT_SERVE_DIR)/$(ATTR_FALLBACK_PATH): $(OUT_SERVE_DIR)/$(COMPRESSED_FROZEN_MODEL_PATH) $(OUT_SERVE_DIR)/$(META_INFO_FINAL)
	@echo "===== validating attribute completions..."
	go run github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/ggnn-attribute-completions \
		--fallbackpath $(TMP_DIR)/attr-fallback-pipeline.txt \
		--exprmodel $(OUT_SERVE_DIR) \
		--runname $(MODEL_NAME)_validate_attribute \
		--exampledir $(TMP_DIR)/$(UNCURATED_EXAMPLES_DIR)/attr \
		--maxevents $(NUM_VALIDATE_SAMPLES) \
		--rundbpath $(RUNDB)
	cat $(TMP_DIR)/attr-fallback-pipeline.txt ./attr-fallback-hardcoded.txt \
    		| sort | uniq > $(TMP_DIR)/$(ATTR_FALLBACK_PATH)
	mv $(TMP_DIR)/$(ATTR_FALLBACK_PATH) $(OUT_SERVE_DIR)


rebuild_curated_examples: $(OUT_DIR)/$(CURATED_EXAMPLES_DIR)/attr.json

$(OUT_DIR)/$(CURATED_EXAMPLES_DIR)/attr.json: $(OUT_SERVE_DIR)/$(ATTR_FALLBACK_PATH)
	@echo "===== making curated examples..."
	mkdir -p $(TMP_DIR)/$(CURATED_EXAMPLES_DIR)
	go run github.com/kiteco/kiteco/kite-go/lang/python/pythoncompletions/cmds/rebuild-attr-examples \
	    --writefile \
	    --exprmodel $(OUT_SERVE_DIR) \
	    ./curated-examples/attr.json \
	    $(TMP_DIR)/$(CURATED_EXAMPLES_DIR)/attr.json
	mkdir -p $(OUT_DIR)/$(CURATED_EXAMPLES_DIR)
	mv $(TMP_DIR)/$(CURATED_EXAMPLES_DIR)/attr.json $(OUT_DIR)/$(CURATED_EXAMPLES_DIR)/attr.json

benchdata/feeds.gob: $(ATTRIBUTE_PIPELINE_BIN)
	mkdir -p ./benchdata
	go run github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/ggnn-attribute-completions \
		--feedpath benchdata/feeds.gob --numfeedrecords 100 --maxevents 1000

bench_model: benchdata/feeds.gob $(OUT_SERVE_DIR)/$(META_INFO)
	go build -o ./bin/preprocess-feeds \
		github.com/kiteco/kiteco/local-pipelines/python-ggnn-expr-completion/bench/preprocess-feeds
	go build -o ./bin/bench-model \
		github.com/kiteco/kiteco/kite-golib/tensorflow/bench/cmds/bench-model
	python bench/bench.py

upload: $(KFSPUT_BIN)
	@echo "===== Uploading frozen model ..."
	$(KFSPUT_BIN) $(OUT_DIR) $(UPLOAD_PATH)/%t

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

FORCE:

validate_calls: FORCE $(OUT_SERVE_DIR)/$(ATTR_FALLBACK_PATH)
	go run github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/ggnn-call-completions-validate \
		--model $(OUT_SERVE_DIR) \
		--maxevents $(NUM_VALIDATE_SAMPLES) \
		--rundb $(RUNDB)
