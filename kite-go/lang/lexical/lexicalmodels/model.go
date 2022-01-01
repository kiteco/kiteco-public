package lexicalmodels

import (
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// ModelBase is an interface that specifies the required API for a lexical model to be used
// by the lexical completion engine
type ModelBase interface {
	// PredictChan returns an asynchronous stream of predictions, as well as an asynchronous error.
	// Neither channel must be drained (the Model correctly buffers the channel).
	// The error channel always yields exactly one error (or nil), and will not be closed. Read it at most once.
	// The predictions channel can yield an arbitrary number of predictions, and will be closed.
	PredictChan(kitectx.Context, predict.Inputs) (chan predict.Predicted, chan error)
	GetLexer() lexer.Lexer
	Unload()

	// NOTE: this should only be used for evaluation binaries
	GetEncoder() *lexicalv0.FileEncoder
}
