package predict

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const numAfterTokensRequiredToMatch = 64

// PrefixSuffixPartialRunModel sets up a new partial run from the model and tracks prediction slots available
type PrefixSuffixPartialRunModel struct {
	hp     HParams
	search SearchConfig
	strict bool

	mgr *partialRunManager

	initIn        PrefixSuffixInputs
	initialScores []float32

	numAfterTokensRequiredToMatch int
}

// NewPrefixSuffixPartialRunModel from the provided model and using the specified context,
// NOTE:
//   - This calls into tensorflow to do the initial embedding
//   - We keep a shallow copy of in
func NewPrefixSuffixPartialRunModel(m *tensorflow.Model, in PrefixSuffixInputs, hp HParams, search SearchConfig, strict bool) (*PrefixSuffixPartialRunModel, error) {
	defer status.NewPartialRunModelDuration.DeferRecord(time.Now())

	if in.numTokensPredict() != 0 {
		return nil, errors.New("cannot initialize partial run with predict tokens")
	}

	if in.numTokensBefore() == 0 {
		return nil, errors.New("cannot initialize partial run with no before tokens")
	}

	model := &PrefixSuffixPartialRunModel{
		hp:                            hp,
		search:                        search,
		strict:                        strict,
		numAfterTokensRequiredToMatch: numAfterTokensRequiredToMatch,
	}

	var fetches, feeds []string
	for i := -1; i < hp.NumPredictionSlots; i++ {
		fetches = append(fetches, prefixSuffixFetchOps.predOp(i, search.UseTemperatureScaling))
		feeds = append(feeds, in.FeedNames(i)...)
	}

	pr, err := m.NewPartialRun(feeds, fetches, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating partial run")
	}

	model.mgr = newPartialRunManager(pr, hp)

	// embed initial context
	err = func() error {
		defer status.EmbedInitialContextDuration.DeferRecord(time.Now())

		ff, err := model.feedFunc(in)
		if err != nil {
			return errors.Wrapf(err, "error creating feed func for initial context")
		}

		res, err := model.mgr.EmbedInitialContext(ff)
		if err != nil {
			return errors.Wrapf(err, "error embedding intial context")
		}

		model.initialScores = res.([][]float32)[0]
		model.initIn = in
		return nil
	}()

	if err != nil {
		return nil, errors.Wrapf(err, "error embedding initial context")
	}

	return model, nil
}

// Close forces the partial run model to cleanup
func (p *PrefixSuffixPartialRunModel) Close() error {
	defer status.ClosePartialRunDuration.DeferRecord(time.Now())

	ff, err := p.feedFunc(PrefixSuffixInputs{Empty: true})
	if err != nil {
		return errors.Wrapf(err, "unable to get feed func for close")
	}

	if err := p.mgr.Close(ff); err != nil {
		return errors.Wrapf(err, "error closing run")
	}
	return nil
}

// Query the model for the probs (logits) for the next token, given the original context
// and the new inputs.
func (p *PrefixSuffixPartialRunModel) Query(predicts [][]int64) ([][]float32, error) {
	defer status.PartialRunQueryDuration.DeferRecord(time.Now())

	in := PrefixSuffixInputs{
		Predict: predicts,
	}

	ff, err := p.feedFunc(in)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating feed func for query")
	}

	res, err := p.mgr.RunNextSlot(ff)
	if err != nil {
		return nil, errors.Wrapf(err, "partial run error")
	}
	return res.([][]float32), nil
}

// MatchAndReserve ...
func (p *PrefixSuffixPartialRunModel) MatchAndReserve(before, after []int64, slots int, search SearchConfig) ([]int64, []int64, bool) {
	embedded, unembedded, match := p.Match(before, after, slots, search)
	if !match {
		return nil, nil, false
	}

	if !p.mgr.Reserve(slots) {
		return nil, nil, false
	}
	return embedded, unembedded, true
}

// Match ...
func (p *PrefixSuffixPartialRunModel) Match(before, after []int64, slots int, search SearchConfig) ([]int64, []int64, bool) {
	if len(after) != 0 {
		if p.initIn.numTokensAfter() == 0 {
			return nil, nil, false
		}

		nAfter := matchPrefix(p.initIn.After, after)
		numRequired := min(p.initIn.numTokensAfter(), len(after), p.numAfterTokensRequiredToMatch)
		if nAfter < numRequired {
			return nil, nil, false
		}
	}

	if len(after) == 0 && p.initIn.numTokensAfter() != 0 {
		return nil, nil, false
	}

	// check if the before contexts match
	embedded, unembedded, match := matchSuffix(p.initIn.Before, before)
	if !match {
		return nil, nil, false
	}

	// TODO: putting too many tokens
	// into the predict context degrades performance.
	if len(unembedded) > 64 {
		return nil, nil, false
	}

	// Check if we have enough positions left in the context,
	// the predict context gets 50% of the sample in training
	// TODO: fix this by adding extra fields to the hparams
	nCtxPredict := p.hp.ContextSize / 2
	if len(embedded)+len(unembedded) > nCtxPredict-slots {
		return nil, nil, false
	}
	return embedded, unembedded, true
}

func (p *PrefixSuffixPartialRunModel) feedFunc(in PrefixSuffixInputs) (feederFunc, error) {
	if p.strict {
		if err := in.Validate(p.hp); err != nil {
			return nil, errors.Wrapf(err, "invalid inputs")
		}
	}

	return func(slot int, isForClose bool) (map[string]interface{}, string) {
		if isForClose {
			in.Before = []int64{0}
			in.After = nil
			in.Predict = nil
			in.Empty = true
		}

		return in.Feed(slot), prefixSuffixFetchOps.predOp(slot, p.search.UseTemperatureScaling)
	}, nil
}

func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= a && b <= c {
		return b
	}
	return c
}
