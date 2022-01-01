# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/

BIN_DIR := /data/kite/keywords_model/bin
OUT_DIR := /data/kite/keywords_model/out
TMP_DIR := $(OUT_DIR)/tmp
VENV_DIR := /data/kite/venv
DIRS_TO_CREATE := $(OUT_DIR) $(TMP_DIR) $(BIN_DIR)

TRAINDATA_BIN := $(BIN_DIR)/traindata

TRAIN_FILE := traindata.json
FREQ_FILE := keyword_frequencies.json
TMP_TRAIN_FILE := keyword_traindata-shard-0-part-0.json
RESAMPLED_FILE := resampled.json
MODEL_PATH := keyword_model
FROZEN_MODEL := keyword_model.frozen.pb
KITE_ML_PATH := ../../kite-python/kite_ml/

$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: train

traindata: $(OUT_DIR)/$(TRAIN_FILE)

train: $(OUT_DIR)/$(FROZEN_MODEL)

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(TMP_DIR)
	rm -rf $(BIN_DIR)

$(TRAINDATA_BIN):
	go build -o $(TRAINDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-keyword-model/traindata

$(TMP_DIR)/$(FREQ_FILE): $(TRAINDATA_BIN)
	$(TRAINDATA_BIN) freq $(TMP_DIR)/$(FREQ_FILE)


$(OUT_DIR)/$(TRAIN_FILE): $(TRAINDATA_BIN) $(TMP_DIR)/$(FREQ_FILE)
	$(TRAINDATA_BIN) collect $(TMP_DIR)/ $(TMP_DIR)/$(FREQ_FILE)
	mv $(TMP_DIR)/$(TMP_TRAIN_FILE) $(OUT_DIR)/$(TRAIN_FILE)

$(OUT_DIR)/$(TRAIN_FILE): $(TRAINDATA_BIN) $(TMP_DIR)/$(FREQ_FILE)
	$(TRAINDATA_BIN) collect $(TMP_DIR)/ $(TMP_DIR)/$(FREQ_FILE)
	mv $(TMP_DIR)/$(TMP_TRAIN_FILE) $(OUT_DIR)/$(TRAIN_FILE)

$(OUT_DIR)/$(FROZEN_MODEL): $(OUT_DIR)/$(TRAIN_FILE)
	( \
	  . $(VENV_DIR)/bin/activate; \
	  which python; \
	  python classifier/train.py $(OUT_DIR)/$(TRAIN_FILE) $(TMP_DIR)/$(MODEL_PATH) $(TMP_DIR)/$(FROZEN_MODEL); \
	  mv $(TMP_DIR)/$(MODEL_PATH) $(OUT_DIR)/$(MODEL_PATH); \
	  mv $(TMP_DIR)/$(FROZEN_MODEL) $(OUT_DIR)/$(FROZEN_MODEL); )
