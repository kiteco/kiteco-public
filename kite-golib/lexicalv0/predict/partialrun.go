package predict

import (
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// PartialRunModel sets up a new partial run from the model and tracks prediction slots available
type PartialRunModel struct {
	hp     HParams
	search SearchConfig
	strict bool

	mgr *partialRunManager

	initialContext []int64
	initialScores  []float32

	langTag int
}

// NewPartialRunModel from the provided model and using the specified context
// NOTE:
//   - This calls into tensorflow to do the initial embedding
func NewPartialRunModel(m *tensorflow.Model, ctx []int64, hp HParams, search SearchConfig, strict bool, langTag int) (*PartialRunModel, error) {
	defer status.NewPartialRunModelDuration.DeferRecord(time.Now())

	model := &PartialRunModel{
		hp:      hp,
		search:  search,
		strict:  strict,
		langTag: langTag,
	}

	var fetches, feeds []string
	for i := -1; i < hp.NumPredictionSlots; i++ {
		if search.UseTemperatureScaling {
			fetches = append(fetches, model.predictionSlotLogits(i))
		} else {
			fetches = append(fetches, model.predictionSlotPreds(i))
		}
		// TODO: kind of nasty
		if i == -1 && langTag > -1 {
			feeds = append(feeds, langsPlaceholderOpName)
		}
		feeds = append(feeds, model.predictionSlotPlaceholder(i))
	}

	pr, err := m.NewPartialRun(feeds, fetches, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating partial run")
	}

	model.mgr = newPartialRunManager(pr, hp)

	// embed initial context
	err = func() error {
		defer status.EmbedInitialContextDuration.DeferRecord(time.Now())

		ff, err := model.feedFunc([][]int64{ctx}, langTag)
		if err != nil {
			return errors.Wrapf(err, "error creating feed func for initial context")
		}

		res, err := model.mgr.EmbedInitialContext(ff)
		if err != nil {
			return errors.Wrapf(err, "error embedding intial context")
		}

		model.initialScores = res.([][]float32)[0]
		model.initialContext = make([]int64, len(ctx))
		copy(model.initialContext, ctx)

		return nil
	}()

	if err != nil {
		return nil, errors.Wrapf(err, "error embedding initial context")
	}

	return model, nil
}

// Close forces the partial run model to cleanup
func (p *PartialRunModel) Close() error {
	defer status.ClosePartialRunDuration.DeferRecord(time.Now())

	ff, err := p.feedFunc(nil, p.langTag)
	if err != nil {
		return errors.Wrapf(err, "unable to get feed func for close")
	}

	if err := p.mgr.Close(ff); err != nil {
		return errors.Wrapf(err, "error closing run")
	}
	return nil
}

// Query the model for the probs (logits) for the next token, given the original context
// and the new added context.
// NOTE:
//   - we do not make a deep copy of newCtx
func (p *PartialRunModel) Query(ctx [][]int64) ([][]float32, error) {
	defer status.PartialRunQueryDuration.DeferRecord(time.Now())

	ff, err := p.feedFunc(ctx, p.langTag)
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
func (p *PartialRunModel) MatchAndReserve(context []int64, slots int, search SearchConfig, langTag int) ([]int64, []int64, bool) {
	if p.langTag != langTag {
		return nil, nil, false
	}

	// Check if the context is a match
	embedded, unembedded, match := matchSuffix(p.initialContext, context)
	if !match {
		return p.initialContext, context, false
	}

	// Check if we have enough positions left in the context
	if len(embedded)+len(unembedded) > p.hp.ContextSize-slots {
		return embedded, unembedded, false
	}

	return embedded, unembedded, p.mgr.Reserve(slots)
}

func (p *PartialRunModel) feedFunc(ctx [][]int64, langTag int) (feederFunc, error) {
	// len(ctx) == 0 is for closing case
	if len(ctx) > 0 {
		if len(ctx[0])+len(p.initialContext) > p.hp.ContextSize {
			return nil, errors.Errorf("new context (%d) + original context (%d) > context size (%d)",
				len(ctx[0]), len(p.initialContext), p.hp.ContextSize)
		}

		if p.strict {
			if err := validate(ctx, p.hp); err != nil {
				return nil, err
			}
		}
	}

	return func(slot int, isForClose bool) (map[string]interface{}, string) {
		if slot == -1 {
			if isForClose {
				ctx = [][]int64{{0}}
			}

			feed := map[string]interface{}{
				contextPlacholderOpName: ctx,
			}
			if langTag > -1 {
				feed[langsPlaceholderOpName] = []int64{int64(langTag)}
			}
			return feed, p.predOp(-1)
		}

		if isForClose {
			// we send -1 in for the context as a flag to make sure we do not run any of
			// the ops and return as quickly as possible
			ctx = [][]int64{{-1}}
		}

		return map[string]interface{}{
			p.predictionSlotPlaceholder(slot): ctx,
		}, p.predOp(slot)
	}, nil
}

func (p *PartialRunModel) predictionSlotPlaceholder(slot int) string {
	if slot < 0 {
		return contextPlacholderOpName
	}

	return fmt.Sprintf("test/prediction%d/placeholders/context", slot)
}

func (p *PartialRunModel) predictionSlotLogits(slot int) string {
	if slot < 0 {
		return "test/predict_context/logits"
	}
	return fmt.Sprintf("test/prediction%d/logits", slot)
}

func (p *PartialRunModel) predictionSlotPreds(slot int) string {
	if slot < 0 {
		return "test/predict_context/preds"
	}
	return fmt.Sprintf("test/prediction%d/preds", slot)
}

func (p *PartialRunModel) predOp(slot int) string {
	if p.search.UseTemperatureScaling {
		return p.predictionSlotLogits(slot)
	}
	return p.predictionSlotPreds(slot)
}
