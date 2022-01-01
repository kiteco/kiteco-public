package main

import (
	"fmt"
	"math/rand"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/internal/utils"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type sampleInputs struct {
	Hash     string
	Src      []byte
	Cursor   int64
	Words    []pythonscanner.Word
	RAST     *pythonanalyzer.ResolvedAST
	NameExpr *pythonast.NameExpr
	// UserTyped contains the buffer starting at the beginning of a name expression to determine which (if any) of
	// completions were actually entered
	UserTyped   []byte
	Completions []mtacconf.Completion
	Idents      []string
}

func (sampleInputs) SampleTag() {}

func getSampleInputs(hash string, src []byte, res utils.Resources, r *rand.Rand) (sampleInputs, error) {
	ast, err := pythonparser.Parse(kitectx.Background(), src, parseOpts)
	if err != nil {
		return sampleInputs{}, fmt.Errorf("unable to parse file: %v", err)
	}
	rast, err := utils.Resolve(ast, res.RM)
	if err != nil {
		return sampleInputs{}, fmt.Errorf("error resolving AST: %v", err)
	}

	// get all name exprs under a call and shuffle them
	exprs := utils.FindNameExprScenarios(rast)
	if len(exprs) == 0 {
		return sampleInputs{}, fmt.Errorf("no NameExprs found")
	}

	// now go through each expr in random order until we successfully generate some inputs
	for _, i := range r.Perm(len(exprs)) {
		expr := exprs[i]

		sampleInputs, err := exprSampleInputs(hash, src, expr, rast, res)
		if err == nil {
			return sampleInputs, nil
		}
	}

	return sampleInputs{}, fmt.Errorf("no valid NameExprs found")
}

func exprSampleInputs(hash string, src []byte, expr *pythonast.NameExpr, rast *pythonanalyzer.ResolvedAST, res utils.Resources) (sampleInputs, error) {
	global := pythonproviders.Global{
		ResourceManager: res.RM,
		Models:          res.Models,
		Product:         licensing.Pro,
	}
	inputs, err := utils.TryName(src, expr, rast, global)
	if err != nil {
		return sampleInputs{}, fmt.Errorf("error getting inputs specific to name: %v", err)
	}
	exprInput := pythonexpr.Input{
		RM:          res.RM,
		RAST:        inputs.ResolvedAST(),
		Words:       inputs.Words(),
		Expr:        inputs.Name,
		MaxPatterns: 10,
	}

	// passing it to the expr model
	exprPred, err := res.Models.Expr.Predict(kitectx.Background(), exprInput)
	if err != nil {
		return sampleInputs{}, fmt.Errorf("expr model can't predict this %v", err)
	}

	var completions []mtacconf.Completion
	var idents []string
	probs, prefixes := []float64{1.}, []string{""}
	pythongraph.Inspect(exprPred.OldPredictorResult, func(n *pythongraph.PredictionTreeNode) bool {
		lastProb := len(probs) - 1
		lastPrefix := len(prefixes) - 1
		if n == nil {
			probs = probs[:lastProb]
			prefixes = prefixes[:lastPrefix]
			return false
		}

		prob := probs[lastProb] * float64(n.Prob)
		probs = append(probs, prob)

		parentPrefix := prefixes[lastPrefix]

		switch {
		case n.AttrBase != "":
			// NOTE: no need to use existing prefix since this is always the root if it exists
			prefixes = append(prefixes, n.AttrBase)
			var c mtacconf.Completion
			if val, err := inputs.TypeValueForName(kitectx.Background(), global.ResourceManager, n.AttrBase); err == nil {
				c.Referent = val
			}
			c.Score = prob
			c.Source = response.ExprModelCompletionsSource
			c.MixData = mtacconf.GetMixData(kitectx.Background(), global.ResourceManager, inputs.Selection, inputs.Words(), inputs.ResolvedAST(), inputs.Name)
			completions = append(completions, c)
			idents = append(idents, n.AttrBase)
		case !n.Attr.Nil():
			prefix := fmt.Sprintf("%s.%s", parentPrefix, n.Attr.Path().Last())
			prefixes = append(prefixes, prefix)

			var c mtacconf.Completion
			c.Referent = pythontype.NewExternal(n.Attr, global.ResourceManager)
			c.Score = prob
			c.Source = response.AttributeModelCompletionSource // TODO: add separate sources for expr generated attributes?
			c.MixData = mtacconf.GetMixData(kitectx.Background(), global.ResourceManager, inputs.Selection, inputs.Words(), inputs.ResolvedAST(), inputs.Name)
			completions = append(completions, c)
			idents = append(idents, prefix)
		default:
			prefixes = append(prefixes, "")
		}
		return true
	})
	return sampleInputs{
		Hash:        hash,
		Src:         []byte(inputs.Text()),
		Cursor:      int64(inputs.Cursor()),
		Words:       inputs.Words(),
		NameExpr:    inputs.Name,
		UserTyped:   inputs.UserTyped,
		Completions: completions,
		Idents:      idents,
	}, nil
}
