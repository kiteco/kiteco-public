package main

import (
	"log"
	"os"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type pipelineArguments struct {
	Filename string
}

// An ArgumentSample describes each expression used as an arg in a function call
// All these samples are written in a json file at the end of the pipeline for further analysis in python
type ArgumentSample struct {
	ArgType      string
	Index        int
	Keyword      string
	ArgSpec      string
	FunctionName string
	Positional   bool
	LiteralValue string
	CallID       uint64
}

// SampleTag implements pipeline.Sample
func (ArgumentSample) SampleTag() {}

func isCompSampleEvent(s pipeline.Sample) bool {
	ev := s.(pythonpipeline.Event)
	return ev.Completions.Failure == pythontracking.CompletionsSample
}

func filterOnePerFile() func(s pipeline.Sample) bool {
	var processedFilesMap sync.Map
	return func(s pipeline.Sample) bool {
		ev := s.(pythonpipeline.Event)

		key := string(ev.UserID) + ev.Filename
		_, present := processedFilesMap.Load(key)
		if !present {
			processedFilesMap.Store(key, true)
		}
		return !present
	}
}

func getLiteralValue(expr pythonast.Expr) string {
	if !pythonast.IsLiteral(expr) {
		return "NOT_LITERAL"
	}
	switch v := expr.(type) {
	case *pythonast.StringExpr:
		return v.Literal()
	case *pythonast.NumberExpr:
		return v.Number.Literal
	case *pythonast.ListExpr:
		if len(v.Values) == 0 {
			return "[]"
		}
		return "LIST"
	case *pythonast.DictExpr:
		if len(v.Items) == 0 {
			return "{}"
		}
		return "DICT"
	case *pythonast.SetExpr:
		if len(v.Values) == 0 {
			return "set()"
		}
		return "SET"
	case *pythonast.ListComprehensionExpr:
		return "LIST_COMP"
	case *pythonast.DictComprehensionExpr:
		return "DICT_COMP"
	case *pythonast.SetComprehensionExpr:
		return "SET_COMP"
	case *pythonast.ComprehensionExpr:
		return "COMPREHENSION"
	case *pythonast.TupleExpr:
		return "TUPLE"
	default:
		return "UNKNOWN_TYPE"
	}
}

func buildArgumentSample(arg *pythonast.Argument, funcSym pythonresource.Symbol,
	spec *pythonimports.ArgSpec, argIdx int, callID uint64) ArgumentSample {
	var name string
	argSpecName := "NO_SPEC"
	if spec != nil {
		if len(spec.Args) > 0 && (spec.Args[0].Name == "self" || spec.Args[0].Name == "cls") {
			// The called function is a method or a class function, we switch the args by 1
			argIdx++
		}
		if !pythonast.IsNil(arg.Name) {
			id, ok := arg.Name.(*pythonast.NameExpr)
			// This cast should be safe but cf ISSUE 7528 (https://github.com/kiteco/kiteco/issues/7528)
			// The parser currently allows for AttributeExpr to be used as keyword, which is illegal in python
			// TODO remove the check once the parser will be fixed
			if ok {
				name = id.Ident.Literal
				nameFound := false
				for _, argSpec := range spec.Args {
					if argSpec.Name == name {
						argSpecName = argSpec.Name
						nameFound = true
						break
					}
				}
				if !nameFound {
					argSpecName = "**KWARGS"
				}
			}
		} else {
			argSpecName = "*ARGS"
			if argIdx < len(spec.Args) {
				argSpecName = spec.Args[argIdx].Name
			}
		}
	}
	if !pythonast.IsNil(arg.Name) {
		id, ok := arg.Name.(*pythonast.NameExpr)
		if ok {
			name = id.Ident.Literal
		}
	}
	literalValue := getLiteralValue(arg.Value)

	return ArgumentSample{
		ArgType:      reflect.TypeOf(arg.Value).String(),
		Positional:   pythonast.IsNil(arg.Name),
		Index:        argIdx,
		Keyword:      name,
		ArgSpec:      argSpecName,
		FunctionName: funcSym.PathString(),
		LiteralValue: literalValue,
		CallID:       callID,
	}
}

func extractArguments(rm pythonresource.Manager) func(s pipeline.Sample) []pipeline.Sample {
	var callCounter uint64
	return func(s pipeline.Sample) []pipeline.Sample {
		expr := s.(pythonpipeline.EventExpr)
		rast := expr.AnalyzedEvent.Context.Resolved
		call, ok := expr.Expr.(*pythonast.CallExpr)
		if !ok {
			return nil
		}
		val := rast.References[call.Func]
		symb, err := python.GetExternalSymbol(kitectx.Background(), rm, val)
		if err != nil {
			return nil
		}
		callID := atomic.AddUint64(&callCounter, 1)

		prototype := rm.ArgSpec(symb)
		var result []pipeline.Sample
		for argIdx, arg := range call.Args {
			result = append(result, buildArgumentSample(arg, symb, prototype, argIdx, callID))
		}
		return result
	}
}

func filterCallExpr(models *pythonmodels.Models, rm pythonresource.Manager) func(sample pipeline.Sample) bool {
	return func(sample pipeline.Sample) bool {
		expr := sample.(pythonpipeline.EventExpr)
		rast := expr.AnalyzedEvent.Context.Resolved
		call, ok := expr.Expr.(*pythonast.CallExpr)
		if !ok {
			return false
		}
		val := rast.References[call.Func]
		symb, err := python.GetExternalSymbol(kitectx.Background(), rm, val)
		//TODO Switch to getExternalSymbols to be sure to cover all possible cases
		if err == nil {
			err = models.Expr.CallSupported(rm, symb)
		}
		return err == nil
	}
}

func buildPipeline(recreator *servercontext.Recreator, models *pythonmodels.Models,
	rm pythonresource.Manager, args *pipelineArguments) (pipeline.Pipeline, *dependent.JSONWriter) {
	compEvents := pythonpipeline.NewTrackingEvents(
		analyze.NewDate(2018, 12, 01),
		analyze.NewDate(2019, 04, 15),
		pythontracking.ServerCompletionsFailureEvent,
		pythonpipeline.TrackingEventsOpts{
			NumReaders: 8,
		})
	outf, err := os.Create(args.Filename)
	if err != nil {
		log.Fatalln(err)
	}
	writer := dependent.NewJSONWriter("writeLogs", outf)

	p := make(pipeline.ParentMap)
	p.Chain(
		compEvents,
		transform.NewFilter("compSampleEvents", isCompSampleEvent),
		transform.NewFilter("onePerFile", filterOnePerFile()),
		transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false)),
		transform.NewMap("exprs", pythonpipeline.Exprs),
		transform.NewFilter("filterResolvedCall", filterCallExpr(models, rm)),
		transform.NewMap("extractArgs", extractArguments(rm)),
		writer)
	pipe := pipeline.Pipeline{
		Name:    "literal-usage-analysis",
		Parents: p,
		Sources: []pipeline.Source{compEvents},
	}
	return pipe, writer
}

func runPipeline(thePipeline pipeline.Pipeline, writer *dependent.JSONWriter, args *pipelineArguments) {
	engine, err := pipeline.NewEngine(thePipeline, pipeline.DefaultEngineOptions)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = engine.Run()
	if err != nil {
		log.Fatalln(err)
	}
	if writer.Errors != 0 {
		log.Fatalf("%d errors encountered when writing to %s", writer.Errors, args.Filename)
	}
	log.Printf("%d record written to %s", writer.Written, args.Filename)
}

func main() {
	args := pipelineArguments{
		Filename: "/var/kite/../notebooks/literal_analysis/literalsCompSample_all.json",
	}
	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	maybeQuit(err)

	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	maybeQuit(err)

	pipe, writer := buildPipeline(recreator, models, recreator.Services.ResourceManager, &args)
	runPipeline(pipe, writer, &args)
}
