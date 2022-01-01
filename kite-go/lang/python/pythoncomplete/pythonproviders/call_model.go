package pythonproviders

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprobcallmodel"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// CallModel is a Provider for the call model
// it completes situations such as foo(‸) with snippet-containing call patterns.
type CallModel struct{}

// returns true if the confidence score is lower than the threshold and should be skipped
func skipCallComp(g Global, mc MetaCompletion) bool {
	if mc.CallModelMeta.CallProb < 0 {
		// Invalid call, should be skipped anyway
		return true
	}
	thresholds := g.Models.CallModelCallProb.Params()
	switch mc.CallModelMeta.NumConcreteArgs {
	case 0:
		if mc.CallModelMeta.CallProb < float64(thresholds.ZeroArgs) {
			return true
		}
	case 1:
		if mc.CallModelMeta.CallProb < float64(thresholds.OneArgs) {
			return true
		}
	case 2:
		if mc.CallModelMeta.CallProb < float64(thresholds.TwoArgs) {
			return true
		}
	}
	return false
}

// Name implements Provider
func (CallModel) Name() data.ProviderName {
	return data.PythonCallModelProvider
}

// Provide implements Provider
func (p CallModel) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	if !ggnnCompletionsEnabled() || applesilicon.Detected {
		return nil
	}

	_, isSmart := SmartProviders[p.Name()]
	if isSmart && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	if g.Models == nil || !g.Models.Expr.IsLoaded() || !g.Models.CallModelCallProb.IsLoaded() {
		return nil
	}

	// for now we only handle cursors, not selections: TODO(naman)
	if in.Selection.Len() > 0 {
		return data.ProviderNotApplicableError{}
	}

	var callExpr *pythonast.CallExpr
	for _, n := range in.UnderSelection() {
		if n, _ := n.(*pythonast.CallExpr); n != nil {
			// if in.Selection.Begin == callExpr.LeftParen.Begin then the selection
			// includes the left paren which we do not want
			if in.Selection.Begin >= int(n.LeftParen.End) {
				// the end of selection is "exclusive" so in.Selection.End == callExpr.Righparen.Begin is ok
				// since the right paren is not included in the selection
				if n.RightParen == nil || in.Selection.End <= int(n.RightParen.Begin) {
					callExpr = n
					break
				}
			}
		}
	}
	if callExpr == nil {
		return data.ProviderNotApplicableError{}
	}

	if !pythonast.IsNil(callExpr.Vararg) || !pythonast.IsNil(callExpr.Kwarg) {
		return data.ProviderNotApplicableError{}
	}

	// deal with cases when the user has already typed part of the call
	argCount := getArgCount(callExpr.Args)
	if argCount > 0 {
		// make sure the user has typed the comma
		// NOTE: the call model expects that len(callExpr.Commas) >= len(callExpr.Args)
		if argCount != len(callExpr.Commas) {
			return data.ProviderNotApplicableError{}
		}

		// make sure the cursor is after the last comma
		lastComma := callExpr.Commas[len(callExpr.Commas)-1]
		if in.Selection.Begin <= int(lastComma.Begin) {
			return data.ProviderNotApplicableError{}
		}
	}

	// deep copy the RAST since we modify it during prediction
	rast, newNodes := in.ResolvedAST().DeepCopy()
	callExpr = newNodes[callExpr].(*pythonast.CallExpr)

	callProbIn := callprobcallmodel.Inputs{
		RM:     g.ResourceManager,
		Cursor: int64(in.Selection.Begin),
		RAST:   rast,
	}

	ggnnNode, err := g.Models.Expr.Predict(ctx, pythonexpr.Input{
		RM:                          g.ResourceManager,
		RAST:                        rast,
		Words:                       in.Words(),
		Expr:                        callExpr,
		MaxPatterns:                 maxCallPatternsForAttrCompletions,
		AlwaysUsePopularityForAttrs: alwaysUsePopularityForAttrs,
	})
	if err != nil {
		return err
	}
	node := ggnnNode.OldPredictorResult
	var c MetaCompletion
	c.Provider = p.Name()

	c.Replace.End = int(callExpr.End())
	if len(callExpr.Args) == 0 {
		c.Replace.Begin = int(callExpr.LeftParen.Begin)
	} else {
		// Use the selection instead of the end of the last comma to avoid weird
		// issues with whitespace between the last comma and the cursor
		c.Replace.Begin = in.Selection.Begin
	}

	c.FromSmartProvider = isSmart

	for _, child := range node.Children {
		if len(child.Call.Predicted) > 0 {
			callProbIn.NumOrigArgs = child.Call.Predicted[0].NumOrigArgs
		}

		sigStats := g.ResourceManager.SigStats(child.Call.Symbol)
		callProbConf := computeCallScoreBatchForCallModel(ctx, g.Models.CallModelCallProb, child.Call, callProbIn)
		for i, call := range child.Call.Predicted {
			// setting a RenderMeta here will cause these completions to be thought of
			// as function completions, even though they're not. We manually fix rendering
			// for the nested case during mixing/rendering.

			argString := argsForCall(call.Args, sigStats, argCount)
			if call.NumOrigArgs == 0 {
				argString = fmt.Sprintf("(%s)", argString)
			} else {
				argString = fmt.Sprintf("%s)", argString)
			}
			c.Snippet = data.BuildSnippet(argString)
			c.Score = float64(call.Prob)
			c.CallModelMeta = &CallModelMeta{
				FunctionSym:     child.Call.Symbol,
				NumArgs:         numCallArgs(call.Args) + call.NumOrigArgs,
				ArgSpec:         g.ResourceManager.ArgSpec(child.Call.Symbol),
				NumOrigArgs:     call.NumOrigArgs,
				NumConcreteArgs: numConcreteArgs(call.Args),
				Call:            &child.Call.Predicted[i],
			}

			// get mixing meta
			c.Source = response.CallModelCompletionSource
			c.CallModelMeta.CallProb = math.NaN()

			if i < len(callProbConf) {
				c.CallModelMeta.CallProb = float64(callProbConf[i])
			}

			if !skipCallComp(g, c) {
				out(ctx, in.SelectedBuffer, c)
			}
		}
	}
	return nil
}

// MarshalJSON implements Provider
func (p CallModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: p.Name(),
	})
}

// CompareCompletions implements CompletionComparator.
func (CallModel) CompareCompletions(comp1, comp2 MetaCompletion) bool {
	if math.IsNaN(comp1.CallModelMeta.CallProb) && math.IsNaN(comp2.CallModelMeta.CallProb) {
		return comp1.Score > comp2.Score
	}
	if math.IsNaN(comp1.CallModelMeta.CallProb) {
		return false
	}
	if math.IsNaN(comp2.CallModelMeta.CallProb) {
		return true
	}
	if math.Abs(comp1.CallModelMeta.CallProb-comp2.CallModelMeta.CallProb) >= 1e-6 {
		return comp1.CallModelMeta.CallProb > comp2.CallModelMeta.CallProb
	}
	return comp1.Score > comp2.Score
}

func numConcreteArgs(args []pythongraph.PredictedCallArg) int {
	var n int
	for _, arg := range args {
		if arg.Stop {
			break
		}
		if arg.Value != pythongraph.PlaceholderPlaceholder && arg.Value != "" {
			n++
		}
	}
	return n
}
