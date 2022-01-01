package languagemodel

import (
	"fmt"
	"math"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/text"
)

type tokenizer func(string) text.Tokens

const (
	wordVecLen = 1001
	alpha      = 0.01
)

// Scorer provides an interface for computing p(T|q) ~ p(q|T) * p(T),
// where p(T) is the prior of observing class T, and p(q|T) is the probability of q being
// an instance of class T.
type Scorer struct {
	Prior          map[string]float64
	LanguageModels map[string]*LanguageModel
	tokenizer      tokenizer
}

// SetTextProcessors sets up text processors used in the language models in a lm scorer
func (lms *Scorer) SetTextProcessors() {
	for _, lm := range lms.LanguageModels {
		lm.setTextProcessors()
	}
}

// TrainScorer takes in docs and the classes that each doc belongs to.
// In the use case, in which the training data consists of code example titles, the classes of the
// titles would be the functions that are in the code example.
func TrainScorer(docs []string, classes [][]string, tokenizer tokenizer) (*Scorer, error) {
	if len(docs) != len(classes) {
		return nil, fmt.Errorf("len of docs != len of classes")
	}
	lms := newScorer(tokenizer)
	for i, doc := range docs {
		lms.addData(doc, classes[i])
	}
	lms.train()
	return lms, nil
}

// TrainScorerFromMap takes in a map from the class of the doc to the doc itself.
func TrainScorerFromMap(corpus map[string][]string, tokenizer tokenizer) (*Scorer, error) {
	lms := newScorer(tokenizer)
	for class, docs := range corpus {
		lms.addDataSet(docs, class)
	}
	lms.train()
	return lms, nil
}

// newScorer returns a pointer to a new Scorer object
func newScorer(tokenizer tokenizer) *Scorer {
	return &Scorer{
		Prior:          make(map[string]float64),
		LanguageModels: make(map[string]*LanguageModel),
		tokenizer:      tokenizer,
	}
}

// addDataSet add a set of strings at once for each class
func (lms *Scorer) addDataSet(docs []string, class string) {
	// increate the count for the method
	lms.Prior[class]++

	var lm *LanguageModel
	var exists bool

	if lm, exists = lms.LanguageModels[class]; !exists {
		lm = NewLanguageModel(lms.tokenizer)
		lms.LanguageModels[class] = lm
	}
	lm.addDataSet(docs)
}

// addData adds training data docs to each class
func (lms *Scorer) addData(doc string, classes []string) {
	for _, class := range classes {
		// increate the count for the method
		lms.Prior[class]++

		var lm *LanguageModel
		var exists bool

		if lm, exists = lms.LanguageModels[class]; !exists {
			lm = NewLanguageModel(lms.tokenizer)
			lms.LanguageModels[class] = lm
		}
		lm.addData(doc)
	}
}

// train Computes the prior probability and train the language model
func (lms *Scorer) train() {
	// compute the prior distribution
	var sum float64
	for _, c := range lms.Prior {
		sum += c
	}
	sum = math.Log(sum)
	for m, lm := range lms.LanguageModels {
		lms.Prior[m] = math.Log(lms.Prior[m]) - sum
		lm.train()
	}
}

// Posterior returns the posterior probability of p(T|q) for each class given q.
func (lms *Scorer) Posterior(tokens []string) map[string]float64 {
	var justScores []float64
	scores := make(map[string]float64)
	for m, prior := range lms.Prior {
		scores[m] = prior + lms.LanguageModels[m].LogLikelihood(tokens)
		justScores = append(justScores, scores[m])
	}
	logSum := logSumExp(justScores)
	for i, s := range scores {
		scores[i] = math.Exp(s - logSum)
	}
	return scores
}

// OneMax normalizes the score by setting the highest probability to 1.
func (lms *Scorer) OneMax(tokens []string) map[string]float64 {
	max := math.Inf(-1)
	scores := make(map[string]float64)
	for m, prior := range lms.Prior {
		scores[m] = prior + lms.LanguageModels[m].LogLikelihood(tokens)
		if scores[m] > max {
			max = scores[m]
		}
	}
	for i, s := range scores {
		scores[i] = math.Exp(s - max)
	}
	return scores
}

// LanguageModel is an unigram language model
type LanguageModel struct {
	WordHashVec [wordVecLen]float64
	processor   *text.Processor
	tokenizer   tokenizer
	lookup      map[string]uint64
}

// NewLanguageModel returns a pointer to a new LanguageModel object
func NewLanguageModel(tokenizer tokenizer) *LanguageModel {
	return &LanguageModel{
		processor: text.NewProcessor(text.Lower, text.RemoveStopWords, text.Stem),
		tokenizer: tokenizer,
		lookup:    make(map[string]uint64),
	}
}

func (lm *LanguageModel) addDataSet(docs []string) {
	for _, doc := range docs {
		lm.addData(doc)
	}
}

func (lm *LanguageModel) addData(doc string) {
	tokens := lm.processor.Apply(lm.tokenizer(doc))

	for _, t := range tokens {
		id, found := lm.lookup[t]
		if !found {
			id = spooky.Hash64([]byte(t))
			lm.lookup[t] = id
		}
		lm.WordHashVec[id%wordVecLen]++
	}
}

func (lm *LanguageModel) alphaSmooth() {
	for i := range lm.WordHashVec {
		lm.WordHashVec[i] += alpha
	}
}

func (lm *LanguageModel) normalize() {
	logTotalWordCount := math.Log(sum(lm.WordHashVec[:]))
	for i := range lm.WordHashVec {
		lm.WordHashVec[i] = math.Log(lm.WordHashVec[i]) - logTotalWordCount
	}
}

// Train smooths the word counts and normalizes the word counts.
func (lm *LanguageModel) train() {
	lm.alphaSmooth()
	lm.normalize()
}

// LogLikelihood returns the log likelihood of an array of words, i.e,
// p(W|model) = \prod p(w_1|model) p(w_2|model) ...
func (lm *LanguageModel) LogLikelihood(ws []string) float64 {
	var score float64
	for _, w := range ws {
		id := spooky.Hash64([]byte(w))
		score += lm.WordHashVec[id%wordVecLen]
	}
	return score
}

func (lm *LanguageModel) setTextProcessors() {
	lm.processor = text.NewProcessor(text.Lower, text.RemoveStopWords, text.Stem)
}
