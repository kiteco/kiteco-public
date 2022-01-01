package tfidf

import (
	"math"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/text"
	"github.com/stretchr/testify/assert"
)

func TestIDFCounter(t *testing.T) {
	doc1 := "hello worlld"
	doc2 := "golang or c++, which one is better?"
	doc3 := "kiteman is going to save the worlld."

	idfCorpus := make(map[string]int)
	for _, dt := range text.SearchTermProcessor.Apply(text.TokenizeWithoutCamelPhrases(doc1)) {
		idfCorpus[dt]++
	}

	for _, dt := range text.SearchTermProcessor.Apply(text.TokenizeWithoutCamelPhrases(doc2)) {
		idfCorpus[dt]++
	}

	for _, dt := range text.SearchTermProcessor.Apply(text.TokenizeWithoutCamelPhrases(doc3)) {
		idfCorpus[dt]++
	}

	idfCounter := TrainIDFCounter(3, idfCorpus)

	exp := math.Log10(3.0 / 2.0)
	act := idfCounter.Weight("worlld")
	assert.Equal(t, exp, act)

	// idfCounter.Weight("is") should return 0 because "is"
	// should be skipped.
	act = idfCounter.Weight("is")
	exp = 0
	assert.Equal(t, exp, act)
}

func TestTFCounter(t *testing.T) {
	processer := text.NewProcessor(text.Lower, text.RemoveStopWords, text.Stem)

	corpus := make(map[string]int)
	doc := "kiteman is going to save the worlld."
	for _, t := range processer.Apply(text.TokenizeWithoutCamelPhrases(doc)) {
		corpus[t]++
	}

	tfCounter := TrainTFCounter(true, corpus)

	// short form tf score
	exp := 1.0
	act := tfCounter.Weight("save")
	assert.Equal(t, exp, act)

	tfCounter = TrainTFCounter(false, corpus)
	// long form tf score
	exp = 0.25
	act = tfCounter.Weight("save")
	assert.Equal(t, exp, act)
}
