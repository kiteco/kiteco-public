package languagemodel

import (
	"math"

	spooky "github.com/dgryski/go-spooky"
)

// UnigramLanguageModel uses a fixed-length of vector to keep counts of words
// and trains a unigram language model using the counts.
// Words are first hashed into ids, and in this implementation, words are identified
// by their ids. This implementation allows comparison between language models that are
// trained on different datasets. Otherwise, we'll have to first figure out the global
// vocabulary by going through all the datasets.
// Note that there will be collisions in this implementation.
type UnigramLanguageModel struct {
	WordHashVec map[uint64]float64
	WordVecLen  uint64
	Unknown     float64
}

// TrainUnigramLanguageModel takes the training data and the given wordVecLen
// to train a UnigramLanguageModel, and returns a pointer to it.
func TrainUnigramLanguageModel(tokens []string, wordVecLen uint64) *UnigramLanguageModel {
	alpha := 1 / float64(wordVecLen)
	words := make(map[uint64]float64)
	for _, t := range tokens {
		id := spooky.Hash64([]byte(t))
		words[id%wordVecLen]++
	}
	total := 0.0
	for i := range words {
		total += words[i]
		words[i] += alpha
	}
	total = math.Log(total + 1)
	for i := range words {
		words[i] = math.Log(words[i]) - total
	}
	return &UnigramLanguageModel{
		WordHashVec: words,
		WordVecLen:  wordVecLen,
		Unknown:     math.Log(alpha) - total,
	}
}

// LogLikelihood returns the log likelihood of an array of words, i.e,
// p(W|model) = \prod p(w_1|model) p(w_2|model) ...
func (lm *UnigramLanguageModel) LogLikelihood(ws []string) float64 {
	var score float64
	for _, w := range ws {
		id := spooky.Hash64([]byte(w))
		if prob, exists := lm.WordHashVec[id%lm.WordVecLen]; exists {
			score += prob
		} else {
			score += lm.Unknown
		}
	}
	return score
}
