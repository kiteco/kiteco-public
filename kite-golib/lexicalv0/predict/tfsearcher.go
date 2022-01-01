package predict

import (
	"fmt"
	"log"
	"path"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// TFSearcher is a predictor that is powered by GPT 2 model
type TFSearcher struct {
	hparams        HParams
	config         SearchConfig
	model          *tensorflow.Model
	encoder        *lexicalv0.FileEncoder
	strict         bool
	debug          bool
	prefixSuffixLM bool
}

// NewTFSearcherFromS3 loads a TFSearcher from a path in S3. It assumes a particular
// naming convention for the model and vocabulary.
func NewTFSearcherFromS3(path string, group lexicalv0.LangGroup) (*TFSearcher, error) {
	modelPath := fileutil.Join(path, "lexical_model.frozen.pb")
	model, err := tensorflow.NewModel(modelPath)
	if err != nil {
		return nil, err
	}

	encoder, params, config, err := LoadModelAssets(path, group)
	if err != nil {
		return nil, err
	}

	return &TFSearcher{
		hparams:        params,
		config:         config,
		model:          model,
		encoder:        encoder,
		prefixSuffixLM: params.ModelType == ModelTypePrefixSuffix,
		//debug:     true,
	}, nil
}

// SetStrictChecking enables stricter checks about the input, search params, etc
func (t *TFSearcher) SetStrictChecking(val bool) {
	t.strict = val
}

// GetHParams implements Predictor
func (t *TFSearcher) GetHParams() HParams {
	return t.hparams
}

// GetEncoder implements Predictor
func (t *TFSearcher) GetEncoder() *lexicalv0.FileEncoder {
	return t.encoder
}

// Unload implements Predictor
func (t *TFSearcher) Unload() {
	t.model.Unload()
}

// GetLexer implements Predictor
func (t *TFSearcher) GetLexer() lexer.Lexer {
	return t.encoder.Lexer
}

type searcherInputs struct {
	Prefix        string
	ContextBefore []int64
	ContextAfter  []int64
	Incremental   chan Predicted
	Config        SearchConfig
}

func newSearcherInputs(in Inputs, enc *lexicalv0.FileEncoder, config SearchConfig, prefixSuffixLM bool) (searcherInputs, error) {
	config = selectSearchConfig(config, in)

	context, _ := buildContextBeforeAndSetPrefix(&in, enc, config)

	searchIn := searcherInputs{
		Prefix:        in.Prefix,
		ContextBefore: context,
		Config:        config,
	}

	if prefixSuffixLM {
		searchIn.ContextAfter = buildContextAfter(in, enc, config)
		if len(searchIn.ContextAfter) == 0 && len(searchIn.ContextBefore) == 0 {
			return searcherInputs{}, errors.New("before and after context empty")
		}
	}

	return searchIn, nil
}

// GetModel ...
func (t *TFSearcher) GetModel() *tensorflow.Model {
	return t.model
}

// PredictChan implements lexicalmodels.ModelBase
func (t *TFSearcher) PredictChan(kctx kitectx.Context, in Inputs) (chan Predicted, chan error) {
	searchIn, err := newSearcherInputs(in, t.encoder, t.config, t.prefixSuffixLM)
	if err != nil {
		return handlePredictChanInitErr(err)
	}

	searchIn.Incremental = make(chan Predicted, searchIn.Config.Depth*searchIn.Config.BeamWidth)
	errChan := kitectx.Go(func() error {
		_, err := t.search(kctx, searchIn)
		return err
	})
	return searchIn.Incremental, errChan
}

// Predict implements performance.Model
func (t *TFSearcher) Predict(kctx kitectx.Context, in Inputs) (Predictions, error) {
	searchIn, err := newSearcherInputs(in, t.encoder, t.config, t.prefixSuffixLM)
	if err != nil {
		return Predictions{}, err
	}

	preds, err := t.search(kctx, searchIn)
	if err != nil {
		return Predictions{}, err
	}

	var filtered []Predicted
	for _, p := range preds {
		if len(p.TokenIDs) == searchIn.Config.Depth {
			filtered = append(filtered, p)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Prob > filtered[j].Prob
	})

	return Predictions{
		Preds: filtered,
		// TODO: we can add other meta info if we decide to use the
		// inspector with this model...
	}, nil
}

// Logits returns the logits for the prediction based on the provided context
// return length: vocab
func (t *TFSearcher) Logits(in Inputs) ([]float32, error) {
	search := selectSearchConfig(t.config, in)

	contextBefore, _ := buildContextBeforeAndSetPrefix(&in, t.encoder, search)

	if t.prefixSuffixLM {
		contextAfter := buildContextAfter(in, t.encoder, search)

		psIn := PrefixSuffixInputs{
			Before: contextBefore,
			After:  contextAfter,
		}

		logits, err := NewPrefixSuffixModel(t.model, t.hparams, t.strict).Query(psIn, true)
		if err != nil {
			return nil, err
		}

		return logits[0], nil
	}

	langTag := t.encoder.LangTagForPath(in.FilePath)

	logits, err := NewModel(t.model, t.hparams, t.strict).Query([][]int64{contextBefore}, true, langTag)
	if err != nil {
		return nil, err
	}

	return logits[0], nil
}

// --

func (t *TFSearcher) search(kctx kitectx.Context, in searcherInputs) ([]Predicted, error) {
	defer status.SearchDuration.DeferRecord(time.Now())

	if in.Incremental != nil {
		defer close(in.Incremental)
	}

	if t.strict {
		if err := t.validateContext(in.ContextBefore); err != nil {
			return nil, err
		}
		if t.prefixSuffixLM {
			if err := t.validateContext(in.ContextAfter); err != nil {
				return nil, err
			}
		}
	}

	var validPrefixIDs []int64
	if in.Prefix != "" {
		validPrefixIDs = idsMatchingPrefixSlice(t.encoder, in.Prefix)
	}

	feeds := searchPlaceholders(in.Config, validPrefixIDs, t.encoder.NumLexical(), t.hparams.VocabSize, t.prefixSuffixLM)
	resultsOp, probsOp := searchFetchOps()

	if t.prefixSuffixLM {
		before, _ := PadContext(in.ContextBefore, in.Config.Window, -1)
		after, _ := PadContext(in.ContextAfter, in.Config.Window, -1)
		in := PrefixSuffixInputs{
			Before: before,
			After:  after,
		}

		for k, v := range in.Feed(-1) {
			feeds[k] = v
		}
	} else {
		context, contextMask := PadContext(in.ContextBefore, in.Config.Window, 0)
		feeds[contextPlacholderOpName] = [][]int64{context}
		feeds[contextMaskPlaceholderOpName] = [][]int64{contextMask}
	}

	res, err := t.model.Run(feeds, []string{resultsOp, probsOp})
	if err != nil {
		return nil, errors.Errorf("model error: %v", err)
	}

	resultsBatched, probsBatched := res[resultsOp].([][][]int64), res[probsOp].([][][]float32)
	results, probs := resultsBatched[0], probsBatched[0]

	predicted, err := buildPredicted(results, probs, in.Config.MinP, in.Prefix, t.encoder)
	if err != nil {
		return nil, err
	}

	if in.Incremental != nil {
		for _, pred := range predicted {
			in.Incremental <- pred
		}
	}

	return predicted, err
}

func (t *TFSearcher) validateContext(context []int64) error {
	return validate([][]int64{context}, t.hparams)
}

func (t *TFSearcher) logf(msg string, args ...interface{}) {
	if t.debug {
		prefix := fmt.Sprintf("[tfsearcher] ")
		log.Printf(prefix+msg, args...)
	}
}

// --

func searchPlaceholders(config SearchConfig, validPrefixIDs []int64, numLexical, vocabSize int, prefixSuffixLM bool) map[string]interface{} {
	name := func(parts ...string) string {
		if prefixSuffixLM {
			parts = append([]string{
				"search",
				"placeholders",
			}, parts...)
		} else {
			parts = append([]string{
				"search",
				"search",
				"placeholders",
			}, parts...)
		}
		return path.Join(parts...)
	}

	prefixMask := PrefixIDMask(validPrefixIDs, vocabSize)

	d := map[string]interface{}{
		//name("minp"):                    config.MinP, // this gets pruned from the graph b/c its currently unused
		name("topk"):                    int64(config.TopK),
		name("width"):                   int64(config.BeamWidth),
		name("valid_prefix_ids"):        [][]int64{prefixMask},
		name("inv_ident_temperature"):   invertScaling(config.IdentTemperature),
		name("inv_lexical_temperature"): invertScaling(config.LexicalTemperature),
		name("num_lexical_tokens"):      int64(numLexical),
	}

	return d
}

func searchFetchOps() (string, string) {
	return "search/search/results", "search/search/probs"
}
