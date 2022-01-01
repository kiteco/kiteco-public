package pythonranker

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	superheros   = []string{"superman", "kiteman", "batman", "spiderman"}
	trainingData = map[string]*MethodTrainingData{
		"superman": &MethodTrainingData{
			Data:     []string{"super", "man"},
			LogPrior: math.Log(0.25)},
		"kiteman": &MethodTrainingData{
			Data:     []string{"kite", "man"},
			LogPrior: math.Log(0.25)},
		"batman": &MethodTrainingData{
			Data:     []string{"bat", "man"},
			LogPrior: math.Log(0.25)},
		"spiderman": &MethodTrainingData{
			Data:     []string{"spider", "man"},
			LogPrior: math.Log(0.25)},
	}
)

func TestIndex(t *testing.T) {
	target := "kiteman"
	act := index(superheros, target)
	exp := 1

	assert.Equal(t, exp, act)
}

func TestNewMatchPredictor(t *testing.T) {
	names := []string{"array", "transpose", "matrix"}
	mp := NewMatchPredictor(names)
	assert.Equal(t, 3, len(mp.Names))
}

func TestMatchPredictorScore(t *testing.T) {
	mp := NewMatchPredictor(superheros)
	tokens := []string{"kiteman"}

	scores := mp.Scores(tokens)

	for _, ms := range scores {
		switch ms.Name {
		case "kiteman":
			exp := (1 + defaultPseudoCount) / (1 + 4*defaultPseudoCount)
			if math.Abs(exp-ms.Score) > 1e-6 {
				t.Errorf("expected %f, but got %f\n", exp, ms.Score)
			}
		default:
			exp := defaultPseudoCount / (1 + 4*defaultPseudoCount)
			if math.Abs(exp-ms.Score) > 1e-6 {
				t.Errorf("expected %f, but got %f\n", exp, ms.Score)
			}
		}
	}
}

func TestNewLmMethodPredictor(t *testing.T) {
	mp := NewLmMethodPredictor(0.5, trainingData)
	assert.Equal(t, math.Log(0.5), mp.LogNotBackgroundProb)

	for i, name := range mp.MethodNames {
		act := mp.MethodLMs[i].LogLikelihood([]string{"kite"})
		var exp float64
		switch name {
		case "kiteman":
			exp = math.Log((1 + 1/float64(wordVecLen)) / 3)
		default:
			exp = math.Log(1 / float64(wordVecLen) / 3)
		}
		if math.Abs(exp-act) > 1e-6 {
			t.Errorf("expected %f, but got %f\n", exp, act)
		}
	}

	act := mp.BackgroundLM.LogLikelihood([]string{"kite"})
	exp := math.Log((1 + 1/float64(wordVecLen)) / 9)

	if math.Abs(exp-act) > 1e-6 {
		t.Errorf("expected %f, but got %f\n", exp, act)
	}
}

func TestSetLogPrior(t *testing.T) {
	newPriors := map[string]float64{
		"kiteman":   math.Log(0.5),
		"batman":    math.Log(0.3),
		"superman":  math.Log(0.2),
		"spiderman": math.Log(0),
	}

	mp := NewLmMethodPredictor(0.5, trainingData)
	mp.SetLogPrior(newPriors)

	for i, prior := range mp.MethodLogPrior {
		assert.Equal(t, newPriors[mp.MethodNames[i]], prior)
	}
}

func TestSetBackgroundWeight(t *testing.T) {
	mp := NewLmMethodPredictor(0.5, trainingData)
	mp.SetBackgroundWeight(0.2)

	assert.Equal(t, 0.2, mp.useBackground)
	assert.Equal(t, 0.8, mp.notUseBackground)
	assert.Equal(t, math.Log(0.2), mp.LogBackgroundProb)
	assert.Equal(t, math.Log(0.8), mp.LogNotBackgroundProb)
}

func TestLmMethodPredictorScore(t *testing.T) {
	mp := NewLmMethodPredictor(0.5, trainingData)

	scores := mp.Scores([]string{"code"})
	for _, ms := range scores {
		if math.Abs(0.25-ms.Score) > 1e-6 {
			t.Errorf("expected %f, but got %f\n", 0.25, ms.Score)
		}
	}

	scores = mp.Scores([]string{"kite"})

	bg := (1 + 1/float64(wordVecLen)) / 9
	kite := (1 + 1/float64(wordVecLen)) / 3
	others := 1 / float64(wordVecLen) / 3

	for _, ms := range scores {
		var exp float64
		switch ms.Name {
		case "kiteman":
			exp = (0.5*bg + 0.5*kite) / (4*0.5*bg + 0.5*kite + 0.5*3*others)
		default:
			exp = (0.5*bg + 0.5*others) / (4*0.5*bg + 0.5*kite + 0.5*3*others)
		}
		if math.Abs(exp-ms.Score) > 1e-6 {
			t.Errorf("expected %f, but got %f\n", exp, ms.Score)
		}
	}
}
