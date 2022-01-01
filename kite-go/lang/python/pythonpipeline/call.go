package pythonpipeline

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/linenumber"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

// ModelCanPredictCall returns true if the provided call can be predicted by the ggnn call models
func ModelCanPredictCall(ctx kitectx.Context, rm pythonresource.Manager, model pythonexpr.Model, rast *pythonanalyzer.ResolvedAST, call *pythonast.CallExpr) bool {
	if call.LeftParen == nil || call.RightParen == nil {
		return false
	}

	if !pythonast.IsNil(call.Kwarg) || !pythonast.IsNil(call.Vararg) {
		return false
	}

	val := rast.References[call.Func]
	for _, sym := range python.GetExternalSymbols(ctx, rm, val) {
		if model.CallSupported(rm, sym) == nil {
			return true
		}
	}

	return false
}

// CallArgSituation describes a situation in which the user is completing a call
type CallArgSituation struct {
	AnalyzedEvent AnalyzedEvent

	CallExpr *pythonast.CallExpr
	Expected string
	Symbol   string
}

// SampleTag implements Sample
func (CallArgSituation) SampleTag() {}

// ExprCallArgSituations builds attribute prediction situations from viable EventExpr samples
func ExprCallArgSituations(s pipeline.Sample) pipeline.Sample {
	expr := s.(EventExpr)

	call, ok := expr.Expr.(*pythonast.CallExpr)
	if !ok {
		return nil
	}

	parentVal := expr.AnalyzedEvent.Context.Resolved.References[call.Func]
	sym, err := python.GetExternalSymbol(kitectx.Background(), expr.AnalyzedEvent.Context.Importer.Global, parentVal)
	if err != nil {
		return nil
	}

	return CallArgSituation{
		AnalyzedEvent: expr.AnalyzedEvent,
		CallExpr:      call,
		Expected:      argumentString(call),
		Symbol:        sym.PathString(),
	}
}

// returns a string of arguments joined together
// NOTE: this should match `predictedCallResult` below
func argumentString(call *pythonast.CallExpr) string {
	var args []string
	for _, arg := range call.Args {
		if pythonast.IsNil(arg.Name) && !pythonast.IsNil(arg.Value) {
			if value, ok := arg.Value.(*pythonast.NameExpr); ok {
				args = append(args, fmt.Sprintf("%s", value.Ident.Literal))
			}
		}
		if !pythonast.IsNil(arg.Name) && !pythonast.IsNil(arg.Value) {
			name := arg.Name.(*pythonast.NameExpr)
			if value, ok := arg.Value.(*pythonast.NameExpr); ok {
				args = append(args, fmt.Sprintf("%s=%s", name.Ident.Literal, value.Ident.Literal))
			}
		}
	}

	return strings.Join(args, ", ")
}

// predictedCallResult gives a prediction result string
// NOTE: this should match `argumentString` above
func predictedCallResult(p pythongraph.PredictedCall) string {
	var args []string
	for _, a := range p.Args {
		if !a.Stop {
			if a.Name == "" {
				args = append(args, fmt.Sprintf("%s", a.Value))
			} else {
				args = append(args, fmt.Sprintf("%s=%s", a.Name, a.Value))
			}
		}
	}

	return fmt.Sprintf("%s", strings.Join(args, ", "))
}

// CallArgCompletion along with its score
type CallArgCompletion struct {
	Identifier string
	Score      float64
}

// CallArgCompletions represents a list of ranked completions along with the expected completion, as well as some
// metadata.
type CallArgCompletions struct {
	Situation    CallArgSituation
	Provided     []CallArgCompletion
	MungedBuffer string
}

// SampleTag implements pipeline.Sample
func (CallArgCompletions) SampleTag() {}

// ToProvided returns an example.Provided struct representing the completions that can be put into an example.Example.
func (a CallArgCompletions) ToProvided() example.Provided {
	cs := make([]example.Completion, 0, len(a.Provided))
	for _, p := range a.Provided {
		cs = append(cs, example.Completion{
			Identifier: p.Identifier,
			Score:      p.Score,
		})
	}
	return example.Provided{
		Completions: cs,
	}
}

// CallArgCompletionsGroup represents completions for a situation given by different providers
type CallArgCompletionsGroup struct {
	Situation CallArgSituation
	Provided  map[string]CallArgCompletions
}

// SampleTag implements pipeline.Sample
func (CallArgCompletionsGroup) SampleTag() {}

// Filter filters the providers to the ones given.
func (a CallArgCompletionsGroup) Filter(providers ...string) CallArgCompletionsGroup {
	filtered := make(map[string]CallArgCompletions)
	for k, v := range a.Provided {
		var found bool
		for _, p := range providers {
			if k == p {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		filtered[k] = v
	}
	return CallArgCompletionsGroup{
		Situation: a.Situation,
		Provided:  filtered,
	}
}

// ToExample creates an example from the given set of completions.
func (a CallArgCompletionsGroup) ToExample() example.Example {
	provided := make(map[string]example.Provided, len(a.Provided))
	for k := range a.Provided {
		provided[k] = a.Provided[k].ToProvided()
	}

	return example.Example{
		Buffer:   a.Situation.AnalyzedEvent.Event.Buffer,
		Cursor:   int64(a.Situation.CallExpr.LeftParen.End),
		Symbol:   a.Situation.Symbol,
		Expected: a.Situation.Expected,
		Provided: provided,
	}
}

// CallArgSituationsAllowedByModel filters call arg situations to ones for which:
// - the Value is a NameExpr
// - the Value of the call expr resolves to a type supported by the various arg tasks
func CallArgSituationsAllowedByModel(model pythonexpr.Model) transform.IncludeFn {
	return func(s pipeline.Sample) bool {
		situation := s.(CallArgSituation)

		rm := situation.AnalyzedEvent.Context.Importer.Global
		return ModelCanPredictCall(kitectx.Background(), rm, model, situation.AnalyzedEvent.Context.Resolved, situation.CallExpr)
	}
}

// GGNNCallArgCompletions returns the call completions predicted by GGNN model
func GGNNCallArgCompletions(recreator *servercontext.Recreator, models *pythonmodels.Models) transform.OneInOneOutFn {
	return func(s pipeline.Sample) pipeline.Sample {
		sit := s.(CallArgSituation)

		callExpr := sit.CallExpr
		src := sit.AnalyzedEvent.Event.Event.Buffer
		trimStart := callExpr.LeftParen.End

		newSrc := src[:callExpr.LeftParen.End] + src[callExpr.RightParen.Begin:]
		mutated := sit.AnalyzedEvent.Event.Event
		mutated.Buffer = newSrc
		mutated.Offset = int64(trimStart)

		newCtx, err := recreator.RecreateContext(&mutated, false)
		if err != nil {
			return nil
		}

		// Find the expression in the new AST that we want to infer on
		var newExpr pythonast.Expr
		pythonast.Inspect(newCtx.AST, func(n pythonast.Node) bool {
			if n, ok := n.(*pythonast.CallExpr); ok {
				if n.Begin() == callExpr.Begin() {
					newExpr = n
				}
			}
			return true
		})

		if pythonast.IsNil(newExpr) {
			return nil
		}

		in := pythonexpr.Input{
			RM:                  newCtx.Importer.Global,
			RAST:                newCtx.Resolved,
			Words:               newCtx.IncrLexer.Words(),
			Src:                 []byte(newSrc),
			Expr:                newExpr,
			MungeBufferForAttrs: true,
		}

		var tree *pythongraph.PredictionTreeNode
		err = kitectx.Background().WithTimeout(3*time.Second, func(ctx kitectx.Context) error {
			ggnn, err := models.Expr.Predict(ctx, in)
			tree = ggnn.OldPredictorResult
			return err
		})
		if err != nil {
			return nil
		}

		var preds []CallArgCompletion

		pythongraph.Inspect(tree, func(n *pythongraph.PredictionTreeNode) bool {
			if n == nil {
				return false
			}
			switch {
			case len(n.Call.Predicted) > 0:
				for _, call := range n.Call.Predicted {
					preds = append(preds, CallArgCompletion{
						Identifier: predictedCallResult(call),
						Score:      float64(call.Prob),
					})
				}
			}
			return true
		})

		sort.Slice(preds, func(i, j int) bool {
			return preds[i].Score > preds[j].Score
		})

		return CallArgCompletions{
			Situation:    sit,
			Provided:     preds,
			MungedBuffer: newSrc,
		}
	}
}

// MungeBufferForCall ...
func MungeBufferForCall(rast *pythonanalyzer.ResolvedAST, lines *linenumber.Map, src []byte, call *pythonast.CallExpr) ([]byte, []pythonscanner.Word, *pythonast.Module, *pythonast.CallExpr, error) {
	stmt := rast.ParentStmts[call]
	line := lines.Line(int(stmt.Begin()))
	if lines.Line(int(stmt.End())) != line {
		return nil, nil, nil, nil, errors.Errorf("multiline statement")
	}

	_, lineEnd := lines.LineBounds(line)

	// remove args and everything up to the end of the line
	newSrc := bytes.Join([][]byte{
		src[:call.LeftParen.End],
		[]byte(")"),
		src[lineEnd:],
	}, nil)

	ast, words, err := Parse(pythonparser.Options{
		ErrorMode:   pythonparser.Recover,
		Approximate: true,
	}, time.Second, sample.ByteSlice(newSrc))

	if err != nil {
		return nil, nil, nil, nil, errors.Errorf("reparse error: %v", err)
	}

	var newCall *pythonast.CallExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if newCall != nil {
			return false
		}
		if c, ok := n.(*pythonast.CallExpr); ok {
			if c.LeftParen.Begin == call.LeftParen.Begin {
				newCall = c
			}
		}
		return true
	})

	if newCall == nil {
		return nil, nil, nil, nil, errors.Errorf("unable to refind call")
	}

	return newSrc, words, ast, newCall, nil
}
