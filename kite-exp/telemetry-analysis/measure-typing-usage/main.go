package main

import (
	"fmt"
	"log"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

func isCompSampleEvent(s pipeline.Sample) bool {
	ev := s.(pythonpipeline.Event)
	return ev.Completions.Failure == pythontracking.CompletionsSample
}

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type typingUsage struct {
	SourceCount     int64
	AnnotationSites int64
	NumAnnotations  int64
	NumTypingUsages int64
}

func (typingUsage) SampleTag() {}

func (t typingUsage) Add(other sample.Addable) sample.Addable {
	o := other.(typingUsage)

	return typingUsage{
		SourceCount:     t.SourceCount + o.SourceCount,
		AnnotationSites: t.AnnotationSites + o.AnnotationSites,
		NumAnnotations:  t.NumAnnotations + o.NumAnnotations,
		NumTypingUsages: t.NumTypingUsages + o.NumTypingUsages,
	}
}

func countTypingUsage(s pipeline.Sample) pipeline.Sample {
	ev := s.(pythonpipeline.AnalyzedEvent)

	isTypingRef := func(expr pythonast.Expr) bool {
		val := ev.Context.Resolved.References[expr]
		for _, sym := range python.GetExternalSymbols(kitectx.Background(), ev.Context.Importer.Global, val) {
			if sym.Path().Head() == "typing" {
				return true
			}
		}
		return false
	}

	usage := typingUsage{
		SourceCount: 1,
	}
	pythonast.Inspect(ev.Context.AST, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}

		switch n := n.(type) {
		case *pythonast.FunctionDefStmt:
			usage.AnnotationSites++
			if !pythonast.IsNil(n.Annotation) {
				usage.NumAnnotations++
				if isTypingRef(n.Annotation) {
					usage.NumTypingUsages++
				}
			}

			for _, param := range n.Parameters {
				usage.AnnotationSites++
				if !pythonast.IsNil(param.Annotation) {
					usage.NumAnnotations++
					if isTypingRef(param.Annotation) {
						usage.NumTypingUsages++
					}
				}
			}
		case *pythonast.AssignStmt:
			// we explicitly ignore assignment statements
			// since this can be pretty noisy, we really should
			// zero in on situations in which the user is
			// defining a symbol and then annotates it
		}
		return true
	})

	return usage
}

func main() {
	args := struct {
		MaxEvents int
	}{
		MaxEvents: 1e5,
	}
	arg.MustParse(&args)

	start := time.Now()
	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	maybeQuit(err)

	opts := pythonpipeline.DefaultTrackingEventsOpts
	opts.MaxEvents = args.MaxEvents

	compEvents := pythonpipeline.NewTrackingEvents(
		analyze.NewDate(2018, 12, 12),
		analyze.NewDate(2019, 01, 15),
		pythontracking.ServerCompletionsFailureEvent,
		opts,
	)

	agg := aggregator.NewSumAggregator("usageAgg", func() sample.Addable {
		return typingUsage{}
	}, func(s pipeline.Sample) sample.Addable {
		return s.(typingUsage)
	})

	p := make(pipeline.ParentMap)

	p.Chain(
		compEvents,
		transform.NewFilter("compSampleEvents", isCompSampleEvent),
		transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false)),
		transform.NewOneInOneOut("countTypingUsage", countTypingUsage),
		agg,
	)

	pipe := pipeline.Pipeline{
		Name:    "exp-measure-typing-usage",
		Parents: p,
		Sources: []pipeline.Source{compEvents},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.DefaultEngineOptions)
	if err != nil {
		log.Fatalln(err)
	}

	res, err := engine.Run()
	if err != nil {
		log.Fatalln(err)
	}

	usage := res[agg].(typingUsage)

	fmt.Printf("Done! Took %v to analyze %d source files\n", time.Since(start), usage.SourceCount)
	fmt.Println("Num possible annotation sites:", usage.AnnotationSites)
	fmt.Println("Num annotations:", usage.NumAnnotations)
	fmt.Println("Num annotations referencing typing:", usage.NumTypingUsages)
}
