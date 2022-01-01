package inspect

import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// Matches determines if the LatestPredictions match the code after the cursor
func Matches(sample Sample) ([]bool, error) {
	predictor, err := getPredictor(sample.Query)
	if err != nil {
		return nil, err
	}
	encoded, err := getEncoded(predictor, sample)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get encoded context from sample")
	}

	encoded = predictor.GetEncoder().RemoveBeforeContextPrefix(encoded, sample.Query.Path)
	before := predictor.GetEncoder().RemoveBeforeContextPrefix(sample.Prediction.Meta.ContextBefore, sample.Query.Path)

	i, found := find(encoded, before)
	if !found {
		return nil, errors.New("context not found in code")
	}
	var matches []bool
	start := i + len(before)
	for _, pred := range sample.Prediction.FinalPredictions {
		if len(pred.TokenIDs) > len(encoded[start:]) {
			matches = append(matches, false)
			continue
		}
		if equal(pred.TokenIDs, encoded[start:start+len(pred.TokenIDs)]) {
			matches = append(matches, true)
			continue
		}
		matches = append(matches, false)
	}
	return matches, nil
}

func getEncoded(predictor predict.Predictor, sample Sample) ([]int, error) {
	// TODO: this is pretty nasty, we should figure out a better way to do this,
	// maybe we should be adding the label when we create the query?
	if predictor.GetEncoder().Lexer.Lang() == lang.Text {
		parts := strings.Split(sample.Query.Code, sample.Query.Cursor)
		if len(parts) != 2 {
			return nil, errors.New("query code must have exactly 1 cursor position")
		}

		toks1, err := predictor.GetEncoder().Lexer.Lex([]byte(parts[0]))
		if err != nil {
			return nil, err
		}
		enc1 := predictor.GetEncoder().EncodeTokens(toks1)

		toks2, err := predictor.GetEncoder().Lexer.Lex([]byte(parts[1]))
		if err != nil {
			return nil, err
		}
		encoded := append(enc1, predictor.GetEncoder().EncodeTokens(toks2)...)
		return encoded, nil
	}
	full := strings.Replace(sample.Query.Code, sample.Query.Cursor, "", len(sample.Query.Code))
	return predictor.GetEncoder().EncodeIdx([]byte(full), sample.Query.Path)
}

func find(haystack []int, needle []int) (int, bool) {
	for i := range haystack {
		if i+len(needle) > len(haystack) {
			return -1, false
		}
		if equal(haystack[i:i+len(needle)], needle) {
			return i, true
		}
	}
	return -1, false
}

func equal(x []int, y []int) bool {
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
