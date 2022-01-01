package tfidf

import (
	"log"
	"math"

	"github.com/kiteco/kiteco/kite-golib/text"
)

// TermCounter defines the functions that a termcounter must implement.
type TermCounter interface {
	Weight(w string) float64
}

// IDFCounter keeps track on the number of docs that contain a certain word in a corpus.
// IDFCounter is used to compute the inverse-document-frequency (idf) score for words.
type IDFCounter struct {
	CorpusSize int
	Scores     map[string]float64
}

// TrainIDFCounter takes in n (the size of the corpus), and counts (the number of
// docs that contain a certain word, and compute the log idf value.
// We use TrainIDFCounter to make IDFCounter a stateless struct.
func TrainIDFCounter(n int, counts map[string]int) *IDFCounter {
	scores := make(map[string]float64)
	for w, count := range counts {
		scores[w] = math.Log10(float64(n) / float64(count))
	}
	c := &IDFCounter{
		CorpusSize: n,
		Scores:     scores,
	}
	return c
}

// TrainIDFCounterFromDocs takes in raw documents, a tokenizer, and a processor
// and builds a IDFCounter based on the data.
func TrainIDFCounterFromDocs(docs []string, tokenizer tokenizer, processor *text.Processor) *IDFCounter {
	corpus := make(map[string]int)
	for _, doc := range docs {
		for _, tok := range processor.Apply(tokenizer(doc)) {
			corpus[tok]++
		}
	}
	return TrainIDFCounter(len(docs), corpus)
}

// Weight returns the idf score of the given token computed.
func (c *IDFCounter) Weight(token string) float64 {
	if c.CorpusSize == 0 {
		log.Println("corpus size is 0. Setting all weight to be 1.0")
		return 1
	}
	if score, exists := c.Scores[token]; exists {
		return score
	}
	return 0
}

// TFCounter keeps track on the number of times a word appears in a doc.
type TFCounter struct {
	Scores    map[string]float64
	MaxCount  int
	Total     int
	ShortType bool
}

// TrainTFCounter takes in the doc type (short doc or not) and word counts in this doc
// and create an object of TFCounter. We use TrainTFCounter to make TFCounter stateless.
func TrainTFCounter(shortType bool, counts map[string]int) *TFCounter {
	var total int
	var max int
	for _, c := range counts {
		if shortType {
			if c > max {
				max = c
			}
		} else {
			total += c
		}
	}
	scores := make(map[string]float64)
	if shortType {
		for t, c := range counts {
			scores[t] = 0.5 + 0.5*float64(c)/float64(max)
		}
	} else {
		for t, c := range counts {
			scores[t] = float64(c) / float64(total)
		}
	}

	return &TFCounter{
		Scores:    scores,
		MaxCount:  max,
		Total:     total,
		ShortType: shortType,
	}
}

// Print prints out the normal (specified by the `false` arg) tf-weight of each word in a doc.
func (c *TFCounter) Print() {
	for w := range c.Scores {
		log.Println(w, c.Weight(w))
	}
}

// Weight returns the tf-weight of a word based on word counts of a document.
// The normal tf score is defined as follows.
// tf(w) = freq(w) / docSize, where docSize is the total number of tokens in the document.
// Note that for short documents, a more accepted definition of tf is defined as follows.
// tf(w) = 0.5 + 0.5 * freq(w) / maxCount, where maxCount is the number of times that the
// most frequent word appears in the document. We allow users to use this definition
// by setting shortType = true.
func (c *TFCounter) Weight(token string) float64 {
	if c.ShortType {
		if c.MaxCount == 0 {
			log.Println("corpus size is 0. Setting all weight to be 0.0")
			return 0
		}
		if score, found := c.Scores[token]; found {
			return score
		}
		return 0.5
	}
	return c.Scores[token]
}
