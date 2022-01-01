package main

import (
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncompletions"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// State contains all the information necessary to build a training sample
type State struct {
	Buffer        []byte
	Words         []pythonscanner.Word
	RAST          *pythonanalyzer.ResolvedAST
	AttributeExpr *pythonast.AttributeExpr
	UserTyped     []byte
	Cursor        int64
}

type compResources struct {
	rm     pythonresource.Manager
	models *pythonmodels.Models
}

func getMixInputs(s State, res compResources) ([]pythoncompletions.MixInput, error) {
	inputs := python.CompletionsInputs{
		Buffer:   s.Buffer,
		Resolved: s.RAST,
		Importer: pythonstatic.Importer{Global: res.rm},
		Models:   res.models,
	}

	cb := python.NewCompletionsEngine(inputs).Callbacks

	compInputs := inputs.EngineInputs(kitectx.Background())
	var result pythoncompletions.ProvisionResult
	var mixInputs []pythoncompletions.MixInput
	var tp string
	var ok bool
	err := kitectx.Background().WithTimeout(10*time.Second, func(ctx kitectx.Context) error {
		result = pythoncompletions.AttributesWithPrefetcher(
			ctx, compInputs, pythoncompletions.Attribute{Node: s.AttributeExpr}, cb)

		prefetcher := result.Prefetcher.(*pythoncompletions.AttributePrefetcher)
		prefetcher.Wait()

		mixInputs, tp, _, ok = prefetcher.MixInputs(ctx, compInputs, cb)

		return nil
	})
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("state incompatible with prefetcher")
	}

	name, ok := s.AttributeExpr.Value.(*pythonast.NameExpr)
	if !ok {
		return nil, fmt.Errorf("base of value is not a name expression")
	}

	start := name.End()
	length := len(name.Ident.Literal)
	fullName := name.Ident.Literal

	attrModelResults, err := cb.AttributePredictions(kitectx.Background(), int64(start), int64(length), fullName)
	if err != nil {
		return nil, fmt.Errorf("error predicting attributes: %v", err)
	}

	attrModelMixInputs := make([]pythoncompletions.MixInput, 0, len(result.Completions))
	completionToScoreMap := make(map[string]float64)
	for _, child := range attrModelResults.Children {
		completionToScoreMap[child.Attr.Path().Last()] = float64(child.Prob)
	}

	for _, c := range filterCompletions(result.Completions, tp) {
		if score, ok := completionToScoreMap[c.Identifier]; ok {
			c.Score = score
			attrModelMixInputs = append(attrModelMixInputs, pythoncompletions.GGNNAttribute{Comp: c})
		}
	}

	mixInputs = append(mixInputs, attrModelMixInputs...)
	return mixInputs, nil
}

func filterCompletions(completions []pythoncompletions.Completion, typedPrefix string) []pythoncompletions.Completion {
	filtered := make([]pythoncompletions.Completion, 0, len(completions))
	for _, comp := range completions {
		if pythoncompletions.CompletionMatches(comp.Identifier, typedPrefix, "") {
			filtered = append(filtered, comp)
		}
	}
	return filtered
}
