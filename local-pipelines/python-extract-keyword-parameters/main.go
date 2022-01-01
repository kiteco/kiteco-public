package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const maxSizeBytes = 1000000
const maxAnalysisInterval = 2 * time.Second
const maxParseInterval = 1 * time.Second

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		In          string
		OutBase     string
		NumReaders  int
		NumAnalysis int
		CacheRoot   string
	}{
		In:          pythoncode.DedupedCodeDumpPath,
		NumReaders:  3,
		NumAnalysis: 8,
		CacheRoot:   "/data/kite/",
	}

	arg.MustParse(&args)

	files, err := aggregator.ListDir(args.In)
	maybeQuit(err)

	sort.Strings(files)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	maybeQuit(<-errc)

	outPath := fileutil.Join(args.OutBase, "stats.gob")

	fmt.Println("datasets loaded, starting processing, writing outputs to:", outPath)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = args.NumReaders
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.CacheRoot = args.CacheRoot

	srcs := source.NewEMRDataset("srcs", emrOpts, files)

	parsed := transform.NewOneInOneOutKeyed("parsed", pythonpipeline.ParsedNonNil(parseOpts, maxParseInterval))

	resolved := transform.NewOneInOneOutKeyed("resolved", pythonpipeline.ResolvedNonNil(rm, maxAnalysisInterval))

	extracted := transform.NewOneInOneOut("extracted", func(s pipeline.Sample) pipeline.Sample {
		return Extract(rm, s.(pipeline.Keyed))
	})

	var fileCount uint64
	var m sync.Mutex
	current := make(pythoncode.KeywordCountsByFunc)
	merged := dependent.NewFromFunc("merged", func(s pipeline.Sample) {
		m.Lock()
		defer m.Unlock()

		syms := s.(Symbols)

		// Symbol (function call) to the map of keyword and counts
		for _, countsWithSymbol := range syms {
			symbol := countsWithSymbol.Symbol.PathString()
			counts := countsWithSymbol.KeywordCounts

			if _, ok := current[symbol]; !ok {
				current[symbol] = counts
				break
			}

			currentCounts := current[symbol]
			for k, c := range counts {
				currentCounts[k] += c
			}
		}

		fileCount++
	})

	pm := make(pipeline.ParentMap)

	pm.Chain(
		srcs,
		parsed,
		resolved,
		extracted,
		merged,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-extract-keyword-parameters",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
	}

	start := time.Now()
	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: args.NumAnalysis,
	})
	maybeQuit(err)

	_, err = engine.Run()
	maybeQuit(err)

	f, err := fileutil.NewBufferedWriter(outPath)
	maybeQuit(err)

	defer f.Close()

	maybeQuit(gob.NewEncoder(f).Encode(current))

	fmt.Printf("Done! Took %v to process %v files, wrote results to %s\n", time.Since(start), fileCount, outPath)
}
