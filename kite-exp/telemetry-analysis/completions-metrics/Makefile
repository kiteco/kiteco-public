START_DATE ?= 2019-01-11

TMP_DIR := ./tmp
OUT_DIR := ./out
RUN_FILE := $(TMP_DIR)/rundb-run.txt
RUNDB_PKG := github.com/kiteco/kiteco/kite-golib/pipeline/cmds/rundb

LOGS_FILE := completions.json
PLOTS_DIR := plots

REMOTE_OUT_DIR := s3://kite-local-pipelines/exp/digest-comp-logs/$(shell date -u +"%Y-%m-%d_%H-%M-%S-%p")
CLUSTER_SIZE ?= 6

DIRS_TO_CREATE := $(TMP_DIR) $(OUT_DIR)

LATEST_LOGS := s3://kite-local-pipelines/exp/digest-comp-logs/2019-06-06_23-10-27-PM/

RUNDB := s3://kite-data/run-db

$(shell mkdir -p $(DIRS_TO_CREATE))

all: plots

logs: $(OUT_DIR)/$(LOGS_FILE)

create_run: $(RUN_FILE)

$(RUN_FILE):
	go run $(RUNDB_PKG) create $(RUNDB) completions-metrics > $(RUN_FILE)

$(OUT_DIR)/$(LOGS_FILE): $(RUN_FILE)
	mkdir -p $(TMP_DIR)/logs
	go run github.com/kiteco/kiteco/kite-exp/telemetry-analysis/completions-metrics/digest-comp-logs \
		--start 2019-04-24 \
		--end 2019-04-25 \
		--outdir $(TMP_DIR)/logs \
		--rundbpath $(shell cat $(RUN_FILE))
	cat $(TMP_DIR)/logs/*.json > $(TMP_DIR)/$(LOGS_FILE)
	mv $(TMP_DIR)/$(LOGS_FILE) $(OUT_DIR)/$(LOGS_FILE)

# after dist_logs is run, the files in $(REMOTE_OUT_DIR) need to be concatenated and copied to ./out/completions.json.
dist_logs: $(RUN_FILE)
	../../../kite-golib/pipeline/util/launch-dist-pipeline.sh \
		github.com/kiteco/kiteco/kite-exp/telemetry-analysis/completions-metrics/digest-comp-logs \
		$(CLUSTER_SIZE) \
		--start $(START_DATE) \
		--outdir $(REMOTE_OUT_DIR) \
		--rundbpath $(shell cat $(RUN_FILE))
	echo "Deployed digest-comp-logs pipeline! Will write logs to $(REMOTE_OUT_DIR)"

merge:
	go run github.com/kiteco/kiteco/kite-golib/pipeline/cmds/merge-output \
		--out $(TMP_DIR)/$(LOGS_FILE) \
		--in $(LATEST_LOGS)
	mv $(TMP_DIR)/$(LOGS_FILE) $(OUT_DIR)/$(LOGS_FILE)

plots: $(OUT_DIR)/$(PLOTS_DIR)

$(OUT_DIR)/$(PLOTS_DIR): $(OUT_DIR)/$(LOGS_FILE) $(RUN_FILE)
	mkdir -p $(TMP_DIR)/$(PLOTS_DIR)
	PYTHONPATH=. python make-plots/plots.py \
		--logs $(OUT_DIR)/$(LOGS_FILE) \
		--plots $(TMP_DIR)/$(PLOTS_DIR)
	go run $(RUNDB_PKG) add-artifact $(shell cat $(RUN_FILE)) $(TMP_DIR)/$(PLOTS_DIR) plots --recursive
	go run $(RUNDB_PKG) set-finished $(shell cat $(RUN_FILE))
	mv $(TMP_DIR)/$(PLOTS_DIR) $(OUT_DIR)

analyze: $(OUT_DIR)/$(LOGS_FILE)
	python ./analyze.py --logs $<

clean:
	rm -rf $(TMP_DIR)
	rm -rf $(OUT_DIR)
