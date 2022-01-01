# See https://quip.com/IpazA83phPyF

# python parameters
PYTHON_VERSION := python2.7

# directory under which intermediate stuff will be created
PIPELINE_DIR := /tmp/validate-type-inference-pipeline
BUILD_TEMPORARIES := $(PIPELINE_DIR)/tmp

# paths to build artifacts
TRACE_FILENAME := $(BUILD_TEMPORARIES)/traced-snippets.json
DIFF_FILENAME := $(BUILD_TEMPORARIES)/report.html

# S3 paths to upload
UPLOAD_DEST := "s3://kite-data/type-inference-models"

# Path to json file containing snippets
SNIPPET_DUMP_FILENAME := s3://kite-emr/datasets/curated-snippets/2015-08-05_10-32-54-AM/curated-snippets-dump.json.gz


ifeq ($(AWS_ACCESS_KEY_ID),)
	echo "AWS_ACCESS_KEY_ID not set"
	exit
endif

ifeq ($(AWS_SECRET_ACCESS_KEY),)
	echo "AWS_SECRET_ACCESS_KEY not set"
	exit
endif


default: all

force:

all: analysis tables

trace: $(TRACE_FILENAME)

report: $(FUNCTION_TABLE_FILENAME) $(TYPE_TABLE_FILENAME) $(ALIASES_FILENAME)

open: report
	open 

$(TRACE_FILENAME):
	go install github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/trace-references
	trace-references -input $(SNIPPET_DUMP_FILENAME) -output $(TRACE_FILENAME)

upload: all
	@echo "==== uploading to $(UPLOAD_DEST)..."
	bash -c "source $(ENVPY2)/bin/activate && timestamped_upload.py --source=$(BUILD_ARTIFACTS) --dest=$(UPLOAD_DEST)"

clean: force
	rm -rf $(BUILD_TEMPORARIES)
	rm -rf $(BUILD_ARTIFACTS)
	rm -f $(ENVPY2_TAG)
	rm -f $(ENVPY3_TAG)

clean_artifacts: force
	rm -rf $(BUILD_ARTIFACTS)
	rm -f $(ENVPY2_TAG)
	rm -f $(ENVPY3_TAG)

clean_temporaries: force
	rm -rf $(BUILD_TEMPORARIES)
	rm -f $(ENVPY2_TAG)
	rm -f $(ENVPY3_TAG)

clean_virtualenvs:
	rm -rf $(ENVPY2)
	rm -rf $(ENVPY3)

virtualenvs: $(ENVPY2_TAG) $(ENVPY3_TAG)

install_deps:
	@echo "=== installing dependencies with apt-get, password may be required...."
	sudo apt-get install gfortran pkg-config liblapack-dev libblas-dev libpython-dev libpython3-dev libfreetype6-dev libpng-dev libxml2-dev libxslt-dev libzmq-dev

$(ENVPY2_TAG): $(ENVPY2_DEPS_TAG)
	@echo "==== installing kite into python2 virtualenv..."
	rm -f $(ENVPY2_TAG)
	bash -c "source $(ENVPY2)/bin/activate && cd ../../kite-python && ./setup.py install && ./setup.py clean"
	touch $(ENVPY2_TAG)

$(ENVPY2_DEPS_TAG): requirements.txt
	@echo "==== building python2 virtualenv..."
	mkdir -p $(dir $(ENVPY2))
	rm -f $(ENVPY2_DEPS_TAG)
	virtualenv -p $(PYTHON_VERSION) $(ENVPY2)
	bash -c "source $(ENVPY2)/bin/activate && pip install numpy==1.9.2 && pip install -r requirements.txt"
	touch $(ENVPY2_DEPS_TAG)

$(ENVPY3_TAG): $(ENVPY3_DEPS_TAG)
	@echo "==== installing kite into python3 virtualenv..."
	rm -f $(ENVPY3_TAG)
	bash -c "source $(ENVPY3)/bin/activate && cd ../../kite-python && ./setup.py install && ./setup.py clean"
	touch $(ENVPY3_TAG)

$(ENVPY3_DEPS_TAG): requirements.txt
	@echo "==== building python3 virtualenv..."
	mkdir -p $(dir $(ENVPY3))
	rm -f $(ENVPY3_DEPS_TAG)
	virtualenv -p $(PYTHON_VERSION) $(ENVPY3)
	bash -c "source $(ENVPY3)/bin/activate && pip install numpy==1.9.2 && pip install -r requirements.txt"
	touch $(ENVPY3_DEPS_TAG)

$(ANALYSIS_FILENAME): packages_to_explore.txt $(ENVPY2_TAG)
	@echo "==== exploring the import tree..."
	mkdir -p $(BUILD_ARTIFACTS) $(BUILD_TEMPORARIES)
	bash -c "source $(ENVPY2)/bin/activate && explore_packages.py $(EXPLORE_PACKAGES_ARGS) --output $(ANALYSIS_RAW_FILENAME)"
	cat $(ANALYSIS_RAW_FILENAME) | gzip > $(ANALYSIS_FILENAME)

$(FILTERED_USAGES_FILENAME):
	@echo "==== grepping the usages dataset..."
	go install github.com/kiteco/kiteco/kite-go/cmds/kfscat
	kfscat $(FULL_USAGES_FILENAME) | grep '$(PACKAGES_REGEX)' > $(FILTERED_USAGES_FILENAME)

$(FUNCTION_TABLE_FILENAME): $(ANALYSIS_FILENAME) $(ENVPY3_TAG) $(FILTERED_USAGES_FILENAME)
	@echo "==== building tables..."
	bash -c "source $(ENVPY3)/bin/activate && learn_types.py --importtree $(ANALYSIS_FILENAME) --usages $(FILTERED_USAGES_FILENAME) --packages $(PACKAGES) --output_funcs $(FUNCTION_TABLE_RAW_FILENAME) --output_types $(TYPE_TABLE_RAW_FILENAME) --output_aliases $(ALIASES_RAW_FILENAME)"
	cat $(FUNCTION_TABLE_RAW_FILENAME) | gzip > $(FUNCTION_TABLE_FILENAME)
	cat $(TYPE_TABLE_RAW_FILENAME) | gzip > $(TYPE_TABLE_FILENAME)
	cat $(ALIASES_RAW_FILENAME) | gzip > $(ALIASES_FILENAME)

$(TYPE_TABLE_FILENAME): $(FUNCTION_TABLE_FILENAME)

$(ALIASES_FILENAME): $(FUNCTION_TABLE_FILENAME)
