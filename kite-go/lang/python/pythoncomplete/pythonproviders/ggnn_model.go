package pythonproviders

import (
	"encoding/json"
	"fmt"
	"go/token"
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprob"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const promotionFactor = 0.5

// GGNNModel is a Provider for model generated completions
type GGNNModel struct {
	ForceDisableFiltering bool
}

// Name implements Provider
func (p GGNNModel) Name() data.ProviderName {
	return data.PythonGGNNModelProvider
}

// implements CompletionsPromoter
func (GGNNModel) promoted() {}

// GetDistanceFromRoot implements CompletionPromoter
func (GGNNModel) GetDistanceFromRoot(mc MetaCompletion) int {
	if mc.GGNNMeta == nil || mc.GGNNMeta.Call == nil {
		return 0
	}
	var numArgs int
	for _, arg := range mc.GGNNMeta.Call.Args {
		if arg.Stop {
			break
		}
		if arg.Value == pythongraph.PlaceholderPlaceholder {
			continue
		}
		numArgs++
	}
	return numArgs
}

// Provide ...
func (p GGNNModel) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	if !ggnnCompletionsEnabled() || applesilicon.Detected {
		return nil
	}

	if _, ok := SmartProviders[p.Name()]; ok && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	if g.Models == nil || !g.Models.Expr.IsLoaded() || !g.Models.FullCallProb.IsLoaded() ||
		!g.Models.PartialCallProb.IsLoaded() ||
		!g.Models.MTACConf.IsLoaded() {
		return nil
	}
	if in.GGNNPredictor != nil {
		// TODO is is a problem to not call the model though the ShardedModel interface?
		// It should use the model that got used during the last expand call
		results, err := in.GGNNPredictor.Expand(ctx)
		if err != nil {
			return err
		}

		return p.processPartialResults(ctx, g, in, results, out, in.SelectedBuffer, in.Selection, nil)
	}
	// TODO: fix the way the ggnn is initialized when we are selecting an argument in the middle of a call and remove this quick fix.
	if in.Selection.Len() > 0 {
		return data.ProviderNotApplicableError{}
	}
	var expr pythonast.Expr
	replace := in.Selection
	for _, n := range in.UnderSelection() {
		if n, _ := n.(*pythonast.CallExpr); n != nil {
			if (n.LeftParen != nil && n.LeftParen.End <= token.Pos(in.End)) &&
				(n.RightParen == nil || n.RightParen.End > token.Pos(in.Begin)) {
				expr = n
			}
		}
		if n, _ := n.(*pythonast.AttributeExpr); n != nil {
			expr = n
			replace = data.Selection{
				Begin: int(n.Dot.End),
				End:   in.Selection.End,
			}
		}
		if n, _ := n.(*pythonast.NameExpr); n != nil {
			// No update of the expr as we don't yet support NameExpr completion
			// So we need to keep the CallExpr surrounding it if there's one
			// And we return NotApplicable if it's only a name at a line begin
			replace = data.Selection{
				Begin: int(n.Begin()),
				End:   int(n.End()),
			}

		}
	}

	if expr == nil {
		return data.ProviderNotApplicableError{}
	}
	bufBytes := []byte(in.Buffer.Text())
	rast := in.ResolvedAST()
	newRAST, newNodes := rast.DeepCopy()
	res, err := g.Models.Expr.Predict(ctx, pythonexpr.Input{
		RM:                          g.ResourceManager,
		RAST:                        newRAST,
		Words:                       in.Words(), // not modified
		Src:                         bufBytes,
		Expr:                        newNodes[expr].(pythonast.Expr),
		MaxPatterns:                 maxCallPatternsForAttrCompletions,
		AlwaysUsePopularityForAttrs: alwaysUsePopularityForAttrs,
		MungeBufferForAttrs:         mungeBufferForAttrs,
		UsePartialDecoder:           in.UsePartialDecoder,
	})
	if err != nil {
		return err
	}
	return p.processPartialResults(ctx, g, in, res.NewPredictorResult, out, in.SelectedBuffer, replace, expr)
}

// MarshalJSON implements Provider
func (p GGNNModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type                  data.ProviderName `json:"type"`
		ForceDisableFiltering bool              `json:"force_disable_filtering"`
	}{
		Type:                  p.Name(),
		ForceDisableFiltering: p.ForceDisableFiltering,
	})
}

func (p GGNNModel) processPartialResults(ctx kitectx.Context, g Global, in Inputs, predictions []pythongraph.Prediction, out OutputFunc, buffer data.SelectedBuffer, replace data.Selection, expr pythonast.Expr) error {
	// Filtering reactivated but not used to decide what to output for now
	// TODO use the result of filtering to block output completions considered as invalid
	callProbIn := callprob.Inputs{
		RM:        g.ResourceManager,
		Cursor:    int64(in.Selection.Begin),
		RAST:      in.ResolvedAST(),
		ScopeSize: getScopeSize(predictions),
		Expr:      expr,
	}
	computeCallProbs(ctx, g, callProbIn, predictions)
pred_loop:
	for _, pred := range predictions {
		var predictedValues []string
		for _, t := range pred.CompTokens {
			// transform placeholder to the right format
			if t.Value == pythongraph.PlaceholderPlaceholder {
				sigStats := g.ResourceManager.SigStats(t.Symbol)
				position := t.Position
				if pred.PredictedCall != nil {
					position += pred.PredictedCall.NumOrigArgs
				}
				renderedPlaceholder, unknownPositional := renderPlaceholder(t.Name, position, sigStats)
				if unknownPositional {
					continue pred_loop
				}
				predictedValues = append(predictedValues, renderedPlaceholder)
				continue
			}
			value := t.Value
			if t.Position != -1 {
				value = data.Hole(value)
			}
			predictedValues = append(predictedValues, value)
		}
		score := pred.Score
		var argSpec *pythonimports.ArgSpec
		var numOrigArgs int
		if pred.PredictedCall != nil {
			score = pred.PredictedCall.CallProb
			argSpec = g.ResourceManager.ArgSpec(pred.PredictedCall.Symbol)
			numOrigArgs = pred.PredictedCall.NumOrigArgs
		}
		compText := strings.Join(predictedValues, "")
		var speculationPlaceholderPresent bool
		if pred.PredictedCall != nil && pred.PredictedCall.PartialCall {
			var paren string
			if !pred.Predictor.ClosingParenthesisPresent() {
				paren = ")"
				pred.Predictor.SetClosingParenthesisPresent(true)
			}
			compText = fmt.Sprintf("%s%s%s", compText, data.Hole(""), paren)
			speculationPlaceholderPresent = true
		}

		var referent pythontype.Value
		if pred.PredictedCall == nil {
			referent = pythontype.NewExternal(pred.Symbol, g.ResourceManager)
		}

		_, isSmart := SmartProviders[p.Name()]

		comp := MetaCompletion{
			Completion: data.Completion{
				Snippet: data.BuildSnippet(compText),
				Replace: replace,
			},
			RenderMeta: RenderMeta{Referent: referent},
			Provider:   p.Name(),
			Source:     response.GGNNModelAttributeSource,
			Score:      score,
			GGNNMeta: &GGNNMeta{
				Predictor:                     pred.Predictor,
				Call:                          pred.PredictedCall,
				ArgSpec:                       argSpec,
				NumOrigArgs:                   numOrigArgs,
				Debug:                         pred.StoppingTask,
				SpeculationPlaceholderPresent: speculationPlaceholderPresent,
			},
			FromSmartProvider: isSmart,
		}

		if pred.PredictedCall != nil {
			if pred.PredictedCall.PartialCall == true {
				comp.Source = response.GGNNModelPartialCallSource
			} else {
				comp.Source = response.GGNNModelFullCallSource
			}
		}
		enableFiltering := !p.ForceDisableFiltering
		if !enableFiltering || comp.GGNNMeta.Call == nil || !comp.GGNNMeta.Call.SkipCall {
			out(ctx, buffer, comp)
		}
	}
	return nil
}

func getScopeSize(predictions []pythongraph.Prediction) int {
	for _, pred := range predictions {
		if pred.PredictedCall != nil {
			return pred.PredictedCall.ScopeSize
		}
	}
	return 0
}

func computeCallProbs(ctx kitectx.Context, g Global, callProbIn callprob.Inputs, predictions []pythongraph.Prediction) {
	type callsForSymbol struct {
		fullCalls    []pythongraph.PredictedCall
		partialCalls []pythongraph.PredictedCall
		fullIndex    []int
		partialIndex []int
		symbol       pythonresource.Symbol
	}

	calls := make(map[string]callsForSymbol)
	for i, p := range predictions {
		if p.PredictedCall != nil {
			c := p.PredictedCall
			path := c.Symbol.Canonical().PathString()
			entry := calls[path]
			entry.symbol = c.Symbol
			if c.PartialCall {
				entry.partialCalls = append(entry.partialCalls, *c)
				entry.partialIndex = append(entry.partialIndex, i)
			} else {
				entry.fullCalls = append(entry.fullCalls, *c)
				entry.fullIndex = append(entry.fullIndex, i)
			}
			calls[path] = entry
		}
	}

	for _, e := range calls {
		sum := pythongraph.PredictedCallSummary{
			Symbol:    e.symbol,
			Predicted: e.fullCalls,
			ScopeSize: callProbIn.ScopeSize,
		}
		callProbs := computeCallScoreBatch(ctx, g.Models.FullCallProb, sum, callProbIn)
		for i, v := range callProbs {
			predIndex := e.fullIndex[i]
			predictions[predIndex].PredictedCall.CallProb = float64(v)
			predictions[predIndex].PredictedCall.MetaData = sum.Predicted[i].MetaData
			predictions[predIndex].PredictedCall.SkipCall = skipCallCompForGGNNModel(g, *predictions[predIndex].PredictedCall, false)
		}
		sum.Predicted = e.partialCalls
		callProbs = computeCallScoreBatch(ctx, g.Models.PartialCallProb, sum, callProbIn)
		for i, v := range callProbs {
			predIndex := e.partialIndex[i]
			predictions[predIndex].PredictedCall.CallProb = float64(v)
			predictions[predIndex].PredictedCall.MetaData = sum.Predicted[i].MetaData
			predictions[predIndex].PredictedCall.SkipCall = skipCallCompForGGNNModel(g, *predictions[predIndex].PredictedCall, true)
		}
	}

}

// returns true if the confidence score is lower than the threshold and should be skipped
func skipCallCompForGGNNModel(g Global, call pythongraph.PredictedCall, partialCall bool) bool {
	if call.CallProb < 0 {
		// Invalid call, should be skipped anyway
		return true
	}

	thresholds := g.Models.FullCallProb.Params()
	if partialCall {
		thresholds = g.Models.PartialCallProb.Params()
	}
	switch numCallArgs(call.Args) {
	case 0:
		if call.CallProb < float64(thresholds.ZeroArgs) {
			return true
		}
	case 1:
		if call.CallProb < float64(thresholds.OneArgs) {
			return true
		}
	case 2:
		if call.CallProb < float64(thresholds.TwoArgs) {
			return true
		}
	}
	return false
}

// CompareCompletions implements CompletionComparator.
func (GGNNModel) CompareCompletions(comp1, comp2 MetaCompletion) bool {
	if comp1.GGNNMeta != nil && comp1.GGNNMeta.Call != nil && comp2.GGNNMeta != nil && comp2.GGNNMeta.Call != nil {
		if math.IsNaN(comp1.GGNNMeta.Call.CallProb) && math.IsNaN(comp2.GGNNMeta.Call.CallProb) {
			return comp1.Score > comp2.Score
		}
		if math.IsNaN(comp1.GGNNMeta.Call.CallProb) {
			return false
		}
		if math.IsNaN(comp2.GGNNMeta.Call.CallProb) {
			return true
		}
		if math.Abs(comp1.GGNNMeta.Call.CallProb-comp2.GGNNMeta.Call.CallProb) >= 1e-6 {
			promotedScore1 := comp1.GGNNMeta.Call.CallProb * math.Pow((1+promotionFactor), float64(comp1.MixingMeta.DistanceFromRoot))
			promotedScore2 := comp2.GGNNMeta.Call.CallProb * math.Pow((1+promotionFactor), float64(comp2.MixingMeta.DistanceFromRoot))
			return promotedScore1 > promotedScore2
		}
	}
	return comp1.Score > comp2.Score
}
