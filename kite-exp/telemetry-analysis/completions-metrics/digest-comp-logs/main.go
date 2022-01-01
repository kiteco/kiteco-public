package main

import (
	"log"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-exp/telemetry-analysis/completions-metrics/event"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type options struct {
	MaxEvents int
	StartDate analyze.Date
	EndDate   analyze.Date
	OutDir    string
	CacheRoot string
}

type pipe struct {
	pipeline.Pipeline
	Writer *aggregator.Writer
}

func createPipeline(opts options) (pipe, error) {
	segOpts := source.DefaultSegmentEventsOpts
	segOpts.MaxEvents = opts.MaxEvents
	segOpts.CacheRoot = opts.CacheRoot
	events := source.NewSegmentEvents("events",
		opts.StartDate, opts.EndDate, segmentsrc.Production, map[string]interface{}{
			"kite_status": event.CompEvent{},
		}, segOpts)

	postprocess := transform.NewOneInOneOut("postprocess", func(s pipeline.Sample) pipeline.Sample {
		ev := s.(sample.SegmentEvent)
		compEvent := ev.Event.(event.CompEvent)

		// filter out events with zero completions
		if compEvent.CompletionsNumShown == 0 {
			return nil
		}

		compEvent.ComputeBreakdowns()

		compEvent.Timestamp = ev.Metadata.Timestamp
		return compEvent
	})

	wOpts := aggregator.DefaultWriterOpts
	wOpts.SamplesPerFile = 10000
	writer := aggregator.NewJSONWriter(wOpts, "writer", opts.OutDir)

	pm := make(pipeline.ParentMap)

	pm.Chain(events, postprocess, writer)

	resFn := func(out map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		var results []rundb.Result
		for _, file := range out[writer].(sample.StringSlice) {
			results = append(results, rundb.Result{
				Name:  "File",
				Value: file,
			})
		}
		return results
	}

	return pipe{
		Pipeline: pipeline.Pipeline{
			Name:      "digest-comp-logs",
			Parents:   pm,
			Sources:   []pipeline.Source{events},
			ResultsFn: resFn,
			Params: map[string]interface{}{
				"StartDate": time.Time(opts.StartDate),
				"EndDate":   time.Time(opts.EndDate),
				"OutDir":    opts.OutDir,
				"MaxEvents": opts.MaxEvents,
			},
		},
		Writer: writer,
	}, nil
}

func main() {
	today := analyze.Today()
	lastWeek := today.Add(0, 0, -7)

	args := struct {
		Start     *analyze.Date
		End       *analyze.Date
		MaxEvents int
		Cache     string
		Role      pipeline.Role
		Port      int
		Endpoints []string
		RunDBPath string
		OutDir    string
	}{
		Start:     &lastWeek,
		End:       &today,
		RunDBPath: rundb.DefaultRunDB,
		OutDir:    "./out",
	}
	arg.MustParse(&args)

	log.Printf("will write results to %s", args.OutDir)

	opts := options{
		StartDate: *args.Start,
		EndDate:   *args.End,
		OutDir:    args.OutDir,
		MaxEvents: args.MaxEvents,
		CacheRoot: args.Cache,
	}

	start := time.Now()
	pipe, err := createPipeline(opts)
	fail(err)

	eOpts := pipeline.DefaultEngineOptions
	eOpts.RunDBPath = args.RunDBPath
	eOpts.Role = args.Role
	eOpts.Port = args.Port
	eOpts.ShardEndpoints = args.Endpoints

	engine, err := pipeline.NewEngine(pipe.Pipeline, eOpts)
	fail(err)

	out, err := engine.Run()
	fail(err)

	log.Printf("files written:")
	for _, filename := range out[pipe.Writer].(sample.StringSlice) {
		log.Printf(filename)
	}

	log.Printf("Done! took %v", time.Since(start))
}
