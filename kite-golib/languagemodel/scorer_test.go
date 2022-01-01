package languagemodel

import (
	"math"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/text"
)

func TestScorer(t *testing.T) {
	docs := []string{"construct 1D array", "construct 2D array"}
	classes := [][]string{[]string{"array", "print"}, []string{"array"}}
	lms, _ := TrainScorer(docs, classes, text.TokenizeWithoutCamelPhrases)

	posteriors := lms.Posterior([]string{"construct"})
	arrayProb := (2.0 / 3.0) * (2.01 / 16.01)
	printProb := (1.0 / 3.0) * (1.01 / 13.01)
	exp := arrayProb / (arrayProb + printProb)

	if math.Abs(posteriors["array"]-exp) > 1e-8 {
		t.Errorf("expected %f, got %f\n", exp, posteriors["array"])
	}
}
