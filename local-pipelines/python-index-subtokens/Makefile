OUT_DIR := ./out
TMP_DIR := ./tmp
BIN_DIR := ./bin

DIRS_TO_CREATE := $(OUT_DIR) $(TMP_DIR) $(BIN_DIR)

INDEX_BIN := $(BIN_DIR)/index
KFSPUT_BIN := $(BIN_DIR)/kfsput

INDEX_PATH := index.json

UPLOAD_PATH := s3://kite-data/python-index-subtokens

DATA_DIR := /data/kite-local-pipelines/

NUM_SUBTOKENS := 30000
ifeq ($(DRY_RUN),1)
	MAX_FILES := 100
else
	MAX_FILES := 1000000
endif

$(shell mkdir -p $(DIRS_TO_CREATE))

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)

$(INDEX_BIN):
	go build -o $(INDEX_BIN) github.com/kiteco/kiteco/local-pipelines/python-index-subtokens/index

$(OUT_DIR)/$(INDEX_PATH): $(INDEX_BIN)
	@echo "=== Indexing subtokens ..."
	$(INDEX_BIN) \
		--maxfiles=$(MAX_FILES) \
		--numsubtokens=$(NUM_SUBTOKENS) \
		--datadir=$(DATA_DIR) \
		--out=$(TMP_DIR)/$(INDEX_PATH)
	mv $(TMP_DIR)/$(INDEX_PATH) $(OUT_DIR)/$(INDEX_PATH)

index: $(OUT_DIR)/$(INDEX_PATH)

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

upload: $(KFSPUT_BIN) $(OUT_DIR)/$(INDEX_PATH)
	@echo "===== Uploading index ..."
	$(KFSPUT_BIN) $(OUT_DIR)/$(INDEX_PATH) $(UPLOAD_PATH)/%t.$(INDEX_PATH)
