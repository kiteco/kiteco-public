package tfidf

import (
	"math"
	"regexp"

	"github.com/kiteco/kiteco/kite-golib/text"
)

type tokenizer func(string) text.Tokens

var (
	funcRegexp = regexp.MustCompile(`([a-zA-Z0-9_]+)\([^\(\)]*\)`)
	selRegexp  = regexp.MustCompile(`(?P<x>[a-zA-Z0-9._]*)\.(?P<sel>[a-zA-Z0-9_]*)`)
)

// RawTFIDFs stores the raw tfidf scores
type RawTFIDFs map[string]float64

// RawTFIDF returns the raw TFIDF score of a particular docID
func (s RawTFIDFs) RawTFIDF(docID string) float64 {
	return s[docID]
}

// LogisticTFIDF returns a score between [0, 1] using logistic function
func (s RawTFIDFs) LogisticTFIDF(docID string) float64 {
	var sum float64
	for _, score := range s {
		sum += math.Exp(score)
	}
	return math.Exp(s[docID]) / sum

}

// Standardize standardizes the tfidf scores
func (s RawTFIDFs) Standardize() {
	var sum float64
	var squareSum float64

	for _, score := range s {
		sum += score
		squareSum += score * score
	}

	n := float64(len(s))
	mean := sum / n
	std := math.Sqrt(squareSum/n - mean*mean)

	if std == 0 {
		for docID := range s {
			s[docID] = 0.0
		}
	} else {
		for docID, raw := range s {
			s[docID] = (raw - mean) / std
		}
	}
}

// GaussianizedTFIDF returns a normalized (gaussianized) tfidf score
func (s RawTFIDFs) GaussianizedTFIDF(docID string) float64 {
	var sum float64
	var squareSum float64

	for _, score := range s {
		sum += score
		squareSum += score * score
	}

	n := float64(len(s))
	mean := sum / n
	std := math.Sqrt(squareSum/n - mean*mean)
	if std == 0 {
		return 0.0
	}
	return (s[docID] - mean) / std
}

// Normalize normalizes the raw TFIDF scores.
func (s RawTFIDFs) Normalize() {
	var sum float64
	for _, score := range s {
		sum += score
	}
	if sum == 0 {
		for docID := range s {
			s[docID] = 0
		}
	} else {

		for docID, raw := range s {
			s[docID] = raw / sum
		}
	}

}

// NormalizedTFIDF returns a score between [0, 1] which is proportional to the raw TFIDF scores.
func (s RawTFIDFs) NormalizedTFIDF(docID string) float64 {
	var sum float64
	for _, score := range s {
		sum += score
	}
	if sum == 0 {
		return 0
	}
	return s[docID] / sum

}

// Scorer provides an interface for computing tfidf scores of a corpus
type Scorer struct {
	// Corpus maps from a doc-id (could be package name or method name, for example) to doc content.
	Corpus map[string][]string

	// IdfCounter is responsible for keeping inverse-doc-frequency (idf) weight of each word
	IdfCounter *IDFCounter

	// TfCounters stores the term-frequency weights for each word and each package
	TfCounters map[string]*TFCounter
	ShortType  bool

	// Norm computes the norm of the vector representation of each doc in the tfidf space
	Norm map[string]float64

	processor *text.Processor
	tokenizer tokenizer
}

// SetTextProcessors sets the processor and the tokenizer that should be used in this Scorer
func (s *Scorer) SetTextProcessors(tokenizer tokenizer) {
	s.processor = text.SearchTermProcessor
	s.tokenizer = tokenizer
}

// DocIDs returns the doc ids in the forpus
func (s *Scorer) DocIDs() []string {
	var ids []string
	for id := range s.Corpus {
		ids = append(ids, id)
	}
	return ids
}

// TrainScorer takes a corpus and return a trained Scorer
func TrainScorer(corpus map[string][]string, shortType bool, tokenizer tokenizer) *Scorer {
	s := newScorer(shortType, tokenizer)
	s.Corpus = corpus
	s.ShortType = shortType
	s.train()
	return s
}

// newScorer returns a pointer to a new Scorer object
func newScorer(shortType bool, tokenizer tokenizer) *Scorer {
	return &Scorer{
		Corpus:     make(map[string][]string),
		TfCounters: make(map[string]*TFCounter),
		Norm:       make(map[string]float64),
		ShortType:  shortType,
		processor:  text.SearchTermProcessor,
		tokenizer:  tokenizer,
	}
}

// Train computes the tfidf scores for the corpus
func (s *Scorer) train() {
	idfCorpus := make(map[string]int)

	for id, docs := range s.Corpus {
		docTokens := make([]string, 0, 10000)
		for _, doc := range docs {
			docTokens = append(docTokens, s.tokenizer(doc)...)
		}

		tfCorpus := make(map[string]int)
		for _, tok := range text.TFProcessor.Apply(docTokens) {
			tfCorpus[tok]++
		}
		s.TfCounters[id] = TrainTFCounter(s.ShortType, tfCorpus)

		s.Corpus[id] = s.processor.Apply(docTokens)
		for _, dt := range s.Corpus[id] {
			idfCorpus[dt]++
		}
	}

	s.IdfCounter = TrainIDFCounter(len(s.Corpus), idfCorpus)

	// Compute the norm of this doc
	for id, tfcounter := range s.TfCounters {
		var norm float64
		for t := range tfcounter.Scores {
			w := s.IdfCounter.Weight(t) * tfcounter.Weight(t)
			norm += w * w
		}
		s.Norm[id] = math.Sqrt(norm)
	}
}

// tfidf returns the raw tfidf scores for the input query (queryTokens) and a specific doc in the corpus
func (s *Scorer) tfidf(queryTokens []string, tfCounter *TFCounter) float64 {
	var score float64
	for _, qt := range queryTokens {
		score += s.IdfCounter.Weight(qt) * tfCounter.Weight(qt)
	}
	return score
}

// TFIDFScores returns the tfidf scores (computed against each doc in the corpus)
// given an array of query tokens.
func (s *Scorer) TFIDFScores(queryTokens []string) RawTFIDFs {
	scores := make(RawTFIDFs)
	for docID, tfCounter := range s.TfCounters {
		scores[docID] = s.tfidf(queryTokens, tfCounter)
	}
	return scores
}

// TFIDFScore returns the cosine distance between the query string and
// the doc identified by the given id in the TFIDF space.
func (s *Scorer) TFIDFScore(queryTokens []string, id string) float64 {
	// check whether the doc id exists in the corpus
	if _, exists := s.Corpus[id]; !exists {
		return 0.0
	}

	// get the tfidf length of the query
	tfCorpus := make(map[string]int)
	for _, tok := range text.TFProcessor.Apply(queryTokens) {
		tfCorpus[tok]++
	}

	tfCounter := TrainTFCounter(true, tfCorpus)

	norm := s.ComputeNorm(tfCounter)
	return s.TFIDFScoreWithTFCounter(tfCounter, norm, id)
}

// ComputeNorm computes the norm of the vector (represented by the given TFCounter)
// in the TFIDF space.
func (s *Scorer) ComputeNorm(tfCounter *TFCounter) float64 {
	var norm float64
	for q := range tfCounter.Scores {
		w := s.IdfCounter.Weight(q) * tfCounter.Weight(q)
		norm += w * w
	}
	return math.Sqrt(norm)
}

// TFIDFScoreWithTFCounter computes the cosine distance between the query and the doc
// by using a precomputed query tfcounter.
func (s *Scorer) TFIDFScoreWithTFCounter(queryCounter *TFCounter, queryNorm float64, id string) float64 {
	// check whether the doc id exists in the corpus
	var docTokens []string
	var exists bool
	if docTokens, exists = s.Corpus[id]; !exists {
		return 0.0
	}

	if queryNorm == 0 || s.Norm[id] == 0 {
		return 0.0
	}

	var score float64
	for qt := range queryCounter.Scores {
		for _, dt := range docTokens {
			if qt == dt {
				qScore := s.IdfCounter.Weight(qt) * queryCounter.Weight(qt)
				dScore := s.IdfCounter.Weight(qt) * s.TfCounters[id].Weight(qt)
				score += qScore * dScore
				break
			}
		}
	}
	return score / (queryNorm * s.Norm[id])
}
