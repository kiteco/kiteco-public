BIN_DIR := ./bin
TMP_DIR := /data/kite-local-pipelines/python-symbol-to-hashes-index/tmp
CACHE_ROOT := /data/kite-local-pipelines
DIRS_TO_CREATE := $(BIN_DIR) $(TMP_DIR) $(CACHE_ROOT)

TMP_EXTRACTED := $(TMP_DIR)/extracted
TMP_MERGED := $(TMP_DIR)/merged

EXTRACT_BIN := $(BIN_DIR)/extract
MERGE_BIN := $(BIN_DIR)/merge
KFSPUT_BIN := $(BIN_DIR)/kfsput

UPLOAD_PATH := s3://kite-local-pipelines/python-symbol-to-hashes-index


$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: merge

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)

extract: $(TMP_EXTRACTED)/DONE

$(EXTRACT_BIN):
	go build -o $(EXTRACT_BIN) github.com/kiteco/kiteco/local-pipelines/python-symbol-to-hashes-index/extract

$(TMP_EXTRACTED)/DONE: $(EXTRACT_BIN)
	@echo "===== Extracting index ..."
	./$(EXTRACT_BIN) --cacheroot=$(CACHE_ROOT) --symbolsout=$(TMP_EXTRACTED)/symbols --canonsymbolsout=$(TMP_EXTRACTED)/canonical-symbols
	touch $(TMP_EXTRACTED)/DONE

merge: $(TMP_MERGED)/DONE

$(MERGE_BIN):
	go build -o $(MERGE_BIN) github.com/kiteco/kiteco/local-pipelines/python-symbol-to-hashes-index/merge

$(TMP_MERGED)/DONE: $(MERGE_BIN) $(TMP_EXTRACTED)/DONE
	@echo "===== Merging symbol index ..."
	./$(MERGE_BIN) --in=$(TMP_EXTRACTED)/symbols --out=$(TMP_MERGED)/symbols
	@echo "===== Merge canonical symbol index ..."
	./$(MERGE_BIN) --in=$(TMP_EXTRACTED)/canonical-symbols --out=$(TMP_MERGED)/canonical-symbols
	@echo "done" > $(TMP_MERGED)/DONE

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

upload: $(KFSPUT_BIN) $(TMP_MERGED)/DONE
	@echo "===== Uploading index ..."
	$(KFSPUT_BIN) $(TMP_MERGED)/ $(UPLOAD_PATH)/%t
