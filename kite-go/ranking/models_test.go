package ranking

import (
	"bytes"
	"log"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var jsondata = `
{
    "ScorerType": "Linear",
    "FeatureLabels": ["tfidf_query_title", "tfidf_query_prelude"],
    "Scorer": {
	"Weights": [1.284058633332359, 0.5225498922557076]
    },
    "Normalizer": {
	"Offset": [-1.1618373628617606, -0.5474899417738636],
	"Scale": [0.9781642001060329, 1.1473641542709838]}
    }
}
`

func TestNormalizer(t *testing.T) {
	normalizer := Normalizer{
		Offset: []float64{-0.5, -1.5},
		Scale:  []float64{0.2, 0.6},
	}
	test := []float64{3, 6}
	act := normalizer.Normalize(test)
	exp := []float64{0.5, 2.7}

	if math.Abs(act[0]-exp[0]) > 1e-8 {
		t.Errorf("expected %f got %f", exp[0], act[0])
	}

	if math.Abs(act[1]-exp[1]) > 1e-8 {
		t.Errorf("expected %f got %f", exp[1], act[1])
	}
}

func TestLoadLinearModel(t *testing.T) {
	buf := bytes.NewBuffer([]byte(jsondata))
	ranker, err := NewRankerFromJSON(buf)
	assert.NoError(t, err, "")

	exp := []string{
		"tfidf_query_title",
		"tfidf_query_prelude",
	}

	assert.Equal(t, exp, ranker.FeatureLabels)

	feat1 := (-1.1618373628617606) * 0.9781642001060329
	feat2 := (-0.5474899417738636) * 1.1473641542709838
	dim1Score := feat1 * 1.284058633332359
	dim2Score := feat2 * 0.5225498922557076
	expScore := dim1Score + dim2Score
	actScore := ranker.Scorer.Evaluate([]float64{feat1, feat2})

	if math.Abs(expScore-actScore) > 1e-8 {
		t.Errorf("expected %f got %f", expScore, actScore)
	}

	data1 := &DataPoint{
		ID:       0,
		Features: []float64{3, 3},
	}
	data2 := &DataPoint{
		ID:       1,
		Features: []float64{10, 10},
	}
	rankedData := []*DataPoint{data1, data2}
	ranker.Rank(rankedData)
	if rankedData[0].ID != 1 {
		t.Errorf("expected data point 1 to rank above data point 0")
	}
}

func TestLoadTreeEnsembleScorer(t *testing.T) {
	buf, err := os.Open("testdata/martmodel.json")
	if err != nil {
		log.Fatal(err)
	}
	ranker, err := NewRankerFromJSON(buf)
	assert.NoError(t, err, "")

	exp := []string{
		"tfidf_query_title",
		"tfidf_query_prelude",
		"matched_package",
		"idf_query_title_unoverlapped",
		"package_prob",
		"max_idf_query_doc",
		"method_lm_prob",
		"code_linenum",
	}

	assert.Equal(t, exp, ranker.FeatureLabels)
}

func TestRbfKernel(t *testing.T) {
	a := []float64{1., 2.}
	b := []float64{3., 4.}
	k := RbfKernel{-.5}
	assert.Equal(t, 0.01831563888873418, k.Evaluate(a, b))
}

func TestKernelScorer(t *testing.T) {
	support := [][]float64{{10., 20., 30.}, {40., 50., 60.}}
	coefs := []float64{-1., 3.}
	s := NewRbfKernelScorer(support, coefs, -1.)
	assert.Equal(t, -0.049787068367863944, s.Evaluate([]float64{11., 21., 31.}))
}
