package main

import (
	"fmt"
	"log"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

const (
	maxCallsPerFile = 5
	minCallsPerDist = 5
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		In        string
		MaxEvents int
		RunDB     string
		CacheRoot string
	}{
		In:        pythoncode.CallPatterns,
		MaxEvents: 1e6,
		RunDB:     rundb.DefaultRunDB,
		CacheRoot: "/data/kite-local-pipelines",
	}

	arg.MustParse(&args)

	start := time.Now()
	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	fail(err)

	src := pythonpipeline.NewTrackingEvents(
		analyze.NewDate(2018, 12, 12),
		analyze.NewDate(2019, 01, 15),
		pythontracking.ServerSignatureFailureEvent,
		pythonpipeline.TrackingEventsOpts{
			NumReaders: 2,
			MaxEvents:  args.MaxEvents,
			CacheRoot:  args.CacheRoot,
		},
	)

	analyzed := transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false))

	extract := transform.NewMap("calls", func(s pipeline.Sample) []pipeline.Sample {
		return extractCalls(s.(pythonpipeline.AnalyzedEvent))
	})

	match := transform.NewOneInOneOut("match", func(s pipeline.Sample) pipeline.Sample {
		return match(recreator.Services.ResourceManager, s.(call))
	})

	agg := aggregator.NewSumAggregator("agg",
		func() sample.Addable { return make(byDist) },
		func(s pipeline.Sample) sample.Addable {
			dc := s.(*distCounts)
			return byDist{
				dc.Dist: dc,
			}
		},
	)

	resFn := func(res map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		return append([]rundb.Result{
			{
				Name:  "Duration",
				Value: fmt.Sprintf("%v", time.Since(start)),
			},
			{
				Name:  "Num functions with patterns",
				Value: countPatterns(recreator.Services.ResourceManager),
			},
		}, res[agg].(byDist).results()...)
	}

	pm := make(pipeline.ParentMap)

	pm.Chain(
		src,
		transform.NewFilter("deduped", pythonpipeline.DedupeEvents()),
		analyzed,
		extract,
		match,
		agg,
	)

	pipe := pipeline.Pipeline{
		Name:      "python-popularsignatures-recall",
		Parents:   pm,
		Sources:   []pipeline.Source{src},
		ResultsFn: resFn,
	}

	opts := pipeline.DefaultEngineOptions
	opts.NumWorkers = 2
	opts.RunDBPath = args.RunDB
	opts.Role = pipeline.Standalone

	engine, err := pipeline.NewEngine(pipe, opts)
	fail(err)

	_, err = engine.Run()
	fail(err)
}

func countPatterns(rm pythonresource.Manager) int {
	var count int
	for _, d := range rm.Distributions() {
		css, err := rm.CanonicalSymbols(d)
		fail(err)
		for _, cs := range css {
			if rm.Kind(cs) != keytypes.FunctionKind {
				continue
			}

			if len(rm.PopularSignatures(cs)) > 0 {
				count++
			}
		}
	}
	return count
}
