package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiteco/kiteco/kite-golib/diskmapindex"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const flushInterval = uint64(2e6)
const maxSizeBytes = 1000000
const maxAnalysisInterval = 2 * time.Second
const maxParseInterval = 1 * time.Second

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type symToHashes map[string][]pythoncode.HashCounts

func flush(sh symToHashes, builder *diskmapindex.Builder) {
	kvs := make([]diskmapindex.KeyValue, 0, len(sh))
	for pathStr, counts := range sh {
		buf, err := pythoncode.EncodeHashes(counts)
		maybeQuit(err)
		kvs = append(kvs, diskmapindex.KeyValue{
			Key:   pathStr,
			Value: buf,
		})
	}

	builder.AddBlock(kvs, true)
}

func main() {
	args := struct {
		SymbolsOut      string
		CanonSymbolsOut string
		NumReaders      int
		NumAnalysis     int
		CacheRoot       string
		Manifest        string
		DistIndex       string
	}{
		NumReaders:  3,
		NumAnalysis: 5,
		CacheRoot:   "/data/kite/",
	}

	arg.MustParse(&args)

	files, err := aggregator.ListDir(pythoncode.DedupedCodeDumpPath)
	maybeQuit(err)

	sort.Strings(files)

	opts, err := pythonresource.DefaultOptions.WithCustomPaths(args.Manifest, args.DistIndex)
	maybeQuit(err)
	rm, errc := pythonresource.NewManager(opts)
	maybeQuit(<-errc)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = args.NumReaders
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.CacheRoot = args.CacheRoot
	emrOpts.Logger = os.Stderr

	srcs := source.NewEMRDataset("srcs", emrOpts, files)

	parsed := transform.NewOneInOneOut("parsed", func(s pipeline.Sample) pipeline.Sample {
		kv := s.(pipeline.Keyed)
		pythonpipeline.ParsedNonNil(parseOpts, maxParseInterval)
		ast, words, _ := pythonpipeline.Parse(parseOpts, maxParseInterval, kv.Sample.(sample.ByteSlice))
		if ast == nil {
			return nil
		}
		return pipeline.Keyed{
			Key: pythoncode.CodeHash(kv.Sample.(sample.ByteSlice)),
			Sample: pythonpipeline.Parsed{
				Mod:   ast,
				Words: words,
			},
		}
	})

	resolved := transform.NewOneInOneOutKeyed("resolved", pythonpipeline.ResolvedNonNil(rm, maxAnalysisInterval))

	extracted := transform.NewOneInOneOut("extracted", func(s pipeline.Sample) pipeline.Sample {
		return Extract(rm, s.(pipeline.Keyed))
	})

	maybeQuit(os.MkdirAll(args.SymbolsOut, os.ModePerm))
	maybeQuit(os.MkdirAll(args.CanonSymbolsOut, os.ModePerm))

	fmt.Println("datasets loaded, starting processing, writing outputs to:", args.SymbolsOut, ", ", args.CanonSymbolsOut)

	builder := diskmapindex.NewBuilder(diskmapindex.BuilderOptions{}, args.SymbolsOut)

	canonBuilder := diskmapindex.NewBuilder(diskmapindex.BuilderOptions{}, args.CanonSymbolsOut)

	symHashes := make(symToHashes)
	canonSymHashes := make(symToHashes)
	reset := func() {
		l := len(symHashes)
		flush(symHashes, builder)
		symHashes = make(symToHashes, l)

		l = len(canonSymHashes)
		flush(canonSymHashes, canonBuilder)
		canonSymHashes = make(symToHashes, l)
	}

	var fileCount uint64
	var m sync.Mutex
	merged := dependent.NewFromFunc("merged", func(s pipeline.Sample) {
		m.Lock()
		defer m.Unlock()

		syms := s.(Symbols)

		for _, counts := range syms.NonCanonCounts {
			pathStr := counts.Symbol.PathString()
			symHashes[pathStr] = append(symHashes[pathStr], pythoncode.HashCounts{
				Hash:   syms.Hash,
				Counts: counts.Counts,
			})
		}

		for _, counts := range syms.CanonCounts {
			pathStr := counts.Symbol.PathString()
			canonSymHashes[pathStr] = append(canonSymHashes[pathStr], pythoncode.HashCounts{
				Hash:   syms.Hash,
				Counts: counts.Counts,
			})
		}

		if atomic.AddUint64(&fileCount, 1)%flushInterval == 0 {
			reset()
		}
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
		Name:    "python-symbol-to-hashes-index",
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

	reset()
	builder.Finalize()
	canonBuilder.Finalize()

	maybeQuit(builder.Err())
	maybeQuit(canonBuilder.Err())

	fmt.Printf("Done! took %v, to procees %v files, wrote results to %s and %s\n",
		time.Since(start), fileCount, args.SymbolsOut, args.CanonSymbolsOut)
}
