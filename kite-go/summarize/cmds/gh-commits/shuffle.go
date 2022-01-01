package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/summarize/data"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

var shuffleCmd = cmdline.Command{
	Name:     "shuffle",
	Synopsis: "shuffle extract gh commits",
	Args: &shuffleArgs{
		Seed: 42,
	},
}

type shuffleArgs struct {
	In     string
	Out    string
	TmpDir string
	Seed   int64
}

// based on https://blog.janestreet.com/how-to-shuffle-a-big-dataset/
func (args *shuffleArgs) Handle() error {
	start := time.Now()
	fail(os.MkdirAll(args.TmpDir, os.ModePerm))

	dumpDir := filepath.Join(args.TmpDir, "dump")
	fail(os.MkdirAll(dumpDir, os.ModePerm))

	fmt.Println("Starting sharding")
	shard(args.In, dumpDir, 100, args.Seed)
	fmt.Printf("Done sharding, took %v\n", time.Since(start))

	start = time.Now()
	fmt.Println("Starting shuffling")
	shuffle(dumpDir, args.Out, args.TmpDir, args.Seed)
	fmt.Printf("Done shuffling, took %v\n", time.Since(start))

	return nil
}

type shardedSample struct {
	Shard  int
	Sample pipeline.Sample
}

func (shardedSample) SampleTag() {}

func shard(inDir, dumpDir string, nShards int, seed int64) {
	commits := newGitCommitsSource(inDir, "commits", source.DatasetOpts{
		NumGo:        2,
		NoCache:      true,
		PanicOnError: true,
	})

	var m sync.Mutex
	rgen := rand.New(rand.NewSource(seed))
	sharder := transform.NewOneInOneOut("sharder", func(s pipeline.Sample) pipeline.Sample {
		m.Lock()
		defer m.Unlock()
		return shardedSample{
			Shard:  rgen.Intn(nShards),
			Sample: s,
		}
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(commits, sharder)

	for shard := 0; shard < nShards; shard++ {
		func(shard int) {
			filter := transform.NewFilter(fmt.Sprintf("filter-shard-%d", shard), func(s pipeline.Sample) bool {
				return s.(shardedSample).Shard == shard
			})

			extract := transform.NewOneInOneOut(fmt.Sprintf("extract-%d", shard), func(s pipeline.Sample) pipeline.Sample {
				return s.(shardedSample).Sample
			})

			writer := newGitCommitsWriter(dumpDir, fmt.Sprintf("shard-writer-%d", shard), aggregator.WriterOpts{
				NumGo:      1,
				FilePrefix: fmt.Sprintf("shard-%d", shard),
			})

			pm.Chain(sharder, filter, extract, writer)
		}(shard)
	}

	pipe := pipeline.Pipeline{
		Name:    "gh-commits-shuffle-shard",
		Parents: pm,
		Sources: []pipeline.Source{commits},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 1,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)
}

func shuffle(dumpDir, out, tmpDir string, seed int64) {
	fs, err := aggregator.ListDir(dumpDir)
	fail(err)

	// we have to load the files manually to avoid memory issues
	var pos int
	chunks := source.Func("chunks", func() pipeline.Record {
		if pos >= len(fs) {
			return pipeline.Record{}
		}
		f := fs[pos]
		r, err := fileutil.NewCachedReader(f)
		fail(err)
		defer r.Close()

		gz, err := gzip.NewReader(r)
		fail(err)

		dec := json.NewDecoder(gz)

		var commits data.GitCommits
		for {
			var commit data.GitCommit
			if err := dec.Decode(&commit); err == io.EOF {
				break
			}
			fail(err)
			commits = append(commits, commit)
		}

		pos++
		return pipeline.Record{
			Key:   f,
			Value: commits,
		}
	})

	rgen := rand.New(rand.NewSource(seed))
	var m sync.Mutex

	shuffle := transform.NewMapChan("shuffle", func(s pipeline.Sample) chan pipeline.Sample {
		res := make(chan pipeline.Sample)
		go func() {
			defer close(res)
			m.Lock()
			defer m.Unlock()

			commits := s.(data.GitCommits)
			rgen.Shuffle(len(commits), func(i, j int) {
				commits[i], commits[j] = commits[j], commits[i]
			})

			for _, c := range commits {
				res <- c
			}
		}()
		return res
	})

	write := newGitCommitsWriter(out, "writer", aggregator.WriterOpts{
		NumGo:  2,
		TmpDir: tmpDir,
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(chunks, shuffle, write)

	pipe := pipeline.Pipeline{
		Name:    "gh-commits-shuffle-shuffle",
		Parents: pm,
		Sources: []pipeline.Source{chunks},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 2,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)
}
