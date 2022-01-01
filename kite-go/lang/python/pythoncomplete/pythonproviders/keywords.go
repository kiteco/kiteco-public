package pythonproviders

import (
	"encoding/json"
	"fmt"
	"go/token"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	minimumKeywordProbThreshold = 0.1
)

// Keywords is a Provider for keyword completions
type Keywords struct{}

// Name implements Provider
func (Keywords) Name() data.ProviderName {
	return data.PythonKeywordsProvider
}

// Provide implements Provider
func (Keywords) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	// We do not support tensorflow models for Apple Silicon
	if applesilicon.Detected {
		return nil
	}

	if in.Selection.Len() > 0 {
		return data.ProviderNotApplicableError{}
	}

	for _, node := range in.UnderSelection() {
		switch t := node.(type) {
		case *pythonast.AttributeExpr:
			// the only case not covered by findApplicableNode
			if token.Pos(in.Selection.Begin) >= t.Attribute.Begin && token.Pos(in.Selection.End) <= t.Attribute.End {
				return data.ProviderNotApplicableError{}
			}
		}
	}

	n, err := findApplicableNode(in.UnderSelection(), in)
	if err != nil {
		return err
	}
	name, _ := n.(*pythonast.NameExpr)
	if name == nil {
		return data.ProviderNotApplicableError{}
	}

	keywordProb, keywordProbs := runModel(ctx, g, in)
	if keywordProb < 0.5 {
		return data.ProviderNotApplicableError{}
	}

	// First, determine whether the user is typing in the beginning of a statement
	// Find the deepest non-BadStmt that the the cursor lies in; prefer a BadStmt
	var curStmt pythonast.Stmt
	for _, n := range in.UnderSelection() {
		if stmt, ok := n.(pythonast.Stmt); ok {
			curStmt = stmt
			if _, bad := n.(*pythonast.BadStmt); bad {
				break
			}
		}
	}

	inBeginning := true

	if curStmt != nil {
		var firstWord pythonscanner.Word
		for _, w := range in.Words() {
			// Find the first non-whitespace token that is part of the statement
			if len(w.Literal) > 0 && w.Begin >= curStmt.Begin() {
				firstWord = w
				break
			}
		}
		inBeginning = in.Selection.Begin <= int(firstWord.End)
	}

	for tok, prob := range keywordProbs {
		kw := pythonkeyword.AllKeywords[tok]

		if prob < minimumKeywordProbThreshold {
			continue
		}
		if inBeginning && !kw.Beginning {
			continue
		}
		if !inBeginning && !kw.Middle {
			continue
		}

		identifier := tok.String() + kw.FollowedBy
		if tok == pythonscanner.Else && !inBeginning {
			// Special case for inline else as we don't want the colon after in this case
			identifier = tok.String() + " "
		}

		replace := in.Selection
		if !pythonast.IsNil(name) {
			replace = data.Selection{Begin: int(name.Begin()), End: int(name.End())}
		}

		c := data.Completion{
			Replace: replace,
			Snippet: data.Snippet{Text: identifier},
		}
		var ok bool
		c, ok = c.Validate(in.SelectedBuffer)
		if !ok {
			c = data.Completion{
				Replace: data.Selection{Begin: replace.Begin, End: in.Selection.End},
				Snippet: data.Snippet{Text: identifier},
			}
			c, ok = c.Validate(in.SelectedBuffer)
			if !ok {
				continue
			}
		}
		meta := MetaCompletion{
			Completion:       c,
			Score:            float64(prob),
			Provider:         Keywords{}.Name(),
			Source:           response.TraditionalCompletionSource, //TODO(moe): Switch that to KeywordModelSource
			KeywordModelMeta: true,
		}
		out(ctx, in.SelectedBuffer, meta)
	}

	return nil
}

// MarshalJSON implements Provider
func (k Keywords) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: k.Name(),
	})
}

func runModel(ctx kitectx.Context, g Global, in Inputs) (float32, map[pythonscanner.Token]float32) {
	if g.Models == nil || g.Models.Keyword == nil {
		return 0.0, nil
	}

	features, err := pythonkeyword.NewFeatures(ctx, newModelInputs(in), pythonkeyword.ModelLookback)
	if err != nil {
		rollbar.Error(fmt.Errorf("error getting model features: %v", err))
		return 0.0, nil
	}

	prob, kwProbs, err := g.Models.Keyword.Infer(features)
	if err != nil {
		rollbar.Error(fmt.Errorf("error running keyword model: %v", err))
		return 0.0, nil
	}

	return prob, kwProbs
}

// NewModelInputs creates the necessary inputs to determine features for the keyword model.
func newModelInputs(inputs Inputs) pythonkeyword.ModelInputs {
	//TODO(Moe) remove the words recomputation once everything will be in mode KeepEOFIndent
	scanOpts := pythonscanner.Options{
		ScanComments:  true,
		ScanNewLines:  true,
		KeepEOFIndent: true,
	}
	// We need to reparse to keep the end of file indent as they have a big impact on keyword selection
	words, err := pythonscanner.Lex([]byte(inputs.SelectedBuffer.Buffer.Text()), scanOpts)

	if err != nil {
		log.Printf("error reparsing buffer for keyword model: %v", err)
		words = inputs.Words() // Decrease the quality of results but will still work
	}

	return pythonkeyword.ModelInputs{
		Buffer:    []byte(inputs.Buffer.Text()),
		Cursor:    int64(inputs.Selection.Begin),
		AST:       inputs.ResolvedAST().Root,
		Words:     words,
		ParentMap: inputs.ResolvedAST().Parent,
	}
}
