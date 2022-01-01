# See https://quip.com/IpazA83phPyF

# directory under which intermediate stuff will be created. To support OSX-based docker-machine this must be under /Users
PIPELINE_DIR := .artifacts
TEMP_DIR := $(PIPELINE_DIR)/tmp
ARTIFACTS_DIR := $(PIPELINE_DIR)/build

# dir in which individual shards of import graph will be created
IMPORT_GRAPH_SHARDS := $(TEMP_DIR)/import-graph-shards

# path to merged import graph
IMPORT_GRAPH_FILENAME := $(ARTIFACTS_DIR)/import-graph.gob.gz
IMPORT_GRAPH_SMALL_FILENAME := $(ARTIFACTS_DIR)/small-import-graph.gob.gz
IMPORT_GRAPH_STRINGS_FILENAME := $(ARTIFACTS_DIR)/import-graph-strings.json.gz
IMPORT_GRAPH_ARG_SPECS_FILENAME := $(ARTIFACTS_DIR)/import-graph-arg-specs.json.gz
IMPORT_GRAPH_SOURCE_FILENAME := $(ARTIFACTS_DIR)/import-graph-source.json.gz
IMPORT_GRAPH_SOURCE_COMPRESSED_FILENAME := $(ARTIFACTS_DIR)/import-graph-source.compressed.gob.gz
IMPORT_DEPS_FILENAME := $(ARTIFACTS_DIR)/import-deps.json.gz

# path to import graph cover
IMPORT_GRAPH_REPORT := $(TEMP_DIR)/import-graph-report.html

# path to type learning models
TYPE_TABLE_FILENAME := $(ARTIFACTS_DIR)/types.json.gz
FUNCTION_TABLE_FILENAME := $(ARTIFACTS_DIR)/functions.json.gz
UPDATE_TABLES_LOG := $(TEMP_DIR)/update-tables.log

TRACE_FILENAME := $(TEMP_DIR)/traced-snippets.json.gz
STATIC_ANALYSIS_REPORT := $(TEMP_DIR)/static-analysis-report.html
TYPE_INDUCTION_REPORT := $(TEMP_DIR)/type-induction-report.html

# Docker image in which import exploration runs
IMPORT_EXPLORATION_IMAGE := kiteco/import-exploration

# Directory containing Dockerfile for import exploration image
IMPORT_EXPLORATION_DEF := ../../docker-images/import-exploration

# The tag used to track updates to the import exploration image
IMPORT_GRAPH_TAG := $(TEMP_DIR)/import-graph-tag
IMPORT_EXPLORATION_IMAGE_TAG := $(TEMP_DIR)/import-exploration-tag

# Path to dynamic usages data (ie dynamic analysis)
DYNAMIC_USAGES_FILENAME := $(TEMP_DIR)/dynamic-usages.json.gz

# Path to the active search index
MODULE_ATTRIBUTES_FILENAME = $(ARTIFACTS_DIR)/module-attributes.json.gz

# Path to the list of identifier names to be indexed
GRAPH_INDEX_FILENAME = $(ARTIFACTS_DIR)/graph-index.json.gz

# Path to static usages data (ie github usages)
STATIC_USAGES_FILENAME := s3://kite-emr/users/tarak/python-code-examples/2015-10-21_13-13-06-PM/merge_group_obj_usages/output/part-00000

# Path to input curated example dump
SNIPPET_DUMP_FILENAME := s3://kite-emr/datasets/curated-snippets/2015-10-14_15-42-14-PM/curated-snippets-dump.json.gz
# S3 paths to upload
UPLOAD_DEST := "s3://kite-data/type-inference-models"

# Args for explore.py
TRACE_ARGS ?=
EVALUATE_ARGS ?=

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

all: import_graph compress_source tables

import_exploration_image: $(IMPORT_EXPLORATION_IMAGE_TAG)

import_graph_small: $(IMPORT_GRAPH_SMALL_FILENAME)

import_graph: $(IMPORT_GRAPH_FILENAME)

compress_source: $(IMPORT_GRAPH_SOURCE_COMPRESSED_FILENAME)

import_deps: $(IMPORT_DEPS_FILENAME)

module_attributes: $(MODULE_ATTRIBUTES_FILENAME)

graph_index: $(GRAPH_INDEX_FILENAME)

import_graph_pipeline: import_graph compress_source import_graph_small import_deps module_attributes graph_index update_tables

function_table: $(FUNCTION_TABLE_FILENAME)

type_table: $(TYPE_TABLE_FILENAME)

tables: function_table type_table

update_tables: import_graph
	@echo "==== updating function tables ..."
	go install github.com/kiteco/kiteco/kite-go/typeinduction/cmds/update-tables
	update-tables --graph $(IMPORT_GRAPH_FILENAME) --verbose $(FUNCTION_TABLE_FILENAME) $(TYPE_TABLE_FILENAME) > $(UPDATE_TABLES_LOG)

dynamic_usages: $(DYNAMIC_USAGES_FILENAME)

trace: $(TRACE_FILENAME)

evaluate_static_analysis: $(STATIC_ANALYSIS_REPORT)

evaluate_type_induction: $(TYPE_INDUCTION_REPORT)

import_graph_coverage: $(IMPORT_GRAPH_REPORT)

show_static_analysis: evaluate_static_analysis
	open $(STATIC_ANALYSIS_REPORT)

show_type_induction: evaluate_type_induction
	open $(TYPE_INDUCTION_REPORT)

show_import_graph: $(IMPORT_GRAPH_FILENAME)
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/import-graph-viewer
	import-graph-viewer --path $(IMPORT_GRAPH_FILENAME) --port ":3021"

upload:
	@echo "==== uploading to $(UPLOAD_DEST)..."
	timestamped_upload.py --source=$(ARTIFACTS_DIR) --dest=$(UPLOAD_DEST)

clean: force
	rm -rf $(TEMP_DIR)
	rm -rf $(ARTIFACTS_DIR)

clean_artifacts: force
	rm -rf $(ARTIFACTS_DIR)

clean_temporaries: force
	rm -rf $(TEMP_DIR)

clean_import_graph:
	rm -f $(IMPORT_GRAPH_TAG)

clean_trace:
	rm -f $(TRACE_FILENAME)

clean_report:
	rm -f $(STATIC_ANALYSIS_REPORT)
	rm -f $(TYPE_INDUCTION_REPORT)
	rm -f $(IMPORT_GRAPH_REPORT)

$(TEMP_DIR):
	mkdir -p $(TEMP_DIR)

$(ARTIFACTS_DIR):
	mkdir -p $(ARTIFACTS_DIR)

$(IMPORT_EXPLORATION_IMAGE_TAG): $(TEMP_DIR)
	@echo "=== building the import graph docker image ..."
	make -f $(IMPORT_EXPLORATION_DEF)/Makefile image
	touch $(IMPORT_EXPLORATION_IMAGE_TAG)

$(IMPORT_GRAPH_TAG): $(TEMP_DIR)
	@echo "==== exploring import graph shards ..."
	rm -rf $(IMPORT_GRAPH_SHARDS) $(IMPORT_GRAPH_TAG)
	mkdir -p $(IMPORT_GRAPH_SHARDS)
	docker run -i --rm \
		-v $(shell pwd):/host \
		-w /host/$(IMPORT_GRAPH_SHARDS) \
		$(IMPORT_EXPLORATION_IMAGE)
	touch $(IMPORT_GRAPH_TAG)

$(IMPORT_GRAPH_FILENAME): $(IMPORT_GRAPH_TAG) $(ARTIFACTS_DIR)
	@echo "==== merging import graph shards ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/merge-import-graphs
	merge-import-graphs \
		--output $(IMPORT_GRAPH_FILENAME) \
		--strings $(IMPORT_GRAPH_STRINGS_FILENAME) \
		--argspecs $(IMPORT_GRAPH_ARG_SPECS_FILENAME) \
		--source $(IMPORT_GRAPH_SOURCE_FILENAME) \
		--shards $(IMPORT_GRAPH_SHARDS)/graph/*.json

$(IMPORT_GRAPH_SOURCE_COMPRESSED_FILENAME): $(IMPORT_GRAPH_FILENAME)
	@echo "=== compressing import graph source ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/compress-import-graph-source
	compress-import-graph-source \
		--input $(IMPORT_GRAPH_SOURCE_FILENAME) \
		--output $(IMPORT_GRAPH_SOURCE_COMPRESSED_FILENAME)

$(IMPORT_GRAPH_SMALL_FILENAME): $(IMPORT_GRAPH_FILENAME)
	@echo "==== building small version of import graph ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/select-subgraph
	select-subgraph \
		--input $(IMPORT_GRAPH_FILENAME) \
		--output $(IMPORT_GRAPH_SMALL_FILENAME) \
		--packages collections json requests

$(IMPORT_DEPS_FILENAME): $(IMPORT_GRAPH_TAG) $(ARTIFACTS_DIR)
	@echo "==== computing dependency graph ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/merge-import-deps
	merge-import-deps \
		-output $(IMPORT_DEPS_FILENAME) \
		$(IMPORT_GRAPH_SHARDS)/deps/*

$(MODULE_ATTRIBUTES_FILENAME): $(IMPORT_GRAPH_FILENAME)
	@echo "==== building attribute index for diff suggestions ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/pythondiffs/cmds/gen-module-attributes
	gen-module-attributes \
		-importgraph $(IMPORT_GRAPH_FILENAME) \
		-output $(MODULE_ATTRIBUTES_FILENAME)

$(GRAPH_INDEX_FILENAME): $(IMPORT_GRAPH_FILENAME)
	@echo "==== building identifier list from graph ..."
	go install github.com/kiteco/kiteco/kite-go/lang/python/pythonindex/cmds/build-graph-index
	build-graph-index \
		-depth 2 \
		-graph $(IMPORT_GRAPH_FILENAME) \
		-output $(GRAPH_INDEX_FILENAME)

$(DYNAMIC_USAGES_FILENAME): $(TEMP_DIR)
	@echo "=== tracing usages using dynamic analysis ..."
	go generate github.com/kiteco/kiteco/kite-go/dynamicanalysis
	go install github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/trace-usages
	trace-usages \
		-input ${SNIPPET_DUMP_FILENAME} \
		-output $(DYNAMIC_USAGES_FILENAME)

$(FUNCTION_TABLE_FILENAME) $(TYPE_TABLE_FILENAME): $(IMPORT_GRAPH_FILENAME) $(IMPORT_DEPS_FILENAME) $(DYNAMIC_USAGES_FILENAME)
	@echo "==== training type models ..."
	go install github.com/kiteco/kiteco/kite-go/typeinduction/cmds/learn-types
	learn-types \
		-dynamicusages $(DYNAMIC_USAGES_FILENAME) \
		-staticusages $(STATIC_USAGES_FILENAME) \
		-importdeps $(IMPORT_DEPS_FILENAME) \
		-importgraph $(IMPORT_GRAPH_FILENAME) \
		-funcs $(FUNCTION_TABLE_FILENAME) \
		-types $(TYPE_TABLE_FILENAME)

$(TRACE_FILENAME): $(TEMP_DIR)
	@echo "=== tracing references using dynamic analysis ==="
	go generate github.com/kiteco/kiteco/kite-go/dynamicanalysis
	go install github.com/kiteco/kiteco/kite-go/dynamicanalysis/cmds/trace-references
	trace-references \
		-input $(SNIPPET_DUMP_FILENAME) \
		-output $(TRACE_FILENAME) \
		$(TRACE_ARGS)

$(STATIC_ANALYSIS_REPORT): $(TRACE_FILENAME) $(FUNCTION_TABLE_FILENAME) $(TYPE_TABLE_FILENAME) $(ARTIFACTS_DIR) $(TEMP_DIR)
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/evaluate-static-analysis
	evaluate-static-analysis \
		$(EVALUATE_ARGS) \
		-snippets $(TRACE_FILENAME) \
		-report $(STATIC_ANALYSIS_REPORT) \
		-typeinduction $(ARTIFACTS_DIR) \
		-importgraph $(IMPORT_GRAPH_FILENAME)

$(TYPE_INDUCTION_REPORT): $(TRACE_FILENAME) $(FUNCTION_TABLE_FILENAME) $(TYPE_TABLE_FILENAME) $(ARTIFACTS_DIR) $(TEMP_DIR)
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/evaluate-type-induction
	evaluate-type-induction \
		$(EVALUATE_ARGS) \
		-snippets $(TRACE_FILENAME) \
		-report $(TYPE_INDUCTION_REPORT) \
		-model $(ARTIFACTS_DIR) \
		-importgraph $(IMPORT_GRAPH_FILENAME)

$(IMPORT_GRAPH_REPORT): $(IMPORT_GRAPH_FILENAME)
	go install github.com/kiteco/kiteco/kite-go/lang/python/cmds/import-graph-coverage
	import-graph-coverage \
		-importgraph $(IMPORT_GRAPH_FILENAME) \
		-usages $(STATIC_USAGES_FILENAME) \
		-report $(TYPE_INDUCTION_REPORT)
