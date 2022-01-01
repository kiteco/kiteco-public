
BIN_DIR := ./bin
TMP_DIR := /data/kite-local-pipelines/python-call-patterns/tmp
DIRS_TO_CREATE := $(BIN_DIR) $(TMP_DIR)

TMP_EXTRACTED := $(TMP_DIR)/extracted
TMP_BUILT := $(TMP_DIR)/built

EXTRACT_BIN := $(BIN_DIR)/extractcalls
BUILD_BIN := $(BIN_DIR)/buildpatterns
KFSPUT_BIN := $(BIN_DIR)/kfsput

UPLOAD_PATH := s3://kite-local-pipelines/python-call-patterns

CACHE_ROOT := /data


$(shell mkdir -p $(DIRS_TO_CREATE))

default: patterns

all: patterns

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(TMP_DIR)

extract: $(TMP_EXTRACTED)/DONE

$(EXTRACT_BIN):
	go build -o $(EXTRACT_BIN) github.com/kiteco/kiteco/local-pipelines/python-call-patterns/extractcalls
$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput
$(BUILD_BIN):
	go build -o $(BUILD_BIN) github.com/kiteco/kiteco/local-pipelines/python-call-patterns/buildpatterns

$(TMP_EXTRACTED)/DONE: $(EXTRACT_BIN)
	@echo "===== Extracting calls ..."
	./$(EXTRACT_BIN) --cacheroot=$(CACHE_ROOT) --out=$(TMP_EXTRACTED)/calls
	touch $(TMP_EXTRACTED)/DONE

patterns: $(TMP_BUILT)/DONE

$(TMP_BUILT)/DONE: $(BUILD_BIN) $(TMP_EXTRACTED)/DONE
	@echo "===== Building patterns ..."
	./$(BUILD_BIN) --cacheroot=$(CACHE_ROOT) --in=$(TMP_EXTRACTED)/calls --out=$(TMP_BUILT)/patterns
	touch $(TMP_BUILT)/DONE

upload: $(KFSPUT_BIN) $(TMP_BUILT)/DONE
	@echo "===== Uploading index ..."
	$(KFSPUT_BIN) $(TMP_BUILT)/patterns $(UPLOAD_PATH)/%t


# Debugging and validation commands
FORCE:

recall: FORCE $(TMP_BUILT)/DONE
	@echo "===== Measuring pattern recall ..."
	# NOTE: for convenience we just build the binary here since it is iterated on relatively often
	go run github.com/kiteco/kiteco/local-pipelines/python-call-patterns/recall \
		--cacheroot=$(CACHE_ROOT) --in=$(TMP_BUILT)/patterns

inspect: FORCE $(TMP_BUILT)/DONE $(TMP_EXTRACTED)/DONE
	# NOTE: for convenience we just build the binary here since it is iterated on relatively often
	go run github.com/kiteco/kiteco/local-pipelines/python-call-patterns/inspect \
		--cacheroot=$(CACHE_ROOT) --calls=$(TMP_EXTRACTED)/calls --patterns=$(TMP_BUILT)/patterns
