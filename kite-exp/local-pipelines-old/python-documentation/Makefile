# directory under which intermediate stuff will be created
PIPELINE_DIR := /tmp/python-documentation-pipeline
BUILD_TEMPORARIES := $(PIPELINE_DIR)/tmp
BUILD_ARTIFACTS := $(PIPELINE_DIR)/build

# paths to python virtualenvs
ENVPY3 := $(PIPELINE_DIR)/envpy3

ENVPY3_CONSTRUCT_TAG := $(ENVPY3)/maketag
ENVPY3_INSTALL_TAG := $(ENVPY3)/installtag

DOCSTRINGS_FILENAME := $(BUILD_ARTIFACTS)/docstrings.json.gz
DOCSTRINGS_HTML_FILENAME := $(BUILD_ARTIFACTS)/docstrings-html.json.gz
DOCSTRINGS_STRUCTURED_FILENAME := $(BUILD_ARTIFACTS)/pythondocstrings.gob.gz

clean:
	rm -rf $(BUILD_TEMPORARIES)
	rm -rf $(BUILD_ARTIFACTS)
	rm -rf $(ENVPY3_INSTALL_TAG)

cleanclean:
	rm -rf $(PIPELINE_DIR)

virtualenvs: $(ENVPY3_INSTALL_TAG)

clean_virtualenvs:
	rm -rf $(ENVPY3)

$(ENVPY3_CONSTRUCT_TAG):
	@echo "==== building python3 virtualenv..."
	mkdir -p $(dir $(ENVPY3))
	rm -f $(ENVPY3_CONSTRUCT_TAG)
	virtualenv -p python3.4 $(ENVPY3)
	bash -c "source $(ENVPY3)/bin/activate && pip install -r requirements.txt && cd ../../kite-python && ./setup.py install"
	touch $(ENVPY3_CONSTRUCT_TAG)

$(ENVPY3_INSTALL_TAG): $(ENVPY3_CONSTRUCT_TAG)
	@echo "==== installing kite into virtualenv..."
	rm -f $(ENVPY3_INSTALL_TAG)
	bash -c "source $(ENVPY3)/bin/activate && cd ../../kite-python && ./setup.py install"
	touch $(ENVPY3_INSTALL_TAG)

upload_docs: $(ENVPY3_INSTALL_TAG) $(DOCSTRINGS_STRUCTURED_FILENAME)
	@echo "==== uploading to datasets/documentation/python..."
	bash -c "source $(ENVPY3)/bin/activate && timestamped_upload.py --source=$(DOCSTRINGS_STRUCTURED_FILENAME) --dest=datasets/documentation/python"

$(DOCSTRINGS_FILENAME):
	@echo "==== downloading docstrings from S3..."
	mkdir -p $(BUILD_ARTIFACTS)
	go install github.com/kiteco/kiteco/kite-go/lang/python/pythondocs/cmds/download-docstrings
	download-docstrings --output $@

$(DOCSTRINGS_HTML_FILENAME): $(ENVPY3_INSTALL_TAG) $(DOCSTRINGS_FILENAME)
	@echo "==== converting docstrings to HTML..."
	bash -c "source $(ENVPY3)/bin/activate && docstrings_to_html.py --input $(DOCSTRINGS_FILENAME) --output $@"

$(DOCSTRINGS_STRUCTURED_FILENAME): $(DOCSTRINGS_HTML_FILENAME)
	@echo "==== parsing HTML into structured docs..."
	go install github.com/kiteco/kiteco/local-pipelines/python-documentation/parse-docstrings
	parse-docstrings --input $(DOCSTRINGS_HTML_FILENAME) --output $@
