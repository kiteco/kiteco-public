package lexicalproviders

import (
	"fmt"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/python"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

// Python is a Provider for Python lexical completions
type Python struct{}

// Name implements Provider
func (Python) Name() data.ProviderName {
	return data.LexicalPythonProvider
}

// Provide implements Provider
func (py Python) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	if lang.FromFilename(g.FilePath) != lang.Python {
		return nil
	}

	if g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	if suppressLang(lang.Python, in) {
		return nil
	}

	modelStart := time.Now()
	predictedChan, errChan := in.Model.PredictChan(ctx, in.PredictInputs)
	for p := range predictedChan {
		modelDuration := time.Since(modelStart)

		if !useCompletionPreRender(lang.Python, p, in) {
			continue
		}

		// Figure out the indentation setup
		indentSym, currentDepth, _ := python.IndentInspect([]byte(in.Text()), in.Selection.Begin)

		var config python.Config
		func() {
			configLock.Lock()
			defer configLock.Unlock()
			config = pyConfig
			if indentSym == "" {
				// Either the indentation is inconsistent or user hasn't indented yet - Use default
				indentSym = config.Indent
			} else {
				// If we are able to get the indentation symbol, update the default config
				config.Indent = indentSym
			}
		}()

		var matchOption render.MatchOption
		if !(p.Prefix == "" && !in.PrecededBySpace) {
			matchOption = render.MatchStart
		}

		decoded, ok := python.Render(in.LineContext, p.Tokens, p.Prefix != "", in.PrecededBySpace, currentDepth, indentSym)
		if !ok {
			continue
		}

		// NOTE(mna): there is a chicken-and-egg issue here in that we need an
		// AST to pretty-print, but we need source code to generate the AST.
		// So I'm reusing the existing python.Render func to generate as
		// best as it can source code from tokens (which are of course missing
		// contextual parse information required to pretty-print - this is
		// the equivalent of the output of a tokenizer while we need the output
		// and contextual information of a parser).
		c := data.Completion{
			Snippet: decoded,
			Replace: data.Selection{
				Begin: in.Selection.Begin - len(p.Prefix),
				End:   in.Selection.End,
			},
		}

		c.Snippet = python.FormatCompletion(in.Text(), c, config, matchOption)

		if !useCompletionPostRender(lang.Python, c, in, p) {
			continue
		}

		score := float64(p.Prob) * computeValue(c, p.Tokens)

		mc := MetaCompletion{
			Completion: c,
			Provider:   Python{}.Name(),
			Score:      score,
			LexicalMeta: LexicalMeta{
				DebugStr: fmt.Sprintf("prob: %.3f, score: %.3f", p.Prob, score),
			},
			Metrics:           newLexicalMetrics(p, c, score, modelDuration),
			FromSmartProvider: true,
			IsServer:          p.IsRemote,
		}
		out(ctx, in.SelectedBuffer, mc)

	}
	if err := <-errChan; err != nil {
		log.Println("Python.Provider error:", err)
	}

	return nil
}
