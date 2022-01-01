package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

var (
	maxSizeBytes = 1 << 18 // 256kb
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		Lang          string
		FilesPerBlock int
		TrainRatio    int
		ValidateRatio int
		TestRatio     int
		MaxFiles      int
		Seed          int
		CacheRoot     string
		TmpDir        string
	}{
		FilesPerBlock: 1e6,
		TrainRatio:    90,
		ValidateRatio: 5,
		TestRatio:     5,
		MaxFiles:      100e6,
		Seed:          42,
		CacheRoot:     "/data/kite",
		TmpDir:        "/data/kite/tmp",
	}

	arg.MustParse(&args)

	err := os.RemoveAll(args.CacheRoot)
	maybeQuit(err)

	err = os.RemoveAll(args.TmpDir)
	maybeQuit(err)

	start := time.Now()

	dataLang := lang.FromName(args.Lang)
	if dataLang == lang.Unknown {
		log.Fatalln("unknown language:", args.Lang)
	}

	input := utils.ShuffledCrawlForLang(dataLang)
	files, err := aggregator.ListDir(input)
	maybeQuit(err)

	sort.Strings(files)

	outputRoot := fmt.Sprintf("s3://kite-local-pipelines/lexical-data-split/%s/", dataLang.Extension())

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = runtime.NumCPU() * 2
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.MaxRecords = args.MaxFiles
	emrOpts.CacheRoot = args.CacheRoot

	srcs := source.NewEMRDataset(fmt.Sprintf("%s-shuffled-corpus", dataLang.Name()), emrOpts, files)

	datasetOpts := utils.NewDatasetOptions(args.TrainRatio, args.ValidateRatio, args.TestRatio, args.Seed)
	if !datasetOpts.CheckValid() {
		log.Fatalln("train/validate/test split is not valid")
	}

	var (
		trainFiles    int
		validateFiles int
		testFiles     int
		vendoredFiles int
	)

	generateFilter := func(filterType utils.DatasetType) *transform.Filter {
		return transform.NewFilter("filter-"+string(filterType), func(s pipeline.Sample) bool {
			kv := s.(pipeline.Keyed)
			repo, fn := utils.SplitKey(kv.Key)
			datasetType := utils.ShardRepo(repo, datasetOpts)
			if datasetType == filterType {
				// Do all the counting here to avoid counting multiple times (from each filter)
				switch dataLang {
				case lang.Golang:
					if strings.Contains(fn, "/vendor/") || strings.Contains(fn, "/Godeps/") {
						vendoredFiles++
						return false
					}
				case lang.JavaScript, lang.JSX, lang.Vue:
					if strings.Contains(fn, "/node_modules/") {
						vendoredFiles++
						return false
					}
				}

				switch datasetType {
				case utils.TrainDataset:
					trainFiles++
				case utils.ValidateDataset:
					validateFiles++
				case utils.TestDataset:
					testFiles++
				}
				return true
			}

			return false
		})
	}

	trainFilter := generateFilter(utils.TrainDataset)
	validateFilter := generateFilter(utils.ValidateDataset)
	testFilter := generateFilter(utils.TestDataset)

	writerOpts := aggregator.DefaultWriterOpts
	writerOpts.NumGo = 1
	writerOpts.FilePrefix = "files"
	writerOpts.SamplesPerFile = args.FilesPerBlock
	writerOpts.TmpDir = args.TmpDir

	timestamp := time.Now().Format("2006-01-02_03-04-05-PM")
	outputRoot = fileutil.Join(outputRoot, timestamp)
	log.Println("writing to", outputRoot)

	trainWriter := aggregator.NewEMRWriter(writerOpts, "train-writer", fileutil.Join(outputRoot, "train"))
	validateWriter := aggregator.NewEMRWriter(writerOpts, "validate-writer", fileutil.Join(outputRoot, "validate"))
	testWriter := aggregator.NewEMRWriter(writerOpts, "test-writer", fileutil.Join(outputRoot, "test"))

	pm := make(pipeline.ParentMap)
	pm.Chain(trainFilter, trainWriter)
	pm.Chain(validateFilter, validateWriter)
	pm.Chain(testFilter, testWriter)
	pm.FanOut(srcs, trainFilter, validateFilter, testFilter)

	pipe := pipeline.Pipeline{
		Name:    fmt.Sprintf("%s-lexical-datasplit", dataLang.Name()),
		Parents: pm,
		Sources: []pipeline.Source{srcs},
		ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
			res := []rundb.Result{
				{
					Name:  "Duration",
					Value: fmt.Sprintf("%v", time.Since(start)),
				},
				{
					Name:  "Num files",
					Value: trainFiles + testFiles + validateFiles,
				},
				{
					Name:  "Test files",
					Value: testFiles,
				},
				{
					Name:  "Train files",
					Value: trainFiles,
				},
				{
					Name:  "Validate files",
					Value: validateFiles,
				},
				{
					Name:  "Vendored files (skipped)",
					Value: vendoredFiles,
				},
			}
			for _, r := range res {
				fmt.Println(r.Name, r.Value)
			}
			return res
		},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 1,
	})

	maybeQuit(err)

	_, err = engine.Run()
	maybeQuit(err)
}
