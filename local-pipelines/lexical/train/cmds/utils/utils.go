package utils

import (
	"fmt"
	"strings"

	"github.com/dgryski/go-spooky"
)

// Golang crawl constants
const (
	DedupedGoCrawl  = "s3://kite-local-pipelines/gh-dump-go/2019-10-11_12-14-57-AM/"
	ShuffledGoCrawl = "s3://kite-local-pipelines/gh-dump-go-shuffled/2019-10-17_04-48-10-PM/"
	GoSplitRoot     = "s3://kite-local-pipelines/lexical-go-datasplit/2019-10-17_05-18-21-PM/"
)

// Python crawl constants
const (
	DedupedPythonCrawl  = "s3://kite-local-pipelines/gh-dump-python/2020-03-12_06-39-56-AM/"
	ShuffledPythonCrawl = "s3://kite-local-pipelines/gh-dump-python-shuffled/2020-03-12_06-39-56-AM/"
	PythonSplitRoot     = "s3://kite-local-pipelines/lexical-data-split/py/2020-03-12_09-48-26-PM/"
)

// JS crawl constants
const (
	DedupedJSCrawl  = "s3://kite-local-pipelines/gh-dump-js/2019-11-19_12-02-10-AM/js/"
	DedupedJSXCrawl = "s3://kite-local-pipelines/gh-dump-js/2019-11-19_12-02-10-AM/jsx/"
	DedupedVueCrawl = "s3://kite-local-pipelines/gh-dump-js/2019-11-19_12-02-10-AM/vue/"

	ShuffledJSCrawl  = "s3://kite-local-pipelines/gh-dump-js-shuffled/2020-01-13_10-30-09-AM/js/"
	ShuffledJSXCrawl = "s3://kite-local-pipelines/gh-dump-js-shuffled/2020-01-13_10-30-09-AM/jsx/"
	ShuffledVueCrawl = "s3://kite-local-pipelines/gh-dump-js-shuffled/2020-01-13_10-30-09-AM/vue/"

	JSSplitRoot  = "s3://kite-local-pipelines/lexical-data-split/js/2020-01-16_08-16-53-PM/"
	JSXSplitRoot = "s3://kite-local-pipelines/lexical-data-split/jsx/2020-01-16_11-40-43-PM/"
	VueSplitRoot = "s3://kite-local-pipelines/lexical-data-split/vue/2020-01-16_11-44-24-PM/"
)

// "Text" crawl variables/constants
var (
	// TextExtensions are all extensions excluding go, python, js, jsx and vue
	TextExtensions = []string{"c", "cc", "cpp", "cs", "css", "h", "hpp", "html", "java",
		"kt", "less", "m", "php", "rb", "scala", "sh", "ts", "tsx"}

	textSplitRootFmt = "s3://kite-local-pipelines/lexical-text-split/2020-09-16_09-17-09-PM/%s/"
)

func supportedTextExt(ext string) bool {
	for _, e := range TextExtensions {
		if e == ext {
			return true
		}
	}
	return false
}

// TextSplitRootForExt ...
func TextSplitRootForExt(ext string) string {
	return fmt.Sprintf(textSplitRootFmt, ext)
}

// SplitKey splits the deduped crawl key into repo and filename
func SplitKey(key string) (repo, fn string) {
	parts := strings.Split(key, ":")
	repo, fn = parts[0], parts[1]
	return
}

// DatasetType ...
type DatasetType string

// DatasetOptions ...
type DatasetOptions struct {
	Train    int
	Validate int
	Test     int
	Seed     uint64
}

// NewDatasetOptions ...
func NewDatasetOptions(train, validate, test, seed int) DatasetOptions {
	return DatasetOptions{
		Train:    train,
		Validate: validate,
		Test:     test,
		Seed:     uint64(seed),
	}
}

// CheckValid ensures options are set correctly
func (d DatasetOptions) CheckValid() bool {
	return d.Train+d.Validate+d.Test == 100
}

var (
	// TrainDataset ...
	TrainDataset = DatasetType("train")

	// ValidateDataset ...
	ValidateDataset = DatasetType("validate")

	// TestDataset ...
	TestDataset = DatasetType("test")
)

// ShardRepo returns the shard the repo belongs to
func ShardRepo(repo string, opts DatasetOptions) DatasetType {
	if !opts.CheckValid() {
		panic("DatasetOptions are not valid (make sure train+validate+test == 100)")
	}
	shard := int(spooky.Hash64Seed([]byte(repo), opts.Seed)) % 100
	if shard <= opts.Train {
		return TrainDataset
	} else if shard <= opts.Train+opts.Validate {
		return ValidateDataset
	} else {
		return TestDataset
	}
}
