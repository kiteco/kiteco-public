package pythonmixing

import (
	"math"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncompletions"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// CompType defines enum to represent which model the completion is from
type CompType int

func (compType CompType) size() int {
	return 4
}

const (
	// SinglePopularityComp describes a popularity-based single attribute completion.
	SinglePopularityComp CompType = 0
	// SingleGGNNAttrComp describes a model-based single attribute completion.
	SingleGGNNAttrComp CompType = 1
	// MultiGGNNAttrComp describes a multi-token model-based attribute completion.
	MultiGGNNAttrComp CompType = 2
	// CallGGNNComp describes a multi-token model-based call completion.
	CallGGNNComp CompType = 3
)

// ContextualFeatures represents a subset of features that can be derived just from the source code.
type ContextualFeatures struct {
	ParentType int `json:"parent_type"` // This is the astNodeType of the ast parent node.
	NumVars    int `json:"num_vars"`
}

func (c ContextualFeatures) vector() []float32 {
	var numVars float32
	vectorizedContext := make([]float32, 62)
	vectorizedContext[c.ParentType-1] = 1.0
	if c.NumVars > 0 {
		numVars = float32(math.Log(float64(c.NumVars)))
	}
	vectorizedContext[61] = numVars
	return vectorizedContext
}

// CompFeatures represents a subset of features that only depends on the completions returned from completion models.
type CompFeatures struct {
	PopularityScore  float64  `json:"popularity_score"`
	SoftmaxScore     float64  `json:"softmax_score"`
	AttrScore        float64  `json:"attr_score"`
	Model            CompType `json:"model"`
	CompletionLength int      `json:"completion_length"`
	NumArgs          int      `json:"num_args"`
}

func (c CompFeatures) vector() []float32 {
	compVector := make([]float32, 0, c.Model.size()+5)

	var popularityScore float32
	if c.PopularityScore > 0.0 {
		popularityScore = float32(math.Log(c.PopularityScore))
	}
	compVector = append(compVector, popularityScore)

	var softmaxScore float32
	if c.SoftmaxScore > 0.0 {
		softmaxScore = float32(math.Log(c.SoftmaxScore))
	}
	compVector = append(compVector, softmaxScore)

	var attrScore float32
	if c.AttrScore > 0.0 {
		attrScore = float32(math.Log(c.AttrScore))
	}
	compVector = append(compVector, attrScore)

	modelVec := make([]float32, c.Model.size())
	modelVec[c.Model] = 1.0
	compVector = append(compVector, modelVec...)
	compVector = append(compVector, float32(c.CompletionLength), float32(c.NumArgs))
	return compVector
}

// Features used for prediction
type Features struct {
	Contextual ContextualFeatures `json:"contextual"`
	Comp       []CompFeatures     `json:"comp"`
}

// NewFeatures calculated from the inputs
func NewFeatures(ctx kitectx.Context, in Inputs) (Features, error) {
	var numVars int
	parentType := pythonkeyword.NodeToCat(in.RAST.Parent[in.AttributeExpr])

	comp := make([]CompFeatures, 0, len(in.MixInputs))
	for _, m := range in.MixInputs {
		c := m.Completion()
		numVars = c.MixData.NumVarsInScope
		cf := CompFeatures{
			CompletionLength: len(c.Identifier),
		}
		switch m.(type) {
		case pythoncompletions.PopularitySingleAttribute:
			cf.Model = SinglePopularityComp
			cf.PopularityScore = c.Score / 1000
		case pythoncompletions.MultiTokenCall:
			cf.Model = CallGGNNComp
			cf.NumArgs = strings.Count(c.Identifier, ",") + 1
			confidenceScore, err := in.CallProb(ctx, int64(in.AttributeExpr.Dot.Begin), c)
			if err == nil {
				cf.SoftmaxScore = float64(confidenceScore)
			} else {
				cf.SoftmaxScore = c.Score
			}
		case pythoncompletions.GGNNAttribute:
			cf.Model = SingleGGNNAttrComp
			cf.AttrScore = c.Score
		}

		comp = append(comp, cf)
	}

	contextual := ContextualFeatures{ParentType: parentType, NumVars: numVars}
	return Features{
		Contextual: contextual,
		Comp:       comp,
	}, nil
}

func (f Features) feeds() map[string]interface{} {
	compFeatures := make([][]float32, 0, len(f.Comp))
	for _, c := range f.Comp {
		compFeatures = append(compFeatures, c.vector())
	}

	return map[string]interface{}{
		"placeholders/contextual_features": [][]float32{f.Contextual.vector()},
		"placeholders/completion_features": compFeatures,
		"placeholders/sample_ids":          make([]int32, len(f.Comp)),
	}
}
