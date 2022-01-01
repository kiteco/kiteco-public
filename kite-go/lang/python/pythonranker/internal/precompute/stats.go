package precompute

import (
	"strings"

	"github.com/kiteco/kiteco/kite-golib/text"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

var (
	noStemProcessor = text.NewProcessor(text.Lower, text.RemoveStopWords, text.Uniquify)
)

// QueryStats contains precomputed query stats.
type QueryStats struct {
	Raw               string
	StemmedTokens     []string
	Tokens            []string
	UnmatchedTokens   []string
	TFCounter         *tfidf.TFCounter
	TFIDFNorm         float64
	Package           string
	StemmedToOriginal map[string][]string
}

// NewQueryStats returns a pointer to a new QueryStats object.
func NewQueryStats(raw, packageName string) *QueryStats {
	packageName = strings.ToLower(packageName)
	tokens := text.RemoveStopWords(text.Lower(text.TokenizeNoCamel(raw)))

	stemmedToOriginal := make(map[string][]string)
	for _, t := range tokens {
		stemmed := text.Stem([]string{t})[0]
		stemmedToOriginal[stemmed] = append(stemmedToOriginal[stemmed], t)
	}

	return &QueryStats{
		Raw:               raw,
		Tokens:            tokens,
		UnmatchedTokens:   tokens,
		StemmedTokens:     text.TFProcessor.Apply(text.TokenizeNoCamel(raw)),
		Package:           strings.ToLower(packageName),
		StemmedToOriginal: stemmedToOriginal,
	}
}

// Reset resets the unmatched tokens, which are modified during the feature generation process.
func (qs *QueryStats) Reset() {
	qs.UnmatchedTokens = qs.Tokens
}

// TargetStats keeps precomputed stats for the targets (i.e., methods in this case).
type TargetStats struct {
	Candidates         []string
	LMScores           map[string]map[string]float64
	WordToMethodScores map[string]float64
}

// NewTargetStats returns a pointer to a new TargetStats object.
func NewTargetStats(candidates []string) *TargetStats {
	return &TargetStats{
		Candidates: candidates,
		LMScores:   make(map[string]map[string]float64),
	}
}
