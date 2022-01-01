package pythonproviders

import (
	"encoding/json"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders/legacy"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Imports is a Provider for import completions
type Imports struct{}

// Name implements Provider
func (Imports) Name() data.ProviderName {
	return data.PythonImportsProvider
}

// Provide implements Provider by calling the old completions Engine
func (Imports) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	product := g.Product.GetProduct()

	importer := pythonstatic.Importer{
		Path:   g.FilePath,
		Global: g.ResourceManager,
	}
	if g.LocalIndex != nil {
		importer.Local = g.LocalIndex.SourceTree
		importer.PythonPaths = g.LocalIndex.PythonPaths
	}
	callbacks := legacy.CompletionsCallbacks{
		Buffer:     []byte(in.Buffer.Text()),
		Cursor:     int64(in.Selection.Begin),
		Words:      in.Words(),
		Resolved:   in.ResolvedAST(),
		LocalIndex: g.LocalIndex,
		Importer:   importer,
		Models:     g.Models,

		ProductGetter: product,
	}
	inputs := callbacks.Inputs(ctx)

	match := legacy.Match(ctx, inputs)
	if match == nil {
		return data.ProviderNotApplicableError{}
	}

	if _, forAlias := match.(legacy.ImportAlias); forAlias && product != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	provided := match.Provide(ctx, inputs, callbacks, false)

	var c MetaCompletion
	c.Replace = data.Selection{
		Begin: in.Selection.Begin - len(provided.TypedPrefix),
		// TypedSuffix should be 0, since we aren't using prefetchers
		End: in.Selection.Begin + len(provided.TypedSuffix),
	}
	if in.Selection.End > c.Replace.End {
		c.Replace.End = in.Selection.End
	}
	for _, compl := range provided.Completions {
		c.Snippet.Text = compl.Identifier
		c.Score = compl.Score
		c.Provider = Imports{}.Name()
		c.Source = response.TraditionalCompletionSource
		if compl.Referent == nil {
			c.KeywordModelMeta = true
		} else {
			c.TraditionalMeta = &TraditionalMeta{
				Situation: match.Name(),
			}
			c.RenderMeta = RenderMeta{
				Referent: compl.Referent,
			}
		}
		if valid, ok := c.Validate(in.SelectedBuffer); ok {
			c.Completion = valid
			out(ctx, in.SelectedBuffer, c)
		}
		// TODO(naman) completions are known to be invalid in the alias case if spacing does not match: add back a rollbar after fixing that
	}
	return nil
}

// MarshalJSON implements Provider
func (i Imports) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: i.Name(),
	})
}
