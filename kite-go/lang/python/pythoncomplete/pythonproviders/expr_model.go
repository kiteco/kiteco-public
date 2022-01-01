package pythonproviders

import (
	"encoding/json"
	"fmt"
	"go/token"
	"math"

	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprobcallmodel"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/threshold"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// ExprModel is a Provider for the expression model at name expression contexts
type ExprModel struct{}

// Name implements Provider
func (ExprModel) Name() data.ProviderName {
	return data.PythonExprModelProvider
}

// Provide implements Provider
func (e ExprModel) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	if !ggnnCompletionsEnabled() || applesilicon.Detected {
		return nil
	}

	_, isSmart := SmartProviders[e.Name()]
	if isSmart && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	if g.Models == nil || !g.Models.Expr.IsLoaded() || !g.Models.CallModelCallProb.IsLoaded() || !g.Models.MTACConf.IsLoaded() {
		return nil
	}
	n, err := findApplicableNode(in.UnderSelection(), in)
	if err != nil {
		return err
	}
	if n == nil {
		return data.ProviderNotApplicableError{}
	}

	nameExpr, ok := n.(*pythonast.NameExpr)
	if !ok {
		if ok, sym := isPlaceholder(ctx, g, in, n); ok {
			return providePlaceholderCompletion(ctx, g, in, out, n, sym)
		}
		return data.ProviderNotApplicableError{}
	}

	bufBytes := []byte(in.Buffer.Text())
	rast := in.ResolvedAST()
	mixData := mtacconf.GetMixData(ctx, g.ResourceManager, in.Selection, in.Words(), in.ResolvedAST(), nameExpr)
	callProbIn := callprobcallmodel.Inputs{
		RM:     g.ResourceManager,
		Cursor: int64(in.Selection.Begin),
		RAST:   rast,
	}

	// deep copy the RAST since we modify it during prediction
	newRAST, newNodes := rast.DeepCopy()
	ggnnRes, err := g.Models.Expr.Predict(ctx, pythonexpr.Input{
		RM:                          g.ResourceManager,
		RAST:                        newRAST,
		Words:                       in.Words(), // not modified
		Src:                         bufBytes,
		Expr:                        newNodes[nameExpr].(*pythonast.NameExpr),
		MaxPatterns:                 maxCallPatternsForAttrCompletions,
		AlwaysUsePopularityForAttrs: alwaysUsePopularityForAttrs,
		MungeBufferForAttrs:         mungeBufferForAttrs,
	})
	if err != nil {
		return err
	}
	res := ggnnRes.OldPredictorResult
	probStack, bufStack := []float64{1.}, []data.SelectedBuffer{in.SelectedBuffer}
	pythongraph.Inspect(res, func(n *pythongraph.PredictionTreeNode) bool {
		lastProb := len(probStack) - 1
		lastBuf := len(bufStack) - 1
		if n == nil {
			probStack = probStack[:lastProb]
			bufStack = bufStack[:lastBuf]
			return false
		}

		prob := probStack[lastProb] * float64(n.Prob)
		probStack = append(probStack, prob)

		curBuf := bufStack[lastBuf]

		switch {
		case n.AttrBase != "":
			var name MetaCompletion
			name.Replace = data.Selection{
				Begin: int(nameExpr.Ident.Begin),
				End:   in.Selection.End,
			}
			name.Snippet.Text = n.AttrBase
			name.Score = float64(n.Prob)
			table, _ := rast.TableAndScope(nameExpr)
			if sym := table.Find(n.AttrBase); sym != nil {
				name.RenderMeta.Referent = sym.Value
			}
			name.Provider = e.Name()
			name.Source = response.ExprModelCompletionsSource
			name.NameModelMeta = &IdentModelMeta{}
			name.NameModelMeta.MTACConfSkip = computeMTACConfSkip(ctx, g, in, name, mixData)
			name.FromSmartProvider = isSmart

			out(ctx, curBuf, name)
			curBuf = curBuf.Select(name.Replace).ReplaceWithCursor(n.AttrBase)

			bufStack = append(bufStack, curBuf)

		case !n.Attr.Nil():
			// generate completion to add a "." (deduped by caller)
			var dot MetaCompletion
			dot.Replace = curBuf.Selection
			dot.Snippet.Text = "."
			dot.FromSmartProvider = isSmart

			out(ctx, curBuf, dot)
			curBuf = curBuf.ReplaceWithCursor(".")

			// generate attribute completion
			attrStr := n.Attr.Path().Last()
			var attr MetaCompletion
			attr.Replace = curBuf.Selection
			attr.Snippet.Text = attrStr
			attr.Score = float64(n.Prob)
			attr.RenderMeta.Referent = pythontype.NewExternal(n.Attr, g.ResourceManager)
			attr.Provider = e.Name()
			attr.Source = response.AttributeModelCompletionSource
			attr.AttrModelMeta = &IdentModelMeta{}
			attr.AttrModelMeta.MTACConfSkip = computeMTACConfSkip(ctx, g, in, attr, mixData)
			attr.FromSmartProvider = isSmart

			out(ctx, curBuf, attr)
			curBuf = curBuf.ReplaceWithCursor(attrStr)

			bufStack = append(bufStack, curBuf)

		default:
			// this may be a leaf node or root node; always push curBuf onto the stack
			bufStack = append(bufStack, curBuf)

			// TODO(naman) can we avoid truncating here and instead truncate during the mixing phase?
			if len(n.Call.Predicted) > maxResultsPerCallPattern {
				n.Call.Predicted = n.Call.Predicted[:maxResultsPerCallPattern]
			}

			// generate completion to add "()" (deduped by caller)
			if len(n.Call.Predicted) > 0 {
				var parens MetaCompletion
				parens.Replace = curBuf.Selection
				parens.Snippet = data.BuildSnippet(fmt.Sprintf("(%s)", data.HoleWithPlaceholderMarks("")))

				out(ctx, curBuf, parens)
				curBuf = curBuf.Replace("()").Select(data.Selection{Begin: curBuf.Selection.Begin, End: curBuf.Selection.Begin + 2})
			}

			callProbConf := computeCallScoreBatchForCallModel(ctx, g.Models.CallModelCallProb, n.Call, callProbIn)
			sigStats := g.ResourceManager.SigStats(n.Call.Symbol)
			for i, p := range n.Call.Predicted {
				var args MetaCompletion
				args.Replace = curBuf.Selection
				args.Snippet = data.BuildSnippet(fmt.Sprintf("(%s)", argsForCall(p.Args, sigStats, 0)))
				args.Score = prob * float64(p.Prob)
				args.CallModelMeta = &CallModelMeta{
					FunctionSym: n.Call.Symbol,
					NumArgs:     len(p.Args),
					ArgSpec:     g.ResourceManager.ArgSpec(n.Call.Symbol),
				}

				args.CallModelMeta.CallProb = math.NaN()

				// check length in case there was an error
				if i < len(callProbConf) {
					args.CallModelMeta.CallProb = float64(callProbConf[i])
				}
				args.Provider = e.Name()
				args.Source = response.CallModelCompletionSource

				args.RenderMeta.Referent = pythontype.NewExternal(n.Call.Symbol, g.ResourceManager)

				args.FromSmartProvider = isSmart

				if !skipCallComp(g, args) {
					out(ctx, curBuf, args)
				}
			}
		}
		return true
	})

	return nil
}

// MarshalJSON implements Provider
func (e ExprModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: e.Name(),
	})
}

func providePlaceholderCompletion(ctx kitectx.Context, g Global, in Inputs, out OutputFunc, expr pythonast.Expr, sym mtacconf.CallSym) error {
	clearedRAST, site, clearedBuffer, err := clearPlaceholders(ctx, g, in, expr, sym)
	if err != nil {
		return err
	}

	parent := clearedRAST.Parent[site]
	filledArg, _ := parent.(*pythonast.Argument)
	if filledArg == nil {
		return errors.New("The parent of the placeholder site is not an Argument")
	}
	parentCall := clearedRAST.Parent[filledArg]
	callExpr, _ := parentCall.(*pythonast.CallExpr)
	if callExpr == nil {
		return errors.New("Can't find the callExpr anymore after removing the placeholder")
	}

	ggnnRes, err := g.Models.Expr.Predict(ctx, pythonexpr.Input{
		RM:                          g.ResourceManager,
		RAST:                        clearedRAST,
		Words:                       in.Words(), // not modified
		Src:                         []byte(clearedBuffer),
		Expr:                        callExpr,
		Arg:                         filledArg,
		MaxPatterns:                 maxCallPatternsForAttrCompletions,
		AlwaysUsePopularityForAttrs: alwaysUsePopularityForAttrs,
		MungeBufferForAttrs:         mungeBufferForAttrs,
	})

	if err != nil {
		return err
	}
	res := ggnnRes.OldPredictorResult

	table, _ := in.ResolvedAST().TableAndScope(expr)

	pythongraph.Inspect(res, func(n *pythongraph.PredictionTreeNode) bool {
		if n == nil {
			return false
		}

		for _, child := range n.Children {
			if n.AttrBase == "" {
				continue
			}

			var args MetaCompletion
			args.Replace = in.Selection
			args.Snippet.Text = child.AttrBase
			args.Score = float64(child.Prob)
			args.ExprModelMeta = &IdentModelMeta{
				MTACConfSkip: false, //TODO: Evaluate confidence for exprModel when filling a placeholder
			}
			args.MixingMeta.DoNotCompose = args.ExprModelMeta.MTACConfSkip
			args.Provider = ExprModel{}.Name()
			args.Source = response.ExprModelCompletionsSource
			// We don't have symbols for the predicted value, RenderMeta is let empty
			if sym := table.Find(n.AttrBase); sym != nil {
				args.RenderMeta.Referent = sym.Value
			}
			out(ctx, in.SelectedBuffer, args)
		}
		return true
	})
	return nil
}

func clearPlaceholders(ctx kitectx.Context, g Global, in Inputs, expr pythonast.Expr, sym mtacconf.CallSym) (*pythonanalyzer.ResolvedAST, *pythonast.NameExpr, string, error) {
	rast, translate := in.ResolvedAST().DeepCopy()
	exprNode := translate[expr]
	arg, _ := rast.Parent[exprNode].(*pythonast.Argument)
	if arg == nil {
		return nil, nil, "", errors.New("Impossible to cast expr parent as an Argument")
	}

	emptyWordPos := token.Pos(in.Selection.Begin)
	emptyWord := &pythonscanner.Word{
		Token:   pythonscanner.Ident,
		Literal: "",
		Begin:   emptyWordPos,
		End:     emptyWordPos,
	}
	name := &pythonast.NameExpr{Ident: emptyWord}

	clearedBuffer := in.Buffer.Text()[:in.Selection.Begin] + in.Buffer.Text()[in.Selection.End:]

	err := pythonast.Replace(arg, exprNode, name)
	rast.Parent[name] = arg
	if err != nil {
		return nil, nil, "", err
	}

	return rast, name, clearedBuffer, nil
}

func isPlaceholder(ctx kitectx.Context, g Global, in Inputs, expr pythonast.Expr) (bool, mtacconf.CallSym) {
	//TODO We would like to have a clean way to detect that the user has selected a placeholder
	sym := mtacconf.GetContainingCallSym(ctx, g.ResourceManager, in.ResolvedAST(), expr)
	return !sym.Sym.Nil(), sym
}

func computeMTACConfSkip(ctx kitectx.Context, g Global, in Inputs, c MetaCompletion, mixData mtacconf.MixData) bool {
	ctx.CheckAbort()

	thresholds := g.Models.MTACConf.Params()

	if !g.Models.MTACConf.IsLoaded() || mixData.Scenario == threshold.Other {
		return false
	}

	inputs := mtacconf.Inputs{
		RM:     g.ResourceManager,
		Cursor: int64(in.Selection.Begin),
		Words:  in.Words(),
		RAST:   in.ResolvedAST(),
	}
	inputs.Comps = []mtacconf.Completion{{
		Score:   c.Score,
		MixData: mixData,
	}}

	switch {
	case c.NameModelMeta != nil:
		inputs.Comps[0].Source = response.ExprModelCompletionsSource
		inputs.Comps[0].Referent = c.RenderMeta.Referent
	case c.AttrModelMeta != nil:
		inputs.Comps[0].Source = response.AttributeModelCompletionSource
		inputs.Comps[0].Referent = c.RenderMeta.Referent
	default:
		return false
	}

	probs, err := g.Models.MTACConf.Infer(ctx, inputs)
	if err != nil {
		err = errors.Wrapf(err, "error running MTAC-confidence model inference")
		rollbar.Error(err)
		return false
	}

	return probs[0] < thresholds[mixData.Scenario]
}
