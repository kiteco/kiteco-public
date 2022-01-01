package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/montanaflynn/stats"

	"github.com/kiteco/kiteco/local-pipelines/python-call-patterns/internal/data"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/diskmapindex"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
)

var filterParams = data.FilterParams{
	MinCallCount:    5,
	MinPatternCount: 2,
	MinSourceCount:  1,
	MinArgCount:     1,
	MinTypeCount:    1,
}

const (
	// long tail of calls with 100000+ instances,
	// 90th percentile is ~400
	maxCallsPerSym = 1000

	// this is just a heuristic used to presize the keys map in the diskmapindex
	numFuncsInIndex = 150000
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type samplePatterns pythonpatterns.Calls

func (samplePatterns) SampleTag() {}

func main() {
	args := struct {
		In            string
		Out           string
		CacheRoot     string
		CallsPerBlock int
		RunDB         string
		NumAnalysis   int
		Compress      bool
	}{
		CallsPerBlock: 1e4,
		RunDB:         rundb.DefaultRunDB,
		NumAnalysis:   5,
		Compress:      true,
	}
	arg.MustParse(&args)

	fail(os.MkdirAll(args.Out, os.ModePerm))

	start := time.Now()
	fmt.Printf("building patterns from %s and outputting to %s\n", args.In, args.Out)

	idx, err := diskmapindex.NewIndex(args.In, args.CacheRoot)
	fail(err)

	keys := make(chan string)
	go func() {
		start := time.Now()
		// NOTE: this can take ~20 minutes depending how large the index is...
		ks, err := idx.Keys(numFuncsInIndex)
		fail(err)

		fmt.Printf("took %v and got keys for %d functions\n", time.Since(start), len(ks))

		for _, k := range ks {
			keys <- k
		}
		close(keys)
	}()

	src := source.Func("calls", func() pipeline.Record {
		k, ok := <-keys
		if !ok {
			return pipeline.Record{}
		}
		return pipeline.Record{
			Key:   k,
			Value: sample.String(k),
		}
	})

	var m sync.Mutex
	var numCalls []float64
	var numPatterns int
	build := transform.NewOneInOneOut("build", func(s pipeline.Sample) pipeline.Sample {
		bufs, err := idx.Get(string(s.(sample.String)))
		fail(err)

		var calls data.Calls
		var total int
		for _, buf := range bufs {
			c := new(data.Calls)
			fail(c.Decode(buf))

			cc := *c
			for _, call := range cc {
				if total >= maxCallsPerSym {
					// Use ResevoirSampling to get an unbiased sampling of calls
					// https://en.wikipedia.org/wiki/Reservoir_sampling
					if j := rand.Intn(total + 1); j < maxCallsPerSym {
						calls[j] = call
					}
				} else {
					calls = append(calls, call)
				}
				total++
			}
		}

		pats := data.BuildPatterns(filterParams, calls)

		m.Lock()
		defer m.Unlock()
		numCalls = append(numCalls, float64(total))
		if len(numCalls)%1e4 == 0 {
			fmt.Printf("Processed %d functions and extracted %d patterns in %v\n",
				len(numCalls), numPatterns, time.Since(start))
		}

		if pats.Func.Nil() {
			return nil
		}
		numPatterns++

		return samplePatterns(pats)
	})

	writerOpts := aggregator.DefaultWriterOpts
	writerOpts.NumGo = 1
	writerOpts.FilePrefix = "patterns"
	writerOpts.SamplesPerFile = args.CallsPerBlock
	writerOpts.Compress = args.Compress

	write := aggregator.NewJSONWriter(writerOpts, "out", args.Out)

	pm := make(pipeline.ParentMap)
	pm.Chain(
		src,
		build,
		write,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-call-patterns-buildpatterns",
		Parents: pm,
		Sources: []pipeline.Source{src},
		ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
			res := []rundb.Result{
				{
					Name:  "Duration",
					Value: fmt.Sprintf("%v", time.Since(start)),
				},
				{
					Name:  "Num input functions",
					Value: len(numCalls),
				},
				{
					Name:  "Num functions with patterns",
					Value: numPatterns,
				},
				{
					Name:  "filter params",
					Value: fmt.Sprintf("%+v", filterParams),
				},
			}

			mean, err := stats.Mean(numCalls)
			fail(err)
			res = append(res, rundb.Result{
				Name:  "Mean num calls per func",
				Value: mean,
			})

			max, err := stats.Max(numCalls)
			fail(err)
			res = append(res, rundb.Result{
				Name:  "max num calls per func",
				Value: max,
			})

			for _, p := range []float64{25, 50, 75, 90, 99} {
				pc, err := stats.Percentile(numCalls, p)
				fail(err)

				res = append(res, rundb.Result{
					Name:  fmt.Sprintf("Num calls per func at %fth percentile", p),
					Value: pc,
				})
			}

			fmt.Println("Done!")
			for _, r := range res {
				fmt.Printf("%s: %v\n", r.Name, r.Value)
			}

			return res
		},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: args.NumAnalysis,
		RunDBPath:  args.RunDB,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)
}
