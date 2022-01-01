package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/diskmapindex"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
)

const maxSizeBytes = 1000000

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		Region        string
		In            string
		OutBase       string
		FilesPerBlock int
		CacheRoot     string
	}{
		Region:        "us-west-1",
		In:            pythoncode.DedupedCodeDumpPath,
		OutBase:       "s3://kite-local-pipelines/python-hash-to-source-index",
		FilesPerBlock: 1e6,
		CacheRoot:     "/data/kite",
	}

	arg.MustParse(&args)

	start := time.Now()

	files, err := aggregator.ListDir(args.In)
	maybeQuit(err)

	sort.Strings(files)

	ts := time.Now().Format("2006-01-02_03-04-05-PM")
	outDir := fileutil.Join(args.OutBase, ts)

	fmt.Println("starting processing, writing outputs to:", outDir)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = 1
	emrOpts.MaxFileSize = maxSizeBytes
	emrOpts.CacheRoot = args.CacheRoot

	srcs := source.NewEMRDataset("srcs", emrOpts, files)

	builder := diskmapindex.NewBuilder(diskmapindex.BuilderOptions{
		Compress: true,
		CacheDir: args.CacheRoot,
	}, outDir)

	var kvs []diskmapindex.KeyValue
	flush := func() {
		l := len(kvs)
		builder.AddBlock(kvs, true)
		kvs = make([]diskmapindex.KeyValue, 0, l)
	}

	var m sync.Mutex
	var count int
	sink := dependent.NewFromFunc("sink", func(s pipeline.Sample) {
		m.Lock()
		defer m.Unlock()

		kv := s.(pipeline.Keyed)

		count++
		if args.FilesPerBlock > 0 && len(kvs) >= args.FilesPerBlock {
			flush()
		}
		bs := []byte(kv.Sample.(sample.ByteSlice))
		kvs = append(kvs, diskmapindex.KeyValue{
			Key:   pythoncode.CodeHash(bs),
			Value: bs,
		})
	})

	pm := make(pipeline.ParentMap)

	pm.Chain(
		srcs,
		sink,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-hash-to-source-index",
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
					Value: count,
				},
				{
					Name:  "Out dir",
					Value: outDir,
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

	flush()

	builder.Finalize()
	maybeQuit(builder.Err())
}
