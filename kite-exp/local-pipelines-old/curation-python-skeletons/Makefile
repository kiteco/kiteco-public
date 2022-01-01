KITECO=$(GOPATH)/src/github.com/kiteco/kiteco

# directory for intermediate stuff
PIPELINE_DIR := .artifacts
ARTIFACTS_DIR := $(PIPELINE_DIR)/build

# paths to build artifacts
SKELETONS_FILENAME := $(ARTIFACTS_DIR)/skeletons.gob.gz

# path to curated skeletons
CURATED_SKELETONS := $(HOME)/curation-team/python-skeletons

# path to curation validation tool binary
CURATED_VALIDATION_TOOL_FILENAME := $(CURATED_SKELETONS)/validate-build/validate-build

default: all

all: skeletons bindata

skeletons: $(SKELETONS_FILENAME)

clean:
	rm -rf $(ARTIFACTS_DIR)

$(PIPELINE_DIR):
	mkdir -p $(PIPELINE_DIR)

$(ARTIFACTS_DIR): $(PIPELINE_DIR)
	mkdir -p $(ARTIFACTS_DIR)

$(SKELETONS_FILENAME): $(ARTIFACTS_DIR)
	@echo "=== building skeletons ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons/cmds/build-skeletons
	build-skeletons $(CURATED_SKELETONS) $(SKELETONS_FILENAME)

validate: skeletons
	@echo "=== validating skeletons ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons/cmds/validate-skeletons
	validate-skeletons $(SKELETONS_FILENAME)

bindata: skeletons
	@echo "=== generating skeleton bindata ..."
	go-bindata -o $(KITECO)/kite-go/lang/python/pythonskeletons/bindata.go --prefix $(ARTIFACTS_DIR) -pkg pythonskeletons $(SKELETONS_FILENAME)

curation-validation-tool: $(CURATED_VALIDATION_TOOL_FILENAME)

$(CURATED_VALIDATION_TOOL_FILENAME):
	@echo "=== generating validation tool for curators ..."
	go build github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons/cmds/validate-build
	cp validate-build $(CURATED_VALIDATION_TOOL_FILENAME)

clean-curation-validation-tool:
	rm $(CURATED_VALIDATION_TOOL_FILENAME)