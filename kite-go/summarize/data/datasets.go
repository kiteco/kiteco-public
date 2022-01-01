package data

import (
	"fmt"
	"sort"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
)

const (
	// RawGHCommitsCrawl is _not_ shuffled, but all fields have their text contents
	// normalized to utf8.
	// SEE: kiteco/local-pipelines/summarize/Makefile.datasets
	RawGHCommitsCrawl = "s3://kite-local-pipelines/gh-commits-all/2020-11-20/raw"

	// ShuffledGHCommitsCrawl is the shuffled version of RawGHCommitsCrawl.
	// SEE: kiteco/local-pipelines/summarize/Makefile.datasets
	ShuffledGHCommitsCrawl = "s3://kite-local-pipelines/gh-commits-all/2020-11-20/shuffled"

	// SplitGHCommitsCrawlRoot is the split (train/validate/test) version of ShuffledGHCommitsCrawl.
	// SEE: kiteco/local-pipelines/summarize/Makefile.datasets
	SplitGHCommitsCrawlRoot = "s3://kite-local-pipelines/gh-commits-all/2020-11-20/split"
)

// DatasetType ...
type DatasetType string

const (
	// RawDataset ...
	RawDataset = DatasetType("raw")

	// TrainDataset ...
	TrainDataset = DatasetType("train")

	// ValidateDataset ...
	ValidateDataset = DatasetType("validate")

	// TestDataset ...
	TestDataset = DatasetType("test")
)

// Dataset ...
func (dt DatasetType) Dataset() string {
	switch dt {
	case RawDataset:
		return RawGHCommitsCrawl
	case TrainDataset, ValidateDataset, TestDataset:
		return fileutil.Join(SplitGHCommitsCrawlRoot, string(dt))
	default:
		panic(fmt.Sprintf("unknown dataset type %v", dt))
	}
}

// Datasets ...
func Datasets(dts ...DatasetType) ([]string, error) {
	var allFiles []string
	for _, dt := range dts {
		fs, err := aggregator.ListDir(dt.Dataset())
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, fs...)
	}
	sort.Strings(allFiles)
	return allFiles, nil
}

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

// ShardRepo returns the shard the repo belongs to
func ShardRepo(owner, name string, opts DatasetOptions) DatasetType {
	if !opts.CheckValid() {
		panic("DatasetOptions are not valid (make sure train+validate+test == 100)")
	}
	shard := int(spooky.Hash64Seed([]byte(owner+":"+name), opts.Seed)) % 100
	if shard <= opts.Train {
		return TrainDataset
	} else if shard <= opts.Train+opts.Validate {
		return ValidateDataset
	} else {
		return TestDataset
	}
}
