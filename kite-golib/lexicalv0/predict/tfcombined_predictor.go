package predict

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

const remoteTimeout = 200 * time.Millisecond

// TFCombinedPredictor queries both local and remote Predictors
type TFCombinedPredictor struct {
	Local  PredictionModel
	Remote *TFServingSearcher
}

// NewTFCombinedPredictor returns a TFCombinedPredictor
func NewTFCombinedPredictor(local PredictionModel, remote *TFServingSearcher) *TFCombinedPredictor {
	return &TFCombinedPredictor{Local: local, Remote: remote}
}

// PredictChan implements the lexicalmodels.ModelBase interface
func (t *TFCombinedPredictor) PredictChan(ctx kitectx.Context, in Inputs) (chan Predicted, chan error) {
	localPredictions, localErr := t.Local.PredictChan(ctx, in)
	if t.Remote == nil {
		return localPredictions, localErr
	}
	remotePredictions, remoteErr := t.Remote.PredictChan(ctx, in)

	maxPredictions := cap(localPredictions)
	if cap(remotePredictions) > maxPredictions {
		maxPredictions = cap(remotePredictions)
	}
	predictions := make(chan Predicted, maxPredictions)

	emit := func(first Predicted, rest chan Predicted, isRemote bool) {
		predictions <- first
		for p := range rest {
			p.IsRemote = isRemote
			predictions <- p
		}
	}

	localFallback := func(errs errors.Errors) error {
		// channel closed without predictions, wait for local predictions
		select {
		case <-ctx.AbortChan():
			ctx.Abort()
		case first := <-localPredictions:
			emit(first, localPredictions, false)
		}

		// always include any remote or local predictor error
		errs = errors.Append(errs, <-remoteErr)
		errs = errors.Append(errs, <-localErr)
		return errs
	}

	return predictions, kitectx.Go(func() error {
		timeout := time.After(remoteTimeout)

		defer close(predictions)
		var errs errors.Errors

		select {
		case <-ctx.AbortChan():
			ctx.Abort()
		case first, ok := <-remotePredictions:
			if !ok {
				// channel closed without predictions, wait for local predictions
				return localFallback(errs)
			}

			emit(first, remotePredictions, true)
			errs = errors.Append(errs, <-remoteErr)

		case <-timeout:
			// remote predictions "timed out"
			errs = errors.Append(errs, errors.Errorf("remote predictions took longer than %s", remoteTimeout))

			// after remoteTimeout, return whichever of local & remote predictions return first
			select {
			case <-ctx.AbortChan():
				ctx.Abort()
			case first, ok := <-remotePredictions:
				if !ok {
					// channel closed without predictions, wait for local predictions
					return localFallback(errs)
				}

				emit(first, remotePredictions, true)
				errs = errors.Append(errs, <-remoteErr)

			case first := <-localPredictions:
				emit(first, localPredictions, false)
				errs = errors.Append(errs, <-localErr)
			}
		}

		return errs
	})
}

// Update amends the remote model and search config
func (t *TFCombinedPredictor) Update(remote *TFServingSearcher) {
	t.Remote = remote
}

// Unload implements the lexicalmodels.ModelBase interface
func (t *TFCombinedPredictor) Unload() {
	// no-op for remote lexicalmodels.Models
	t.Local.Unload()
}

// GetLexer implements the lexicalmodels.ModelBase interface
func (t *TFCombinedPredictor) GetLexer() lexer.Lexer {
	// stub with Local implementation
	return t.Local.GetLexer()
}

// GetEncoder implements the lexicalmodels.ModelBase interface
func (t *TFCombinedPredictor) GetEncoder() *lexicalv0.FileEncoder {
	// stub with Local implementation
	return t.Local.GetEncoder()
}
