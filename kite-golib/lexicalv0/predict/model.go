package predict

import (
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// NewModel from the provided model
func NewModel(m *tensorflow.Model, hp HParams, strict bool) Model {
	return Model{
		model:  m,
		hp:     hp,
		strict: strict,
	}
}

// Model encapsulates the core tensorflow model
type Model struct {
	model  *tensorflow.Model
	hp     HParams
	strict bool
}

// Query the model for the probs (logits) for the next token, given the original context
// and the new added context.
// NOTE:
//   - we do not make a deep copy of context
func (m Model) Query(context [][]int64, getLogits bool, langTag int) ([][]float32, error) {
	if m.strict {
		if err := validate(context, m.hp); err != nil {
			return nil, err
		}
	}

	feed := make(map[string]interface{})
	feed[contextPlacholderOpName] = context

	if langTag > -1 {
		feed[langsPlaceholderOpName] = []int64{int64(langTag)}
	}

	predOp := "prediction/last"
	if getLogits {
		predOp = "prediction/last_logits"
	}

	res, err := m.model.Run(feed, []string{predOp})
	if err != nil {
		return nil, err
	}

	if langTag > -1 {
		return res[predOp].([][][]float32)[0], nil
	}
	return res[predOp].([][]float32), nil
}

// TODO: make this a method on HParams?
func validate(context [][]int64, hp HParams) error {
	if len(context) == 0 {
		if hp.ModelType == ModelTypePrefixSuffix {
			return nil
		}
		return errors.Errorf("no context provided")
	}
	l := len(context[0])

	if l > hp.ContextSize {
		return errors.Errorf("context size %d >= max context size %d", l, hp.ContextSize)
	}

	for _, ctx := range context {
		if len(ctx) == 0 && hp.ModelType != ModelTypePrefixSuffix {
			return errors.Errorf("got context of length 0")
		}

		if len(ctx) != l {
			return errors.Errorf("provided batch of contexts is not square, %d != %d", l, len(ctx))
		}

		for i, id := range ctx {
			if id < 0 || id >= int64(hp.VocabSize) {
				return errors.Errorf("context encoding at pos %d is invalid (%d)", i, id)
			}
		}
	}
	return nil
}

// NewPrefixSuffixModel from the provided model
func NewPrefixSuffixModel(m *tensorflow.Model, hp HParams, strict bool) PrefixSuffixModel {
	return PrefixSuffixModel{
		model:  m,
		hp:     hp,
		strict: strict,
	}
}

// PrefixSuffixModel encapsulates the core tensorflow model
type PrefixSuffixModel struct {
	model  *tensorflow.Model
	hp     HParams
	strict bool
}

// Query the model for the probs (logits) for the next token
// NOTE:
//   - we do not make a deep copy of any of the fields of in
func (m PrefixSuffixModel) Query(in PrefixSuffixInputs, getLogits bool) ([][]float32, error) {
	if m.strict {
		if err := in.Validate(m.hp); err != nil {
			return nil, err
		}
	}

	feed := in.Feed(-1)
	predOp := prefixSuffixFetchOps.predOp(-1, getLogits)

	res, err := m.model.Run(feed, []string{predOp})
	if err != nil {
		return nil, err
	}

	return res[predOp].([][]float32), nil
}

// FetchTokenEmbeddings ...
func FetchTokenEmbeddings(predictor Predictor) ([][]float32, error) {
	fetchOp := "wte"
	res, err := predictor.GetModel().Run(nil, []string{fetchOp})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch results")
	}

	return res[fetchOp].([][]float32), nil
}
