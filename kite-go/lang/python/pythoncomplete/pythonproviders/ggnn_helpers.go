package pythonproviders

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// GGNNModelAccumulator is a wrapper around GGNN model for testing purpose
type GGNNModelAccumulator struct {
	ForceDisableFiltering  bool
	ForceUsePartialDecoder bool
}

// MarshalJSON implements Provider
func (g GGNNModelAccumulator) MarshalJSON() ([]byte, error) {
	panic("Not implemented, GGNNModelAccumulator is only a provider for testing purpose")
}

type partialComp struct {
	comp   MetaCompletion
	pred   pythongraph.PredictorNew
	inputs Inputs
}

// Provide simulates the scheduler loop for GGNNModel provider and returns completions when they are emitted by GGNN model
func (g GGNNModelAccumulator) Provide(ctx kitectx.Context, global Global, in Inputs, out OutputFunc) error {
	provider := GGNNModel{ForceDisableFiltering: g.ForceDisableFiltering}
	if g.ForceUsePartialDecoder {
		in.UsePartialDecoder = true
	}

	workingQueue := []partialComp{partialComp{inputs: in, comp: MetaCompletion{
		Score: 1,
	}}}
	var err error
	for len(workingQueue) > 0 {
		// pop
		inputs, baseComp := workingQueue[0].inputs, workingQueue[0].comp
		workingQueue = workingQueue[1:]
		err = provider.Provide(kitectx.Background(), global, inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc MetaCompletion) {
			if baseComp.Snippet.Text != "" {
				composedComp, err := mc.After(baseComp.Completion)
				if err != nil {
					panic(err)
				}
				mc.Completion = composedComp
			}
			out(ctx, sb, mc)

			if p := mc.GGNNMeta.Predictor; p != nil {

				var nextBuffer data.SelectedBuffer
				if mc.GGNNMeta.SpeculationPlaceholderPresent {
					placeholders := mc.Snippet.Placeholders()
					nextBuffer = inputs.Select(mc.Replace).Replace(mc.Snippet.Text).Select(data.Cursor(mc.Replace.Begin + placeholders[len(placeholders)-1].Begin))
				} else {
					nextBuffer = inputs.Select(mc.Replace).ReplaceWithCursor(mc.Snippet.Text)
				}

				nextInputs, err := NewInputs(kitectx.Background(), global, nextBuffer, false, true)
				if err != nil {
					panic(err)
				}
				nextInputs.GGNNPredictor = p
				workingQueue = append(workingQueue, partialComp{
					inputs: nextInputs,
					comp:   mc,
				})
			}
		})
	}
	return err
}

// Name implements Provider interface (this provider should only be used for testing purpose)
func (GGNNModelAccumulator) Name() data.ProviderName {
	return -1
}
