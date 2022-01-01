package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/montanaflynn/stats"

	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type argCounts map[int]int64

func (ac argCounts) Sum() int64 {
	var s int64
	for _, v := range ac {
		s += v
	}
	return s
}

func (argCounts) SampleTag() {}

func (ac argCounts) Add(other sample.Addable) sample.Addable {
	for k, v := range other.(argCounts) {
		ac[k] += v
	}
	return ac
}

type situations struct {
	SupportedToday argCounts
	Keywords       argCounts
	Placeholders   argCounts
	Unsupported    argCounts
}

func newSituations() situations {
	return situations{
		SupportedToday: make(argCounts),
		Keywords:       make(argCounts),
		Placeholders:   make(argCounts),
		Unsupported:    make(argCounts),
	}
}

func (situations) SampleTag() {}

func (s situations) Add(other sample.Addable) sample.Addable {
	os := other.(situations)
	s.SupportedToday.Add(os.SupportedToday)
	s.Keywords.Add(os.Keywords)
	s.Placeholders.Add(os.Placeholders)
	s.Unsupported.Add(os.Unsupported)
	return s
}

func count(model pythonexpr.Model, evt pythonpipeline.AnalyzedEvent) sample.Addable {
	s := newSituations()
	pythonast.Inspect(evt.Context.AST, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}

		call, ok := n.(*pythonast.CallExpr)
		if !ok {
			return true
		}

		nArgs := len(call.Args)
		if !pythonast.IsNil(call.Vararg) || !pythonast.IsNil(call.Kwarg) {
			s.Unsupported[nArgs]++
			return true
		}

		val := evt.Context.Resolved.References[call.Func]
		var supported bool
		for _, sym := range python.GetExternalSymbols(kitectx.Background(), evt.Context.Importer.Global, val) {
			if model.CallSupported(evt.Context.Importer.Global, sym) == nil {
				supported = true
				break
			}
		}

		if !supported {
			s.Unsupported[nArgs]++
			return true
		}

		var sawKW, sawPlaceholder bool
		for _, arg := range call.Args {
			if !pythonast.IsNil(arg.Name) {
				sawKW = true
			}
			if _, ok := arg.Value.(*pythonast.NameExpr); !ok {
				sawPlaceholder = true
			}
		}

		switch {
		case sawPlaceholder:
			s.Placeholders[nArgs]++
		case sawKW:
			s.Keywords[nArgs]++
		default:
			s.SupportedToday[nArgs]++
		}

		return true
	})

	return s
}

func main() {
	args := struct {
		MaxEvents int
	}{
		MaxEvents: 3e5,
	}
	arg.MustParse(&args)

	maybeQuit(datadeps.Enable())

	start := time.Now()
	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	maybeQuit(err)

	model, err := pythonexpr.NewModel(pythonmodels.DefaultOptions.ExprModelShards[0].ModelPath, pythonexpr.DefaultOptions)
	maybeQuit(err)

	opts := pythonpipeline.DefaultTrackingEventsOpts
	opts.MaxEvents = args.MaxEvents
	opts.NumReaders = 2

	compEvents := pythonpipeline.NewTrackingEvents(
		analyze.NewDate(2018, 12, 12),
		analyze.NewDate(2019, 01, 15),
		pythontracking.ServerCompletionsFailureEvent,
		opts,
	)

	deduped := transform.NewFilter("deduped", pythonpipeline.DedupeEvents())

	analyzed := transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false))

	agg := aggregator.NewSumAggregator("counts", func() sample.Addable {
		return newSituations()
	}, func(s pipeline.Sample) sample.Addable {
		return count(model, s.(pythonpipeline.AnalyzedEvent))
	})

	p := make(pipeline.ParentMap)

	p.Chain(
		compEvents,
		deduped,
		analyzed,
		agg,
	)

	pipe := pipeline.Pipeline{
		Name:    "exp-count-call-situations",
		Parents: p,
		Sources: []pipeline.Source{compEvents},
	}

	eopts := pipeline.DefaultEngineOptions
	eopts.NumWorkers = 2
	engine, err := pipeline.NewEngine(pipe, eopts)
	maybeQuit(err)

	res, err := engine.Run()
	maybeQuit(err)

	s := res[agg].(situations)

	total := float64(s.Unsupported.Sum() + s.SupportedToday.Sum() + s.Keywords.Sum() + s.Placeholders.Sum())
	fmt.Printf("Fraction of calls supported today: %v\n", float64(s.SupportedToday.Sum())/total)
	fmt.Printf("Fraction of calls supported after keyword args: %v\n", float64(s.SupportedToday.Sum()+s.Keywords.Sum())/total)
	fmt.Printf("Fraction of calls supported after keywords and placeholders: %v\n", float64(s.SupportedToday.Sum()+s.Keywords.Sum()+s.Placeholders.Sum())/total)

	acs := []argCounts{
		s.Keywords,
		s.Placeholders,
		s.SupportedToday,
	}

	var numArgs []float64
	for _, ac := range acs {
		for na, c := range ac {
			for i := 0; i < int(c); i++ {
				numArgs = append(numArgs, float64(na))
			}
		}
	}

	meanNumArgs, err := stats.Mean(numArgs)
	maybeQuit(err)
	fmt.Printf("Mean num args: %v\n", meanNumArgs)

	var ps []float64
	for _, p := range []float64{25, 50, 75, 95} {
		percentile, err := stats.Percentile(numArgs, p)
		maybeQuit(err)
		ps = append(ps, percentile)
	}

	padding := 3
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.Debug)

	fmt.Println("Percentiles for number of arguments in a call")
	fmt.Fprintln(w, "25th\t50th\t75th\t95th")
	fmt.Fprintln(w, "--\t--\t--\t--\t")
	fmt.Fprintf(w, "%f\t%f\t%f\t%f\t\n", ps[0], ps[1], ps[2], ps[3])
	w.Flush()

	fmt.Println("Done! took", time.Since(start))
}
