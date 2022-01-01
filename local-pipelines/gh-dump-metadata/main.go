package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"

	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"

	"github.com/kiteco/kiteco/kite-golib/pipeline"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		Crawl string
		Out   string
		RunDB string
	}{
		Crawl: "s3://github-crawl-kite/2019-03",
		Out:   "s3://kite-local-pipelines/gh-dump-metadata/",
		RunDB: rundb.DefaultRunDB,
	}
	arg.MustParse(&args)

	start := time.Now()

	crawlOpts := source.DefaultRawGHOpts
	crawlOpts.MetaDataOnly = true
	crawlOpts.NumGo = runtime.NumCPU()
	crawlOpts.Logger = os.Stdout
	crawlOpts.NoCache = true
	crawlOpts.MaxRepos = 1e6

	repos, err := fileutil.ListDir(args.Crawl)
	maybeQuit(err)

	fmt.Printf("Found %d repos to extract metadata from\n", len(repos))

	crawl := source.NewRawGHCrawl(crawlOpts, "crawl", repos)

	var count int64
	metadata := transform.NewOneInOneOut("metadata", func(s pipeline.Sample) pipeline.Sample {
		atomic.AddInt64(&count, 1)
		return s.(sample.GHRepo).Meta
	})

	timestamp := time.Now().Format("2006-01-02_03-04-05-PM")
	out := fileutil.Join(args.Out, timestamp)

	wOpts := aggregator.DefaultWriterOpts
	wOpts.Logger = os.Stdout
	wOpts.NumGo = 2
	wOpts.SamplesPerFile = 1e6
	wOpts.FilePrefix = "metadata"
	wOpts.Compress = true

	writer := aggregator.NewJSONWriter(wOpts, "writer", out)
	pm := make(pipeline.ParentMap)
	pm.Chain(
		crawl,
		metadata,
		writer,
	)

	opts := pipeline.DefaultEngineOptions
	opts.NumWorkers = 1
	opts.Role = pipeline.Standalone
	opts.RunName = args.Crawl
	opts.RunDBPath = args.RunDB

	pipe := pipeline.Pipeline{
		Name:    "gh-dump-metadata",
		Parents: pm,
		Sources: []pipeline.Source{crawl},
		ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
			return []rundb.Result{
				rundb.Result{
					Name:  "runtime",
					Value: fmt.Sprintf("%v", time.Since(start)),
				},
				rundb.Result{
					Name:  "num repos",
					Value: count,
				},
				rundb.Result{
					Name:  "crawl",
					Value: rundb.RenderS3DirLink(args.Crawl, args.Crawl),
				},
				rundb.Result{
					Name:  "results",
					Value: rundb.RenderS3DirLink(out, out),
				},
			}
		},
	}

	engine, err := pipeline.NewEngine(pipe, opts)
	maybeQuit(err)

	_, err = engine.Run()
	maybeQuit(err)
}
