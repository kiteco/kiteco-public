package pythonranker

import (
	"math"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker/internal/precompute"
	"github.com/stretchr/testify/assert"
)

var (
	keywords = []string{"kiteman", "superman", "spiderman", "batman", "kiteman"}
	names    = map[string][]string{
		"numpy.ones":  []string{"numpy.ones"},
		"numpy.zeros": []string{"numpy.zeros"}}

	logPrior = map[string]float64{
		"numpy.ones":  math.Log(0.5),
		"numpy.zeros": math.Log(0.5),
	}
	soCorpus = map[string][]string{
		"numpy.ones":  []string{"Numpy's ones array creation"},
		"numpy.zeros": []string{"How do I create an empty array/matrix in NumPy?"},
	}
	docCorpus = map[string][]string{
		"numpy.ones":  []string{"Return a new array of given shape and type, filled with ones."},
		"numpy.zeros": []string{"Return a new array of given shape and type, filled with zeros."},
	}
	curationCorpus = map[string][]string{
		"numpy.ones":  []string{"Create a 1D array of all ones"},
		"numpy.zeros": []string{"Create a 1D array of all zeros"},
	}
)

func TestKeywordMatcher(t *testing.T) {
	km := newKeywordMatcher(keywords)
	assert.Equal(t, 4, len(km.Keywords))
}

func TestKeywordMatcherMatch(t *testing.T) {
	km := newKeywordMatcher(keywords)
	score, matched := km.match([]string{"kiteman"})

	assert.EqualValues(t, 1, score)
	assert.Equal(t, struct{}{}, matched["kiteman"])
	assert.Equal(t, 1, len(matched))
}

func TestFuzzyKeywordMatcher(t *testing.T) {
	km := newFuzzyKeywordMatcher(keywords)
	assert.Equal(t, 4, len(km.Keywords))

	score, matched := km.match([]string{"man", "men", "kite"})

	assert.EqualValues(t, 2, score)
	assert.Equal(t, 2, len(matched))
}

func TestBuildWordCount(t *testing.T) {
	methods, wordCounts := buildWordCount(soCorpus, docCorpus, curationCorpus)

	expectedWordCounts := map[string]int{
		"i":        1,
		"s":        1,
		"matrix":   1,
		"empti":    1,
		"creation": 1,
		"do":       1,
		"numpi":    2,
		"zero":     2,
		"1d":       2,
		"all":      2,
		"return":   2,
		"new":      2,
		"given":    2,
		"shape":    2,
		"type":     2,
		"fill":     2,
		"creat":    3,
		"on":       3,
		"arrai":    6,
	}

	for _, wc := range wordCounts {
		assert.Equal(t, expectedWordCounts[wc.word], wc.count)
	}

	assert.Equal(t, len(expectedWordCounts), len(wordCounts))
	assert.Equal(t, 2, len(methods))
}

func TestCountWordToMethods(t *testing.T) {
	methods, wordCounts := buildWordCount(soCorpus, docCorpus, curationCorpus)

	wordToMethods := make(map[string][]float64)
	for _, wc := range wordCounts {
		wordToMethods[wc.word] = make([]float64, len(methods))
	}

	methodToID := make(map[string]int)
	for i, m := range methods {
		methodToID[m] = i
	}

	countWordToMethods(wordToMethods, methodToID, soCorpus, docCorpus, curationCorpus)

	expectedWordToMethods := map[string][]float64{
		"i":        []float64{1, 0},
		"s":        []float64{0, 1},
		"matrix":   []float64{1, 0},
		"empti":    []float64{1, 0},
		"creation": []float64{0, 1},
		"do":       []float64{1, 0},
		"numpi":    []float64{1, 1},
		"zero":     []float64{2, 0},
		"1d":       []float64{1, 1},
		"all":      []float64{1, 1},
		"return":   []float64{1, 1},
		"new":      []float64{1, 1},
		"given":    []float64{1, 1},
		"shape":    []float64{1, 1},
		"type":     []float64{1, 1},
		"fill":     []float64{1, 1},
		"creat":    []float64{2, 1},
		"on":       []float64{0, 3},
		"arrai":    []float64{3, 3},
	}

	for w, counts := range expectedWordToMethods {
		assert.Equal(t, counts, []float64{wordToMethods[w][methodToID["numpy.zeros"]], wordToMethods[w][methodToID["numpy.ones"]]})
	}
}

func TestWordToMethodFeaturer(t *testing.T) {
	featurer := newWordToMethodFeaturer(soCorpus, docCorpus, curationCorpus, "word_to_method")

	assert.Equal(t, 1, len(featurer.Word2MethodModel))
	assert.Equal(t, 2, len(featurer.MethodToID))
	assert.Equal(t, "word_to_method", featurer.Label())
}

func TestWordToMethodFeaturerFeatures(t *testing.T) {
	augCorpus := map[string][]string{
		"numpy.ones":  []string{"Numpy's ones array creation", "zeros"},
		"numpy.zeros": []string{"How do I create an empty array/matrix in NumPy?", "zeros", "zeros", "zeros"},
	}

	featurer := newWordToMethodFeaturer(augCorpus, docCorpus, curationCorpus, "word_to_method")

	raw := "create an array of all zeros"
	query := precompute.NewQueryStats(raw, "numpy")
	targets := precompute.NewTargetStats([]string{"numpy.ones", "numpy.zeros"})

	act := featurer.Features("numpy.ones", query, targets)
	exp := 1.0 / 3.0
	if math.Abs(act-exp) > 1e-8 {
		t.Errorf("expected %f, got %f\n", exp, act)
	}

}
