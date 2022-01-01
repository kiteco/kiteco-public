package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

const maxTraceDepth = 4

func isCompSampleEvent(s pipeline.Sample) bool {
	ev := s.(pythonpipeline.Event)
	return ev.Completions.Failure == pythontracking.CompletionsSample
}

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type situations map[string]int64

func (situations) SampleTag() {}

func (s situations) Add(other sample.Addable) sample.Addable {
	os := other.(situations)

	for k, v := range os {
		s[k] += v
	}
	return s
}

func binaryOpCategory(t pythonscanner.Token) string {
	switch t {
	case pythonscanner.Add, pythonscanner.Sub, pythonscanner.Mul,
		pythonscanner.Pow, pythonscanner.Div, pythonscanner.Truediv, pythonscanner.Pct:
		return "Arithmetic"
	case pythonscanner.BitAnd, pythonscanner.BitOr, pythonscanner.BitXor, pythonscanner.BitNot,
		pythonscanner.BitLshift, pythonscanner.BitRshift:
		return "Bitwise"
	default:
		return "Logical"
	}
}

func countSituations(parsed pythonpipeline.Parsed) sample.Addable {
	sits := make(situations)
	pythonast.InspectEdges(parsed.Mod, func(parent, child pythonast.Node, edge string) bool {
		if pythonast.IsNil(child) {
			return false
		}

		if pythonast.IsNil(parent) {
			// at module
			return true
		}

		if _, ok := parent.(*pythonast.PrintStmt); ok {
			// skip print statements to avoid skewing stats
			// and because the contents are highly variable
			return false
		}

		if c, ok := child.(*pythonast.CallExpr); ok {
			if n, ok := c.Func.(*pythonast.NameExpr); ok {
				// hacky but should be fine for this analysis
				if n.Ident.Literal == "print" {
					return false
				}
			}
		}

		var s string
		switch p := parent.(type) {
		case *pythonast.ForStmt, *pythonast.IfStmt, *pythonast.Branch:
			if _, ok := child.(pythonast.Stmt); !ok {
				// immediate parent is one of the above and we are not a statement
				// so we must be in either a condition, target, or iterable
				return false
			}
		case *pythonast.AssignStmt:
			if child == p.Value {
				s = pythonpipeline.TypeName(p) + "RHS"
			} else {
				s = pythonpipeline.TypeName(p) + "LHS"
			}
		}

		s += pythonpipeline.TypeName(child)

		switch c := child.(type) {
		case *pythonast.ExprStmt, *pythonast.FunctionDefStmt, *pythonast.ClassDefStmt, *pythonast.AssignStmt, *pythonast.ReturnStmt:
			return true
		case *pythonast.BinaryExpr:
			s += binaryOpCategory(c.Op.Token)
		case *pythonast.StringExpr:
			if _, ok := parent.(*pythonast.ExprStmt); ok {
				s += "DocString"
			} else {
				s += "NotADocString"
			}
		case *pythonast.NameExpr:
			switch parent.(type) {
			case *pythonast.ClassDefStmt, *pythonast.FunctionDefStmt, *pythonast.Parameter:
				s += "Def"
			default:
				s += pythonast.GetUsage(c).String()
			}
		}

		sits[s]++

		switch child.(type) {
		case *pythonast.ImportFromStmt, *pythonast.ImportNameStmt,
			*pythonast.Parameter, *pythonast.BadStmt, *pythonast.BinaryExpr,
			*pythonast.DictExpr, *pythonast.DictComprehensionExpr,
			*pythonast.ListExpr, *pythonast.ListComprehensionExpr,
			*pythonast.SetExpr, *pythonast.SetComprehensionExpr:
			return false
		}

		return true
	})

	return sits
}

func main() {
	args := struct {
		MaxEvents int
	}{
		MaxEvents: 1e5,
	}
	arg.MustParse(&args)

	start := time.Now()

	opts := pythonpipeline.DefaultTrackingEventsOpts
	opts.MaxEvents = args.MaxEvents
	opts.ShardByUMF = true
	opts.NumReaders = 2

	compEvents := pythonpipeline.NewTrackingEvents(
		analyze.NewDate(2018, 12, 12),
		analyze.NewDate(2019, 01, 15),
		pythontracking.ServerCompletionsFailureEvent,
		opts,
	)

	agg := aggregator.NewSumAggregator("situations-agg", func() sample.Addable {
		return make(situations)
	}, func(s pipeline.Sample) sample.Addable {
		return countSituations(s.(pythonpipeline.Parsed))
	})

	p := make(pipeline.ParentMap)

	p.Chain(
		compEvents,
		transform.NewFilter("comp-events", isCompSampleEvent),
		transform.NewOneInOneOut("content", func(s pipeline.Sample) pipeline.Sample {
			return sample.ByteSlice(s.(pythonpipeline.Event).Buffer)
		}),
		transform.NewOneInOneOut("parsed", pythonpipeline.ParsedNonNil(pythonparser.Options{
			ErrorMode: pythonparser.Recover,
		}, time.Second)),
		agg,
	)

	pipe := pipeline.Pipeline{
		Name:    "exp-count-situations",
		Parents: p,
		Sources: []pipeline.Source{compEvents},
	}

	eopts := pipeline.DefaultEngineOptions
	eopts.NumWorkers = 2
	engine, err := pipeline.NewEngine(pipe, eopts)
	maybeQuit(err)

	res, err := engine.Run()
	maybeQuit(err)

	situations := res[agg].(situations)

	type kv struct {
		Key   string
		Value int64
	}

	var kvs []kv
	var total int64
	for k, v := range situations {
		kvs = append(kvs, kv{Key: k, Value: v})
		total += v
	}

	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].Value > kvs[j].Value
	})

	if len(kvs) > 100 {
		kvs = kvs[:100]
	}

	fmt.Printf("Took %v and got %d situations a total of %d times\n", time.Since(start), len(kvs), total)

	padding := 3
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', tabwriter.Debug)

	fmt.Fprintln(w, "Situation\tCount\tPercent\t")
	fmt.Fprintln(w, "--\t--\t--\t")
	for _, kv := range kvs {
		avg := float64(kv.Value) / float64(total)
		fmt.Fprintf(w, "%s\t%d\t%f\t\n", kv.Key, kv.Value, avg)
	}
	w.Flush()
}
