package predict

import (
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// PredictionModel duplicates lexicalmodels.Model
type PredictionModel interface {
	// PredictChan returns an asynchronous stream of predictions, as well as an asynchronous error.
	// Neither channel must be drained (the Model correctly buffers the channel).
	// The error channel always yields exactly one error (or nil), and will not be closed. Read it at most once.
	// The predictions channel can yield an arbitrary number of predictions, and will be closed.
	PredictChan(kitectx.Context, Inputs) (chan Predicted, chan error)
	GetLexer() lexer.Lexer
	Unload()

	// NOTE: this should only be used for evaluation binaries
	GetEncoder() *lexicalv0.FileEncoder
}

// Predictor is an interface that specifies the required API for a lexical model to be used
// by the lexical completion engine as well as other lexical pipelines (e.g minp, temp scaling, etc)
type Predictor interface {
	PredictionModel
	SetStrictChecking(bool)
	Predict(kitectx.Context, Inputs) (Predictions, error)
	GetHParams() HParams
	GetModel() *tensorflow.Model
	Logits(Inputs) ([]float32, error)
}

// NewPredictor ...
func NewPredictor(path string, group lexicalv0.LangGroup) (Predictor, error) {
	paramsPath := fileutil.Join(path, "config.json")
	params, err := NewHParams(paramsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to load hyperparameters")
	}

	switch params.ModelType {
	case ModelTypeLexical:
		return NewTFPredictorFromS3(path, group)
	case ModelTypePrefixSuffix:
		return NewPrefixSuffixPredictorFromS3(path, group)
	default:
		return nil, errors.New("unsupported model type %s", params.ModelType)
	}
}

// PredictedMetrics tracks metrics for a prediction
type PredictedMetrics struct {
	CuratedContextExists bool `json:"curated_context_exists"`
	CuratedContextUsed   bool `json:"curated_context_used"`
}

// Predicted represents a series of predictions (of any length)
type Predicted struct {
	TokenIDs []int

	Prob float32

	IsRemote bool

	Metrics PredictedMetrics

	// Everything below is meta information
	// that is used to help render / filter the predictions
	// when we use these predictions for compeltions.
	// TODO: better spot for these?

	// Prefix used for the prediction
	Prefix string

	// the decoded tokens that we predicted
	Tokens []lexer.Token

	// EndsWithIncompleteTok is true if the predicted tokens ends with an
	// incomplete token
	// TODO: nasty, but also weird that clients should care about the Encoder that is used
	// with the model ...
	EndsWithIncompleteTok bool
}

func newPredicted(tokenIDs []int64, prob float32, prefix string, enc *lexicalv0.FileEncoder, isRemote bool) Predicted {
	tids := toInt(tokenIDs)

	tokens := enc.Decode(tids)

	var endsWithIncompleteTok bool
	if len(tids) > 0 {
		last := tids[len(tids)-1]
		if !enc.IsLexical(last) && enc.Lexer.IsIncompleteToken(enc.IDToString[last]) {
			endsWithIncompleteTok = true
		}
	}

	return Predicted{
		TokenIDs:              tids,
		Prob:                  prob,
		Prefix:                prefix,
		Tokens:                tokens,
		EndsWithIncompleteTok: endsWithIncompleteTok,
		IsRemote:              isRemote,
	}
}

// Expansion ...
type Expansion struct {
	AtDepth             int
	Input               []Predicted
	RawPredictions      [][]Predicted
	SelectedPredictions [][]Predicted
	BeamPredictions     []Predicted
	Before              [][]int
	After               [][]int
	Predict             [][]int
}

// PredictionsMeta stores extra meta information for a set of predictions,
// typically used for debugging or training.
type PredictionsMeta struct {
	Prefix         string
	ContextBefore  []int
	ContextAfter   []int
	ContextPredict []int
	Expansions     []Expansion
}

// Predictions ...
type Predictions struct {
	Preds []Predicted

	// Meta information for the prediction set,
	// only populated if `Inputs.IncludeMetaInfo == true`
	Meta PredictionsMeta
}

// EditorEvent ...
type EditorEvent struct {
	Tokens []lexer.Token
}

// EditorEvents ...
type EditorEvents []EditorEvent

// Inputs ...
type Inputs struct {
	// FilePath is required
	FilePath string

	// Tokens to encode (typically the full file)
	// NOTE: this field is required.
	Tokens []lexer.Token

	// CursorTokenIdx is the index of the token that contains the cursor.
	// NOTE:
	//   - DO NOT MODIFY this field since it is also used to render completions
	//     see lexical/lexicalcomplete/lexicalproviders/Data_inputs.go
	//   - This field is required.
	//   - The cursor token is _not_ included in the context, unless a prefix is
	//     set, see context.go/buildLeftContextAndSetPrefix.
	CursorTokenIdx int

	// NOTE: this field is required for the prefix_suffix model
	Buffer data.SelectedBuffer

	// Prefix suggested by clients to use for prediction, the model may
	// use a subset of this prefix so client's should check Predicted.Prefix for the actual
	// value. If this value is empty then the model will not use _any_ prefix.
	Prefix string

	RandomSeed int64

	// IncludeMetaInfo in the `Predictions` response, this can be slow
	// so only use this for debugging
	IncludeMetaInfo bool

	// AllowContextAfterCursor to be used for prediction, unfortunately we need a flag for this
	// because we need to update minp/calibration/performance binaries to handle this properly
	AllowContextAfterCursor bool

	// Sorted from least recent to most recent (e.g oldest to newest)
	Events EditorEvents

	// SearchConfig used to override default if set; default is empty -- primarily used for various
	// model evaluation tasks
	SearchConfig SearchConfig
}
