package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const (
	maxSizeBytes     = 1000000
	maxParseInterval = 1 * time.Second
)

var parseOpts = pythonparser.Options{
	ErrorMode:   pythonparser.Recover,
	Approximate: true,
}

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		Out        string
		NumReaders int
		CacheRoot  string
	}{
		Out:        "imports.json.gz",
		NumReaders: runtime.NumCPU(),
		CacheRoot:  "/data/kite/",
	}

	start := time.Now()
	arg.MustParse(&args)

	files, err := aggregator.ListDir(pythoncode.DedupedCodeDumpPath)
	maybeQuit(err)

	sort.Strings(files)

	files = files[:5]
	log.Printf("total files: %d", len(files))

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = args.NumReaders
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.CacheRoot = args.CacheRoot

	srcs := source.NewEMRDataset("srcs", emrOpts, files)

	var count int32
	parsed := transform.NewOneInOneOut("parsed", func(s pipeline.Sample) pipeline.Sample {
		atomic.AddInt32(&count, 1)
		if count%100000 == 0 {
			d := time.Since(start)
			fps := float64(count) / d.Seconds()
			log.Printf("%.02f files/sec", fps)
		}

		kv := s.(pipeline.Keyed)
		ast, _, _ := pythonpipeline.Parse(parseOpts, maxParseInterval, kv.Sample.(sample.ByteSlice))
		if ast == nil {
			return nil
		}
		return pipeline.Keyed{
			Key: pythoncode.CodeHash(kv.Sample.(sample.ByteSlice)),
			Sample: pythonpipeline.Parsed{
				Mod: ast,
			},
		}
	})

	var m sync.Mutex
	f, err := os.Create(args.Out)
	maybeQuit(err)
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	out := json.NewEncoder(gz)

	extract := dependent.NewFromFunc("extract", func(s pipeline.Sample) {
		imports := make(map[string]bool)

		ks := s.(pipeline.Keyed)
		parsed := ks.Sample.(pythonpipeline.Parsed)
		pythonast.Inspect(parsed.Mod, func(node pythonast.Node) bool {
			if pythonast.IsNil(node) {
				return false
			}

			switch node := node.(type) {
			case *pythonast.ImportNameStmt:
				if len(node.Names) > 0 && node.Names[0].External != nil && len(node.Names[0].External.Names) > 0 {
					imports[node.Names[0].External.Names[0].Ident.Literal] = true
				}
				return false
			case *pythonast.ImportFromStmt:
				if node.Package != nil && len(node.Package.Names) > 0 {
					imports[node.Package.Names[0].Ident.Literal] = true
				}
				return false
			}

			return true
		})

		if len(imports) == 0 {
			return
		}

		var unique []string
		for k := range imports {
			unique = append(unique, k)
		}

		sort.Strings(unique)

		m.Lock()
		defer m.Unlock()
		maybeQuit(out.Encode(unique))
	})

	pm := make(pipeline.ParentMap)

	pm.Chain(
		srcs,
		parsed,
		extract,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-imports-per-file",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: args.NumReaders,
	})
	maybeQuit(err)

	_, err = engine.Run()
	maybeQuit(err)

	fmt.Printf("done! took %v, to procees %v files", time.Since(start), count)
}
