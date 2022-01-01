# directory under which intermediate stuff will be created
PIPELINE_DIR := .pipeline
TEMP_DIR := $(PIPELINE_DIR)/tmp
ARTIFACTS_DIR := $(PIPELINE_DIR)/artifacts

# the December 2016 stackoverflow data dump, after
# processing with so-transcode to convert to gob:
# https://archive.org/details/stackexchange
POSTS := "s3://kite-data/stackoverflow/2017-03-09_01-11-09-PM/posts.gob.gz"

# path to index
INDEX_FILENAME := $(ARTIFACTS_DIR)/index.json.diskmap

# S3 paths to upload
INDEX_DEST := "s3://kite-data/stackoverflow/%t/index.json.diskmap"


default: all

force:

all: index

index: $(INDEX_FILENAME)

clean: force
	rm -rf $(TEMP_DIR)
	rm -rf $(ARTIFACTS_DIR)

$(TEMP_DIR):
	mkdir -p $(TEMP_DIR)

$(ARTIFACTS_DIR):
	mkdir -p $(ARTIFACTS_DIR)

$(INDEX_FILENAME): $(ARTIFACTS_DIR)
	@echo "=== building the index ..."
	go install github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/so-index-nodes
	go install github.com/kiteco/kiteco/kite-go/cmds/kfsput
	so-index-nodes $(POSTS) "$(INDEX_FILENAME).tmp"
	mv "$(INDEX_FILENAME).tmp" $(INDEX_FILENAME)
	kfsput $(INDEX_FILENAME) $(INDEX_DEST)
