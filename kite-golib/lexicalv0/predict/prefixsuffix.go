package predict

import (
	"fmt"
	"math/rand"
	"path"
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

type prefixSuffixFetchOpsT struct {
	// Useful for getting logits or preds for a single prediction,
	// only allowed for the "CPU searcher"
	FirstLogits string
	FirstPreds  string
}

var prefixSuffixFetchOps = prefixSuffixFetchOpsT{
	FirstLogits: "test/predict_init/logits",
	FirstPreds:  "test/predict_init/preds",
}

func (p prefixSuffixFetchOpsT) predOp(slot int, getLogits bool) string {
	if slot == -1 {
		if getLogits {
			return p.FirstLogits
		}
		return p.FirstPreds
	}

	prefix := fmt.Sprintf("test/prediction%d/", slot)
	if getLogits {
		return prefix + "logits"
	}
	return prefix + "preds"
}

// PrefixSuffixInputs ...
type PrefixSuffixInputs struct {
	Before  []int64
	After   []int64
	Predict [][]int64
	Empty   bool
}

func (in PrefixSuffixInputs) batchSize() int {
	return len(in.Predict)
}

func (in PrefixSuffixInputs) numTokensBefore() int {
	return len(in.Before)
}

func (in PrefixSuffixInputs) numTokensAfter() int {
	return len(in.After)
}

func (in PrefixSuffixInputs) numTokensPredict() int {
	if len(in.Predict) > 0 {
		return len(in.Predict[0])
	}
	return 0
}

func (in PrefixSuffixInputs) fullName(slot int, name string) string {
	if slot == -1 {
		return path.Join("placeholders", name)
	}
	return path.Join("test", fmt.Sprintf("prediction%d", slot), "placeholders", name)
}

// FeedNames ...
func (in PrefixSuffixInputs) FeedNames(slot int) []string {
	names := []string{
		in.fullName(slot, "context_predict"),
	}

	if slot == -1 {
		names = append(names, in.fullName(slot, "context_before"))
		names = append(names, in.fullName(slot, "context_after"))
	} else {
		names = append(names, in.fullName(slot, "empty"))
	}
	return names
}

// Feed ...
func (in PrefixSuffixInputs) Feed(slot int) map[string]interface{} {
	emptyVal := int64(0)
	if in.Empty {
		emptyVal = 1
	}

	fd := map[string]interface{}{
		in.fullName(slot, "context_predict"): in.Predict,
	}

	if slot == -1 {
		fd[in.fullName(slot, "context_before")] = [][]int64{in.Before}
		fd[in.fullName(slot, "context_after")] = [][]int64{in.After}
	} else {
		fd[in.fullName(slot, "empty")] = emptyVal
	}
	return fd
}

// Validate inputs
func (in PrefixSuffixInputs) Validate(hp HParams) error {
	if in.Empty {
		return nil
	}

	if err := validate([][]int64{in.Before}, hp); err != nil {
		return errors.Wrapf(err, "error validating before context")
	}

	if err := validate([][]int64{in.After}, hp); err != nil {
		return errors.Wrapf(err, "error validating after context")
	}

	if err := validate(in.Predict, hp); err != nil {
		return errors.Wrapf(err, "error validating predict context")
	}

	b := len(in.Before)
	a := len(in.After)
	p := len(in.Predict)

	if a == 0 && b == 0 && p == 0 {
		return errors.New("no context provided: before, after, and predict all empty")
	}

	return nil
}

// PrefixSuffixPredictor is a predictor that is powered by a prefix suffix model.
type PrefixSuffixPredictor struct {
	hparams HParams
	search  SearchConfig
	model   *tensorflow.Model
	encoder *lexicalv0.FileEncoder

	m         sync.Mutex
	prCache   *lru.Cache
	predCache *lru.Cache
	strict    bool
	useCache  bool
}

// NewPrefixSuffixPredictorFromS3 loads a PrefixSuffixPredictor from a path in S3. It assumes a particular
// naming convention for the model and vocabulary.
func NewPrefixSuffixPredictorFromS3(path string, group lexicalv0.LangGroup) (*PrefixSuffixPredictor, error) {
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
		prm := obj.(*PrefixSuffixPartialRunModel)
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

	return &PrefixSuffixPredictor{
		hparams:   params,
		search:    search,
		model:     model,
		encoder:   encoder,
		predCache: predcache,
		prCache:   prcache,
		useCache:  true,
	}, nil
}

// GetModel ...
func (t *PrefixSuffixPredictor) GetModel() *tensorflow.Model {
	return t.model
}

// SetUseCache enables prm cache use, only set this at initializtion
func (t *PrefixSuffixPredictor) SetUseCache(val bool) {
	t.useCache = val
}

// SetStrictChecking enables stricter checks about the input, search params, etc
func (t *PrefixSuffixPredictor) SetStrictChecking(val bool) {
	t.strict = val
}

// GetHParams implements Predictor
func (t *PrefixSuffixPredictor) GetHParams() HParams {
	return t.hparams
}

// GetEncoder implements Predictor
func (t *PrefixSuffixPredictor) GetEncoder() *lexicalv0.FileEncoder {
	return t.encoder
}

// Unload implements Predictor
func (t *PrefixSuffixPredictor) Unload() {
	t.model.Unload()
	t.prCache.Purge()
	t.predCache.Purge()
}

// GetLexer implements Predictor
func (t *PrefixSuffixPredictor) GetLexer() lexer.Lexer {
	return t.encoder.Lexer
}

type prefixSuffixInputs struct {
	PrefixSuffixInputs
	// used by predictor
	prefix      string
	incremental chan Predicted
	rand        *rand.Rand
	prm         *PrefixSuffixPartialRunModel
	embedded    []int64
	search      SearchConfig
}

func (t *PrefixSuffixPredictor) newInputs(in Inputs, setupPartialRun bool) (prefixSuffixInputs, error) {
	defer status.NewPredictStateDuration.DeferRecord(time.Now())

	search := selectSearchConfig(t.search, in)

	before, _ := buildContextBeforeAndSetPrefix(&in, t.encoder, search)
	after := buildContextAfter(in, t.encoder, search)

	psIn := prefixSuffixInputs{
		PrefixSuffixInputs: PrefixSuffixInputs{
			Before: before,
			After:  after,
		},
		prefix: in.Prefix,
		rand:   rand.New(rand.NewSource(in.RandomSeed)),
		search: search,
	}

	if !setupPartialRun {
		return psIn, nil
	}

	// This lock wraps cache lookup & insertion, preventing a race between eviction (created by an Add) event
	// and lookup, which prevents looking up a partial run we *just* evicted
	t.m.Lock()
	defer t.m.Unlock()
	if t.useCache {
		for _, key := range t.prCache.Keys() {
			obj, ok := t.prCache.Peek(key)
			if !ok {
				continue
			}

			currentPRM := obj.(*PrefixSuffixPartialRunModel)
			slots := currentPRM.mgr.SlotsRemaining()
			if embedded, unembedded, match := currentPRM.MatchAndReserve(before, after, search.Depth, search); match {
				currentPRM.mgr.logf("reusing partial run %d with %d -> %d slots, unembedded: %d",
					currentPRM.mgr.id, slots, currentPRM.mgr.SlotsRemaining(), len(unembedded),
				)
				psIn.prm = currentPRM
				if len(unembedded) > 0 {
					psIn.Predict = [][]int64{unembedded}
				}
				psIn.embedded = embedded

				t.prCache.Get(key) // just to update recency status within the lru
				break
			}
		}

		if psIn.prm != nil {
			status.PartialRunReuseRate.Hit()
			return psIn, nil
		}
		status.PartialRunReuseRate.Miss()
	}

	prm, err := NewPrefixSuffixPartialRunModel(t.model, psIn.PrefixSuffixInputs, t.hparams, search, t.strict)
	if err != nil {
		return prefixSuffixInputs{}, errors.Wrapf(err, "error creating partial run model")
	}

	if !prm.mgr.Reserve(search.Depth) {
		return prefixSuffixInputs{}, errors.Wrapf(err, "unable to reserve slots after creating partial run model")
	}

	psIn.prm = prm
	t.prCache.Add(prm.mgr.id, prm)
	return psIn, nil
}

// PredictChan ...
func (t *PrefixSuffixPredictor) PredictChan(kctx kitectx.Context, in Inputs) (chan Predicted, chan error) {
	psIn, err := t.newInputs(in, true)
	if err != nil {
		return handlePredictChanInitErr(err)
	}

	psIn.incremental = make(chan Predicted, psIn.search.Depth*psIn.search.BeamWidth)

	errChan := kitectx.Go(func() error {
		_, err := t.predict(kctx, psIn, false)
		return err
	})
	return psIn.incremental, errChan
}

// Predict ...
func (t *PrefixSuffixPredictor) Predict(kctx kitectx.Context, in Inputs) (Predictions, error) {
	psIn, err := t.newInputs(in, true)
	if err != nil {
		return Predictions{}, err
	}

	res, err := t.predict(kctx, psIn, in.IncludeMetaInfo)
	if err != nil {
		return Predictions{}, err
	}

	if in.IncludeMetaInfo {
		res.Meta.Prefix = psIn.prefix
		// TODO: this is not quite right since technically we perform
		// prediction on the psIn.prm.initIn + psIn.Predict[0],
		// ideally we would surface this in the inspector
		// but we also use these fields in `minp/minp.go` when we compute
		// matches and that breaks if we do not send back the original context.
		res.Meta.ContextBefore = toInt(psIn.Before)
		res.Meta.ContextAfter = toInt(psIn.After)
		if len(psIn.Predict) > 0 {
			res.Meta.ContextPredict = toInt(psIn.Predict[0])
		}
	}

	return res, nil
}

func (t *PrefixSuffixPredictor) predict(kctx kitectx.Context, initIn prefixSuffixInputs, includeMetaInfo bool) (Predictions, error) {
	var queryCalled int
	defer func() {
		// Calls to query consume a slot. We always reserve search.Depth slots. If
		// we used fewer slots, return them to the partial run.
		if queryCalled < initIn.search.Depth {
			initIn.prm.mgr.Release(initIn.search.Depth - queryCalled)
		}
	}()

	var exps []PrefixSuffixInputs

	nextLogitsFn := func(kctx kitectx.Context, step int, candidates [][]int64) (res [][]float32, err error) {
		kctx.CheckAbort()

		var psIn PrefixSuffixInputs
		noUnembeddedContext := len(initIn.Predict) == 0
		initStep := step == 0
		switch {
		case initStep && noUnembeddedContext:
			if includeMetaInfo {
				exps = append(exps, initIn.PrefixSuffixInputs)
			}
			return [][]float32{initIn.prm.initialScores}, nil
		case initStep:
			psIn.Predict = [][]int64{initIn.Predict[0]}
		case noUnembeddedContext:
			psIn.Predict = candidates
		default:
			psIn.Predict = make([][]int64, 0, len(candidates))
			for _, cand := range candidates {
				psIn.Predict = append(psIn.Predict, copyAndAppend(initIn.Predict[0], cand))
			}
		}

		if includeMetaInfo {
			exps = append(exps, psIn)
		}

		if t.useCache {
			key := t.predCacheKey(initIn, psIn.Predict)
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

		queryCalled++
		return initIn.prm.Query(psIn.Predict)
	}

	results, err := searcher{
		config:          initIn.search,
		nextLogits:      nextLogitsFn,
		predictChan:     initIn.incremental,
		enc:             t.encoder,
		prefix:          initIn.prefix,
		rand:            initIn.rand,
		strict:          t.strict,
		includeMetaInfo: includeMetaInfo,
	}.Search(kctx)

	if err != nil {
		return Predictions{}, errors.Wrapf(err, "search error")
	}

	if includeMetaInfo {
		for i, exp := range exps {
			results.Meta.Expansions[i].Predict = toInt2d(exp.Predict)
			// tile before and after to match batch size of prefix to make it easier for clients
			for range exp.Predict {
				results.Meta.Expansions[i].Before = append(results.Meta.Expansions[i].Before, toInt(initIn.Before))
				results.Meta.Expansions[i].After = append(results.Meta.Expansions[i].After, toInt(initIn.After))
			}
		}
	}

	return results, nil
}

func (t *PrefixSuffixPredictor) predCacheKey(initIn prefixSuffixInputs, predicts [][]int64) string {
	var strs []string
	for _, ctx := range predicts {
		// Use the before we are actually predicting on
		base := toStrings(initIn.embedded)
		base = append(base, toStrings(ctx)...)

		// Use the after we are actually predicting on ...
		base = append(base, toStrings(initIn.prm.initIn.After)...)

		strs = append(strs, strings.Join(base, ","))
	}

	// sort to ensure keys are consistent
	sort.Strings(strs)

	return strings.Join(strs, ":")
}

// Logits returns the logits for the prediction based on the provided context
// return length: vocab
func (t *PrefixSuffixPredictor) Logits(in Inputs) ([]float32, error) {
	psIn, err := t.newInputs(in, false)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create inputs")
	}

	logits, err := NewPrefixSuffixModel(t.model, t.hparams, t.strict).Query(psIn.PrefixSuffixInputs, true)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get logits")
	}

	return logits[0], nil
}
