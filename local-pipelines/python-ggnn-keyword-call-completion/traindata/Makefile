# Adapted from python-ggnn-call-completion
# First set up the Kite ML Python environment per these instructions:
# https://github.com/kiteco/kiteco/tree/master/kite-python/kite_ml/

OUT_DIR := ./out
TMP_DIR := ./tmp
BIN_DIR := ./bin
DIRS_TO_CREATE := $(OUT_DIR) $(TMP_DIR) $(BIN_DIR)

TRAIN_DATA_BIN := $(BIN_DIR)/traindata


KEYWORD_COUNT := kwcntdata.json

PACKAGES = ./packagelist.txt

SOURCE_ENDPOINT = http://ml-training-2.kite.com:3039/symbol/sources

SCORE_ENDPOINT = http://ml-training-2.kite.com:3039/symbol/scores

$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: train

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)

traindata: $(OUT_DIR)/$(KEYWORD_COUNT)

train: $(OUT_DIR)/$(MODEL_PATH)

$(TRAIN_DATA_BIN):
	go build -o $(TRAIN_DATA_BIN) github.com/kiteco/kiteco/local-pipelines/call-keyword-data

$(OUT_DIR)/$(SYMBOL_DIST): $(TRAIN_DATA_BIN)
	@echo "===== Making symbol keyword arguments count table ..."
	./$(TRAIN_DATA_BIN) --packages=$(PACKAGES) --out=$(TMP_DIR)/$(KEYWORD_COUNT) --sourcesendpoint=$(SOURCE_ENDPOINT) --scoresendpoint=$(SCORE_ENDPOINT)
	mv $(TMP_DIR)/$(KEYWORD_COUNT) $(OUT_DIR)/$(KEYWORD_COUNT)
