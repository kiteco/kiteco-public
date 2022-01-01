package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"runtime/pprof"
	"sort"
	"time"

	arg "github.com/alexflint/go-arg"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/status"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

var (
	section = status.NewSection("python-offline-metrics/ggnn-call-completions")

	predictionDuration = section.SampleDuration("AttributeCallPredictions")
)

const (
	inferTimeout = 30 * time.Second
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type validCall struct {
	Call  *pythonast.CallExpr
	Evt   pythonpipeline.AnalyzedEvent
	Lines *linenumber.Map
}

func (validCall) SampleTag() {}

func validCalls(model pythonexpr.Model, ev pythonpipeline.AnalyzedEvent) []pipeline.Sample {
	lm := linenumber.NewMap(ev.Context.Buffer)
	var valid []pipeline.Sample
	pythonast.Inspect(ev.Context.AST, func(n pythonast.Node) bool {
		call, ok := n.(*pythonast.CallExpr)
		if !ok {
			return true
		}

		// only pick calls that are not nested within other calls
		// and not nested as the base of an attribute expr
		if _, ok := ev.Context.Resolved.Parent[call].(pythonast.Stmt); !ok {
			return false
		}

		if pythonpipeline.ModelCanPredictCall(kitectx.Background(), ev.Context.Importer.Global, model, ev.Context.Resolved, call) {
			valid = append(valid, validCall{
				Call:  call,
				Evt:   ev,
				Lines: lm,
			})
		}

		return true
	})

	return valid
}

func newLabel(call *pythonast.CallExpr) pythongraph.PredictedCall {
	var label pythongraph.PredictedCall
	for _, arg := range call.Args {
		var name string
		if !pythonast.IsNil(arg.Name) {
			name = arg.Name.(*pythonast.NameExpr).Ident.Literal
		}
		if val, ok := arg.Value.(*pythonast.NameExpr); ok {
			label.Args = append(label.Args, pythongraph.PredictedCallArg{
				Value: val.Ident.Literal,
				Name:  name,
				Prob:  1.,
			})
		} else {
			label.Args = append(label.Args, pythongraph.PredictedCallArg{
				Value: pythongraph.PlaceholderPlaceholder,
				Name:  name,
				Prob:  1.,
			})
		}
	}
	// add a stop token in the label so we can compare with the predicted args that have stop token as the last argument.
	label.Args = append(label.Args, pythongraph.PredictedCallArg{
		Stop: true,
		Prob: 1,
	})
	return label
}

func newPredicted(ptn *pythongraph.PredictionTreeNode) pythongraph.PredictedCallSummary {
	var pcs pythongraph.PredictedCallSummary
	for _, c := range ptn.Children {
		if len(c.Call.Predicted) > 0 {
			// there should be exactly one predicted call
			pcs = c.Call
			break
		}
	}

	return pcs
}

func makeArgMap(args []pythongraph.PredictedCallArg, argSpecArgs []pythonimports.Arg) map[string]string {
	argMap := make(map[string]string)
	for i, a := range args {
		if a.Stop {
			break
		}
		if a.Name == "" {
			if i < len(argSpecArgs) {
				argMap[argSpecArgs[i].Name] = a.Value
			} else {
				// if this is a vararg then we set the key such that
				// varargs are matched based on ordering and value.
				// thus for example `plt.plot(x,y)` will not match `plt.plot(y,x)`
				argMap[fmt.Sprintf("vararg_%d", i)] = a.Value
			}
		} else {
			argMap[a.Name] = a.Value
		}
	}
	return argMap
}

type predictedCall struct {
	Predicted   pythongraph.PredictedCallSummary
	Label       pythongraph.PredictedCall
	ArgSpecArgs []pythonimports.Arg
}

func (predictedCall) SampleTag() {}

func inferCallTrimSrc(model pythonexpr.Model, maxPatterns int, vc validCall) pipeline.Sample {
	stmt := vc.Evt.Context.Resolved.ParentStmts[vc.Call]
	line := vc.Lines.Line(int(stmt.Begin()))
	if vc.Lines.Line(int(stmt.End())) != line {
		return pipeline.NewError(fmt.Sprintf("unable to trim source: multiline statement"))
	}

	_, lineEnd := vc.Lines.LineBounds(line)
	if lineEnd < len(vc.Evt.Event.Buffer) {
		lineEnd++ // +1 so we start at newline
	}

	// remove args and everything up to the end of the line
	src := bytes.Join([][]byte{
		vc.Evt.Context.Buffer[:vc.Call.LeftParen.End],
		[]byte(")"),
		vc.Evt.Context.Buffer[lineEnd:],
	}, nil)

	ast, words, err := pythonpipeline.Parse(pythonparser.Options{
		ErrorMode:   pythonparser.Recover,
		Approximate: true,
	}, time.Second, sample.ByteSlice(src))

	if err != nil {
		return pipeline.WrapError("reparse error", err)
	}

	var call *pythonast.CallExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if call != nil || pythonast.IsNil(n) {
			return false
		}

		c, ok := n.(*pythonast.CallExpr)
		if !ok {
			return true
		}

		if c.LeftParen.Begin == vc.Call.LeftParen.Begin {
			call = c
		}

		return true
	})

	if call == nil || len(call.Args) != 0 || call.RightParen == nil {
		return pipeline.NewError("unable to refind call after trimming source")
	}

	rast, err := pythonpipeline.Resolve(vc.Evt.Context.Importer.Global, time.Second, pythonpipeline.Parsed{
		Mod:   ast,
		Words: words,
	})

	if err != nil {
		return pipeline.WrapError("error reanalyzing source", err)
	}

	// predict back call arguments
	start := time.Now()
	var pred *pythongraph.PredictionTreeNode
	err = kitectx.Background().WithTimeout(inferTimeout, func(ctx kitectx.Context) error {
		res, err := model.Predict(ctx, pythonexpr.Input{
			RM:          vc.Evt.Context.Importer.Global,
			RAST:        rast,
			Words:       words,
			Expr:        call,
			MaxPatterns: maxPatterns,
		})

		if err != nil {
			return err
		}
		pred = res.OldPredictorResult
		return nil
	})

	if err != nil {
		if _, ok := err.(kitectx.ContextExpiredError); ok {
			return pipeline.NewError("prediction took too long")
		}
		return pipeline.WrapError("infer error", err)
	}

	pcs := newPredicted(pred)
	if len(pcs.Predicted) == 0 {
		return pipeline.NewError("no predictions for call")
	}

	// only record duration for sucessful predictions
	predictionDuration.RecordDuration(time.Since(start))

	rm := vc.Evt.Context.Importer.Global
	val := rast.References[call.Func]
	sym, err := python.GetExternalSymbol(kitectx.Background(), rm, val)
	if err != nil {
		return pipeline.NewError("could not get function symbol")
	}

	argSpecs := rm.ArgSpec(sym)
	if argSpecs == nil {
		return pipeline.WrapError("no arg spec for symbol", errors.Errorf("no arg spec for symbol %v", sym))
	}

	return predictedCall{
		Predicted:   pcs,
		Label:       newLabel(vc.Call),
		ArgSpecArgs: argSpecs.Args,
	}
}

type callComparison struct {
	// Match will only be true if it's an exact match (including positional and keyword arguments)
	Match                    bool
	NumArgs                  int
	MatchPos                 int
	IsPlaceholder            bool
	MisPredictPlaceholder    bool
	MisPredictPlaceholderPos int
}

func (callComparison) SampleTag() {}

func newCallComparison(p predictedCall) callComparison {
	cc := callComparison{
		// subtract one here because of the artificially added stop token
		NumArgs: len(p.Label.Args) - 1,
	}

	labelArgMap := makeArgMap(p.Label.Args, p.ArgSpecArgs)

	for _, label := range p.Label.Args {
		if label.Value == pythongraph.PlaceholderPlaceholder {
			cc.IsPlaceholder = true
			break
		}
	}

	for i, call := range p.Predicted.Predicted {
		if len(call.Args) != len(p.Label.Args) {
			continue
		}

		for j := range p.Label.Args {
			// We mostly care about false negative (ie putting a name when it should be a placeholder).
			// As in this case, the user has to manually edit back the completion
			// In the false positive case (placeholder instead of name), the user will just have to fill the placeholder.
			if cc.IsPlaceholder && call.Args[j].Value != pythongraph.PlaceholderPlaceholder {
				// Get the highest place that mispredicts
				if !cc.MisPredictPlaceholder {
					cc.MisPredictPlaceholderPos = i
					cc.MisPredictPlaceholder = true
					break
				}
			}
		}

		callArgMap := makeArgMap(call.Args, p.ArgSpecArgs)
		if reflect.DeepEqual(labelArgMap, callArgMap) {
			cc.Match = true
			cc.MatchPos = i
			return cc
		}
	}

	return cc
}

type callsSummaryStats struct {
	Calls    int
	Match    int
	MatchPos int
}

type placeholderStats struct {
	MisPredictPlaceholder    int
	MisPredictPlaceholderPos int
}

type callsSummary struct {
	PlaceholderCall callsSummaryStats
	ConcreteCall    callsSummaryStats
	Placeholder     placeholderStats
}

func newCallsSummary(cc callComparison) callsSummary {
	var cs callsSummary
	if cc.IsPlaceholder {
		cs.PlaceholderCall.Calls++
		if cc.Match {
			cs.PlaceholderCall.Match++
			cs.PlaceholderCall.MatchPos += cc.MatchPos
		}
	} else {
		cs.ConcreteCall.Calls++
		if cc.Match {
			cs.ConcreteCall.Match++
			cs.ConcreteCall.MatchPos += cc.MatchPos
		}
		if cc.MisPredictPlaceholder {
			cs.Placeholder.MisPredictPlaceholder++
			cs.Placeholder.MisPredictPlaceholderPos += cc.MisPredictPlaceholderPos
		}
	}
	return cs
}

func (cs callsSummary) Add(cso callsSummary) callsSummary {
	cs.PlaceholderCall.Calls += cso.PlaceholderCall.Calls
	cs.PlaceholderCall.Match += cso.PlaceholderCall.Match
	cs.PlaceholderCall.MatchPos += cso.PlaceholderCall.MatchPos
	cs.ConcreteCall.Calls += cso.ConcreteCall.Calls
	cs.ConcreteCall.Match += cso.ConcreteCall.Match
	cs.ConcreteCall.MatchPos += cso.ConcreteCall.MatchPos
	cs.Placeholder.MisPredictPlaceholder += cso.Placeholder.MisPredictPlaceholder
	cs.Placeholder.MisPredictPlaceholderPos += cso.Placeholder.MisPredictPlaceholderPos
	return cs
}

func (cs callsSummary) Results(name string) []rundb.Result {
	var results []rundb.Result
	addResults := func(metricName string, count, total int) {
		results = append(results, rundb.Result{
			Name:  fmt.Sprintf("%s: count %s", name, metricName),
			Value: count,
		})

		var avg float64
		if total > 0 {
			avg = float64(count) / float64(total)
		}

		results = append(results, rundb.Result{
			Name:  fmt.Sprintf("%s: avg %s", name, metricName),
			Value: avg,
		})
	}

	results = append(results, rundb.Result{
		Name:  fmt.Sprintf("%s: num concrete calls", name),
		Value: cs.ConcreteCall.Calls,
	})

	results = append(results, rundb.Result{
		Name:  fmt.Sprintf("%s: num placeholder calls", name),
		Value: cs.PlaceholderCall.Calls,
	})

	addResults("exact match of arguments for concrete calls", cs.ConcreteCall.Match, cs.ConcreteCall.Calls)

	addResults("match pos for concrete calls", cs.ConcreteCall.MatchPos, cs.ConcreteCall.Match)

	addResults("exact match of arguments for placeholder calls", cs.PlaceholderCall.Match, cs.PlaceholderCall.Calls)

	addResults("match pos for placeholder calls", cs.PlaceholderCall.MatchPos, cs.PlaceholderCall.Match)

	addResults("mispredict placeholder calls", cs.Placeholder.MisPredictPlaceholder, cs.ConcreteCall.Calls)

	addResults("mispredict placeholder position", cs.Placeholder.MisPredictPlaceholderPos, cs.Placeholder.MisPredictPlaceholder)

	return results
}

type summary struct {
	Calls map[int]callsSummary
}

func newSummary(cc callComparison) summary {
	return summary{
		Calls: map[int]callsSummary{cc.NumArgs: newCallsSummary(cc)},
	}
}

func (s summary) Add(so summary) summary {
	for k, v := range so.Calls {
		s.Calls[k] = s.Calls[k].Add(v)
	}
	return s
}

type keyedSummary map[string]summary

// SampleTag implements sample.Addable
func (keyedSummary) SampleTag() {}

// Add implements sample.Addable
func (k keyedSummary) Add(o sample.Addable) sample.Addable {
	for key, s := range o.(keyedSummary) {
		if existing, ok := k[key]; ok {
			k[key] = existing.Add(s)
		} else {
			k[key] = s
		}
	}
	return k
}

func main() {
	fail(datadeps.Enable())
	args := struct {
		MaxEvents            int
		RunDB                string
		RunName              string
		Model                string
		UseUncompressed      bool
		NumTensorflowThreads int
		NumReader            int
		NumWorkers           int
		MaxCallsPerFile      int
		Benchmark            bool
	}{
		MaxEvents:            1000,
		RunDB:                "s3://kite-data/run-db",
		RunName:              "",
		Model:                pythonmodels.DefaultOptions.ExprModelShards[0].ModelPath,
		UseUncompressed:      false,
		NumTensorflowThreads: 1,
		NumReader:            2,
		NumWorkers:           4,
		MaxCallsPerFile:      5,
	}
	arg.MustParse(&args)

	var runDB string
	if args.RunName != "" {
		runDB = args.RunDB
	}

	tensorflow.SetTensorflowThreadpoolSize(args.NumTensorflowThreads)

	start := time.Now()
	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	fail(err)

	mopts := pythonexpr.DefaultOptions
	mopts.UseUncompressed = args.UseUncompressed

	model, err := pythonexpr.NewModel(args.Model, mopts)
	fail(err)

	startDate := analyze.NewDate(2018, 12, 12)
	endDate := analyze.NewDate(2019, 01, 15)

	compEvents := pythonpipeline.NewTrackingEvents(
		startDate,
		endDate,
		pythontracking.ServerSignatureFailureEvent,
		pythonpipeline.TrackingEventsOpts{
			NumReaders: args.NumReader,
			MaxEvents:  args.MaxEvents,
			DedupByUMF: true,
		},
	)

	pm := make(pipeline.ParentMap)

	comps := pm.Chain(
		compEvents,
		transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false)),
		transform.NewMap("valid-calls", func(s pipeline.Sample) []pipeline.Sample {
			calls := validCalls(model, s.(pythonpipeline.AnalyzedEvent))
			rand.Shuffle(len(calls), func(i, j int) { calls[i], calls[j] = calls[j], calls[i] })
			if len(calls) > args.MaxCallsPerFile {
				calls = calls[:args.MaxCallsPerFile]
			}
			return calls
		}),
		transform.NewOneInOneOut("call-completions", func(s pipeline.Sample) pipeline.Sample {
			res := inferCallTrimSrc(model, 2, s.(validCall))
			if _, ok := res.(predictedCall); !ok {
				// we got an error so we cannot wrap it in the keyed struct
				// TODO: should the pipeline handle checking keyed values for errors?
				return res
			}
			return pipeline.Keyed{
				Key:    "trim-src",
				Sample: res,
			}
		}))

	summaries := pm.Chain(
		comps,
		transform.NewOneInOneOutKeyed("call-comparison", func(s pipeline.Sample) pipeline.Sample {
			return newCallComparison(s.(predictedCall))
		}),
	)

	agg := aggregator.NewSumAggregator(
		"summary-agg",
		func() sample.Addable { return make(keyedSummary) },
		func(s pipeline.Sample) sample.Addable {
			keyed := s.(pipeline.Keyed)
			cc := keyed.Sample.(callComparison)
			return keyedSummary{
				keyed.Key: newSummary(cc),
			}
		},
	)

	pm.Chain(comps, summaries, agg)

	resFn := func(res map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		results := []rundb.Result{{
			Name:  "runtime",
			Value: fmt.Sprintf("%v", time.Since(start)),
		}}

		ks := res[agg].(keyedSummary)

		var keys []string
		for k := range ks {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			summary := ks[k]
			var args []int
			for a := range summary.Calls {
				args = append(args, a)
			}

			sort.Ints(args)
			for _, a := range args {
				// disregard numarg = -1
				if a >= 0 {
					cs := summary.Calls[a]
					results = append(results, cs.Results(fmt.Sprintf("%s-num-args-%d", k, a))...)
				}
			}

		}

		addPercentileResults := func(name string, ps []float64, values []int64) {
			for i, p := range ps {
				results = append(results, rundb.Result{
					Name:  fmt.Sprintf("%s-%v", name, p),
					Value: fmt.Sprintf("%v", time.Duration(values[i])),
				})
			}
		}

		addPercentileResults("Attribute call predictions", section.Percentiles(), predictionDuration.Values())

		graphSection := status.Get().Sections[pythongraph.StatusSectionName]

		var names []string
		for name := range graphSection.SampleDurations {
			names = append(names, name)
		}
		sort.Strings(names)

		ps := graphSection.Percentiles()
		for _, name := range names {
			vs := graphSection.SampleDurations[name].Values()
			addPercentileResults(name, ps, vs)
		}

		return results
	}

	pipe := pipeline.Pipeline{
		Name:      "ggnn-call-completions-validate",
		Parents:   pm,
		Sources:   []pipeline.Source{compEvents},
		ResultsFn: resFn,
		Params: map[string]interface{}{
			"ExprModel":       args.Model,
			"MaxEvents":       args.MaxEvents,
			"StartDate":       time.Time(startDate),
			"EndDate":         time.Time(endDate),
			"UseUncompressed": fmt.Sprintf("%v", args.UseUncompressed),
		},
	}

	opts := pipeline.DefaultEngineOptions
	opts.NumWorkers = args.NumWorkers
	opts.RunDBPath = runDB
	opts.RunName = args.RunName
	opts.Role = pipeline.Standalone

	engine, err := pipeline.NewEngine(pipe, opts)
	fail(err)

	if args.Benchmark {
		f, err := os.Create("profile")
		fail(err)
		fail(pprof.StartCPUProfile(f))
		defer pprof.StopCPUProfile()
	}

	_, err = engine.Run()
	fail(err)
}
