
OUT_DIR := /data/kite-local-pipelines/python-extract-param-constants
BIN_DIR := ./bin
DIRS_TO_CREATE := $(OUT_DIR) $(BIN_DIR)

EXTRACT_BIN := $(BIN_DIR)/extract
KFSPUT_BIN := $(BIN_DIR)/kfsput

UPLOAD_PATH := s3://kite-local-pipelines/python-extract-param-constants

EXTRACT_TAG := extract-completed


$(shell mkdir -p $(DIRS_TO_CREATE))

default: all

all: extract

clean:
	rm -rf $(OUT_DIR)
	rm -rf $(BIN_DIR)

extract: $(OUT_DIR)/$(EXTRACT_TAG)

$(EXTRACT_BIN):
	go build -o $(EXTRACT_BIN) github.com/kiteco/kiteco/local-pipelines/python-extract-param-constants

$(OUT_DIR)/$(EXTRACT_TAG): $(EXTRACT_BIN)
	@echo "===== Extracting keyword param constants ..."
	./$(EXTRACT_BIN) --outbase=$(OUT_DIR)
	# need to put contents into the tag otherwise kfsput breaks for some reason
	echo "done" > $(OUT_DIR)/$(EXTRACT_TAG)

$(KFSPUT_BIN):
	go build -o $(KFSPUT_BIN) github.com/kiteco/kiteco/kite-go/cmds/kfsput

upload: $(KFSPUT_BIN) $(OUT_DIR)/$(EXTRACT_TAG)
	@echo "===== Uploading index ..."
	$(KFSPUT_BIN) $(OUT_DIR) $(UPLOAD_PATH)/%t
