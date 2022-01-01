package lexicalproviders

import (
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// LexicalMeta stores information about lexical tokens
type LexicalMeta struct {
	DebugStr string
}

// LexicalMetrics stores information about lexical metrics
type LexicalMetrics struct {
	Score           float64 `json:"score"`
	Probability     float32 `json:"probability"`
	NumVocabTokens  int     `json:"num_vocab_tokens"`
	ModelDurationMS int64   `json:"model_duration_ms"`
	NumNewlines     int     `json:"num_newlines"`

	// TODO: should we move everything into this struct?
	predict.PredictedMetrics
}

func newLexicalMetrics(pred predict.Predicted, c data.Completion, score float64, modelDuration time.Duration) *LexicalMetrics {
	return &LexicalMetrics{
		Score:            score,
		Probability:      pred.Prob,
		NumVocabTokens:   len(pred.TokenIDs),
		ModelDurationMS:  int64(modelDuration) / int64(time.Millisecond),
		NumNewlines:      strings.Count(c.Snippet.Text, "\n"),
		PredictedMetrics: pred.Metrics,
	}
}

// MetaCompletion pairs a completion with metadata used for rendering and/or mixing
type MetaCompletion struct {
	data.Completion

	LexicalMeta
	Provider          data.ProviderName
	Source            response.EditorCompletionSource
	Score             float64
	ExperimentalScore float64

	Metrics interface{}

	FromSmartProvider bool
	IsServer          bool
}
