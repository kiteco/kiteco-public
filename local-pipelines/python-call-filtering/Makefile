# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/

MODEL_NAME := logistic

BIN_DIR := ./bin
DATA_DIR := ./data
TMP_DIR := ./tmp
OUT_DIR := ./out
OUT_SERVE_DIR := ./out/serve
TMP_DATA_DIR := ./tmp/data

OUT_TRUNCATED_DIR := ./out/truncated_calls
OUT_FULL_DIR := ./out/full_calls

TRUNC_TRAINDATA_FILE := truncated_traindata.json
FULL_TRAINDATA_FILE := full_traindata.json
TMP_TRAINDATA_DIR := $(TMP_DATA_DIR)/traindata

GTDATA_FILE := gt_data.json
TMP_GTDATA_DIR := $(TMP_DATA_DIR)/gtdata

PARAMS_FILE := params.json
METRIC_DATA_FILE := metric_data.json



DIRS_TO_CREATE := $(BIN_DIR) $(DATA_DIR) $(TMP_DIR) $(OUT_DIR) $(OUT_SERVE_DIR) $(TMP_DATA_DIR)


GTDATA_BIN := $(BIN_DIR)/gtdata
TRAINDATA_BIN := $(BIN_DIR)/traindata
METRIC_BIN := $(BIN_DIR)/metric
THRESHOLD_BIN := $(BIN_DIR)/threshold
KFSPUT_BIN := $(BIN_DIR)/kfsput


SAVED_TRUNC_MODEL_PATH := saved_model_trunc
SAVED_FULL_MODEL_PATH := saved_model_full
FROZEN_FULL_MODEL_PATH := full_call_prob_model.frozen.pb
FROZEN_TRUNC_MODEL_PATH := truncated_call_prob_model.frozen.pb



KITE_ML_PATH := ../../kite-python/kite_ml/
UPLOAD_PATH := s3://kite-data/python-call-filtering
MODEL_UPLOAD_PATH := s3://kite-data/python-call-prob

INSTANCE_COUNT := 2
REMOTE_OUT_DIR := s3://kite-data/python-call-prob/traindata/$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

MAX_FILES_GT := 5000

ifeq ($(DRY_RUN),1)
	MAX_FILES := 10
else
	MAX_FILES := 1000
endif

$(shell mkdir -p $(DIRS_TO_CREATE))

.PHONY: metrics

default: all

all: metrics

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(DATA_DIR)
	rm -rf $(TMP_DIR)
	rm -rf $(OUT_DIR)
	rm -rf $(OUT_SERVE_DIR)
	rm -rf $(TMP_DATA_DIR)

traindata: $(DATA_DIR)/$(TRUNC_TRAINDATA_FILE) $(DATA_DIR)/$(FULL_TRAINDATA_FILE)

train: $(OUT_SERVE_DIR)/$(FROZEN_TRUNC_MODEL_PATH) $(OUT_SERVE_DIR)/$(FROZEN_FULL_MODEL_PATH)

gtdata: $(GTDATA_BIN) $(DATA_DIR)/$(GTDATA_FILE)

threshold: $(OUT_SERVE_DIR)/$(FROZEN_TRUNC_MODEL_PATH) $(OUT_SERVE_DIR)/$(FROZEN_FULL_MODEL_PATH) $(OUT_SERVE_DIR)/$(PARAMS_FILE)

dist_traindata:
	../../kite-golib/pipeline/util/launch-dist-pipeline.sh \
		github.com/kiteco/kiteco/local-pipelines/python-call-filtering/python-call-prob/traindata \
		$(INSTANCE_COUNT) \
		--outdir=$(REMOTE_OUT_DIR) \
		--maxfiles=$(MAX_FILES)

$(DATA_DIR)/$(TRUNC_TRAINDATA_FILE) $(DATA_DIR)/$(FULL_TRAINDATA_FILE):
	@echo "==== Making train data ..."
	go build -o $(TRAINDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-call-filtering/python-call-prob/traindata
	./$(TRAINDATA_BIN) --outdir=$(TMP_TRAINDATA_DIR) --maxfiles=$(MAX_FILES)
	cat $(TMP_TRAINDATA_DIR)/truncated_call*.json > $(DATA_DIR)/$(TRUNC_TRAINDATA_FILE)
	cat $(TMP_TRAINDATA_DIR)/full_call*.json > $(DATA_DIR)/$(FULL_TRAINDATA_FILE)


$(OUT_SERVE_DIR)/$(FROZEN_TRUNC_MODEL_PATH): $(DATA_DIR)/$(TRUNC_TRAINDATA_FILE)
	@echo "==== Training model ..."
	PYTHONPATH=$(KITE_ML_PATH) python python-call-prob/train/train.py \
			--traindata=$(DATA_DIR)/$(TRUNC_TRAINDATA_FILE) \
			--out_dir=$(TMP_DIR)/$(SAVED_TRUNC_MODEL_PATH) \
			--frozen_model=$(TMP_DIR)/$(FROZEN_TRUNC_MODEL_PATH)
	mv $(TMP_DIR)/$(SAVED_TRUNC_MODEL_PATH) $(OUT_TRUNCATED_DIR)
	mv $(TMP_DIR)/$(FROZEN_TRUNC_MODEL_PATH) $(OUT_SERVE_DIR)/$(FROZEN_TRUNC_MODEL_PATH)

$(OUT_SERVE_DIR)/$(FROZEN_FULL_MODEL_PATH): $(DATA_DIR)/$(FULL_TRAINDATA_FILE)
	@echo "==== Training model ..."
	PYTHONPATH=$(KITE_ML_PATH) python python-call-prob/train/train.py \
			--traindata=$(DATA_DIR)/$(FULL_TRAINDATA_FILE) \
			--out_dir=$(TMP_DIR)/$(SAVED_FULL_MODEL_PATH) \
			--frozen_model=$(TMP_DIR)/$(FROZEN_FULL_MODEL_PATH)
	mv $(TMP_DIR)/$(SAVED_FULL_MODEL_PATH) $(OUT_FULL_DIR)
	mv $(TMP_DIR)/$(FROZEN_FULL_MODEL_PATH) $(OUT_SERVE_DIR)/$(FROZEN_FULL_MODEL_PATH)


$(DATA_DIR)/$(GTDATA_FILE):
	@echo "==== Making ground truth data ..."
	go build -o $(GTDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-call-filtering/gtdata
	./$(GTDATA_BIN) --out=$(TMP_GTDATA_DIR) --maxfiles=$(MAX_FILES_GT)
	cat $(TMP_GTDATA_DIR)/*.json > $(DATA_DIR)/$(GTDATA_FILE)


metrics: $(DATA_DIR)/$(GTDATA_FILE)
	@echo "===== computing metrics..."
	go build -o $(METRIC_BIN) github.com/kiteco/kiteco/local-pipelines/python-call-filtering/metrics
	./$(METRIC_BIN) --input=$(DATA_DIR)/$(GTDATA_FILE)

$(OUT_SERVE_DIR)/$(PARAMS_FILE): $(DATA_DIR)/$(GTDATA_FILE)
	@echo "===== finding thresholds..."
	go build -o $(THRESHOLD_BIN) github.com/kiteco/kiteco/local-pipelines/python-call-filtering/threshold
	./$(THRESHOLD_BIN) --input $(DATA_DIR)/$(GTDATA_FILE) \
	--modelspath $(OUT_SERVE_DIR) \
	--outanalytic $(TMP_DATA_DIR)/$(METRIC_DATA_FILE) \
	--outparams $(TMP_DATA_DIR)/$(PARAMS_FILE)
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
