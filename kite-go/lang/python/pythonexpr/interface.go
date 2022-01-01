package pythonexpr

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Model is the API exported by the Expr model
type Model interface {
	// IsLoaded returns true if the underlying model was successfully loaded.
	IsLoaded() bool

	// AttrSupported returns nil if the Model is able to provide completions for the specified parent.
	AttrSupported(rm pythonresource.Manager, parent pythonresource.Symbol) error

	// AttrCandidates for the specified parent symbol
	AttrCandidates(rm pythonresource.Manager, parent pythonresource.Symbol) ([]int32, []pythonresource.Symbol, error)

	// CallSupported returns nil if the model is able to provide call completions for the
	// specified symbol.
	CallSupported(rm pythonresource.Manager, sym pythonresource.Symbol) error

	// Dir returns the directory from which the model was loaded.
	Dir() string

	// FuncInfo gets all the needed info for call completion
	FuncInfo(rm pythonresource.Manager, sym pythonresource.Symbol) (*pythongraph.FuncInfo, error)

	// Predict an expression completion
	Predict(ctx kitectx.Context, in Input) (*GGNNResults, error)

	// MetaInfo for the model
	MetaInfo() MetaInfo

	// Reset unloads the model
	Reset()

	// Load loads the model
	Load() error
}
