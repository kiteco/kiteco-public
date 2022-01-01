# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/

KITECO_ROOT := $(GOPATH)/src/github.com/kiteco/kiteco
VOCAB_SIZE := 13500
CONTEXT_SIZE := 600
EMBEDDING_SIZE := 180
NUM_HEADS := 6
NUM_LAYERS := 4
NUM_PREDICTION_SLOTS := 10
BATCH := 20
LANG := go

ifeq ($(DRY_RUN),1)
	STEPS := 10
	SELECT_FILES := 1
else
	STEPS := 100000
	SELECT_FILES := 50
endif

VOCAB_FILE := ident-vocab-$(VOCAB_SIZE)-entries.bpe
VOCAB_PATH := out_vocab_$(LANG)/vocab-checkpoints/$(VOCAB_FILE)

ifeq ($(LANG),go)
	# VOCAB_RUNDB := s3://kite-data/run-db/2020-02-11T22:18:53Z_lexical-vocabgen-go
	VOCAB_RUNDB := s3://kite-data/run-db/2020-08-09T21:24:05Z_lexical-vocabgen-go
endif
ifeq ($(LANG),javascript)
	VOCAB_RUNDB := s3://kite-data/run-db/2020-02-05T02:03:19Z_lexical-vocabgen-javascript
endif
ifeq ($(LANG),text)
	# VOCAB_RUNDB := s3://kite-data/run-db/2020-08-28T19:10:58Z_lexical-vocabgen-text

	# golang text vocab, allow keywords to merge with following hspace only
	VOCAB_RUNDB := s3://kite-data/run-db/2020-09-15T21:29:59Z_lexical-vocabgen-text
endif

MODEL_TYPE := prefix_suffix
MODEL_NAME := $(LANG)_$(MODEL_TYPE)_context_$(CONTEXT_SIZE)_embedding_$(EMBEDDING_SIZE)_layer_$(NUM_LAYERS)_head_$(NUM_HEADS)_vocab_$(VOCAB_SIZE)_steps_$(STEPS)

KITECO := $(GOPATH)/src/github.com/kiteco/kiteco
TENSORBOARD_DIR := ./tensorboard
OUT_DIR := ./out
GLOBAL_DATA_DIR := /data/lexical-training-data
VOCABS_DIR := ./vocabs/$(LANG)
BIN_DIR := ./bin
TMP_DIR := ./tmp
TRAIN_DIR := train
VALIDATE_DIR := validate
UPLOAD_PATH := s3://kite-data/lexical-completions-model

DIRS_TO_CREATE := $(OUT_DIR) $(TMP_DIR) $(TENSORBOARD_DIR)

TENSORBOARD_PATH := $(TENSORBOARD_DIR)/$(MODEL_NAME)
SAVED_MODEL_PATH := saved_model
TFSERVING_PATH := tfserving
FROZEN_MODEL_PATH := lexical_model.frozen.pb
VOCAB_FILE := ident-vocab-$(VOCAB_SIZE).bpe
CONFIG_FILE := config.json
TEST_RESULT := result.csv
CKPT_PATH := ckpt
TEST_DIRS := test_dirs.txt
SEARCH_CONFIG_FILE := searchconfig.json

$(shell mkdir -p $(DIRS_TO_CREATE))

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(OUT_DIR)
	rm -rf $(TMP_DIR)
	rm -rf $(TENSORBOARD_PATH)

fetch_vocab:
	rundb get-artifact $(VOCAB_RUNDB) $(VOCAB_PATH) $(VOCABS_DIR)/$(VOCAB_FILE)

datagen:
	go build -o $(BIN_DIR)/datagen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/datagen
	$(BIN_DIR)/datagen \
		--lang=$(LANG) \
		--vocab=$(VOCABS_DIR)/$(VOCAB_FILE) \
		--contextsize=$(CONTEXT_SIZE) \
		--traindir=$(GLOBAL_DATA_DIR)/$(TRAIN_DIR) \
		--validatedir=$(GLOBAL_DATA_DIR)/$(VALIDATE_DIR)

$(OUT_DIR)/$(CONFIG_FILE):
	go build -o $(BIN_DIR)/configgen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/configgen
	$(BIN_DIR)/configgen \
		--vocabsize=$(VOCAB_SIZE) \
		--embeddingsize=$(EMBEDDING_SIZE) \
		--contextsize=$(CONTEXT_SIZE) \
		--numheads=$(NUM_HEADS) \
		--lang=$(LANG) \
		--numlayers=$(NUM_LAYERS) \
		--numpredictionslots=$(NUM_PREDICTION_SLOTS) \
		--output=$(TMP_DIR)/$(CONFIG_FILE) \
		--modeltype=$(MODEL_TYPE)
	mv $(TMP_DIR)/$(CONFIG_FILE) $(OUT_DIR)

configgen: $(OUT_DIR)/$(CONFIG_FILE)


train: $(OUT_DIR)/$(SAVED_MODEL_PATH)

$(OUT_DIR)/$(SAVED_MODEL_PATH): $(GLOBAL_DATA_DIR)/$(TRAIN_DIR) $(OUT_DIR)/$(CONFIG_FILE)
	python train.py \
		--steps=$(STEPS) \
		--config=$(OUT_DIR)/$(CONFIG_FILE) \
		--train_samples=$(GLOBAL_DATA_DIR)/$(TRAIN_DIR) \
		--validate_samples=$(GLOBAL_DATA_DIR)/$(VALIDATE_DIR) \
		--checkpoint_path=$(TMP_DIR)/$(CKPT_PATH) \
		--out_dir=$(TMP_DIR)/$(SAVED_MODEL_PATH) \
		--tensorboard=$(TENSORBOARD_PATH) \
		--train_batch_size=$(BATCH) \
		--validate_batch_size=$(BATCH) \
		--model_type=$(MODEL_TYPE)
	mv $(TMP_DIR)/$(SAVED_MODEL_PATH) $(OUT_DIR)
	cp $(VOCABS_DIR)/$(VOCAB_FILE) $(OUT_DIR)/ident-vocab.bpe

generate_local_model: $(OUT_DIR)/$(SAVED_MODEL_PATH)
	python local_model.py \
		--in_saved_model=$(OUT_DIR)/$(SAVED_MODEL_PATH) \
		--config=$(OUT_DIR)/$(CONFIG_FILE) \
		--out_frozen_model=$(TMP_DIR)/$(FROZEN_MODEL_PATH)
	mv $(TMP_DIR)/$(FROZEN_MODEL_PATH) $(OUT_DIR)

generate_tfserving_model: $(OUT_DIR)/$(SAVED_MODEL_PATH) $(OUT_DIR)/$(SEARCH_CONFIG_FILE)
	python tfserve_model.py \
		--in_saved_model=$(OUT_DIR)/$(SAVED_MODEL_PATH) \
		--config=$(OUT_DIR)/$(CONFIG_FILE) \
		--search_config=$(OUT_DIR)/$(SEARCH_CONFIG_FILE) \
		--out_saved_model=$(TMP_DIR)/$(TFSERVING_PATH)/1
	mv $(TMP_DIR)/$(TFSERVING_PATH) $(OUT_DIR)

searchconfiggen: $(OUT_DIR)/$(SEARCH_CONFIG_FILE)

$(OUT_DIR)/$(SEARCH_CONFIG_FILE): $(OUT_DIR)/$(FROZEN_MODEL_PATH)
	# initial search config
	go build -o $(BIN_DIR)/searchconfiggen github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/searchconfiggen
	$(BIN_DIR)/searchconfiggen \
		--lang=$(LANG) \
		--out=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--modeltype=$(MODEL_TYPE)

	# calibration before temperature scaling
	go build -o $(BIN_DIR)/calibration github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/calibration
	$(BIN_DIR)/calibration \
		--modelpath=$(OUT_DIR) \
		--outdir=$(TMP_DIR)/calibration_original \
		--search=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--lang=$(LANG)

	# temperature scaling (updates search config)
	go build -o $(BIN_DIR)/traindata_temperature_scaling \
		github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/calibrate-temperature-scaling/traindata_temperature_scaling
	$(BIN_DIR)/traindata_temperature_scaling \
		--modelpath=$(OUT_DIR) \
		--traindir=$(TMP_DIR)/temperature_scaling/train_samples \
		--validatedir=$(TMP_DIR)/temperature_scaling/validate_samples \
		--search=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--lang=$(LANG)
	python cmds/calibrate-temperature-scaling/train/train_temperature_scaling.py \
		--insearch=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--outsearch=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--train_samples=$(TMP_DIR)/temperature_scaling/train_samples \
		--validate_samples=$(TMP_DIR)/temperature_scaling/validate_samples

	# calibration after temperature scaling
	$(BIN_DIR)/calibration \
		--modelpath=$(OUT_DIR) \
		--outdir=$(TMP_DIR)/calibration_final \
		--search=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--lang=$(LANG)

	# min p computation (updates search config)
	go build -o $(BIN_DIR)/minp github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/minp
	$(BIN_DIR)/minp \
		--modelpath=$(OUT_DIR) \
		--insearch=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--outsearch=$(TMP_DIR)/$(SEARCH_CONFIG_FILE) \
		--language=$(LANG)

	# done
	mv $(TMP_DIR)/calibration_original $(OUT_DIR)/calibration_original
	mv $(TMP_DIR)/calibration_final $(OUT_DIR)/calibration_final
	mv $(TMP_DIR)/$(SEARCH_CONFIG_FILE) $(OUT_DIR)/$(SEARCH_CONFIG_FILE)

test: $(OUT_DIR)/$(TEST_RESULT)

$(OUT_DIR)/$(TEST_RESULT): $(OUT_DIR)/$(FROZEN_MODEL_PATH) $(OUT_DIR)/$(SEARCH_CONFIG_FILE)
	go build -o $(BIN_DIR)/performance github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/performance
	$(BIN_DIR)/performance \
		--modelpath=$(OUT_DIR) \
		--resultpath=$(TMP_DIR)/$(TEST_RESULT) \
		--search=$(OUT_DIR)/$(SEARCH_CONFIG_FILE) \
		--lang=$(LANG)
	mv $(TMP_DIR)/$(TEST_RESULT) $(OUT_DIR)

upload: $(BIN_DIR)/kfsput
	@echo "===== Uploading frozen model ..."
	$(BIN_DIR)/kfsput $(OUT_DIR) $(UPLOAD_PATH)/%t/out

$(BIN_DIR)/kfsput:
	go build -o $(BIN_DIR)/kfsput github.com/kiteco/kiteco/kite-go/cmds/kfsput

MODEL_DIR := ./out
RESULT := custom.txt
custom_test:
	go build -o $(BIN_DIR)/performance github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/performance
	$(BIN_DIR)/performance \
		--modelpath=$(MODEL_DIR) \
		--resultpath=$(RESULT) \
		--lang=$(LANG) \
		--usesearcher=false \
		--modelconfig

# Local code fine tuning
LANG := go
VOCAB_ITER := 400
MODEL_NAME := lang_$(LANG)_vocab_$(VOCAB_ITER)

LOCAL := ./local_fine_tune_$(MODEL_NAME)
LOCAL_OUT := $(LOCAL)/out
LOCAL_TMP := $(LOCAL)/tmp
LOCAL_DATA := $(LOCAL)/data
LOCAL_INITIAL := $(LOCAL)/initial
LOCAL_UPDATED_SAVED_MODEL := $(LOCAL)/updated_saved_model
LOCAL_TENSORBOARD := $(LOCAL)/tensorboard/$(MODEL_NAME)

LOCAL_DIRS_TO_CREATE := $(LOCAL) $(LOCAL_OUT) $(LOCAL_TMP) $(LOCAL_DATA) $(LOCAL_INITIAL) $(LOCAL_UPDATED_SAVED_MODEL) $(LOCAL_TENSORBOARD)
$(shell mkdir -p $(LOCAL_DIRS_TO_CREATE))

ORIGINAL_MODEL := s3://kite-data/run-db/2019-11-21T01:05:16Z_lexical-model-experiments-3/out_lexical_context_128_embedding_180_layer_4_head_6_vocab_13500_steps_2000000
LOCAL_STEPS := 2000

localtrain_clean:
	rm -rf $(LOCAL_OUT)
	rm -rf $(LOCAL_TMP)
	rm -rf $(LOCAL_DATA)
	rm -rf $(LOCAL_UPDATED_SAVED_MODEL)
	rm -rf $(LOCAL_TENSORBOARD)

get_global_model:
	aws s3 cp --recursive $(ORIGINAL_MODEL) $(LOCAL_INITIAL)


localtrain_prepare:
	cp $(LOCAL_INITIAL)/$(SEARCH_CONFIG_FILE) $(LOCAL_OUT)/$(SEARCH_CONFIG_FILE)
	go build -o $(BIN_DIR)/localtrain github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/localtrain
	$(BIN_DIR)/localtrain \
		--maxtrainiters=$(LOCAL_STEPS) \
		--vocabiters=$(VOCAB_ITER) \
		--localdataroot=$(KITECO_ROOT) \
		--language=$(LANG) \
		--globalmodel=$(ORIGINAL_MODEL) \
		--outvocabidents=$(LOCAL_OUT)/ident-vocab-entries.bpe \
		--outvocabembeddings=$(LOCAL_OUT)/wte-updated.json \
		--outvalidatesamples=$(LOCAL_DATA)/$(VALIDATE_DIR) \
		--outtrainsamples=$(LOCAL_DATA)/$(TRAIN_DIR)

	python update_vocab_embedding.py \
		--in_saved_model=$(LOCAL_INITIAL)/$(SAVED_MODEL_PATH) \
		--extra_vocab_size=$(VOCAB_ITER) \
		--original_config=$(LOCAL_INITIAL)/$(CONFIG_FILE) \
		--new_embedding=$(LOCAL_OUT)/wte-updated.json \
		--out_saved_model=$(LOCAL_UPDATED_SAVED_MODEL) \
		--out_config=$(LOCAL_OUT)/$(CONFIG_FILE)


localtrain:
	python train.py \
		--steps=$(LOCAL_STEPS) \
		--config=$(LOCAL_OUT)/$(CONFIG_FILE) \
		--train_samples=$(LOCAL_DATA)/$(TRAIN_DIR) \
		--validate_samples=$(LOCAL_DATA)/$(VALIDATE_DIR) \
		--checkpoint_path=$(LOCAL_TMP)/$(CKPT_PATH) \
		--out_dir=$(LOCAL_TMP)/$(SAVED_MODEL_PATH) \
		--tensorboard=$(LOCAL_TENSORBOARD) \
		--train_batch_size=$(BATCH) \
		--validate_batch_size=100 \
		--model_type=lexical \
		--load_model=$(LOCAL_UPDATED_SAVED_MODEL)

	python model_from_checkpoint.py \
		--load_checkpoint=$(LOCAL_TMP) \
		--config=$(LOCAL_OUT)/$(CONFIG_FILE) \
		--out_dir=$(LOCAL_OUT)/$(SAVED_MODEL_PATH)

	python local_model.py \
  	--in_saved_model=$(LOCAL_OUT)/$(SAVED_MODEL_PATH) \
    --config=$(LOCAL_OUT)/$(CONFIG_FILE) \
    --out_frozen_model=$(LOCAL_TMP)/$(FROZEN_MODEL_PATH)
	mv $(LOCAL_TMP)/$(FROZEN_MODEL_PATH) $(LOCAL_OUT)


# For github data extraction
GITHUB_REPO := "go-apache"
GITHUB_OWNER := "airflow"
PULLS_PATH := "pulls.json"

githubdataextract:
	go build -o $(BIN_DIR)/githubdataextract github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/githubdataextract
	$(BIN_DIR)/githubdataextract \
		--owner=$(GITHUB_OWNER) \
		--repo=$(GITHUB_REPO) \
		--pullspath=$(PULLS_PATH) \
		--repoclonedir=$(TMP)
