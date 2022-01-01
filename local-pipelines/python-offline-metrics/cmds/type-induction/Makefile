OUT_DIR := ./out
TMP_DIR := ./tmp
BIN_DIR := ./bin
ATTRS_DIR := ./attrs

SAMPLES_DIR := /data/kite-local-pipelines/type-induction/traindata
MODEL_DIR := model
VALIDATE_DATA := validate-examples
REPORT_DIR := report

DIRS_TO_CREATE := $(OUT_DIR) $(BIN_DIR) $(ATTRS_DIR) $(SAMPLES_DIR) $(TMP_DIR)/$(VALIDATE_DATA) $(TMP_DIR)/$(REPORT_DIR)

GOPATH ?= ${HOME}/go
ifeq ($(origin GOPATH), undefined)
	GOPATH := ${HOME}/go
endif
export GOPATH
KITECO := github.com/kiteco/kiteco/
KITEPATH := ${GOPATH}/src/${KITECO}

TRAINDATA_BIN := $(BIN_DIR)/traindata
SPLITDATA_BIN := $(BIN_DIR)/splitdata
TRAIN_BIN := $(BIN_DIR)/train
VALIDATEDATA_BIN := $(BIN_DIR)/validatedata
VALIDATE_BIN := $(BIN_DIR)/validate
KFSPUT_BIN := $(BIN_DIR)/kfsput

REMOTE_TRAINDATA := s3://kite-local-pipelines/type-induction/traindata/$(shell date -u +"%Y-%m-%d_%H-%M-%S")
REMOTE_VALIDATEDATA := s3://kite-local-pipelines/type-induction/validatedata/$(shell date -u +"%Y-%m-%d_%H-%M-%S")
MODEL_UPLOAD_PATH := s3://kite-local-pipelines/type-induction/model

PACKAGES_FILE := packages.txt
PACKAGES_PATH := ./$(PACKAGES_FILE)

MAX_RECORDS := 1000000
STEPS := 1000
MAX_EVENTS := 2000000
INSTANCE_COUNT := 2

$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: validate

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)

$(TRAINDATA_BIN):
	go build -o $(TRAINDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/traindata

traindata: $(TRAINDATA_BIN)
	@echo "===== Extracting training samples ..."
	./$(TRAINDATA_BIN) \
		--out=$(TMP_DIR)/$(SAMPLES_DIR) \
		--packages=$(PACKAGES_PATH) \
		--maxrecords=$(MAX_RECORDS)
	mv $(TMP_DIR)/$(SAMPLES_DIR) $(OUT_DIR)
	cp $(PACKAGES_PATH) $(OUT_DIR)/$(PACKAGES_FILE)

dist_traindata:
	$(KITE_PATH)/kite-golib/pipeline/util/launch-dist-pipeline.sh \
		github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/traindata \
		$(INSTANCE_COUNT) \
		--out=$(REMOTE_TRAINDATA) \
		--maxrecords=0

$(TRAIN_BIN):
	go build -o $(TRAIN_BIN) github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/train

$(SPLITDATA_BIN):
	go build -o $(SPLITDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/splitdata

$(SAMPLES_DIR): $(SPLITDATA_BIN)
	@echo "===== Splitting training data by packages ..."
	./$(SPLITDATA_BIN) --samplesout=$(TMP_DIR)/samples
	mv $(TMP_DIR)/samples $(SAMPLES_DIR)

$(OUT_DIR)/$(MODEL_DIR): $(TRAIN_BIN) $(SAMPLES_DIR)
	@echo "===== Training model ..."
	./$(TRAIN_BIN) \
		--modelpath=$(TMP_DIR)/$(MODEL_DIR) \
		--attrdist=$(ATTRS_DIR) \
		--steps=$(STEPS) \
		--samplesdir=$(SAMPLES_DIR) \
		--packages=$(PACKAGES_PATH)
	mv $(TMP_DIR)/$(MODEL_DIR) $(OUT_DIR)

train: $(OUT_DIR)/$(MODEL_DIR)

$(VALIDATEDATA_BIN):
	go build -o $(VALIDATEDATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/validatedata

validatedata: $(VALIDATEDATA_BIN)
	@echo "===== Generating validation examples from segment data ..."
	./$(VALIDATEDATA_BIN) \
		--packages=$(PACKAGES_PATH) \
		--maxevents=$(MAX_EVENTS) \
		--out=$(TMP_DIR)/$(VALIDATE_DATA)
	mv $(TMP_DIR)/$(VALIDATE_DATA) $(OUT_DIR)

$(VALIDATE_BIN):
	go build -o $(VALIDATE_BIN) github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/validate

$(OUT_DIR)/$(REPORT_DIR): $(OUT_DIR)/$(MODEL_DIR) $(VALIDATE_BIN)
	@echo "==== Validating model ..."
	./$(VALIDATE_BIN) \
		--packages=$(PACKAGES_PATH) \
		--modeldir=$(OUT_DIR)/$(MODEL_DIR) \
		--outdir=$(TMP_DIR)/$(REPORT_DIR)
	mv $(TMP_DIR)/$(REPORT_DIR) $(OUT_DIR)

validate: $(OUT_DIR)/$(REPORT_DIR)

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

upload: $(KFSPUT_BIN) $(OUT_DIR)/$(MODEL_DIR)
	@echo "===== Uploading results ..."
	$(KFSPUT_BIN) $(OUT_DIR) $(UPLOAD_PATH)/%t
