package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/data"
)

const maxSizeBytes = 1000000
const maxAnalysisInterval = 2 * time.Second
const maxParseInterval = 1 * time.Second

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		In          string
		Out         string
		NumReaders  int
		NumAnalysis int
		CacheRoot   string
		Packages    string
		MaxRecords  int
		RunDBPath   string
		RunName     string
		Role        pipeline.Role
		Port        int
		Endpoints   []string
	}{
		In:          pythoncode.DedupedCodeDumpPath,
		NumReaders:  2,
		NumAnalysis: 2,
		CacheRoot:   "/data/kite/",
		MaxRecords:  0,
		Port:        0,
		RunDBPath:   rundb.DefaultRunDB,
		Packages:    "all-packages.txt",
	}

	arg.MustParse(&args)
	// Load package list from local or s3
	pkgs, err := traindata.LoadPackageList(args.Packages)
	fail(err)

	files, err := aggregator.ListDir(args.In)
	fail(err)

	sort.Strings(files)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = args.NumReaders
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.CacheRoot = args.CacheRoot
	emrOpts.MaxRecords = args.MaxRecords

	srcs := source.NewEMRDataset("srcs", emrOpts, files)

	srcFiltered := transform.NewFilter("src-filtered", func(s pipeline.Sample) bool {
		k := s.(pipeline.Keyed)
		return len(k.Sample.(sample.ByteSlice)) < maxSizeBytes
	})

	parseOpts := pythonparser.Options{
		ErrorMode:   pythonparser.Recover,
		Approximate: true,
	}

	parsed := transform.NewOneInOneOutKeyed("parsed", pythonpipeline.ParsedNonNil(parseOpts, maxParseInterval))

	resolved := transform.NewOneInOneOutKeyed("resolved", pythonpipeline.ResolvedNonNil(rm, maxAnalysisInterval))

	extractor := data.NewExtractor(rm, pkgs)

	extracted := transform.NewMap("extracted", func(s pipeline.Sample) []pipeline.Sample {
		rast := s.(pipeline.Keyed).Sample.(pythonpipeline.Resolved).RAST
		e := extractor.Extract(rast)
		return e
	})

	wOpts := aggregator.DefaultWriterOpts
	wOpts.FilePrefix = "traindata"
	wOpts.NumGo = 2
	wOpts.SamplesPerFile = 1e7
	out := aggregator.NewJSONWriter(wOpts, "out", args.Out)

	pm := make(pipeline.ParentMap)
	pm.Chain(
		srcs,
		srcFiltered,
		parsed,
		resolved,
		extracted,
		out,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-type-induction-training-samples",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
	}

	var runDBPath string
	if args.RunName != "" {
		runDBPath = args.RunDBPath
	}

	eOpts := pipeline.DefaultEngineOptions
	eOpts.RunDBPath = runDBPath
	eOpts.RunName = args.RunName
	eOpts.Role = args.Role
	eOpts.Port = args.Port
	eOpts.ShardEndpoints = args.Endpoints
	eOpts.NumWorkers = args.NumAnalysis

	start := time.Now()
	engine, err := pipeline.NewEngine(pipe, eOpts)
	fail(err)

	_, err = engine.Run()
	fail(err)

	fmt.Printf("Done! Took %v to extract samples.\n", time.Since(start))
}
