package callprobutils

import (
	"math/rand"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"

	"github.com/kiteco/kiteco/kite-golib/linenumber"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/local-pipelines/python-call-filtering/internal/utils"
)

const (
	maxCallsPerFile = 5
)

// CallToPredict contains all info about a call location that will then be used to generate samples
type CallToPredict struct {
	Hash      string
	Src       []byte
	Call      *pythonast.CallExpr
	Src2      []byte
	Call2     *pythonast.CallExpr
	RAST2     *pythonanalyzer.ResolvedAST
	Words2    []pythonscanner.Word
	UserTyped string
}

// SampleTag implements pipeline.Sample
func (CallToPredict) SampleTag() {}

// GetCallsToPredict find a set of valid call where we can make prediction in the provided source file
func GetCallsToPredict(hash string, src []byte, res utils.Resources, r *rand.Rand) ([]CallToPredict, error) {
	ast, _, err := utils.Parse(src)
	if err != nil {
		return nil, pipeline.WrapErrorAsError("unable to parse ast", err)
	}

	calls := findCalls(ast)
	if len(calls) == 0 {
		return nil, pipeline.NewErrorAsError("no calls found")
	}

	rast, err := utils.Resolve(ast, res.RM)
	if err != nil {
		return nil, pipeline.WrapErrorAsError("error resolving AST", err)
	}

	lm := linenumber.NewMap(src)

	var samples []CallToPredict
	for _, i := range r.Perm(len(calls)) {
		c, err := tryCall(hash, src, lm, calls[i], rast, res)
		if err == nil {
			samples = append(samples, c)
		}

		if len(samples) >= maxCallsPerFile {
			break
		}
	}

	if len(samples) == 0 {
		return nil, pipeline.NewErrorAsError("no valid calls found")
	}
	return samples, nil
}

func tryCall(hash string, src []byte, lm *linenumber.Map, call *pythonast.CallExpr, rast *pythonanalyzer.ResolvedAST, res utils.Resources) (CallToPredict, error) {
	val := rast.References[call.Func]
	if val == nil {
		return CallToPredict{}, errors.Errorf("value is unresolved")
	}

	if !utils.ValueSupported(res, val) {
		return CallToPredict{}, errors.Errorf("unsupported attribute value")
	}

	src2, words2, ast2, call2, err := pythonpipeline.MungeBufferForCall(rast, lm, src, call)
	if err != nil {
		return CallToPredict{}, err
	}

	rast2, err := utils.Resolve(ast2, res.RM)
	if err != nil {
		return CallToPredict{}, errors.Errorf("unable to re-resolve file: %v", err)
	}

	return CallToPredict{
		Hash:   hash,
		Src:    src,
		Call:   call,
		Src2:   src2,
		Call2:  call2,
		RAST2:  rast2,
		Words2: words2,
	}, nil
}

// Predict computes prediction for the provided call location
func Predict(res utils.Resources, c CallToPredict, r *rand.Rand) pipeline.Sample {
	preds, sym, scopeSize, err := res.PredictCall(c.Src2, c.Words2, c.RAST2, c.Call2)
	if err != nil {
		return pipeline.WrapError("infer call error", err)
	}

	if len(preds) == 0 {
		return pipeline.NewError("no predictions")
	}

	const (
		maxPred = 10
		topPred = 5
	)

	if len(preds) > maxPred {
		// More than 10 preds, we take the top 5 and 5 random in the remaining
		var result []pythongraph.PredictedCall
		if topPred > 0 {
			sort.Slice(preds, func(i, j int) bool {
				return preds[i].Prob > preds[j].Prob
			})
			result = append(result, preds[:topPred]...)
		}
		for _, p := range r.Perm(len(preds) - topPred)[:maxPred-topPred] {
			result = append(result, preds[p+topPred])
		}
		preds = result
	}
	return SampleInputs{
		Hash:      c.Hash,
		Cursor:    int64(c.Call.LeftParen.End),
		RAST:      c.RAST2,
		Sym:       sym,
		UserTyped: c.Src[c.Call.LeftParen.End:c.Call.RightParen.Begin],
		UserCall:  c.Call,
		CallComps: preds,
		ScopeSize: scopeSize,
	}
}

func findCalls(ast *pythonast.Module) []*pythonast.CallExpr {
	var calls []*pythonast.CallExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		switch n := n.(type) {
		case *pythonast.CallExpr:
			calls = append(calls, n)
		}
		return true
	})
	return calls
}
