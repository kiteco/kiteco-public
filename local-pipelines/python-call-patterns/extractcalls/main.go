package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-golib/diskmapindex"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/local-pipelines/python-call-patterns/internal/data"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

var parseOpts = pythonparser.Options{
	ErrorMode: pythonparser.Recover,
}

const (
	timeout               = time.Second
	maxCallsPerSymPerFile = 5
)

func fail(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type bundle struct {
	Src  []byte
	AST  *pythonast.Module
	RAST *pythonanalyzer.ResolvedAST
}

func (bundle) SampleTag() {}

func main() {
	fail(datadeps.Enable())
	args := struct {
		Out               string
		NumAnalysis       int
		NumReaders        int
		CacheRoot         string
		RunDB             string
		FileFlushInterval uint64
		MaxFiles          int
		Debug             bool
	}{
		NumAnalysis:       5,
		NumReaders:        3,
		CacheRoot:         "/data/kite-local-pipelines",
		RunDB:             rundb.DefaultRunDB,
		FileFlushInterval: 150000,
	}
	arg.MustParse(&args)

	files, err := aggregator.ListDir(pythoncode.DedupedCodeDumpPath)
	fail(err)

	sort.Strings(files)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = args.NumReaders
	emrOpts.CacheRoot = args.CacheRoot
	emrOpts.MaxRecords = args.MaxFiles

	srcs := source.NewEMRDataset("srcs", emrOpts, files)

	parsed := transform.NewOneInOneOut("parsed", func(s pipeline.Sample) pipeline.Sample {
		bs := s.(pipeline.Keyed).Sample.(sample.ByteSlice)
		ast, _, _ := pythonpipeline.Parse(parseOpts, timeout, bs)
		if ast == nil {
			return nil
		}
		return bundle{
			Src: []byte(bs),
			AST: ast,
		}
	})

	resolved := transform.NewOneInOneOut("resolved", func(s pipeline.Sample) pipeline.Sample {
		b := s.(bundle)
		rast, err := pythonpipeline.Resolve(rm, timeout, pythonpipeline.Parsed{Mod: b.AST})
		if err != nil {
			return nil
		}
		b.RAST = rast
		return b
	})

	extracted := transform.NewOneInOneOut("extracted", func(s pipeline.Sample) pipeline.Sample {
		b := s.(bundle)
		var calls data.Calls
		err := kitectx.Background().WithTimeout(timeout, func(ctx kitectx.Context) error {
			calls = data.Extract(ctx, rm, maxCallsPerSymPerFile, b.Src, b.RAST)
			return nil
		})
		if err != nil {
			return nil
		}
		return calls
	})

	fail(os.MkdirAll(args.Out, os.ModePerm))
	fmt.Println("datasets loaded, starting processing, writing outputs to:", args.Out)

	builder := diskmapindex.NewBuilder(diskmapindex.BuilderOptions{}, args.Out)

	var fileCount uint64
	var m sync.Mutex
	byHash := make(map[pythonimports.Hash]data.Calls, 1e6)
	reset := func() {
		kvs := make([]diskmapindex.KeyValue, 0, len(byHash))
		for h, c := range byHash {
			buf, err := c.Encode()
			fail(err)
			kvs = append(kvs, diskmapindex.KeyValue{
				Key:   h.String(),
				Value: buf,
			})
		}

		builder.AddBlock(kvs, true)

		if args.Debug {
			dumpDebugJSON(args.Out, byHash)
		}

		byHash = make(map[pythonimports.Hash]data.Calls, len(byHash))
	}

	merged := dependent.NewFromFunc("merged", func(s pipeline.Sample) {
		m.Lock()
		defer m.Unlock()

		for _, call := range s.(data.Calls) {
			h := call.Func.Hash()
			byHash[h] = append(byHash[h], call)
		}

		fileCount++
		if fileCount%args.FileFlushInterval == 0 {
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
		Name:    "python-call-patterns-extractcalls",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
	}

	start := time.Now()
	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: args.NumAnalysis,
		RunDBPath:  args.RunDB,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)

	reset()

	builder.Finalize()

	fail(builder.Err())

	fmt.Printf("Done! took %v, to procees %v files, wrote results to %s\n",
		time.Since(start), fileCount, args.Out)
}

func dumpDebugJSON(outDir string, byHash map[pythonimports.Hash]data.Calls) {
	f, err := os.Create(filepath.Join(outDir, "debug.json"))
	fail(err)
	defer f.Close()

	fail(json.NewEncoder(f).Encode(byHash))
}
