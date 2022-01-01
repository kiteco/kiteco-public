package lang

import (
	"math"
	"path"
	"testing"
)

// Pre-define useful variables
var (
	testDir   = "test"
	testFile  = path.Join(testDir, "naive_bayes_training.txt")
	wordScore = math.Log10(2.0 / (wordVecLen + 4.0))
)

// Test addOneSmooth() of naiveBayesModel.
// The model is not trained on any data; therefore, nb.wordHashVec
// should be [1, 1, ..., 1] after calling addOneSmooth()
func TestNaiveBayesModel_addOneSmooth(t *testing.T) {
	nb := newNaiveBayesModel()
	nb.addOneSmooth()
	for _, v := range nb.wordHashVec {
		if v != 1 {
			t.Errorf("Expected 1, got %f in wordHashVec of an empty model", v)
		}
	}
}

// Test normalize() of naiveBayesModel.
// The model is not trained on any data; thereofre, nb.wordHashVec
// should be [-log10(wordVecLen), -log10(wordVecLen), ..., -log10(wordVecLen)]
// after calling addOneSmooth() and normalize()
func TestNaiveBayesModel_normalize(t *testing.T) {
	nb := newNaiveBayesModel()
	nb.addOneSmooth()
	nb.normalize()
	for _, v := range nb.wordHashVec {
		if v != -math.Log10(wordVecLen) {
			t.Errorf("Expected %f, got %f in wordHashVec of an empty model"+
				"after normalization", -math.Log10(wordVecLen), v)
		}
	}
}

// Test logLikelihood(string) of naiveBayesModel.
// The model is not trained on any data; therefore, the probability of
// any single word should be -log10(wordVecLen).
func TestNaiveBayesModel_logLikelihood(t *testing.T) {
	nb := newNaiveBayesModel()
	nb.addOneSmooth()
	nb.normalize()
	if nb.logLikelihood("hello") != -math.Log10(wordVecLen) {
		t.Errorf("Expected %f, got %f for log p(hello|D) from an empty model",
			-math.Log10(wordVecLen), nb.logLikelihood("hello"))
	}
}

// Test trainFromData(string) of naiveBayesModel.
// The model is trained on a file that contains 'package java.io.IOException;';
// therefore, log p('package'|D) should be wordScore.
func TestNaiveBayesModel_TrainFromData(t *testing.T) {
	nb := newNaiveBayesModel()
	nb.TrainFromData(testDir)
	if nb.logLikelihood("package") != wordScore {
		t.Errorf("Expected %f, got %f for log p('package'|D) from a model"+
			"trained with 'package java.io.IOException;'",
			wordScore, nb.logLikelihood("package"))
	}
}

// Test sum([]float64) on [1.0, 2.4, 2.5, 6.8, -1.0, 3.9]
func TestUtils_sum(t *testing.T) {
	a := []float64{1.0, 2.4, 2.5, 6.8, -1.0, 3.9}
	if sum(a) != 15.6 {
		t.Errorf("Expected 15.6, got %f for"+
			"summing(1.0, 2.4, 2.5, 6.8, -1.0, 3.9)", sum(a))
	}
}

// Test readLines(string) on testFile.
func TestUtils_readLines(t *testing.T) {
	lines, _ := readLines(testFile)
	if len(lines) != 1 {
		t.Errorf("Expected to read 1 line from %s, got %d",
			testFile, len(lines))
	}
}

// Test getTokens(string) on "package java.io.IOException;"
func TestUtils_getTokens(t *testing.T) {
	tokens := getTokens("package java.io.IOException;")
	truth := []string{"package", "java", "io", "IOException;"}
	for i, w := range tokens {
		if w != truth[i] {
			t.Errorf("Expected %s, got %s", truth[i], w)
		}
	}
}

// Test logLikelihood(string) of languageDetector.
// The naiveBayesModel inside ld is trained a file that contains
// 'package java.io.IOException;'; therefore,
// p('package java.io.IOException;'|D) should be 4 * wordScore.
func TestLanguageDetector_logLikelihood(t *testing.T) {
	ld := NewLanguageDetector()
	ld.Model.TrainFromData(testDir)
	score := ld.logLikelihood("package java.io.IOException;")
	if score != 4*wordScore {
		t.Errorf("Expected %f, got %f",
			4*wordScore, score)
	}
}

// Test getMostLikelyLanguage(map[Language]float64), which is used
// in DetectByContents. By testing this method (
// along with all other tests above), we know DetectByContents is also
// logically correct. Whether DetectByContents is functionally correct
// depends on the quality of the naive Bayes models.
func TestUtils_getMostLikelyLanguage(t *testing.T) {
	scores := map[Language]float64{
		Golang: -123.456,
		Java:   -1246.35,
		Python: -772.13,
		Cpp:    -1.976,
	}

	languageID := getMostLikelyLanguage(scores)

	if languageID != Cpp {
		t.Errorf("cpp should be the most likely (awesome) language,"+
			"but got %s", languageID.Extension())
	}

	scores2 := map[Language]float64{
		Golang: -1.456,
		Java:   -1.455,
	}

	languageID = getMostLikelyLanguage(scores2)

	if languageID != Unknown {
		t.Errorf("Expected Unknown, but got %s", languageID.Extension())
	}
}
