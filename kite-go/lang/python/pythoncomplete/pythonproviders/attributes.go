package pythonproviders

import (
	"encoding/json"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Attributes is a Provider for attribute completions
type Attributes struct {
	// UseDefaultReferences make the provider use the non refined union when looking for attributes
	// Default is to use the refinedValues
	UseDefaultReferences bool
}

// Name implements Provider
func (Attributes) Name() data.ProviderName {
	return data.PythonAttributesProvider
}

// Provide implements Provider
func (a Attributes) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	var attrExpr *pythonast.AttributeExpr
	for _, node := range in.UnderSelection() {
		if a, _ := node.(*pythonast.AttributeExpr); a != nil {
			if in.Begin >= int(a.Attribute.Begin) && in.End <= int(a.Attribute.End) {
				attrExpr = a
				break
			}
		}
	}
	if attrExpr == nil {
		return data.ProviderNotApplicableError{}
	}

	ref := in.ResolvedAST().RefinedValue(attrExpr.Value)
	if a.UseDefaultReferences {
		ref = in.ResolvedAST().References[attrExpr.Value]
	}
	if ref == nil {
		return nil
	}

	replace := data.Selection{
		Begin: int(attrExpr.Attribute.Begin),
		End:   int(attrExpr.Attribute.End),
	}
	isValid := func(attr string) bool {
		if strings.HasPrefix(attr, "__") {
			return false
		}
		c := data.Completion{
			Snippet: data.Snippet{Text: attr},
			Replace: replace,
		}
		var ok bool
		_, ok = c.Validate(in.SelectedBuffer)
		if !ok {
			c = data.Completion{
				Snippet: data.Snippet{Text: attr},
				Replace: data.Selection{
					Begin: replace.Begin,
					End:   in.Selection.End,
				},
			}
			_, ok = c.Validate(in.SelectedBuffer)
		}
		return ok
	}

	type valScore struct {
		val   pythontype.Value
		score float64
	}
	valScores := make(map[string]valScore)
	emit := func(attr string, val pythontype.Value, score float64) {
		vs := valScores[attr]
		vs.val = pythontype.Unite(ctx, vs.val, val)
		vs.score += score
		valScores[attr] = vs
	}

	// TODO(naman) ideally we would use Bayes' to consider the completions for each disjunct of `ref` separately.
	// But when I tried this with a uniform distribution across the disjuncts, py2/3 collisions made the results screwy.
	// However, the code is left structured so that we can call `provideAttributesFor` once per disjuct.
	provideAttributesFor(ctx, g, ref, isValid, emit)

	for attr, vs := range valScores {
		c := data.Completion{
			Snippet: data.Snippet{Text: attr},
			Replace: replace,
		}
		var ok bool
		c, ok = c.Validate(in.SelectedBuffer)
		if !ok {
			c = data.Completion{
				Snippet: data.Snippet{Text: attr},
				Replace: data.Selection{Begin: replace.Begin, End: in.Selection.End},
			}
			c, ok = c.Validate(in.SelectedBuffer)
			if !ok {
				continue
			}
		}
		// If we considered multiple disjuncts, we'd multiply P(disjunct) with this score.
		score := vs.score
		meta := MetaCompletion{
			Completion: c,
			Score:      score,
			Provider:   a.Name(),
			Source:     response.TraditionalCompletionSource,
			TraditionalMeta: &TraditionalMeta{
				// update this once we get fully replace of the "Traditional" Provider
				Situation: "Attribute",
			},
			RenderMeta: RenderMeta{Referent: vs.val},
		}
		out(ctx, in.SelectedBuffer, meta)
	}

	return nil
}

// MarshalJSON implements Provider
func (a Attributes) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type                 data.ProviderName `json:"type"`
		UseDefaultReferences bool              `json:"user_default_references"`
	}{
		Type:                 a.Name(),
		UseDefaultReferences: a.UseDefaultReferences,
	})
}

// provideAttributesFor provides attributes completions via callbacks.
// The scores are normalized to sum to 1, since otherwise, these scores may not be comparable with other scores.
func provideAttributesFor(ctx kitectx.Context, g Global, val pythontype.Value, isValid func(string) bool, emit func(string, pythontype.Value, float64)) {
	val = pythontype.Translate(ctx, pythontype.WidenConstants(val), g.ResourceManager)
	if val == nil {
		return
	}

	var scorer freqScorer
	// TODO(naman) stop doing value choice here.
	if _, ok := pythontype.MostSpecific(ctx, val).(pythontype.GlobalValue); ok {
		scorer = globalFreqScorer(g.ResourceManager, true)
	} else {
		scorer = localFreqScorer(g.LocalIndex)
	}

	members := pythontype.Members(ctx, g.ResourceManager, val)

	var totalFreq int
	freqs := make(map[string]int, len(members))

	for attr, val := range members {
		if !isValid(attr) {
			continue
		}

		// We take the maximum of the frequencies for each of the disjuncts.
		// This choice is somewhat arbitrary, and we may want to experiment.
		disjuncts := pythontype.Disjuncts(ctx, val)
		var maxFreq int
		for _, val := range disjuncts {
			val = pythontype.Translate(ctx, pythontype.WidenConstants(val), g.ResourceManager)
			score := scorer(ctx, val)
			if score > maxFreq {
				maxFreq = score
			}
		}

		// so that we don't emit scores of 0
		maxFreq++

		totalFreq += maxFreq
		freqs[attr] = maxFreq
	}

	for attr, freq := range freqs {
		emit(attr, members[attr], float64(freq)/float64(totalFreq))
	}
}

type freqScorer = func(kitectx.Context, pythontype.Value) int

func localFreqScorer(local *pythonlocal.SymbolIndex) freqScorer {
	if local == nil {
		return func(kitectx.Context, pythontype.Value) int { return 0 }
	}
	return func(ctx kitectx.Context, val pythontype.Value) int {
		if val == nil {
			return 0
		}
		if score, err := local.ValueCount(ctx, val); err == nil {
			return score
		}
		return 0
	}
}

func globalFreqScorer(rm pythonresource.Manager, withAttribute bool) freqScorer {
	return func(ctx kitectx.Context, val pythontype.Value) int {
		if val == nil {
			return 0
		}

		var sym pythonresource.Symbol
		switch val := val.(type) {
		case pythontype.ExternalInstance:
			sym = val.TypeExternal.Symbol()
		case pythontype.External:
			sym = val.Symbol()
		}
		counts := rm.SymbolCounts(sym)
		if counts == nil {
			return 0
		}

		total := counts.Import
		if withAttribute {
			total += counts.Attribute
		}
		return total
	}
}
