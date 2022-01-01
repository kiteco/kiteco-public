package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kiteco/kiteco/kite-go/summarize/data"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

var splitCmd = cmdline.Command{
	Name:     "split",
	Synopsis: "split gh commits into train, validate, and test sets",
	Args: &splitArgs{
		TrainPct:    90,
		ValidatePct: 5,
		TestPct:     5,
		Seed:        42,
	},
}

type splitArgs struct {
	In          string
	Out         string
	TmpDir      string
	TrainPct    int
	ValidatePct int
	TestPct     int
	Seed        int
}

func (args *splitArgs) Handle() error {
	start := time.Now()
	fail(os.MkdirAll(args.TmpDir, os.ModePerm))

	commits := newGitCommitsSource(args.In, "commits", source.DatasetOpts{
		NumGo:        2,
		NoCache:      true,
		PanicOnError: true,
	})

	dsOpts := data.NewDatasetOptions(args.TrainPct, args.ValidatePct, args.TestPct, args.Seed)

	pm := make(pipeline.ParentMap)
	for _, dt := range []data.DatasetType{data.TrainDataset, data.ValidateDataset, data.TestDataset} {
		func(dt data.DatasetType) {
			out := fileutil.Join(args.Out, string(dt))
			filter := transform.NewFilter(fmt.Sprintf("filter-%v", dt), func(s pipeline.Sample) bool {
				c := s.(data.GitCommit)
				return dt == data.ShardRepo(c.RepoOwner, c.RepoName, dsOpts)
			})

			write := newGitCommitsWriter(out, fmt.Sprintf("write-%v", dt), aggregator.WriterOpts{
				NumGo:  2,
				TmpDir: args.TmpDir,
			})

			pm.Chain(commits, filter, write)
		}(dt)
	}

	pipe := pipeline.Pipeline{
		Name:    "gh-commits-split",
		Parents: pm,
		Sources: []pipeline.Source{commits},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 2 * runtime.NumCPU(),
	})
	fail(err)

	_, err = engine.Run()
	fail(err)

	fmt.Printf("done splitting files, took %v\n", time.Since(start))

	return nil
}
