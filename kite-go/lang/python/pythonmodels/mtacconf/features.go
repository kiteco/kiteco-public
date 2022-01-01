package mtacconf

import (
	"fmt"
	"math"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/threshold"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// ContextualFeatures represents a subset of features that can be derived just from the source code.
type ContextualFeatures struct {
	NumVars int `json:"num_vars"` // number of variables in scope at the prediction site
}

func (c ContextualFeatures) vector() []float32 {
	var numVars float32
	if c.NumVars > 0 {
		numVars = float32(math.Log(float64(c.NumVars)))
	}

	return []float32{1.0, numVars}
}

// CompFeatures represents a subset of features derived from completions
type CompFeatures struct {
	Score                    float64                `json:"score"`
	ReturnTypePercentMatched float64                `json:"return_type_percent_matched"`
	IsIterable               int                    `json:"is_iterable"`
	NoneRatio                float64                `json:"none_ratio"`
	CompTypesEmpty           int                    `json:"comp_types_empty"`
	Scenario                 threshold.MTACScenario `json:"scenario"`
}

func (c CompFeatures) vector() []float32 {
	scenarioVector := make([]float32, c.Scenario.Size())
	scenarioVector[c.Scenario] = 1
	compVec := []float32{
		float32(c.Score),
		float32(c.ReturnTypePercentMatched),
		float32(c.IsIterable),
		float32(c.NoneRatio),
		float32(c.CompTypesEmpty),
	}
	compVec = append(compVec, scenarioVector...)
	return compVec
}

// Features to the MTAC confidence model
type Features struct {
	Contextual ContextualFeatures `json:"contextual"`
	Comps      []CompFeatures     `json:"comp"`
}

func typesForParameter(rm pythonresource.Manager, pe *editorapi.ParameterExample) ([]pythonresource.Symbol, error) {
	if len(pe.Types) == 0 {
		return nil, fmt.Errorf("we don't have popular types")
	}

	var ts []pythonresource.Symbol
	for _, t := range pe.Types {
		symbol, err := rm.PathSymbol(pythonimports.NewDottedPath(t.ID.LanguageSpecific()))
		if err != nil {
			continue
		}
		ts = append(ts, symbol)

	}
	return ts, nil
}

func returnTypePercentPos(argTypes []pythonresource.Symbol, compTypes []pythonresource.Symbol) float64 {
	if argTypes == nil {
		return 0.0
	}

	if len(argTypes) == 0 && len(compTypes) == 0 {
		return 1.0
	}
	if len(compTypes) == 0 {
		return 0.0
	}

	var numMatched int
	for _, at := range argTypes {
		for _, ct := range compTypes {
			if ct.Equals(at) {
				numMatched++
			}
		}
	}
	return float64(numMatched) / float64(len(compTypes))
}

func typeOfComp(c Completion, rm pythonresource.Manager) ([]pythonresource.Symbol, error) {
	valType := c.Referent
	valType = pythontype.Translate(kitectx.Background(), valType, rm)
	ext, ok := valType.(pythontype.External)
	if !ok {
		return nil, fmt.Errorf("error getting symbol")
	}
	sym := ext.Symbol()
	switch c.Source {
	case response.ExprModelCompletionsSource:
		return []pythonresource.Symbol{sym}, nil
	case response.AttributeModelCompletionSource:
		return rm.ReturnTypes(sym), nil
	default:
		return nil, fmt.Errorf("not yet handling this type of completion")
	}
}

// ratio of the return type being None over all types
func ratioReturnTypeNone(compTypes []pythonresource.Symbol) float64 {
	if len(compTypes) == 0 {
		return 0
	}

	for _, ct := range compTypes {
		if ct.Canonical().PathString() == "builtins.None.__class__" {
			return float64(1) / float64(len(compTypes))
		}
	}
	return 0
}

// This function finds out if in either the return types or the expr types, anything is iterable.
func isIterable(compTypes []pythonresource.Symbol, rm pythonresource.Manager) int {
	for _, c := range compTypes {
		val := pythontype.TranslateExternal(c, rm)
		if _, ok := val.(pythontype.Iterable); ok {
			return 1
		}
	}

	return 0
}

// This function computes percentage match between:
// 1. The type of an expression or the return type if the expression is function.
// 2. The argument type (signature pattern) of the parent function.
func returnTypePercentMatched(c Completion, rm pythonresource.Manager, sigPatterns []*editorapi.Signature, compTypes []pythonresource.Symbol) float64 {
	var numPattern int
	var percentMatch float64
	for _, sp := range sigPatterns {
		if len(sp.Args) == 0 {
			continue
		}

		if c.MixData.Call.Pos >= len(sp.Args) {
			continue
		}
		numPattern++

		argTypes, err := typesForParameter(rm, sp.Args[c.MixData.Call.Pos])
		if err != nil {
			continue
		}
		score := returnTypePercentPos(argTypes, compTypes)
		percentMatch += score
	}
	if numPattern < 1 {
		return 0
	}
	return percentMatch / float64(numPattern)
}

// NewFeatures builds new features from MTAC inputs
func NewFeatures(in Inputs) (Features, error) {
	comp := make([]CompFeatures, 0, len(in.Comps))
	for _, c := range in.Comps {
		// type of the completions
		compTypes, err := typeOfComp(c, in.RM)
		if err != nil {
			comp = append(comp, CompFeatures{
				Score: c.Score,
			})
			continue
		}

		var compTypesEmpty int
		if len(compTypes) == 0 {
			compTypesEmpty = 1
		}

		switch c.MixData.Scenario {
		case threshold.InCall:
			sym := c.MixData.Call.Sym
			if sym.Nil() || c.MixData.Call.Pos == -1 {
				comp = append(comp, CompFeatures{
					Score:          c.Score,
					CompTypesEmpty: compTypesEmpty,
					Scenario:       c.MixData.Scenario,
				})
				continue
			}

			// get pop patterns
			sigPatterns := in.RM.PopularSignatures(sym)

			// check if patterns are empty
			if len(sigPatterns) == 0 {
				comp = append(comp, CompFeatures{
					Score:          c.Score,
					CompTypesEmpty: compTypesEmpty,
					Scenario:       c.MixData.Scenario,
				})
				continue
			}
			comp = append(comp, CompFeatures{
				Score:                    c.Score,
				ReturnTypePercentMatched: returnTypePercentMatched(c, in.RM, sigPatterns, compTypes),
				CompTypesEmpty:           compTypesEmpty,
				Scenario:                 c.MixData.Scenario,
			})
		case threshold.InIf, threshold.InWhile:
			noneRatio := ratioReturnTypeNone(compTypes)
			comp = append(comp, CompFeatures{
				Score:          c.Score,
				NoneRatio:      noneRatio,
				CompTypesEmpty: compTypesEmpty,
				Scenario:       c.MixData.Scenario,
			})
		case threshold.InFor:
			comp = append(comp, CompFeatures{
				Score:          c.Score,
				IsIterable:     isIterable(compTypes, in.RM),
				CompTypesEmpty: compTypesEmpty,
				Scenario:       c.MixData.Scenario,
			})
		}

	}

	if len(comp) == 0 {
		return Features{}, fmt.Errorf("no completion features")
	}
	return Features{Comps: comp}, nil
}

func (f Features) feeds() map[string]interface{} {
	compFeatures := make([][]float32, 0, len(f.Comps))
	for _, c := range f.Comps {
		compFeatures = append(compFeatures, c.vector())
	}
	return map[string]interface{}{
		"placeholders/contextual_features": [][]float32{f.Contextual.vector()},
		"placeholders/completion_features": compFeatures,
		"placeholders/sample_ids":          make([]int32, len(f.Comps)),
	}
}
