package pythonproviders

import (
	"encoding/json"

	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// AttributeModel is a Provider for the attribute model
// it completes situations such as foo.‸ with attributes.
type AttributeModel struct{}

// Name implements Provider
func (AttributeModel) Name() data.ProviderName {
	return data.PythonAttributeModelProvider
}

// Provide implements Provider
func (a AttributeModel) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	if !ggnnCompletionsEnabled() || applesilicon.Detected {
		return nil
	}

	_, isSmart := SmartProviders[a.Name()]
	if isSmart && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	if g.Models == nil || !g.Models.Expr.IsLoaded() {
		return nil
	}

	// find the deepest node under the position, that is not entirely contained within the position:
	// so in foo(‸bar()‸), call expr bar() is not chosen, but rather the outer call expr.
	var attrExpr *pythonast.AttributeExpr
	underPos := in.UnderSelection()
	for i := len(underPos) - 1; i >= 0; i-- {
		n := underPos[i]
		if attr, _ := n.(*pythonast.AttributeExpr); attr != nil {
			if in.Selection.Begin == int(attr.Dot.End) {
				attrExpr = attr
				break
			}
		}
	}
	if attrExpr == nil {
		return data.ProviderNotApplicableError{}
	}

	// deep copy the RAST since we modify it during prediction
	rast, newNodes := in.ResolvedAST().DeepCopy()

	ggnnNode, err := g.Models.Expr.Predict(ctx, pythonexpr.Input{
		RM:                          g.ResourceManager,
		RAST:                        rast,
		Words:                       in.Words(),
		Expr:                        newNodes[attrExpr].(*pythonast.AttributeExpr),
		AlwaysUsePopularityForAttrs: alwaysUsePopularityForAttrs,
	})
	if err != nil {
		return err
	}
	node := ggnnNode.OldPredictorResult

	mixData := mtacconf.GetMixData(ctx, g.ResourceManager, in.Selection, in.Words(), in.ResolvedAST(), attrExpr)
	var c MetaCompletion
	c.Provider = a.Name()
	c.Replace = in.Selection
	for _, child := range node.Children {
		c.Snippet.Text = child.Attr.Path().Last()
		c.Score = float64(child.Prob)
		c.Source = response.AttributeModelCompletionSource
		c.RenderMeta.Referent = pythontype.NewExternal(child.Attr, g.ResourceManager)
		c.AttrModelMeta = &IdentModelMeta{}
		c.AttrModelMeta.MTACConfSkip = computeMTACConfSkip(ctx, g, in, c, mixData)
		c.MixingMeta.DoNotCompose = c.AttrModelMeta.MTACConfSkip
		c.FromSmartProvider = isSmart
		out(ctx, in.SelectedBuffer, c)
	}

	return nil
}

// MarshalJSON implements Provider
func (a AttributeModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: a.Name(),
	})
}
