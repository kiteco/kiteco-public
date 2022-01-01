package main

import (
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

const (
	numTokens = 104
)

type feature struct {
	name       string
	hideIfZero bool
}

// features gives names for each component of the feature vector ("x") that goes into the classification
var allFeatures []feature

func init() {
	allFeatures = createFeatures()
}

// createFeatureNames creates the feature mapping. This is tightly coupled to the implementation of the actual model.
func createFeatures() []feature {
	var feats []feature

	nodeTypes := pythonkeyword.AllNodeTypes()

	for _, nf := range []string{"last_sibling", "parent_node"} {
		for _, typ := range nodeTypes {
			feats = append(feats,
				feature{
					name:       fmt.Sprintf("%s:%s", nf, typ),
					hideIfZero: true,
				})
		}
	}

	for tok := 0; tok < numTokens; tok++ {
		feats = append(feats,
			feature{
				name:       fmt.Sprintf("first_token:%s", pythonscanner.Token(tok).String()),
				hideIfZero: true,
			})
	}

	feats = append(feats, []feature{
		{name: "rel_indent:equal", hideIfZero: true},
		{name: "rel_indent:less", hideIfZero: true},
		{name: "rel_indent:greater", hideIfZero: true},
	}...)

	for lookback := 0; lookback < pythonkeyword.ModelLookback; lookback++ {
		for tok := 0; tok < numTokens; tok++ {
			feats = append(feats,
				feature{
					name:       fmt.Sprintf("prev:%d:%s", lookback+1, pythonscanner.Token(tok).String()),
					hideIfZero: true,
				})
		}
	}
	feats = append(feats, []feature{
		{name: "prefix:invalid_prefix", hideIfZero: true},
	}...)

	for letter := 'a'; int(letter) <= int('z'); letter++ {
		feats = append(feats, feature{name: fmt.Sprintf("prefix_%c", rune(letter)), hideIfZero: true})
	}
	//Extraneous input feature that is never used
	feats = append(feats, feature{name: "not_used", hideIfZero: true})

	//Previous keywords
	for i := uint(0); i < pythonkeyword.NumKeywords(); i++ {
		feats = append(feats, feature{name: fmt.Sprintf("kw:%s_in_doc", pythonkeyword.KeywordCatToToken(int(i+1))), hideIfZero: true})
	}

	return feats
}

// featureValues returns a map of the feature name to its value as used for input in the model.
// For one-hot features, only the one-hot component gets put in the map.
func featureValues(x [][]float32) map[string]float32 {
	vals := make(map[string]float32)

	if len(allFeatures) != len(x[0]) {
		log.Fatalf("mismatch between feature length (%d) and x size (%d)", len(allFeatures), len(x[0]))
	}

	for i, val := range x[0] {
		f := allFeatures[i]
		if f.hideIfZero && val == 0 {
			continue
		}
		vals[f.name] = val
	}

	return vals
}

// contributions returns a map of the feature name to its contribution to outIdx component of the the pre-softmax output
// of the classifier.
// i.e. if y_j = softmax(sum_i(W_i_j * x_i) + b_j), contribution_i_j = W_i_j * x_i
// For one-hot features, only the one-hot component gets put in the map.
func contributions(weights [][]float32, x [][]float32, outIdx int) map[string]float32 {
	c := make(map[string]float32)

	for i, val := range x[0] {
		f := allFeatures[i]
		if f.hideIfZero && val == 0 {
			continue
		}
		c[f.name] = val * weights[i][outIdx]
	}

	return c
}
