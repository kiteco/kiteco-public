package languagemodel

import (
	"math"
	"testing"
)

func TestSum(t *testing.T) {
	entries := []float64{1.7, 8.9 - 0.2}
	act := sum(entries)
	exp := 10.4
	if math.Abs(act-exp) > 1e-8 {
		t.Errorf("expected %f, got %f", exp, act)
	}
}

func TestLogLikelihood(t *testing.T) {
	tokens := []string{"this", "is", "a", "test"}
	lm := TrainUnigramLanguageModel(tokens, 10)

	act := lm.LogLikelihood(tokens)
	exp := 2*math.Log(1.1/5.0) + 2*math.Log(2.1/5.0)

	if math.Abs(act-exp) > 1e-8 {
		t.Errorf("expected %f, but got %f\n", exp, act)
	}
}
