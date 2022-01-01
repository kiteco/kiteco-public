package pythonproviders

import (
	"encoding/json"
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	thresholdEmptyCall = 0.9
)

// EmptyCalls is a very simple Provider that completes call expressions from arbitrary function expressions: json.dumps‸ -> json.dumps()
type EmptyCalls struct{}

// Name implements Provider
func (EmptyCalls) Name() data.ProviderName {
	return data.PythonEmptyCallsProvider
}

func noCallArgs(call *pythonast.CallExpr) bool {
	for _, arg := range call.Args {
		if arg.Begin() != arg.End() {
			return false
		}
	}
	return true
}

// Provide implements Provider
func (EmptyCalls) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	if in.Selection.Len() > 0 {
		return data.ProviderNotApplicableError{}
	}

	var expr pythonast.Expr
search:
	for _, n := range in.UnderSelection() {
		switch n := n.(type) {
		case *pythonast.ImportNameStmt:
			break search
		case *pythonast.ImportFromStmt:
			break search
		case *pythonast.FunctionDefStmt:
			// break if the selection overlaps the name of a function def
			if in.Selection.End > int(n.Def.End) && in.Selection.Begin <= int(n.Name.End()) {
				break search
			}
		case *pythonast.LambdaExpr:
			// continue the search, but don't consider lambdas as valid candidates for empty calls
			continue search
		case pythonast.Expr:
			// important to not break the search in these cases, as we want the *deepest* expression.
			// e.g. for the code `lambda: foo|`, we should match `foo`, not the entire lambda.
			// (even though the lambda case is handled above, we should still choose the deepest)
			pos := int(n.End())
			if pos == in.Selection.Begin {
				if in.Buffer.Len() <= pos || in.Buffer.TextAt(data.Selection{Begin: pos, End: pos + 1}) != "(" {
					// there is not a call already following the selection, emit one
					expr = n
					continue search
				}
			}
		}
	}
	if expr == nil {
		return data.ProviderNotApplicableError{}
	}

	// check for the incomplete call case
	if call, _ := expr.(*pythonast.CallExpr); call != nil && call.RightParen == nil && noCallArgs(call) {
		var c MetaCompletion
		c.Snippet = data.BuildSnippet(fmt.Sprintf("%s)", data.Hole("")))
		c.Replace = in.Selection
		c.Score = 1
		c.Provider = EmptyCalls{}.Name()
		c.Source = response.EmptyCallCompletionSource
		out(ctx, in.SelectedBuffer, c)
		return nil
	}

	// only add parens for functions and types
	resolved := in.ResolvedAST()
	val := resolved.References[expr]
	if val == nil || !mightBeKind(ctx, val, pythontype.FunctionKind, pythontype.TypeKind) {
		return nil
	}

	isType := mightBeKind(ctx, val, pythontype.TypeKind)
	var c MetaCompletion
	c.Completion = data.Completion{
		Snippet: getEmptyCallSnippet(ctx, g, val),
		Replace: in.Selection,
	}
	c.Score = 1
	c.RenderMeta = RenderMeta{Referent: val}
	c.Source = response.EmptyCallCompletionSource
	c.EmptyCallMeta = &EmptyCallMeta{IsTypeKind: isType}
	out(ctx, in.SelectedBuffer, c)

	return nil
}

// MarshalJSON implements Provider
func (e EmptyCalls) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: e.Name(),
	})
}

func getEmptyCallSnippet(ctx kitectx.Context, g Global, val pythontype.Value) data.Snippet {
	var noArgInSpec bool
	for _, val := range pythontype.Disjuncts(ctx, val) {
		ext, ok := pythontype.TranslateGlobal(val, g.ResourceManager).(pythontype.External)
		if !ok {
			if sourceFunc, ok := val.(*pythontype.SourceFunction); ok && !hasNonReceiverArgs(sourceFunc) {
				noArgInSpec = true
				break
			}
			continue
		}

		if spec := g.ResourceManager.ArgSpec(ext.Symbol()); spec != nil {
			if len(spec.NonReceiverArgs()) == 0 && spec.Kwarg == "" && spec.Vararg == "" {
				noArgInSpec = true
				break
			}
		}
	}
	if noArgInSpec {
		return data.NewSnippet("()")
	}
	return data.BuildSnippet(fmt.Sprintf("(%s)", data.Hole("")))

}

func hasNonReceiverArgs(function *pythontype.SourceFunction) bool {
	if function.Vararg != nil || function.Kwarg != nil {
		return true
	}
	if function.KwargDict != nil && len(function.KwargDict.Entries) > 0 {
		return true
	}

	if len(function.Parameters) == 0 {
		return false
	}

	if len(function.Parameters) == 1 && (function.HasClassReceiver || function.HasReceiver) {
		return false
	}
	return true
}
