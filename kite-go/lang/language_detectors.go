package lang

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"math"
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// wordVecLen defines the size of the hash table used to score word counts
// modelDir points to the directory where the pre-trained models for language
// detection are
const (
	wordVecLen = 100003
)

// naiveBayesModel learns p(w|l;D), the probability of w being
// used in programming language l. D denotes the training data.
// We use a hash table to store the word counts. Using hash tables allows us
// to deal with the out-of-vocabulary problem, which may occur when
// new programming languages are added, and it also allows us
// to train the model for each language independently.
type naiveBayesModel struct {
	wordHashVec []float64
}

func newNaiveBayesModel() naiveBayesModel {
	return naiveBayesModel{
		wordHashVec: make([]float64, wordVecLen),
	}
}

// TrainFromData loads training data and learns p(w|l;D) by
// gathering word counts, smoothing word counts, and normalizing the counts.
func (nb *naiveBayesModel) TrainFromData(dataDir string) {
	// Load data from the given directory
	files, _ := ioutil.ReadDir(dataDir)
	for _, file := range files {
		lines, err := readLines(path.Join(dataDir, file.Name()))
		if err != nil {
			log.Println("Error loading files from", dataDir, err)
		} else {
			for _, l := range lines {
				tokens := getTokens(l)
				for _, t := range tokens {
					nb.wordHashVec[(hashStrToUint64(t)%wordVecLen)]++
				}
			}
		}
	}

	// Add one smoothing
	nb.addOneSmooth()

	// Normalize the counts
	nb.normalize()
}

// SaveModel writes the model into filename in binary
func (nb *naiveBayesModel) SaveModel(filename string) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, nb.wordHashVec)
	if err == nil {
		ioutil.WriteFile(filename, buf.Bytes(), 0644)
	} else {
		log.Println("Can't save file into", filename)
	}
}

// loadModel loads the model from filename
func (nb *naiveBayesModel) loadModel(filename string) error {
	f, err := fileutil.NewCachedReader(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	buf := bytes.NewReader(contents)
	err = binary.Read(buf, binary.LittleEndian, &nb.wordHashVec)
	if err != nil {
		log.Println("Can't load file from", filename)
	} else {
		log.Println("Loaded model from", filename)
	}
	return err
}

// defaultModel builds a uniform model for p(w|l)
func (nb *naiveBayesModel) defaultModel() {
	// Add one smoothing
	nb.addOneSmooth()

	// Normalize the counts
	nb.normalize()
}

// normalize normalizes the word counts stored in the hash table.
func (nb *naiveBayesModel) normalize() {
	totalWordCount := sum(nb.wordHashVec[:])
	for i := range nb.wordHashVec {
		nb.wordHashVec[i] = math.Log10(nb.wordHashVec[i] / totalWordCount)
	}
}

// addOneSmooth adds one pseudo count to each bucket in the hash table to avoid
// getting p(w|l;D) = 0, which is troublesome when computing the probability
// of a string that contains out-of-vocab words.
func (nb *naiveBayesModel) addOneSmooth() {
	for i := range nb.wordHashVec {
		nb.wordHashVec[i]++
	}
}

// logLikelihood returns p(w|l;D)
func (nb *naiveBayesModel) logLikelihood(w string) float64 {
	return nb.wordHashVec[(hashStrToUint64(w) % wordVecLen)]
}

// LanguageDetector wraps naiveBayeModel because in the future we may want to
// add more models to LanguageDetector. For example, LanguageDetector could
// contain both a logistic regression classifier and a naive Bayes classifier.
type LanguageDetector struct {
	Model naiveBayesModel
}

// NewLanguageDetector returns a pointer to an initialized LanguageDetector.
func NewLanguageDetector() *LanguageDetector {
	return &LanguageDetector{
		Model: newNaiveBayesModel(),
	}
}

// logLikelihood returns log p(editor contents|language)
func (ld *LanguageDetector) logLikelihood(contents string) float64 {
	var score float64
	lines := strings.Split(contents, "\n")
	for _, l := range lines {
		tokens := getTokens(l)
		for _, t := range tokens {
			score += ld.Model.logLikelihood(t)
		}
	}
	return score
}
