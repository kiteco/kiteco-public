package main

import (
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

const maxCallsPerEvent = 10

type analysisPipeline struct {
	Pipeline pipeline.Pipeline
	WriteAgg pipeline.Aggregator
}

type resources struct {
	rm        pythonresource.Manager
	sti       traindata.SubtokenIndex
	recreator *servercontext.Recreator
}

type options struct {
	StartDate analyze.Date
	EndDate   analyze.Date
	MaxEvents int
	OutDir    string
}

func createPipeline(res resources, opts options) analysisPipeline {
	pm := make(pipeline.ParentMap)

	trackOpts := pythonpipeline.DefaultTrackingEventsOpts
	trackOpts.MaxEvents = opts.MaxEvents
	trackOpts.ShardByUMF = true
	trackOpts.NumReaders = 2
	trackOpts.Logger = os.Stderr

	events := pythonpipeline.NewTrackingEvents(
		opts.StartDate, opts.EndDate, pythontracking.ServerSignatureFailureEvent, trackOpts)

	records := pm.Chain(
		events,
		transform.NewFilter("deduped", pythonpipeline.DedupeEvents()),
		transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(res.recreator, true)),
		transform.NewMap("records", func(s pipeline.Sample) []pipeline.Sample {
			return getRecords(res, s.(pythonpipeline.AnalyzedEvent))
		}))

	summaryAgg := aggregator.NewSumAggregator("summary",
		func() sample.Addable {
			return paramSummary{}
		}, func(s pipeline.Sample) sample.Addable {
			return newParamSummary(s.(funcRecord))
		})

	pm.Chain(records, summaryAgg)

	var writeAgg pipeline.Aggregator
	if opts.OutDir != "" {
		writeAgg = aggregator.NewJSONWriter(aggregator.DefaultWriterOpts, "write-dir", opts.OutDir)
		pm.Chain(records, writeAgg)
	}

	resFn := func(res map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		s := res[summaryAgg].(paramSummary)
		percent := func(num, denom int) float64 { return 100. * float64(num) / float64(denom) }
		result := func(name string, value interface{}) rundb.Result {
			return rundb.Result{Aggregator: summaryAgg.Name(), Name: name, Value: value}
		}

		return []rundb.Result{
			result("Count", s.Count),
			result("Resolved %", percent(s.Resolved, s.Count)),
			result("Resolved to Global %", percent(s.ResolvedToGlobal, s.Count)),
			result("Resolved to Local %", percent(s.ResolvedToLocal, s.Count)),
			result("All Name Subtokens Recognized %", percent(s.AllSubtokensRecognized, s.Count)),
		}
	}

	return analysisPipeline{
		Pipeline: pipeline.Pipeline{
			Name:    "local-code-analysis",
			Parents: pm,
			Sources: []pipeline.Source{events},
			Params: map[string]interface{}{
				"StartDate": time.Time(opts.StartDate),
				"EndDate":   time.Time(opts.EndDate),
				"MaxEvents": opts.MaxEvents,
				"OutDir":    opts.OutDir,
			},
			ResultsFn: resFn,
		},
		WriteAgg: writeAgg,
	}
}

func main() {
	args := struct {
		MaxEvents int
		OutDir    string
		RunDBPath string
	}{
		MaxEvents: 100000,
		RunDBPath: rundb.DefaultRunDB,
	}
	arg.MustParse(&args)

	datadeps.Enable()

	sti, err := traindata.NewSubtokenIndex(traindata.NewSubtokenIndexPath)
	fail(err)

	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	fail(err)

	startDate, err := analyze.ParseDate("2018-10-20")
	fail(err)

	endDate, err := analyze.ParseDate("2018-12-20")
	fail(err)

	opts := options{
		StartDate: startDate,
		EndDate:   endDate,
		MaxEvents: args.MaxEvents,
		OutDir:    args.OutDir,
	}

	resources := resources{
		rm:        recreator.Services.ResourceManager,
		recreator: recreator,
		sti:       sti,
	}

	pipe := createPipeline(resources, opts)

	eOpts := pipeline.DefaultEngineOptions
	eOpts.RunDBPath = args.RunDBPath
	engine, err := pipeline.NewEngine(pipe.Pipeline, eOpts)
	fail(err)

	res, err := engine.Run()
	fail(err)

	if pipe.WriteAgg != nil {
		for _, filename := range res[pipe.WriteAgg].(sample.StringSlice) {
			log.Printf("written: %s", filename)
		}
	}
}

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
