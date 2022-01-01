package tfidf

import (
	"math"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/text"
	"github.com/stretchr/testify/assert"
)

func TestRawTFIDF(t *testing.T) {
	scores := RawTFIDFs(map[string]float64{
		"numpy": 1.35,
		"scipy": 3.8,
		"array": 0.4,
	})

	act := scores.RawTFIDF("numpy")
	exp := 1.35

	assert.Equal(t, exp, act)

	act = scores.LogisticTFIDF("numpy")
	exp = math.Exp(1.35) / sum([]float64{math.Exp(1.35), math.Exp(3.8), math.Exp(0.4)})
	assert.Equal(t, exp, act)

	mean := sum([]float64{1.35, 3.8, 0.4}) / 3.0
	std := math.Sqrt(sum([]float64{1.35 * 1.35, 3.8 * 3.8, 0.4 * 0.4})/3.0 - mean*mean)
	act = scores.GaussianizedTFIDF("numpy")
	exp = (1.35 - mean) / std

	if math.Abs(act-exp) > 1e-8 {
		t.Error("error in computing GaussianizedTFIDF")
	}

	act = scores.NormalizedTFIDF("numpy")
	exp = 1.35 / sum([]float64{1.35, 3.8, 0.4})

	if math.Abs(act-exp) > 1e-8 {
		t.Error("error in computing GaussianizedTFIDF")
	}
}

func TestTFIDFScorer(t *testing.T) {
	corpus := make(map[string][]string)
	loadData(corpus)
	scorer := TrainScorer(corpus, true, text.TokenizeWithoutCamelPhrases)

	docIDs := scorer.DocIDs()
	assert.Equal(t, 2, len(docIDs))

	scores := scorer.TFIDFScores([]string{"kiteman"})
	rawTFIDF := scores.RawTFIDF("doc1")
	exp := 0

	assert.EqualValues(t, exp, rawTFIDF)
	assert.EqualValues(t, 0, scores.RawTFIDF("doc3"))

	scores = scorer.TFIDFScores([]string{"yolo"})
	rawTFIDF = scores.RawTFIDF("doc1")
	expTFIDF := math.Log10(2) * 0.75

	if math.Abs(rawTFIDF-expTFIDF) > 1e-8 {
		t.Errorf("expected %f, got %f\n", expTFIDF, rawTFIDF)
	}
}

func loadData(corpus map[string][]string) {
	corpus["doc1"] = []string{"kiteman v.s. superman", "yolo v.s. helo"}

	corpus["doc2"] = []string{"kiteman is apple"}
}

func TestComputeNorm(t *testing.T) {
	corpus := make(map[string][]string)
	loadData(corpus)
	scorer := TrainScorer(corpus, true, text.TokenizeWithoutCamelPhrases)

	queryTokens := []string{"superman", "superman", "apple"}

	// get the tfidf length of the query
	tfCorpus := make(map[string]int)
	for _, tok := range text.TFProcessor.Apply(queryTokens) {
		tfCorpus[tok]++
	}
	tfCounter := TrainTFCounter(false, tfCorpus)

	act := scorer.ComputeNorm(tfCounter)
	v1 := math.Log10(2.0) * (2.0 / 3.0)
	v2 := math.Log10(2.0) * (1.0 / 3.0)
	exp := math.Sqrt(v1*v1 + v2*v2)

	if math.Abs(act-exp) > 1e-8 {
		t.Errorf("expected %f, got %f\n", exp, act)
	}

	tfCounter = TrainTFCounter(true, tfCorpus)
	act = scorer.ComputeNorm(tfCounter)

	v1 = math.Log10(2.0)
	v2 = math.Log10(2.0) * 0.75
	exp = math.Sqrt(v1*v1 + v2*v2)

	if math.Abs(act-exp) > 1e-8 {
		t.Errorf("expected %f, got %f\n", exp, act)
	}
}

func TestTFIDFScore(t *testing.T) {
	corpus := make(map[string][]string)
	loadData(corpus)
	scorer := TrainScorer(corpus, true, text.TokenizeWithoutCamelPhrases)

	queryTokens := []string{"superman", "superman", "apple"}

	act := scorer.TFIDFScore(queryTokens, "doc1")

	v11 := math.Log10(2.0)
	v12 := math.Log10(2.0) * 0.75
	norm1 := math.Sqrt(v11*v11 + v12*v12)

	v21 := math.Log10(2.0) * 0.75
	v22 := 0.0
	v23 := math.Log10(2.0)
	v24 := math.Log10(2.0) * 0.75
	v25 := math.Log10(2.0)
	v26 := math.Log10(2.0) * 0.75

	norm2 := math.Sqrt(v21*v21 + v22*v22 + v23*v23 + v24*v24 + v25*v25 + v26*v26)

	exp := v11 * v21 / (norm1 * norm2)

	if math.Abs(act-exp) > 1e-8 {
		t.Errorf("expected %f, got %f\n", exp, act)
	}

	if math.Abs(norm2-scorer.Norm["doc1"]) > 1e-8 {
		t.Errorf("expected %f, got %f\n", norm2, scorer.Norm["doc1"])
	}
}
