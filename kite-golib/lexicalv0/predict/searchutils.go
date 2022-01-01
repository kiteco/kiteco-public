package predict

import (
	"sort"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

// PadContext pads the context with `padValue` such that it is the provided length. It returns
// the padded context and a mask over the padded context corresponding to the added padding.
// e.g [1,2,3,4] w/ window 6 -> [padValue,padValue,1,2,3,4], [0,0,1,1,1,1]
func PadContext(context []int64, window int, padValue int64) ([]int64, []int64) {
	if len(context) >= window {
		return context, ones(window, 0)
	}

	nPad := window - len(context)

	padding := make([]int64, nPad, nPad+len(context))
	if padValue != 0 {
		for i := range padding {
			padding[i] = padValue
		}
	}

	padding = append(padding, context...)
	return padding, ones(window, nPad)
}

// PrefixIDMask creates a mask over the vocab corresponding to the ids to keep.
// TODO(tarak): this works but worth optimizing? maybe just store the all 1's case
func PrefixIDMask(prefixIDs []int64, vocabSize int) []int64 {
	if len(prefixIDs) == 0 {
		return ones(vocabSize, 0)
	}
	vector := make([]int64, vocabSize)
	for _, id := range prefixIDs {
		vector[id] = 1
	}
	return vector
}

func ones(size, offset int) []int64 {
	v := make([]int64, size)
	for i := 0; i < len(v); i++ {
		if i >= offset {
			v[i] = 1
		}
	}
	return v
}

func buildPredicted(results [][]int64, probs [][]float32, minp float32, prefix string, enc *lexicalv0.FileEncoder) ([]Predicted, error) {
	if len(results) != len(probs) {
		return nil, errors.Errorf("bad times, len(results) = %d, len(probs) = %d", len(results), len(probs))
	}

	// These shapes are the initial results/probs states, no results returned
	// TOOD(tarak): is this an error in the beam search?
	if len(results) == 1 && len(results[0]) == 0 &&
		len(probs) == 1 && len(probs[0]) == 0 {
		return nil, nil
	}

	var predicted []Predicted
	for i := 0; i < len(results); i++ {
		hypothesis := results[i]
		chainedProbs := probs[i]

		if len(hypothesis) != len(chainedProbs) {
			return nil, errors.Errorf("bad times, len(hypothesis) = %d, len(chainedProbs) = %d\n",
				len(hypothesis), len(chainedProbs))
		}

		// Find end-of-prediction
		var eop int
		for eop = 0; eop < len(hypothesis) && hypothesis[eop] >= 0; eop++ {
		}

		if eop == 0 {
			continue
		}

		if chainedProbs[eop-1] >= minp {
			tokens := make([]int64, 0, len(hypothesis[:eop]))
			for _, t := range hypothesis[:eop] {
				tokens = append(tokens, t)
			}

			prob := chainedProbs[eop-1]
			pred := newPredicted(tokens, prob, prefix, enc, true)

			predicted = append(predicted, pred)
		}
	}

	// TODO: this sort is to make results consistent,
	// but user's shouldn't rely on it as a global ordering
	// since comparing probabilities for predictions of different
	// lengths is not neccesarily desirable.
	sort.Slice(predicted, func(i, j int) bool {
		return predicted[i].Prob > predicted[j].Prob
	})

	return predicted, nil
}

func invertScaling(val float32) float32 {
	if val == 0.0 {
		return 1.0
	}
	return 1.0 / val
}
