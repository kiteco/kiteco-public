# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/
OUT_DIR := ./out
TMP_DIR := ./tmp
BIN_DIR := ./bin

DIRS_TO_CREATE := $(OUT_DIR) $(TMP_DIR) $(BIN_DIR)

TRAIN_DATA_BIN := $(BIN_DIR)/traindata
SEGMENT_DATA_BIN := $(BIN_DIR)/segment-data
KFSPUT_BIN := $(BIN_DIR)/kfsput

COOCCURS_NAME := cooccurs.json
SEGMENT_COOCCURS_NAME := segment_cooccurs.json
CLUSTERS_NAME := clusters.json
FREQ_NAME := freq.txt
ORDERED_NAME := ordered.txt
ALL_PKG_NAME := all_packages.txt
DEPS_NAME := deps.json
DIST_NAME := dist.json
STANDARD_LIBS_NAME := stand_libs.txt

GOPATH ?= ${HOME}/go
ifeq ($(origin GOPATH), undefined)
	GOPATH := ${HOME}/go
endif
export GOPATH
KITECO := github.com/kiteco/kiteco/
KITEPATH := ${GOPATH}/src/${KITECO}

KITE_ML_PATH := $(KITEPATH)/kite-python/kite_ml/

UPLOAD_PATH := s3://kite-data/python-cluster-packages

NUM_CLUSTERS := 4

MAX_EVENTS := 1000000

ifeq ($(REMOTE),1)
	SERVER := http://ml-training-0.kite.com:3039
else
	SERVER := http://localhost:3039
endif

ifeq ($(DRY_RUN),1)
	MAX_PACKAGES := 20
	MAX_FILES := 20
else
	MAX_PACKAGES := 1500
	MAX_FILES := 500
endif


SEGMENT_COOCCURS_PATH := $(OUT_DIR)/$(SEGMENT_COOCCURS_NAME)
COOCCURS_PATH := $(OUT_DIR)/$(COOCCURS_NAME)

$(shell mkdir -p $(DIRS_TO_CREATE))

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)

$(TRAIN_DATA_BIN):
	go build -o $(TRAIN_DATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-cluster-packages/traindata

$(COOCCURS_PATH): $(TRAIN_DATA_BIN)
	@echo "===== Getting co-occurences ..."
	./$(TRAIN_DATA_BIN) --endpoint=$(SERVER) --maxpackages=$(MAX_PACKAGES) --maxfiles=$(MAX_FILES) --out=$(TMP_DIR)/$(COOCCURS_NAME)
	mv $(TMP_DIR)/$(COOCCURS_NAME) $(COOCCURS_PATH)

$(SEGMENT_DATA_BIN):
	go build -o $(SEGMENT_DATA_BIN) github.com/kiteco/kiteco/local-pipelines/python-cluster-packages/segment-data

$(SEGMENT_COOCCURS_PATH): $(SEGMENT_DATA_BIN)
	@echo "===== Getting co-occurences from segment data ..."
	./$(SEGMENT_DATA_BIN) --out=$(TMP_DIR)/$(SEGMENT_COOCCURS_NAME) --maxevents=$(MAX_EVENTS) --freq=$(TMP_DIR)/$(FREQ_NAME)
	mv $(TMP_DIR)/$(SEGMENT_COOCCURS_NAME) $(SEGMENT_COOCCURS_PATH)
	mv $(TMP_DIR)/$(FREQ_NAME) $(OUT_DIR)/$(FREQ_NAME)

$(OUT_DIR)/$(CLUSTERS_PATH): $(CO_OCCURS)
	@echo "===== Clustering ..."
	PYTHONPATH=$(KITE_ML_PATH) python cluster/cluster.py \
		--endpoint=$(SERVER) \
		--cooccurs=$(COOCCURS_PATH) \
		--num_clusters=$(NUM_CLUSTERS) \
		--out=$(TMP_DIR)/$(CLUSTERS_NAME)
	mv $(TMP_DIR)/$(CLUSTERS_NAME) $(OUT_DIR)

cluster: $(OUT_DIR)/$(CLUSTERS_PATH)

order:
	python collect_packages.py \
		--deps_path=./$(DEPS_NAME) \
        --dist_path=./$(DIST_NAME) \
        --freq_path=./$(OUT_DIR)/$(FREQ_NAME) \
        --cooc_path=./$(COOCCURS_PATH) \
        --ggnn_path=$(KITEPATH)/local-pipelines/python-ggnn-expr-completion/packagelist.txt \
        --stan_path=./$(STANDARD_LIBS_NAME) \
        --ordered_path=$(TMP_DIR)/$(ORDERED_NAME) \
        --all_path=$(TMP_DIR)/$(ALL_PKG_NAME)
	mv $(TMP_DIR)/$(ORDERED_NAME) $(OUT_DIR)
	mv $(TMP_DIR)/$(ALL_PKG_NAME) $(OUT_DIR)

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

upload: $(KFSPUT_BIN) $(OUT_DIR)/$(ORDERED_NAME)
	@echo "===== Uploading list of packages collected ..."
	$(KFSPUT_BIN) $(OUT_DIR) $(UPLOAD_PATH)/%t
