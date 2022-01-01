package predict

import (
	"fmt"
	"os"
	"reflect"
	"text/tabwriter"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

// TFMultiPredictor wraps multiple models to test for discrepancies
type TFMultiPredictor struct {
	model1 *TFSearcher
	model2 *TFServingSearcher
}

// NewTFMultiPredictor creates a new multi predictor
func NewTFMultiPredictor(model1 *TFSearcher, model2 *TFServingSearcher) *TFMultiPredictor {
	return &TFMultiPredictor{
		model1: model1,
		model2: model2,
	}
}

// PredictChan ...
func (t *TFMultiPredictor) PredictChan(kctx kitectx.Context, in Inputs) (chan Predicted, chan error) {
	// TODO: could make this work if we needed to
	return t.model1.PredictChan(kctx, in)
}

// GetHParams implements Predictor
func (t *TFMultiPredictor) GetHParams() HParams {
	return t.model1.GetHParams()
}

// GetEncoder implements Predictor
func (t *TFMultiPredictor) GetEncoder() *lexicalv0.FileEncoder {
	return t.model1.GetEncoder()
}

// Unload implements Predictor
func (t *TFMultiPredictor) Unload() {
	// no-op
}

// GetLexer implements Predictor
func (t *TFMultiPredictor) GetLexer() lexer.Lexer {
	return t.model1.GetLexer()
}

// TargetProbability implements performance.Model
func (t *TFMultiPredictor) TargetProbability(context []int, target int, search SearchConfig) (float64, error) {
	// This can be made to work by using the tfserving client's Probs API
	return 0.0, nil
}

// Predict implements performance.Model
func (t *TFMultiPredictor) Predict(kctx kitectx.Context, in Inputs) (Predictions, error) {
	res1, err1 := t.model1.Predict(kctx, in)
	if err1 != nil {
		return Predictions{}, errors.Wrapf(err1, "model1 error")
	}
	preds1 := res1.Preds

	res2, err2 := t.model2.Predict(kctx, in)
	if err2 != nil {
		return Predictions{}, errors.Wrapf(err1, "model2 error")
	}
	preds2 := res2.Preds

	var diff bool
	if len(preds1) != len(preds2) {
		diff = true
	}

	if !diff {
		for i := 0; i < len(preds1); i++ {
			if !reflect.DeepEqual(preds1[i].TokenIDs, preds2[i].TokenIDs) {
				diff = true
			}
		}
	}

	if diff {
		fmt.Println("====== detected difference =======")
		tabw := tabwriter.NewWriter(os.Stdout, 16, 4, 4, ' ', 0)
		fmt.Fprintln(tabw, "idx\tmodel1\tprob\tmodel2\tprob")
		for j := 0; j < len(preds1) || j < len(preds2); j++ {
			var s, r Predicted
			if j < len(preds1) {
				s = preds1[j]
			}
			if j < len(preds2) {
				r = preds2[j]
			}
			fmt.Fprintf(tabw, "%d\t%v\t%.04f\t%v\t%.04f\n", j,
				s.TokenIDs, s.Prob, r.TokenIDs, r.Prob)
		}
		tabw.Flush()
	}

	return res1, nil
}
