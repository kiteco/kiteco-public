package predict

import (
	"sort"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

var (
	errAllPredictionsFilteredMinP   = errors.New("all prediction candidates have been removed by minp filtering")
	errAllPredictionsFilteredPrefix = errors.New("all prediction candidates have been remove by prefix filtering")

	// ErrUnableToReserveSlots ...
	ErrUnableToReserveSlots = errors.New("unable to reserve slots")
)

const (
	predictCacheSize    = 10
	partialRunCacheSize = 10
)

// TFPredictor is a predictor that is powered by GPT 2 model
type TFPredictor struct {
	model *tensorflow.Model

	m sync.Mutex

	encoder *lexicalv0.FileEncoder
	hparams HParams
	search  SearchConfig

	prCache   *lru.Cache
	predCache *lru.Cache
	strict    bool
	useCache  bool
}

// NewTFPredictorFromS3 loads a TFPredictor from a path in S3. It assumes a particular
// naming convention for the model and vocabulary.
func NewTFPredictorFromS3(path string, group lexicalv0.LangGroup) (*TFPredictor, error) {
	modelPath := fileutil.Join(path, "lexical_model.frozen.pb")
	model, err := tensorflow.NewModel(modelPath)
	if err != nil {
		return nil, err
	}

	encoder, params, search, err := LoadModelAssets(path, group)
	if err != nil {
		return nil, err
	}

	predcache, err := lru.New(predictCacheSize)
	if err != nil {
		return nil, err
	}

	prcache, err := lru.NewWithEvict(partialRunCacheSize, func(_ interface{}, obj interface{}) {
		prm := obj.(*PartialRunModel)
		prm.mgr.logf("evicted")
		// We close in a goroutine because Close might wait for active runs to complete before
		// it actually closes the partial run.
		kitectx.Go(func() error {
			return prm.Close()
		})
	})
	if err != nil {
		return nil, err
	}

	return &TFPredictor{
		model:     model,
		encoder:   encoder,
		hparams:   params,
		search:    search,
		predCache: predcache,
		prCache:   prcache,
		useCache:  true,
	}, nil
}

// GetModel ...
func (t *TFPredictor) GetModel() *tensorflow.Model {
	return t.model
}

// SetUseCache enables prm cache use, only set this at initializtion
func (t *TFPredictor) SetUseCache(val bool) {
	t.useCache = val
}

// SetStrictChecking enables stricter checks about the input, search params, etc
func (t *TFPredictor) SetStrictChecking(val bool) {
	t.strict = val
}

// GetHParams implements Predictor
func (t *TFPredictor) GetHParams() HParams {
	return t.hparams
}

// GetEncoder implements Predictor
func (t *TFPredictor) GetEncoder() *lexicalv0.FileEncoder {
	return t.encoder
}

// Unload implements Predictor
func (t *TFPredictor) Unload() {
	t.model.Unload()
	t.prCache.Purge()
	t.predCache.Purge()
}

// GetLexer implements Predictor
func (t *TFPredictor) GetLexer() lexer.Lexer {
	return t.encoder.Lexer
}

// newPredictState ...
func (t *TFPredictor) newPredictState(in Inputs) (*State, error) {
	defer status.NewPredictStateDuration.DeferRecord(time.Now())

	// Check for search config override
	search := selectSearchConfig(t.search, in)

	// This lock wraps cache lookup & insertion, preventing a race between eviction (created by an Add) event
	// and lookup, which prevents looking up a partial run we *just* evicted
	t.m.Lock()
	defer t.m.Unlock()

	langTag := t.encoder.LangTagForPath(in.FilePath)

	context, curatedTokens := buildContextBeforeAndSetPrefix(&in, t.encoder, search)
	if t.useCache {

		var state *State
		keys := t.prCache.Keys()
		for _, key := range keys {
			obj, ok := t.prCache.Peek(key)
			if !ok {
				continue
			}
			currentPRM := obj.(*PartialRunModel)
			slots := currentPRM.mgr.SlotsRemaining()
			if embedded, unembedded, match := currentPRM.MatchAndReserve(context, search.Depth, search, langTag); match {
				currentPRM.mgr.logf("reusing partial run %d with %d -> %d slots, unembedded: %d",
					currentPRM.mgr.id, slots, currentPRM.mgr.SlotsRemaining(), len(unembedded),
				)
				state = newPredictState(context, in.Prefix, in.RandomSeed, search)
				state.EmbeddedContext = embedded
				state.UnembeddedContext = unembedded
				state.prm = currentPRM
				t.prCache.Get(key) // just to update recency status within the lru
				break
			}
		}

		if state != nil && state.prm != nil {
			status.PartialRunReuseRate.Hit()
			state.curatedTokens = curatedTokens
			return state, nil
		}
		status.PartialRunReuseRate.Miss()
	}

	state, err := t.newPartialRunStateLocked(context, in.Prefix, in.RandomSeed, search, langTag)
	if err != nil {
		return nil, err
	}
	state.curatedTokens = curatedTokens
	return state, nil
}

func (t *TFPredictor) newPartialRunStateLocked(context []int64, prefix string, randomSeed int64, search SearchConfig, langTag int) (*State, error) {
	state := newPredictState(context, prefix, randomSeed, search)

	prm, err := NewPartialRunModel(t.model, context, t.hparams, search, t.strict, langTag)
	if err != nil {
		return nil, err
	}

	reserved := prm.mgr.Reserve(search.Depth)
	if !reserved {
		return nil, ErrUnableToReserveSlots
	}

	state.prm = prm

	t.prCache.Add(prm.mgr.id, prm)

	return state, nil
}

// PredictChan ...
func (t *TFPredictor) PredictChan(kctx kitectx.Context, in Inputs) (chan Predicted, chan error) {
	search := selectSearchConfig(t.search, in)
	predsChan := make(chan Predicted, search.Depth*search.BeamWidth)

	errChan := kitectx.Go(func() error {
		state, err := t.newPredictState(in)
		state.incremental = predsChan
		if err != nil {
			close(predsChan)
			return err
		}
		_, err = t.predict(kctx, state, false)
		return err
	})
	return predsChan, errChan
}

// Predict ...
func (t *TFPredictor) Predict(kctx kitectx.Context, in Inputs) (Predictions, error) {
	state, err := t.newPredictState(in)
	if err != nil {
		return Predictions{}, err
	}

	res, err := t.predict(kctx, state, in.IncludeMetaInfo)
	if err != nil {
		return Predictions{}, err
	}

	if in.IncludeMetaInfo {
		res.Meta.Prefix = state.Prefix
		// make sure to use original context otherwise it can be confusing for clients
		res.Meta.ContextBefore = toInt(state.originalContext)
	}

	return res, nil
}

func (t *TFPredictor) predict(kctx kitectx.Context, state *State, includeMetaInfo bool) (Predictions, error) {
	defer func() {
		// Calls to query consume a slot. We always reserve search.Depth slots. If
		// we used fewer slots, return them to the partial run.
		if state.queryCalled < state.Search.Depth {
			state.prm.mgr.Release(state.Search.Depth - state.queryCalled)
		}
	}()

	var exps [][][]int64

	nextLogitsFn := func(kctx kitectx.Context, step int, candidates [][]int64) (res [][]float32, err error) {
		kctx.CheckAbort()

		var contexts [][]int64
		noUnembeddedContext := len(state.UnembeddedContext) == 0
		initStep := step == 0
		switch {
		case initStep && noUnembeddedContext:
			if includeMetaInfo {
				exps = append(exps, [][]int64{state.originalContext})
			}
			return [][]float32{state.prm.initialScores}, nil
		case initStep:
			contexts = [][]int64{state.UnembeddedContext}
		case noUnembeddedContext:
			contexts = candidates
		default:
			contexts = make([][]int64, 0, len(candidates))
			for _, cand := range candidates {
				newCand := copyAndAppend(state.UnembeddedContext, cand)
				contexts = append(contexts, newCand)
			}
		}

		if includeMetaInfo {
			var fullContext [][]int64
			for _, context := range contexts {
				fullContext = append(fullContext, copyAndAppend(state.originalContext, context))
			}
			exps = append(exps, fullContext)
		}

		if t.useCache {
			key := t.predCacheKey(state.EmbeddedContext, contexts)
			preds, ok := t.predCache.Get(key)
			if ok {
				status.PredictionReuseRate.Hit()
				return preds.([][]float32), nil
			}
			status.PredictionReuseRate.Miss()

			defer func() {
				if err == nil {
					t.predCache.Add(key, res)
				}
			}()
		}

		state.queryCalled++
		res, err = state.prm.Query(contexts)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	metricsFn := func(p Predicted) PredictedMetrics {
		return PredictedMetrics{
			CuratedContextExists: len(state.curatedTokens) > 0,
			CuratedContextUsed:   curatedContextUsed(state.curatedTokens, p.TokenIDs),
		}
	}

	results, err := searcher{
		config:          state.Search,
		nextLogits:      nextLogitsFn,
		predictChan:     state.incremental,
		enc:             t.encoder,
		prefix:          state.Prefix,
		rand:            state.Rand,
		strict:          t.strict,
		metricsFn:       metricsFn,
		includeMetaInfo: includeMetaInfo,
	}.Search(kctx)

	if includeMetaInfo {
		for i, exp := range exps {
			results.Meta.Expansions[i].Before = toInt2d(exp)
		}
	}

	return results, err
}

func (t *TFPredictor) predCacheKey(embeddedCtx []int64, contexts [][]int64) string {
	var strs []string
	for _, ctx := range contexts {
		base := toStrings(embeddedCtx)
		base = append(base, toStrings(ctx)...)
		strs = append(strs, strings.Join(base, ","))
	}

	// sort to ensure keys are consistent
	sort.Strings(strs)

	return strings.Join(strs, ":")
}

// Logits returns the logits for the prediction based on the provided context
// return length: vocab
func (t *TFPredictor) Logits(in Inputs) ([]float32, error) {
	search := selectSearchConfig(t.search, in)

	context, _ := buildContextBeforeAndSetPrefix(&in, t.encoder, search)

	langTag := t.encoder.LangTagForPath(in.FilePath)

	logits, err := NewModel(t.model, t.hparams, t.strict).Query([][]int64{context}, true, langTag)
	if err != nil {
		return nil, err
	}

	return logits[0], nil
}
