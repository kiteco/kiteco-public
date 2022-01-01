# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/

MODEL_NAME := skeleton

TENSORBOARD_DIR := ./tensorboard
BIN_DIR := ./bin
OUT_DIR := ./out
OUT_SERVE_DIR := ./out/serve
DATA_DIR := ./data
TMP_DIR := ./tmp

DIRS_TO_CREATE := $(OUT_DIR) $(OUT_SERVE_DIR) $(TMP_DIR) $(BIN_DIR) $(TENSORBOARD_DIR) $(DATA_DIR)

TRAINDATA_BIN := $(BIN_DIR)/traindata
VISSERVER_BIN := $(BIN_DIR)/visserver
KFSPUT_BIN := $(BIN_DIR)/kfsput

TENSORBOARD_PATH := $(TENSORBOARD_DIR)/$(MODEL_NAME)
SAVED_MODEL_PATH := saved_model
FROZEN_MODEL_PATH := mix_model.frozen.pb
CKPT_PATH := ckpt

TRAINDATA := train_data.json
VISDATA := visdata.json

KITE_ML_PATH := ../../kite-python/kite_ml/

UPLOAD_PATH := s3://kite-data/python-completion-mixing

ifeq ($(DRY_RUN),1)
	STEPS := 10
	MAX_FILES := 5000
	MAX_VIS_SAMPLES := 10
else
	STEPS := 50000
	MAX_FILES := 1000000
	MAX_VIS_SAMPLES := 1000
endif

$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: train

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)
	rm -rf $(DATA_DIR)
	rm -rf $(TENSORBOARD_PATH)

traindata: $(DATA_DIR)/$(TRAINDATA)

train: $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH)

visdata: $(DATA_DIR)/$(VISDATA)

$(TRAINDATA_BIN):
	go build -o $(TRAINDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-completion-mixing/traindata

$(DATA_DIR)/$(TRAINDATA): $(TRAINDATA_BIN)
	@echo "==== Making train data ..."
	./$(TRAINDATA_BIN) --out=$(TMP_DIR)/$(TRAINDATA) --maxfiles=$(MAX_FILES)
	mv $(TMP_DIR)/$(TRAINDATA) $(DATA_DIR)/$(TRAINDATA)

$(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH): $(DATA_DIR)/$(TRAINDATA)
	@echo "==== Training model ..."
	PYTHONPATH=$(KITE_ML_PATH) python train/train.py \
    		--traindata=$(DATA_DIR)/$(TRAINDATA) \
    		--out_dir=$(TMP_DIR)/$(SAVED_MODEL_PATH) \
    		--frozen_model=$(TMP_DIR)/$(FROZEN_MODEL_PATH) \
    		--checkpoint_path=$(TMP_DIR)/$(CKPT_PATH) \
    		--tensorboard=$(TENSORBOARD_PATH) \
    		--steps=$(STEPS)
	mv $(TMP_DIR)/$(SAVED_MODEL_PATH) $(OUT_DIR)
	mv $(TMP_DIR)/$(FROZEN_MODEL_PATH) $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH)

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

upload: $(KFSPUT_BIN) $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH)
	@echo "===== Uploading frozen model ..."
	$(KFSPUT_BIN) $(OUT_DIR) $(UPLOAD_PATH)/%t

$(DATA_DIR)/$(VISDATA): $(DATA_DIR)/$(TRAINDATA) $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH)
	PYTHONPATH=$(KITE_ML_PATH) python visdata/visdata.py \
			--traindata=$(DATA_DIR)/$(TRAINDATA) \
			--checkpoint_path=$(OUT_DIR)/$(SAVED_MODEL_PATH)/variables/variables \
			--max_samples=$(MAX_VIS_SAMPLES) \
			--out_path=$(TMP_DIR)/$(VISDATA)
	mv $(TMP_DIR)/$(VISDATA) $(DATA_DIR)/$(VISDATA)


$(VISSERVER_BIN):
	go build -o $(VISSERVER_BIN) github.com/kiteco/kiteco/local-pipelines/python-completion-mixing/visserver

visserver: $(VISSERVER_BIN) $(DATA_DIR)/$(VISDATA) nonexistent_file.txt
	$(VISSERVER_BIN) --visdata=$(DATA_DIR)/$(VISDATA)

nonexistent_file.txt:
	true

