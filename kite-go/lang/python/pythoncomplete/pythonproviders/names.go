package pythonproviders

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/response"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Names is a Provider for name completions
type Names struct{}

// Name implements Provider
func (Names) Name() data.ProviderName {
	return data.PythonNamesProvider
}

// Provide implements Provider
func (Names) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	rast := in.ResolvedAST()
	underSel := in.UnderSelection()

	var table *pythontype.SymbolTable
	var nameExpr *pythonast.NameExpr

	if n, err := findApplicableNode(underSel, in); err == nil {
		nameExpr, _ = n.(*pythonast.NameExpr)
		table, _ = rast.TableAndScope(n)
	} else {
		return err
	}

	if table == nil {
		// TODO(naman) what if rast.TableAndScope returned nil above: rollbar?
		return data.ProviderNotApplicableError{}
	}

	vals := make(map[string]pythontype.Value)
	var names []string
	for ; table != nil; table = table.Parent {
		start := len(names)

		for _, s := range table.Table {
			name := s.Name.Path.Last()

			// no private symbol completions
			if strings.HasPrefix(name, "_") {
				continue
			}

			// If nameExpr is in a negative AST position (i.e. Usage == Assign)
			// we'll end up generating a completion for exactly what the user has typed.
			// There's no way of knowing precisely when this is desirable without augmenting analysis,
			// since we don't know if the same name has been assigned to in some other location.
			// For now, we simply filter out completions that exactly match expr if expr.Usage == Assign.
			if nameExpr != nil && nameExpr.Usage == pythonast.Assign && name == nameExpr.Ident.Literal {
				continue
			}

			// Given symbols of the same name, prefer ones in an inner scope
			if _, found := vals[name]; found {
				continue
			}

			vals[name] = pythontype.Translate(ctx, s.Value, g.ResourceManager)
			names = append(names, name)
		}

		// sort just this scope's names alphabetically
		sort.Slice(names[start:], func(i, j int) bool {
			iName, jName := names[start+i], names[start+j]
			return iName < jName
		})
	}

	replace := in.Selection
	if nameExpr != nil {
		replace = data.Selection{Begin: int(nameExpr.Ident.Begin), End: int(nameExpr.Ident.End)}
	}
	var validated []string
	completions := make(map[string]data.Completion)
	for _, name := range names {
		// attempt to validate completion that replaces entire nameExpr
		c := data.Completion{
			Replace: replace,
			Snippet: data.Snippet{Text: name},
		}
		var ok bool
		c, ok = c.Validate(in.SelectedBuffer)
		if !ok {
			// attempt to validate completion that replaces only the prefix of nameExpr
			c = data.Completion{
				Replace: data.Selection{Begin: replace.Begin, End: in.Selection.End},
				Snippet: data.Snippet{Text: name},
			}
			c, ok = c.Validate(in.SelectedBuffer)
			if !ok {
				continue
			}
		}
		validated = append(validated, name)
		completions[name] = c
	}

	n := float64(len(validated)) + 1
	for i, name := range validated {
		meta := MetaCompletion{
			Completion: completions[name],
			// set the score between 1/n and 1/(n+1).
			// note 1 / len(validated) is the probability if we assume one of
			// the validated names will be selected and they are equally likely.
			// the + 1 accounts for cases where another name is used.
			// the scores need to be different and the monotonic ordering is important,
			// but otherwise the exact values are not important.
			// so the range is an arbitrary choice and we are free to change it.
			Score:    1. / (n + float64(i)/n),
			Provider: Names{}.Name(),
			Source:   response.TraditionalCompletionSource,
			TraditionalMeta: &TraditionalMeta{
				// update this once we get fully replace of the "Traditional" Provider
				Situation: "NameExpr",
			},
			RenderMeta: RenderMeta{Referent: vals[name]},
		}
		out(ctx, in.SelectedBuffer, meta)
	}

	return nil
}

// MarshalJSON implements Provider
func (n Names) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: n.Name(),
	})
}
