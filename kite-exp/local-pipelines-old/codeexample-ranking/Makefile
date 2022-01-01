# directory under which intermediate stuff will be created
PIPELINE_DIR := /tmp/codeexample-ranking-pipleline
BUILD_TEMPORARIES := $(PIPELINE_DIR)/tmp
BUILD_ARTIFACTS := $(PIPELINE_DIR)/build

# python parameters
PYTHON_VERSION := python2.7

# paths to python virtualenvs
ENVPY2 := $(PIPELINE_DIR)/envpy2
ENVPY2_CONSTRUCT_TAG := $(ENVPY2)/construct_tag
ENVPY2_INSTALL_TAG := $(ENVPY2)/install_tag

# paths to build artifacts
SNIPPET_ID_FILENAME := $(BUILD_TEMPORARIES)/snapshot-ids.txt
RANKING_DUMP_FILENAME := $(BUILD_ARTIFACTS)/ranked-snippets-dump.json.gz
RANKING_CURATED_FILENAME := $(BUILD_ARTIFACTS)/ranked-snippets.emr
RANKING_ANNOTATE_FILENAME := $(BUILD_ARTIFACTS)/ranked-annotated-asts.json.gz
RANKING_ATTRIBUTES_FILENAME := $(BUILD_ARTIFACTS)/ranked-attributes.json.gz
RANKING_WITH_ATTRIBUTES_FILENAME := $(BUILD_ARTIFACTS)/ranked-snippets-attributes.emr
RANKING_ERRORS_FILENAME := $(BUILD_ARTIFACTS)/stderr.txt

DUMP_FILENAME := $(BUILD_ARTIFACTS)/curated-snippets-dump.json.gz
CURATED_FILENAME := $(BUILD_ARTIFACTS)/curated-snippets.emr

# paths to build artifacts
ANALYSIS_RAW_FILENAME := $(BUILD_TEMPORARIES)/import-analysis.json
ANALYSIS_FILENAME := $(BUILD_ARTIFACTS)/import-analysis.json.gz

# args to pass to dump-snippets
DUMP_SNIPPETS_ARGS ?= -annotations=0
ATTRIBUTES_ARGS ?= -perSnippet

# args to pass to build-features
BASENAME ?= test
TRAIN_FEATFILE := $(BUILD_ARTIFACTS)/$(BASENAME)-train.json
TEST_FEATFILE := $(BUILD_ARTIFACTS)/$(BASENAME)-test.json

ifeq ($(AWS_ACCESS_KEY_ID),)
	echo "AWS_ACCESS_KEY_ID not set"
	exit
endif

ifeq ($(AWS_SECRET_ACCESS_KEY),)
	echo "AWS_SECRET_ACCESS_KEY not set"
	exit
endif

ifeq ($(CODEEXAMPLE_DB_DRIVER),)
	echo "CODEEXAMPLE_DB_DRIVER not set"
	exit
endif

ifeq ($(CODEEXAMPLE_DB_URI),)
	echo "CODEEXAMPLE_DB_URI not set"
	exit
endif


ifeq ($(RANKING_DB_DRIVER),)
	echo "RANKING_DB_DRIVER not set"
	exit
endif

ifeq ($(RANKING_DB_URI),)
	echo "RANKING_DB_URI not set"
	exit
endif


default: all

force:

all: ids ranking_dump ranking_snippets ranking_attributes dump snippets featurer

ids: $(SNIPPET_ID_FILENAME)

dump: $(DUMP_FILENAME)

snippets: $(CURATED_FILENAME)

ranking_dump: $(RANKING_DUMP_FILENAME)

ranking_snippets: $(RANKING_CURATED_FILENAME)

ranking_attributes: $(RANKING_WITH_ATTRIBUTES_FILENAME)

featurer: test.txt exclude.txt features.txt $(RANKING_CURATED_FILENAME) $(CURATED_FILENAME) 
	@echo "==== generating features..."
	go install github.com/kiteco/kiteco/kite-go/codeexample/cmds/build-features
	build-features --testq test.txt --exclude exclude.txt --labelled $(RANKING_CURATED_FILENAME) --curated $(CURATED_FILENAME) --basename $(BASENAME) --root $(BUILD_ARTIFACTS)

upload_test: $(RANKING_CURATED_FILENAME) $(CURATED_FILENAME) $(ENVPY3_TAG) 
	@echo "==== uploading to testing/rankers/code-examples..."
	bash -c "source $(ENVPY3)/bin/activate && timestamped_upload.py --source=$(BUILD_ARTIFACTS) --dest=testing/rankers/code-examples"

upload_prod: $(RANKING_CURATED_FILENAME) $(CURATED_FILENAME) $(ENVPY3_TAG)
	@echo "==== uploading to ranker/code-examples..."
	bash -c "source $(ENVPY3)/bin/activate && timestamped_upload.py --source=$(BUILD_ARTIFACTS) --dest=rankers/code-examples"

$(SNIPPET_ID_FILENAME):
	mkdir -p $(dir $(SNIPPET_ID_FILENAME))
	@echo "==== dumping snapshot ids from the ranking database..."
	go install github.com/kiteco/kiteco/kite-go/ranking/cmds/get-snippet-ids
	get-snippet-ids -output $(SNIPPET_ID_FILENAME)
 
$(RANKING_DUMP_FILENAME): $(SNIPPET_ID_FILENAME)
	@echo "==== dumping curated snippets from database to $@..."
	go install github.com/kiteco/kiteco/kite-go/curation/cmds/dump-snippets
	mkdir -p $(BUILD_ARTIFACTS)
	dump-snippets -output=$@ -snapshotids $(SNIPPET_ID_FILENAME) 

$(RANKING_CURATED_FILENAME): $(RANKING_DUMP_FILENAME) $(ENVPY2_INSTALL_TAG)
	@echo "==== parsing curated snippets..."
	go install github.com/kiteco/kiteco/kite-go/cmds/jsongz_to_emr
	bash -c "source $(ENVPY2)/bin/activate && cat $(RANKING_DUMP_FILENAME) | jsongz_to_emr | build_curated_snippets.py > $(RANKING_CURATED_FILENAME)"

$(DUMP_FILENAME):
	@echo "==== dumping curated snippets from database to $@..."
	go install github.com/kiteco/kiteco/kite-go/curation/cmds/dump-snippets
	mkdir -p $(BUILD_ARTIFACTS)
	dump-snippets -output=$@ $(DUMP_SNIPPETS_ARGS)

$(CURATED_FILENAME): $(DUMP_FILENAME) $(ENVPY2_INSTALL_TAG)
	@echo "==== parsing curated snippets..."
	go install github.com/kiteco/kiteco/kite-go/cmds/jsongz_to_emr
	bash -c "source $(ENVPY2)/bin/activate && cat $(DUMP_FILENAME) | jsongz_to_emr | build_curated_snippets.py > $(CURATED_FILENAME)"

$(RANKING_ANNOTATE_FILENAME): $(RANKING_CURATED_FILENAME)
	@echo "==== infering types on curated snippets..."
	go build github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/runtime-inferencer
	./runtime-inferencer --examples $(RANKING_CURATED_FILENAME) --image=kiteco/pythonsandbox --output $(RANKING_ANNOTATE_FILENAME) 2> $(RANKING_ERRORS_FILENAME)

$(RANKING_ATTRIBUTES_FILENAME): $(RANKING_ANNOTATE_FILENAME)
	@echo "=== extracting mapping from types to attributes called on them..."
	go build github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/attributes-transformer
	./attributes-transformer --input $(RANKING_ANNOTATE_FILENAME) --output $(RANKING_ATTRIBUTES_FILENAME) $(ATTRIBUTES_ARGS)

$(RANKING_WITH_ATTRIBUTES_FILENAME): $(RANKING_CURATED_FILENAME) $(RANKING_ATTRIBUTES_FILENAME) 
	@echo "=== populating snippets with attribute data..."
	go run attributes/main.go --snippets $(RANKING_CURATED_FILENAME) --attributes $(RANKING_ATTRIBUTES_FILENAME) --output $(RANKING_WITH_ATTRIBUTES_FILENAME)
clean: force
	rm -rf $(BUILD_TEMPORARIES)
	rm -rf $(BUILD_ARTIFACTS)

clean_virtualenv:
	rm -rf $(ENVPY2)

$(ENVPY2_INSTALL_TAG): $(ENVPY2_CONSTRUCT_TAG)
	@echo "==== installing kite into virtualenv..."
	rm -f $(ENVPY2_INSTALL_TAG)
	bash -c "source $(ENVPY2)/bin/activate && cd ../../kite-python && ./setup.py install"
	touch $(ENVPY2_INSTALL_TAG)

$(ENVPY2_CONSTRUCT_TAG): requirements.txt
	@echo "==== building virtualenv..."
	mkdir -p $(dir $(ENVPY2))
	rm -f $(ENVPY2_CONSTRUCT_TAG)
	virtualenv -p $(PYTHON_VERSION) $(ENVPY2)
	bash -c "source $(ENVPY2)/bin/activate && pip install numpy==1.9.2 && pip install -r requirements.txt"
	touch $(ENVPY2_CONSTRUCT_TAG)
