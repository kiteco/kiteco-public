package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/status"
)

const (
	// predict the call completions with mechanism ArgType|KwargName|KwargValue
	ggnnArgsLabel = "ggnn-args-label"
)

var (
	section         = status.NewSection("python-offline-metrics/ggnn-call-completions-examples")
	predictDuration = section.SampleDuration("Prediction duration")
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type exampleWriter struct {
	MaxExamples int

	m        sync.Mutex
	count    int
	examples []example.Example
}

func (e *exampleWriter) maybeAdd(ex example.Example) {
	e.m.Lock()
	defer e.m.Unlock()
	e.count++

	if len(e.examples) < e.MaxExamples {
		e.examples = append(e.examples, ex)
		return
	}

	// reservoir sampling to ensure uniform distribution of examples
	// https://en.wikipedia.org/wiki/Reservoir_sampling
	if i := rand.Intn(e.count); i < len(e.examples) {
		e.examples[i] = ex
	}
}

// callExprs returns a deterministic pseudo-random sampling of up to max CallExprs in an analyzed event.
func callExprs(s pipeline.Sample, max int) []pipeline.Sample {
	ev := s.(pythonpipeline.AnalyzedEvent)

	var exprs []*pythonast.CallExpr
	pythonast.Inspect(ev.Context.AST, func(n pythonast.Node) bool {
		if expr, ok := n.(*pythonast.CallExpr); ok {
			exprs = append(exprs, expr)
		}
		return true
	})

	if len(exprs) == 0 {
		return nil
	}

	if len(exprs) <= max {
		max = len(exprs)
	}

	h := fnv.New64()
	h.Write([]byte(ev.Event.Buffer))
	r := rand.New(rand.NewSource(int64(h.Sum64())))

	var samples []pipeline.Sample
	for _, i := range r.Perm(len(exprs))[:max] {
		samples = append(samples, pythonpipeline.EventExpr{
			AnalyzedEvent: ev,
			Expr:          exprs[i],
		})
	}

	return samples
}

type options struct {
	MaxEvents int
	StartDate analyze.Date
	EndDate   analyze.Date
}

func createPipeline(recreator *servercontext.Recreator, models *pythonmodels.Models, opts options, ew *exampleWriter) pipeline.Pipeline {
	trackOpts := pythonpipeline.DefaultTrackingEventsOpts
	trackOpts.MaxEvents = opts.MaxEvents
	trackOpts.ShardByUMF = true
	trackOpts.NumReaders = 2
	trackOpts.Logger = os.Stdout

	events := pythonpipeline.NewTrackingEvents(
		opts.StartDate, opts.EndDate, pythontracking.ServerSignatureFailureEvent, trackOpts)

	pm := make(pipeline.ParentMap)
	pm.Chain(
		events,
		transform.NewFilter("deduped", pythonpipeline.DedupeEvents()),
		transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false)),
		transform.NewMap("exprs", func(s pipeline.Sample) []pipeline.Sample {
			return callExprs(s, 5)
		}),
		transform.NewOneInOneOut("call-arg-situations", pythonpipeline.ExprCallArgSituations),
		transform.NewFilter("chosen-situations", pythonpipeline.CallArgSituationsAllowedByModel(models.Expr)),
		transform.NewOneInOneOut("call-arg-completions", func(s pipeline.Sample) pipeline.Sample {
			sit := s.(pythonpipeline.CallArgSituation)

			mc := pythonpipeline.CallArgCompletionsGroup{
				Situation: sit,
				Provided:  make(map[string]pythonpipeline.CallArgCompletions),
			}

			start := time.Now()
			callComps := pythonpipeline.GGNNCallArgCompletions(recreator, models)(s)
			if callComps != nil && len(callComps.(pythonpipeline.CallArgCompletions).Provided) > 0 {
				predictDuration.RecordDuration(time.Since(start))
			}

			if callComps != nil {
				mc.Provided[ggnnArgsLabel] = callComps.(pythonpipeline.CallArgCompletions)
			}

			return mc
		}),
		dependent.NewFromFunc("maybe-add-example", func(s pipeline.Sample) {
			mc := s.(pythonpipeline.CallArgCompletionsGroup)
			if filtered := mc.Filter(ggnnArgsLabel); len(filtered.Provided) == 1 {
				// only add the example if all providers gave completions
				ew.maybeAdd(filtered.ToExample())
			}
		}),
	)

	return pipeline.Pipeline{
		Name:    "ggnn-call-arg-completions",
		Parents: pm,
		Sources: []pipeline.Source{events},
		Params: map[string]interface{}{
			"ExprModel": models.Expr.Dir(),
			"MaxEvents": opts.MaxEvents,
			"StartDate": time.Time(opts.StartDate),
			"EndDate":   time.Time(opts.EndDate),
		},
	}
}

func main() {
	args := struct {
		ExprModel   string
		MaxEvents   int
		OutDir      string
		ExampleDir  string
		NumExamples int
		RunDBPath   string
		RunName     string
		Role        pipeline.Role
		Port        int
		Endpoints   []string
	}{
		ExprModel:   pythonmodels.DefaultOptions.ExprModelShards[0].ModelPath,
		MaxEvents:   0,
		NumExamples: 1000,
		RunDBPath:   rundb.DefaultRunDB,
		ExampleDir:  "/data/arg-examples",
	}

	arg.MustParse(&args)

	modelOpts := pythonmodels.DefaultOptions
	modelOpts.ExprModelShards = pythonexpr.ShardsFromModelPath(args.ExprModel)
	models, err := pythonmodels.New(modelOpts)
	maybeQuit(err)

	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	maybeQuit(err)

	start := time.Now()

	startDate, err := analyze.ParseDate("2018-10-20")
	maybeQuit(err)

	endDate, err := analyze.ParseDate("2018-12-20")
	maybeQuit(err)

	opts := options{
		MaxEvents: args.MaxEvents,
		StartDate: startDate,
		EndDate:   endDate,
	}

	ew := &exampleWriter{MaxExamples: args.NumExamples}
	pipe := createPipeline(recreator, models, opts, ew)

	var runDBPath string
	if args.RunName != "" {
		runDBPath = args.RunDBPath
	}

	eOpts := pipeline.DefaultEngineOptions
	eOpts.RunDBPath = runDBPath
	eOpts.RunName = args.RunName
	eOpts.Role = args.Role
	eOpts.Port = args.Port
	eOpts.ShardEndpoints = args.Endpoints
	eOpts.NumWorkers = 2

	engine, err := pipeline.NewEngine(pipe, eOpts)
	maybeQuit(err)

	_, err = engine.Run()
	maybeQuit(err)

	// Write the examples to dir
	maybeQuit(os.MkdirAll(args.ExampleDir, os.ModePerm))
	coll := example.Collection{Examples: ew.examples}
	maybeQuit(coll.WriteToDir(args.OutDir))
	log.Printf("wrote %d examples to %s", len(ew.examples), args.OutDir)

	fmt.Println("Done! Took", time.Since(start))
}
