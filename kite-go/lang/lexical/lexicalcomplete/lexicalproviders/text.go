package lexicalproviders

import (
	"fmt"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

// Text is a Provider for text based lexical completions
type Text struct{}

// Name implements Provider
func (Text) Name() data.ProviderName {
	return data.LexicalTextProvider
}

// Provide implements Provider
func (Text) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	if applesilicon.Detected {
		return nil
	}

	nativeLang := lang.FromFilename(g.FilePath)

	if nativeLang == lang.Python && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	if in.LangGroup.Lexer != lang.Text || !in.LangGroup.Contains(nativeLang) {
		return nil
	}

	if suppressLang(nativeLang, in) {
		return nil
	}

	if in.SelectedBuffer.Buffer.Len() == 0 {
		// empty files can sometimes give weird results for multilingual models
		return nil
	}

	// predict
	modelStart := time.Now()
	predictedChan, errChan := in.Model.PredictChan(ctx, in.PredictInputs)
	for p := range predictedChan {
		modelDuration := time.Since(modelStart)

		if !useCompletionPreRender(nativeLang, p, in) {
			continue
		}

		rendered, ok := text.Render(nativeLang, in.LineContext, p.Tokens)
		if !ok {
			continue
		}

		c := data.Completion{
			Snippet: rendered,
			Replace: data.Selection{
				Begin: in.Selection.Begin - len(p.Prefix),
				End:   in.Selection.End,
			},
		}

		if !useCompletionPostRender(nativeLang, c, in, p) {
			continue
		}

		score := float64(p.Prob) * computeValue(c, p.Tokens)

		mc := MetaCompletion{
			Completion: c,
			Score:      score,
			Provider:   data.TextProviderNameFromPath(g.FilePath),
			LexicalMeta: LexicalMeta{
				DebugStr: fmt.Sprintf("prob: %.3f, score: %.3f", p.Prob, score),
			},
			Metrics:           newLexicalMetrics(p, c, score, modelDuration),
			FromSmartProvider: nativeLang == lang.Python,
			IsServer:          p.IsRemote,
		}

		out(ctx, in.SelectedBuffer, mc)
	}
	if err := <-errChan; err != nil {
		log.Println("LexicalTextProvider.Provider error:", err)
	}

	return nil
}
