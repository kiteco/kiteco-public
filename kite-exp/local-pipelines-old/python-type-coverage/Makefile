KITECO=$(GOPATH)/src/github.com/kiteco/kiteco

# directory for intermediate stuff
PIPELINE_DIR := .artifacts

# directory containing downloaded user code
CORPUS_DIR := $(PIPELINE_DIR)/corpus

# path to file containing counts
COUNTS := $(PIPELINE_DIR)/counts.json.gz

# userIDs to get code from
IDS := 208 \
	216 \
	257 \
	290 \
	298 \
	330 \
	355 \
	371 \
	450 \
	463 \
	485 \
	526 \
	567 \
	589 \
	591 \
	636 \
	647 \
	676 \
	692 \
	694 \
	730 \
	772 \
	773 \
	783 \
	787 \
	809 \
	813 \
	820 \
	821 \
	832 \
	845 \
	861 \
	869 \
	872 \
	876 \
	881 \
	882 \
	910 \
	932 \
	947 \
	949 \
	950 \
	958 \
	961 \
	1027 \


# place holder to signal that downloads have finished succesfully
DOWNLOAD := $(PIPELINE_DIR)/downloaded

# place holder to signal that coverage has finished succesfully
COVERAGE := $(PIPELINE_DIR)/coverage

default: all

all: download count

download: $(DOWNLOAD)

$(DOWNLOAD): $(CORPUS_DIR)
	@echo "=== downloading user files ..."
	go install github.com/kiteco/kiteco/local-pipelines/python-type-coverage/download
	download --corpus $(CORPUS_DIR) $(IDS) && touch $(DOWNLOAD)

count: $(COUNTS)

coverage: $(COVERAGE)

$(COVERAGE): $(DOWNLOAD)
	@echo "=== measuring coverage for user files ..."
	go install github.com/kiteco/kiteco/local-pipelines/python-type-coverage/coverage
	coverage measure $(CORPUS_DIR) && touch $(COVERAGE)

$(COUNTS): $(DOWNLOAD)
	@echo "=== counting stats for user files ..."
	go install github.com/kiteco/kiteco/local-pipelines/python-type-coverage/submodules
	submodules count $(CORPUS_DIR) $(COUNTS)

clean:
	rm -rf $(PIPELINE_DIR)

$(PIPELINE_DIR):
	mkdir -p $(PIPELINE_DIR)

$(CORPUS_DIR): $(PIPELINE_DIR)
	mkdir -p $(CORPUS_DIR)