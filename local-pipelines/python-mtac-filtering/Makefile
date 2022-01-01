# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/

MODEL_NAME := logistic

BIN_DIR := ./bin
DATA_DIR := ./data
TMP_DIR := ./tmp
OUT_DIR := ./out
OUT_SERVE_DIR := ./out/serve
TMP_DATA_DIR := ./tmp/data
TRAINDATA_FILE := traindata.json
GTDATA_FILE := gt_data.json
PARAMS_FILE := params.json
METRIC_DATA_FILE := metric_data.json


DIRS_TO_CREATE := $(BIN_DIR) $(DATA_DIR) $(TMP_DIR) $(OUT_DIR) $(OUT_SERVE_DIR) $(TMP_DATA_DIR)

GTDATA_BIN := $(BIN_DIR)/gtdata
TRAINDATA_BIN := $(BIN_DIR)/traindata
METRIC_BIN := $(BIN_DIR)/metric
THRESHOLD_BIN := $(BIN_DIR)/threshold
KFSPUT_BIN := $(BIN_DIR)/kfsput

SAVED_MODEL_PATH := saved_model
FROZEN_MODEL_PATH := mtac_conf_model.frozen.pb

KITE_ML_PATH := ../../kite-python/kite_ml/
UPLOAD_PATH := s3://kite-data/python-mtac-filtering
MODEL_UPLOAD_PATH := s3://kite-data/python-mtac-confidence

INSTANCE_COUNT := 2
REMOTE_OUT_DIR := s3://kite-data/python-mtac-confidence/traindata/$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

MAX_FILES_GT := 1000000

ifeq ($(DRY_RUN),1)
	MAX_FILES := 5000
else
	MAX_FILES := 200
endif

$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: metrics

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(DATA_DIR)
	rm -rf $(TMP_DIR)
	rm -rf $(OUT_DIR)
	rm -rf $(OUT_SERVE_DIR)
	rm -rf $(TMP_DATA_DIR)

traindata: $(DATA_DIR)/$(TRAINDATA_FILE)

train: $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH)

gtdata: $(GTDATA_BIN) $(DATA_DIR)/$(GTDATA_FILE)

threshold: $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH) $(OUT_SERVE_DIR)/$(PARAMS_FILE)

dist_traindata:
	../../kite-golib/pipeline/util/launch-dist-pipeline.sh \
		github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/python-mtac-confidence/traindata \
		$(INSTANCE_COUNT) \
		--outdir=$(REMOTE_OUT_DIR) \
		--maxfiles=$(MAX_FILES)

$(TRAINDATA_BIN):
	go build -o $(TRAINDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/python-mtac-confidence/traindata

$(DATA_DIR)/$(TRAINDATA_FILE): $(TRAINDATA_BIN)
	@echo "==== Making train data ..."
	./$(TRAINDATA_BIN) --outdir=$(TMP_DATA_DIR) --maxfiles=$(MAX_FILES)
	cat $(TMP_DATA_DIR)/*.json > $(DATA_DIR)/$(TRAINDATA_FILE)

$(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH): $(DATA_DIR)/$(TRAINDATA_FILE)
	@echo "==== Training model ..."
	PYTHONPATH=$(KITE_ML_PATH) python python-mtac-confidence/train/train.py \
			--traindata=$(DATA_DIR)/$(TRAINDATA_FILE) \
			--out_dir=$(TMP_DIR)/$(SAVED_MODEL_PATH) \
			--frozen_model=$(TMP_DIR)/$(FROZEN_MODEL_PATH)
	mv $(TMP_DIR)/$(SAVED_MODEL_PATH) $(OUT_DIR)
	mv $(TMP_DIR)/$(FROZEN_MODEL_PATH) $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH)

$(GTDATA_BIN):
	go build -o $(GTDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/gtdata

$(DATA_DIR)/$(GTDATA_FILE): $(GTDATA_BIN)
	@echo "==== Making ground truth data ..."
	./$(GTDATA_BIN) --out=$(TMP_DATA_DIR)/$(GTDATA_FILE) --maxfiles=$(MAX_FILES_GT)
	mv $(TMP_DATA_DIR)/$(GTDATA_FILE) $(DATA_DIR)/$(GTDATA_FILE)

$(METRIC_BIN):
	go build -o $(METRIC_BIN) github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/metrics

metrics: $(METRIC_BIN) $(DATA_DIR)/$(GTDATA_FILE)
	@echo "===== computing metrics..."
	./$(METRIC_BIN) --input=$(DATA_DIR)/$(GTDATA_FILE)

$(THRESHOLD_BIN):
	go build -o $(THRESHOLD_BIN) github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/threshold

$(OUT_SERVE_DIR)/$(PARAMS_FILE): $(DATA_DIR)/$(GTDATA_FILE) $(THRESHOLD_BIN)
	@echo "===== finding thresholds..."
	./$(THRESHOLD_BIN) --input $(DATA_DIR)/$(GTDATA) --modelpath $(OUT_SERVE_DIR)\
	--outanalytic $(TMP_DATA_DIR)/$(METRIC_DATA_FILE) --outparams $(TMP_DATA_DIR)/$(PARAMS_FILE)
	mv $(TMP_DATA_DIR)/$(PARAMS_FILE) $(OUT_SERVE_DIR)/$(PARAMS_FILE)
	mv $(TMP_DATA_DIR)/$(METRIC_DATA_FILE) $(DATA_DIR)/$(METRIC_DATA_FILE)

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

uploadgt: $(KFSPUT_BIN) $(DATA_DIR)/$(GTDATA_FILE)
	@echo "===== Uploading ground truth data ..."
	$(KFSPUT_BIN) $(DATA_DIR) $(UPLOAD_PATH)/%t

upload: $(KFSPUT_BIN) $(OUT_SERVE_DIR)/$(FROZEN_MODEL_PATH)
	@echo "===== Uploading frozen model ..."
	$(KFSPUT_BIN) $(OUT_DIR) $(MODEL_UPLOAD_PATH)/%t
