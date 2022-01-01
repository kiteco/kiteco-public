package lexicalproviders

import (
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// OutputFunc is the callback for returning results from a Provider
type OutputFunc func(kitectx.Context, data.SelectedBuffer, MetaCompletion)

// Provider is a function that provides Completions for a SelectedBuffer by passing them to an OutputFunc
type Provider interface {
	Provide(kitectx.Context, Global, Inputs, OutputFunc) error
	Name() data.ProviderName
}
