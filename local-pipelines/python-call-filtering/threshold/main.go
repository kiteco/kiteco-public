package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"

	"github.com/kiteco/kiteco/kite-golib/tensorflow"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/threshold"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprob"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/local-pipelines/python-call-filtering/internal/utils"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// get sorted score and original ID slice for each call.
func getSortedScoreAndID(res utils.Resources, mip metricInputs, partialCalls bool) (revFloat32Slice, error) {
	model := res.Models.FullCallProb
	if partialCalls {
		model = res.Models.PartialCallProb
	}
	callprobPred, err := model.Infer(kitectx.Background(), mip.Input)
	if err != nil {
		return revFloat32Slice{}, errors.Errorf("callProb cannot do inference: %v", err)
	}

	idSlice := make([]int, len(callprobPred))
	for i := 0; i < len(callprobPred); i++ {
		idSlice[i] = i
	}
	slice := revFloat32Slice{
		Floats: callprobPred,
		Idxs:   idSlice,
	}
	sort.Sort(slice)
	return slice, nil
}

func matchCalls(res utils.Resources, m *sync.Mutex, sc *segmentedCalls, analyticsEnc *json.Encoder, partialCalls bool) func(s pipeline.Sample) pipeline.Sample {
	return func(s pipeline.Sample) pipeline.Sample {

		cin := s.(callProbIn)

		matched, err := utils.IsPredicted(cin.Preds, cin.Original, partialCalls)
		if err != nil {
			return pipeline.WrapError("no matching prediction", err)
		}

		callprobInput := callprob.Inputs{
			RM:        res.RM,
			RAST:      cin.In.RAST,
			CallComps: cin.Preds,
			Sym:       cin.Symbol,
			Cursor:    cin.In.Cursor,
		}

		// Add segmentation to data
		mip := metricInputs{Input: callprobInput, Matches: matched}
		ssID, err := getSortedScoreAndID(res, mip, partialCalls)
		if err != nil {
			return pipeline.WrapError("unable to get scores", err)
		}

		// initializes numArgsComps to hold number of arguments for the corresponding completion.
		var numArgsComps []int
		for _, compID := range ssID.Idxs {
			numArgsComps = append(numArgsComps, utils.NumCallArgs(cin.Preds[compID].Args))
		}

		comps := ReturnedComps{
			Completions:  ssID.Idxs,
			NumArgsArray: numArgsComps,
			Scores:       ssID.Floats,
		}

		// serialize data to json file to be analyzed indepedently
		fail(analyticsEnc.Encode(metricData{Labels: mip.Matches, NumArgs: len(cin.In.Call.Args), CandComps: comps}))

		// unpack the data by completion.
		m.Lock()
		defer m.Unlock()
		for i, compID := range comps.Completions {
			matching := contains(compID, mip.Matches)
			switch numArgs := comps.NumArgsArray[i]; numArgs {
			case 0:
				sc.ZeroArgs = append(sc.ZeroArgs, threshold.Comp{IsMatched: matching, Score: comps.Scores[i]})
			case 1:
				sc.OneArgs = append(sc.OneArgs, threshold.Comp{IsMatched: matching, Score: comps.Scores[i]})
			case 2:
				sc.TwoArgs = append(sc.TwoArgs, threshold.Comp{IsMatched: matching, Score: comps.Scores[i]})
			}
		}
		return nil
	}
}

func contains(target int, s []int) bool {
	for _, i := range s {
		if target == i {
			return true
		}
	}
	return false
}

// revFloat32Slice wraps sort to get it to work with float32 and in reverse order
type revFloat32Slice struct {
	Idxs   []int
	Floats []float32
}

func (s revFloat32Slice) Swap(i, j int) {
	s.Floats[i], s.Floats[j] = s.Floats[j], s.Floats[i]
	s.Idxs[i], s.Idxs[j] = s.Idxs[j], s.Idxs[i]
}

func (s revFloat32Slice) Less(i, j int) bool {
	return s.Floats[j] < s.Floats[i]
}

func (s revFloat32Slice) Len() int {
	return len(s.Floats)
}

type metricInputs struct {
	Input   callprob.Inputs
	Matches []int
}

type segmentedCalls struct {
	ZeroArgs []threshold.Comp
	OneArgs  []threshold.Comp
	TwoArgs  []threshold.Comp
}

type srcAndCall struct {
	Src  string
	Call *pythonast.CallExpr
}

func (srcAndCall) SampleTag() {}

type callProbIn struct {
	Original  *pythonast.CallExpr
	In        utils.Input
	Symbol    pythonresource.Symbol
	Preds     []pythongraph.PredictedCall
	ScopeSize int
}

func (callProbIn) SampleTag() {}

type predictInputs struct {
	Original *pythonast.CallExpr
	In       utils.Input
}

func (predictInputs) SampleTag() {}

// TruncateCallsTransformer filter predictions to only keep partial call (without closing parenthesis)
func TruncateCallsTransformer(sample pipeline.Sample) pipeline.Sample {
	si := sample.(callProbIn)
	si.Preds = utils.TruncateCalls(si.Preds)
	return si
}

// RemoveIncompleteCalls filter predictions to only keep complete call (with closing parenthesis)
func RemoveIncompleteCalls(sample pipeline.Sample) pipeline.Sample {
	si := sample.(callProbIn)
	si.Preds = utils.KeepOnlyCompleteCalls(si.Preds)
	return si
}

func main() {
	datadeps.Enable()
	args := struct {
		Input                 string
		OutAnalytic           string
		OutParams             string
		ModelsPath            string
		MinExamplePerArgCount int
		MaxCallPerFile        int
		NumTensorflowThreads  int
		MaxSamples            int
		ExprShards            string
		NumAnalysis           int
		RunDB                 string
		RunName               string
	}{
		Input:                 "../gtdata/out",
		OutAnalytic:           "./metric_data.json",
		MinExamplePerArgCount: 500,
		MaxCallPerFile:        5,
		NumTensorflowThreads:  8,
		NumAnalysis:           4,
		RunDB:                 rundb.DefaultRunDB,
		MaxSamples:            1000,
	}

	arg.MustParse(&args)

	var runDB string
	if args.RunName != "" {
		runDB = args.RunDB
	}

	tensorflow.SetTensorflowThreadpoolSize(args.NumTensorflowThreads)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	partialCallProbModel, err := callprob.NewBaseModel(args.ModelsPath, true)
	fail(err)

	fullCallProbModel, err := callprob.NewBaseModel(args.ModelsPath, false)
	fail(err)

	modelOpts := pythonmodels.DefaultOptions
	if args.ExprShards != "" {
		shards, err := pythonexpr.ShardsFromFile(args.ExprShards)
		if err != nil {
			log.Fatalln(err)
		}
		modelOpts.ExprModelShards = shards
	}

	exprModel, err := pythonexpr.NewShardedModel(context.Background(), modelOpts.ExprModelShards, modelOpts.ExprModelOpts)
	fail(err)

	res := utils.Resources{RM: rm, Models: &pythonmodels.Models{PartialCallProb: partialCallProbModel,
		FullCallProb: fullCallProbModel,
		Expr:         exprModel}}

	outf, err := os.Create(args.OutAnalytic)
	fail(err)
	defer outf.Close()

	analyticsEnc := json.NewEncoder(outf)

	samples := make(chan *utils.Sample)
	go func() {
		var count int
		defer func() {
			close(samples)
		}()

		file, err := fileutil.NewCachedReader(args.Input)
		fail(err)
		defer file.Close()

		decoder := json.NewDecoder(file)

		for {
			var s utils.Sample
			err := decoder.Decode(&s)
			switch err {
			case nil:
				samples <- &s
				count++
				if args.MaxSamples > 0 && count > args.MaxSamples {
					return
				}
			case io.EOF:
				return
			default:
				fail(err)
			}
		}
	}()

	samplesSource := source.Func("samples", func() pipeline.Record {
		s := <-samples
		if s == nil {
			return pipeline.Record{}
		}
		return pipeline.Record{
			Value: *s,
		}
	})

	calls := transform.NewMap("calls", func(s pipeline.Sample) []pipeline.Sample {
		sample := s.(utils.Sample)
		calls, err := utils.FindCalls(sample)
		if err != nil {
			return []pipeline.Sample{pipeline.WrapError("error finding calls", err)}
		}

		if len(calls) > args.MaxCallPerFile {
			rand.Shuffle(len(calls), func(i, j int) {
				calls[i], calls[j] = calls[j], calls[i]
			})
			calls = calls[:args.MaxCallPerFile]
		}

		var samples []pipeline.Sample
		for _, c := range calls {
			samples = append(samples, srcAndCall{
				Src:  string(sample.Source),
				Call: c,
			})
		}
		return samples
	})

	inputs := transform.NewOneInOneOut("inputs", func(s pipeline.Sample) pipeline.Sample {
		sc := s.(srcAndCall)

		// do all the munging then return expr input and pass to expr model to get the prediction
		inp, err := utils.TryCall(sc.Call, []byte(sc.Src), res)
		if err != nil {
			return pipeline.WrapError("error getting inputs", err)
		}

		return predictInputs{
			Original: sc.Call,
			In:       inp,
		}
	})

	predict := transform.NewOneInOneOut("predict", func(s pipeline.Sample) pipeline.Sample {
		in := s.(predictInputs)

		preds, sym, scopeSize, err := res.PredictCall(in.In.Src, in.In.Words, in.In.RAST, in.In.Call)
		if err != nil {
			return pipeline.WrapError("predict error", err)
		}

		return callProbIn{
			Original:  in.Original,
			In:        in.In,
			Symbol:    sym,
			Preds:     preds,
			ScopeSize: scopeSize,
		}
	})

	callTruncator := transform.NewOneInOneOut("Call truncator", TruncateCallsTransformer)
	fullCallFilter := transform.NewOneInOneOut("Full Call filter", RemoveIncompleteCalls)

	var mPartial, mFull sync.Mutex
	var scPartial, scFull segmentedCalls
	matchPartial := transform.NewOneInOneOut("matchPartial", matchCalls(res, &mPartial, &scPartial, analyticsEnc, true))
	matchFull := transform.NewOneInOneOut("matchFull", matchCalls(res, &mFull, &scFull, analyticsEnc, false))

	pm := make(pipeline.ParentMap)

	callPredictor := pm.Chain(
		samplesSource,
		calls,
		inputs,
		predict,
	)

	pm.Chain(callPredictor,
		callTruncator,
		matchPartial)

	pm.Chain(callPredictor,
		fullCallFilter,
		matchFull)

	eOpts := pipeline.DefaultEngineOptions
	eOpts.RunDBPath = runDB
	eOpts.RunName = args.RunName
	eOpts.NumWorkers = args.NumAnalysis

	pipe := pipeline.Pipeline{
		Name:    "call-prob-threshold",
		Parents: pm,
		Sources: []pipeline.Source{samplesSource},
	}

	engine, err := pipeline.NewEngine(pipe, eOpts)
	fail(err)

	_, err = engine.Run()
	fail(err)

	thresholdsPartial := threshold.Thresholds{
		ZeroArgs: threshold.GetOptimalThreshold(scPartial.ZeroArgs),
		OneArgs:  threshold.GetOptimalThreshold(scPartial.OneArgs),
		TwoArgs:  threshold.GetOptimalThreshold(scPartial.TwoArgs),
	}
	thresholdsFull := threshold.Thresholds{
		ZeroArgs: threshold.GetOptimalThreshold(scFull.ZeroArgs),
		OneArgs:  threshold.GetOptimalThreshold(scFull.OneArgs),
		TwoArgs:  threshold.GetOptimalThreshold(scFull.TwoArgs),
	}

	outf2, err := os.Create(args.OutParams)
	fail(err)
	defer outf2.Close()

	encoder2 := json.NewEncoder(outf2)
	fail(encoder2.Encode(threshold.Set{
		FullCall:    thresholdsFull,
		PartialCall: thresholdsPartial,
	}))

	fmt.Println("Partial calls :\n", thresholdsPartial.String(len(scPartial.ZeroArgs), len(scPartial.OneArgs), len(scPartial.TwoArgs)))
	fmt.Println("Full calls :\n", thresholdsFull.String(len(scFull.ZeroArgs), len(scFull.OneArgs), len(scFull.TwoArgs)))
}
