package pythonkeyword

import (
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const (
	featuresOp           = "features/features"
	isKeywordLogitsOp    = "classifiers/is_keyword/logits"
	whichKeywordLogitsOp = "classifiers/which_keyword/logits"
)

var (
	// dummyResp is the response that is assumed to be returned from the underlying Tensorflow model if the model
	// is not available. This response signifies an is-keyword probability of 0.
	dummyResp = map[string]interface{}{
		isKeywordLogitsOp:    [][]float32{{1.0, 0.0}},
		whichKeywordLogitsOp: [][]float32{make([]float32, len(pythonscanner.KeywordTokens))},
	}
)

// Model is used to determine whether a current editing situation is for a keyword, and provides a set of keyword
// rankings.
type Model struct {
	model *tensorflow.Model
}

// NewModel loads the keyword model from the given path and initializes it.
func NewModel(path string) (*Model, error) {
	model, err := tensorflow.NewModel(path)
	if err != nil {
		return nil, err
	}

	return &Model{
		model: model,
	}, nil
}

// Reset unloads data
func (m *Model) Reset() {
	m.model.Unload()
}

// GetFeeds returns the feeds needed for inference in the Tensorflow model.
func (*Model) GetFeeds(features Features) map[string]interface{} {
	return map[string]interface{}{
		featuresOp: [][]int64{features.Vector()},
	}
}

// Infer returns:
// - the probability that the user is about to type a keyword given the inputs.
// - a map of token type to the softmax output of the model for each keyword.
func (m *Model) Infer(features Features) (float32, map[pythonscanner.Token]float32, error) {
	start := time.Now()
	defer func() {
		modelInferDuration.RecordDuration(time.Since(start))
	}()

	res := dummyResp
	if m.model != nil {
		feeds := m.GetFeeds(features)
		fetches := []string{isKeywordLogitsOp, whichKeywordLogitsOp}

		var err error
		res, err = m.model.Run(feeds, fetches)
		if err != nil {
			return 0, nil, err
		}
	}

	isKeywordProb := res[isKeywordLogitsOp].([][]float32)[0][1]

	kwProbs := make(map[pythonscanner.Token]float32, NumKeywords())
	kwLogits := res[whichKeywordLogitsOp].([][]float32)
	for _, tok := range pythonscanner.KeywordTokens {
		cat := KeywordTokenToCat(tok)
		if cat > 0 {
			// Category are 1-indexed as 0 is the nul value for int in go
			// But the output array is 0-indexed so we need to do cat-1 to have a match
			kwProbs[tok] = kwLogits[0][cat-1]
		}
	}

	if isKeywordProb > 0.5 {
		modelIsKeywordRatio.Hit()
	} else {
		modelIsKeywordRatio.Miss()
	}

	return isKeywordProb, kwProbs, nil
}

// GetFetches returns the desired fetches.
func (m *Model) GetFetches(features Features, fetches []string) (map[string]interface{}, error) {
	feeds := m.GetFeeds(features)
	res, err := m.model.Run(feeds, fetches)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("model not implemented")
	}
	return res, nil
}
