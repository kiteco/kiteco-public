# See https://quip.com/IpazA83phPyF

# directory under which intermediate stuff will be created
PIPELINE_DIR := /var/kite/tmp/curation-related-examples-pipeline
BUILD_TEMPORARIES := $(PIPELINE_DIR)/tmp
BUILD_ARTIFACTS := $(PIPELINE_DIR)/build

# paths to python virtualenvs
ENVPY2 := $(PIPELINE_DIR)/envpy2
ENVPY3 := $(PIPELINE_DIR)/envpy3

ENVPY2_CONSTRUCT_TAG := $(ENVPY2)/maketag
ENVPY2_INSTALL_TAG := $(ENVPY2)/installtag
ENVPY3_CONSTRUCT_TAG := $(ENVPY3)/maketag
ENVPY3_INSTALL_TAG := $(ENVPY3)/installtag

# paths to build artifacts
DUMP_FILENAME := $(BUILD_ARTIFACTS)/curated-snippets-dump.json.gz
SAMPLE_FILES_FILENAME := $(BUILD_ARTIFACTS)/sample-files.json.gz
CURATED_FILENAME := $(BUILD_ARTIFACTS)/curated-snippets.emr
CURATED_WITH_ATTRIBUTES_FILENAME := $(BUILD_ARTIFACTS)/curated-snippets-attributes.emr
RELATED_FILENAME := $(BUILD_ARTIFACTS)/related-examples.json.gz
ANNOTATE_FILENAME := $(BUILD_ARTIFACTS)/annotated-asts.json.gz
FUNCTION_NAMES_FILENAME := $(BUILD_ARTIFACTS)/function-names.json.gz
RETURN_TYPES_FILENAME := $(BUILD_ARTIFACTS)/return-types.json.gz
ATTRIBUTES_FILENAME := $(BUILD_ARTIFACTS)/attributes.json.gz
ERRORS_FILENAME := $(BUILD_ARTIFACTS)/stderr.txt
TRACE_FILENAME := $(BUILD_ARTIFACTS)/traced-references.json.gz

# paths for word2vec
WORD2VEC_MODEL_FILENAME := $(BUILD_TEMPORARIES)/word2vec.model
WORD2VEC_DATA_FILENAME := $(BUILD_TEMPORARIES)/full-train.txt
WORD2VEC_STD_FILENAME := /var/kite/data/code_examples/word2vec/python-train.txt

# paths for relatedness classifier
SVM_KERNAL := linear
SVM_DIR := $(BUILD_TEMPORARIES)/svm
SVM_MODEL_FILENAME := $(SVM_DIR)/$(SVM_KERNAL)_svm_model.pkl
SVM_TRAIN_DATA := /var/kite/data/relatedness_classifier/data/train.txt

# path to the uncompressed related examples dict
RELATED_FILENAME_RAW := $(BUILD_TEMPORARIES)/related-examples.json

DUMP_SNIPPETS_ARGS ?= -annotations -dockerimage=kiteco/pythonsandbox -approvedOnly
ATTRIBUTES_ARGS ?= -perSnippet
RETURN_TYPES_ARGS ?= --collate=false

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


default: all

force:

all: related dump snippets types trace_references

dump: $(DUMP_FILENAME)

sample_files: $(SAMPLE_FILES_FILENAME)

snippets: $(CURATED_FILENAME)

snippet_attributes: $(CURATED_WITH_ATTRIBUTES_FILENAME)

classifier: $(SVM_MODEL_FILENAME)

related: $(RELATED_FILENAME)

annotate: $(ANNOTATE_FILENAME)

fqn: $(FUNCTION_NAMES_FILENAME)

types: $(RETURN_TYPES_FILENAME)

trace_references: $(TRACE_FILENAME)

attributes: $(ATTRIBUTES_FILENAME)

upload_test: $(CURATED_FILENAME) $(SAMPLE_FILES_FILENAME) $(RELATED_FILENAME) $(ENVPY3_INSTALL_TAG)
	@echo "==== uploading to testing/curated-snippets..."
	bash -c "source $(ENVPY3)/bin/activate && timestamped_upload.py --source=$(BUILD_ARTIFACTS) --dest=testing/curated-snippets"

upload_prod: $(CURATED_FILENAME) $(SAMPLE_FILES_FILENAME) $(RELATED_FILENAME) $(ENVPY3_INSTALL_TAG)
	@echo "==== uploading to datasets/curated-snippets..."
	bash -c "source $(ENVPY3)/bin/activate && timestamped_upload.py --source=$(BUILD_ARTIFACTS) --dest=datasets/curated-snippets"

clean: force
	rm -rf $(BUILD_TEMPORARIES)
	rm -rf $(BUILD_ARTIFACTS)
	rm -rf $(ENVPY2_INSTALL_TAG)
	rm -rf $(ENVPY3_INSTALL_TAG)

cleanclean: force
	rm -rf $(PIPELINE_DIR)

install_deps:
	@echo "Installing with apt-get, password may be required...."
	sudo apt-get install gfortran pkg-config liblapack-dev libblas-dev libpython-dev libpython3-dev

virtualenvs: $(ENVPY2_INSTALL_TAG) $(ENVPY3_INSTALL_TAG)

clean_virtualenvs:
	rm -rf $(ENVPY2) $(ENVPY3)

$(ENVPY3_CONSTRUCT_TAG): requirements.txt
	@echo "==== building python3 virtualenv..."
	mkdir -p $(dir $(ENVPY3))
	rm -f $(ENVPY3_CONSTRUCT_TAG)
	virtualenv -p python3.4 $(ENVPY3)
	bash -c "source $(ENVPY3)/bin/activate && pip install numpy==1.9.2 && pip install scipy==0.15.1 && pip install -r requirements.txt && cd ../../kite-python && ./setup.py install"
	touch $(ENVPY3_CONSTRUCT_TAG)

$(ENVPY3_INSTALL_TAG): $(ENVPY3_CONSTRUCT_TAG)
	@echo "==== installing kite into virtualenv..."
	rm -f $(ENVPY3_INSTALL_TAG)
	bash -c "source $(ENVPY3)/bin/activate && cd ../../kite-python && ./setup.py install"
	touch $(ENVPY3_INSTALL_TAG)

$(ENVPY2_CONSTRUCT_TAG): requirements.txt
	@echo "==== building python2 virtualenv..."
	mkdir -p $(dir $(ENVPY2))
	rm -f $(ENVPY2_CONSTRUCT_TAG)
	virtualenv -p python2.7 $(ENVPY2)
	bash -c "source $(ENVPY2)/bin/activate && pip install numpy==1.9.2 && pip install scipy==0.15.1 && pip install -r requirements.txt && cd ../../kite-python && pip install -r requirements-emr.txt && ./setup.py install"
	touch $(ENVPY2_CONSTRUCT_TAG)

$(ENVPY2_INSTALL_TAG): $(ENVPY2_CONSTRUCT_TAG)
	@echo "==== installing kite into virtualenv..."
	rm -f $(ENVPY2_INSTALL_TAG)
	bash -c "source $(ENVPY2)/bin/activate && cd ../../kite-python && ./setup.py install"
	touch $(ENVPY2_INSTALL_TAG)

$(DUMP_FILENAME):
	@echo "==== dumping curated snippets from database to $@..."
	go install github.com/kiteco/kiteco/kite-go/curation/cmds/dump-snippets
	mkdir -p $(BUILD_ARTIFACTS)
	dump-snippets -output=$@ $(DUMP_SNIPPETS_ARGS)

$(SAMPLE_FILES_FILENAME):
	@echo "==== dumping sample files from S3 to $@..."
	go install github.com/kiteco/kiteco/kite-go/curation/cmds/sample-files
	mkdir -p $(BUILD_ARTIFACTS)
	sample-files -output=$@

$(ANNOTATE_FILENAME): $(CURATED_FILENAME)
	@echo "==== infering types on curated snippets..."
	go build github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/runtime-inferencer
	./runtime-inferencer --examples $(CURATED_FILENAME) --image=kiteco/pythonsandbox --output $(ANNOTATE_FILENAME) 2> $(ERRORS_FILENAME)

$(FUNCTION_NAMES_FILENAME): $(ANNOTATE_FILENAME)
	@echo "=== extracting mapping from expressions to fully qualified names..."
	go build github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/live-transformer
	./live-transformer --input $(ANNOTATE_FILENAME) --output $(FUNCTION_NAMES_FILENAME) $(RETURN_TYPES_ARGS)

$(RETURN_TYPES_FILENAME): $(ANNOTATE_FILENAME)
	@echo "=== extracting mapping from fully qualified names to return types..."
	go build github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/github-transformer
	./github-transformer --input $(ANNOTATE_FILENAME) --output $(RETURN_TYPES_FILENAME)

$(ATTRIBUTES_FILENAME): $(ANNOTATE_FILENAME)
	@echo "=== extracting mapping from types to attributes called on them..."
	go build github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/attributes-transformer
	./attributes-transformer --input $(ANNOTATE_FILENAME) --output $(ATTRIBUTES_FILENAME) $(ATTRIBUTES_ARGS)

$(CURATED_WITH_ATTRIBUTES_FILENAME): $(CURATED_FILENAME) $(ATTRIBUTES_FILENAME)
	@echo "=== populating snippets with attribute data..."
	go run attributes/main.go --snippets $(CURATED_FILENAME) --attributes $(ATTRIBUTES_FILENAME) --output $(CURATED_WITH_ATTRIBUTES_FILENAME)

$(CURATED_FILENAME): $(DUMP_FILENAME) $(ENVPY2_INSTALL_TAG)
	@echo "==== parsing curated snippets..."
	go install github.com/kiteco/kiteco/kite-go/cmds/jsongz_to_emr
	bash -c "source $(ENVPY2)/bin/activate && cat $(DUMP_FILENAME) | jsongz_to_emr | build_curated_snippets.py > $(CURATED_FILENAME)"

$(WORD2VEC_DATA_FILENAME): $(CURATED_FILENAME) word2vec-data/main.go
	@echo "==== generating training data from snippets for word2vec model..."
	mkdir -p $(dir $(WORD2VEC_DATA_FILENAME))
	go run word2vec-data/main.go -curated $(CURATED_FILENAME) -output $(WORD2VEC_DATA_FILENAME)

check_fast_word2vec: $(ENVPY3_INSTALL_TAG)
	bash -c "source $(ENVPY3)/bin/activate && check_fast_word2vec.py"

$(WORD2VEC_MODEL_FILENAME): $(WORD2VEC_DATA_FILENAME) $(ENVPY3_INSTALL_TAG) check_fast_word2vec
	@echo "==== training word2vec model..."
	bash -c "source $(ENVPY3)/bin/activate && train_word2vec.py --input $(WORD2VEC_STD_FILENAME) $(WORD2VEC_DATA_FILENAME) --output $(WORD2VEC_MODEL_FILENAME)"

$(SVM_MODEL_FILENAME): $(WORD2VEC_MODEL_FILENAME) $(ENVPY3_INSTALL_TAG)
	@echo "==== building classifier..."
	mkdir -p $(SVM_DIR)
	bash -c "source $(ENVPY3)/bin/activate && build_relatedness_classifier.py --train_input $(SVM_TRAIN_DATA) --word2vec_model $(WORD2VEC_MODEL_FILENAME) --output_dir $(SVM_DIR)"

$(RELATED_FILENAME_RAW): $(SVM_MODEL_FILENAME) $(WORD2VEC_MODEL_FILENAME) $(CURATED_FILENAME) classifier/classification.py dispatcher/main.go $(ENVPY3_INSTALL_TAG)
	@echo "==== building related examples map..."
	bash -c "source $(ENVPY3)/bin/activate && go run dispatcher/main.go --file $(CURATED_FILENAME) | sort -k1,1 | python classifier/classification.py --fout $(RELATED_FILENAME_RAW) --svm $(SVM_MODEL_FILENAME) --word2vec $(WORD2VEC_MODEL_FILENAME)"

$(RELATED_FILENAME): $(RELATED_FILENAME_RAW)
	# Do not gzip the file "in place" because that will trigger the original to be rebuilt by make next time
	gzip -c $(RELATED_FILENAME_RAW) > $(RELATED_FILENAME)

$(TRACE_FILENAME):
	@echo "=== tracing references using dynamic analysis ==="
	go generate github.com/kiteco/kiteco/kite-go/dynamicanalysis
	go install github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/trace-references
	trace-references \
		-input $(DUMP_FILENAME) \
		-output $(TRACE_FILENAME) \
		$(TRACE_ARGS)
