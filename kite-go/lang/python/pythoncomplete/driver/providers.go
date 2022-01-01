package driver

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

var zeroDepthProviders = map[pythonproviders.Provider]struct{}{
	pythonproviders.EmptyCalls{}: struct{}{},
}

var allProviders = map[pythonproviders.Provider]struct{}{
	pythonproviders.AttributeModel{}: struct{}{},
	pythonproviders.Attributes{}:     struct{}{},
	pythonproviders.CallModel{}:      struct{}{},
	pythonproviders.CallPatterns{}:   struct{}{},
	pythonproviders.DictKeys{}:       struct{}{},
	pythonproviders.EmptyCalls{}:     struct{}{},
	pythonproviders.Imports{}:        struct{}{},
	pythonproviders.Keywords{}:       struct{}{},
	pythonproviders.KWArgs{}:         struct{}{},
	pythonproviders.Lexical{}:        struct{}{},
	pythonproviders.Names{}:          struct{}{},
}

var blockingProviders = map[pythonproviders.Provider]struct{}{
	pythonproviders.Attributes{}:   struct{}{},
	pythonproviders.CallPatterns{}: struct{}{},
	pythonproviders.DictKeys{}:     struct{}{},
	pythonproviders.EmptyCalls{}:   struct{}{},
	pythonproviders.Imports{}:      struct{}{},
	pythonproviders.Keywords{}:     struct{}{},
	pythonproviders.KWArgs{}:       struct{}{},
	pythonproviders.Lexical{}:      struct{}{},
	pythonproviders.Names{}:        struct{}{},
}

var speculationProviders = map[pythonproviders.Provider]struct{}{
	pythonproviders.Attributes{}:   struct{}{},
	pythonproviders.CallModel{}:    struct{}{},
	pythonproviders.CallPatterns{}: struct{}{},
	pythonproviders.EmptyCalls{}:   struct{}{},
}

// prioritizedProviders is the provider priority used for mixing:
// - dedupe during collection
// - sorting
// Earlier in the list indicates a higher priority.
var prioritizedProviders = []pythonproviders.Provider{
	pythonproviders.Lexical{},
	pythonproviders.EmptyCalls{},

	// "traditional" completions
	pythonproviders.Attributes{},
	pythonproviders.Imports{},

	// call completions goes before name. The assumption that it never overlap with traditional is no longer true for partial completions
	// ordered according to product spec
	pythonproviders.CallModel{},

	// names goes before keywords
	pythonproviders.Names{}, pythonproviders.Keywords{},

	pythonproviders.CallPatterns{},
	pythonproviders.KWArgs{},

	// dict provider goes last according to product spec
	pythonproviders.DictKeys{},
	pythonproviders.AttributeModel{},
}

// prioritizedSpeculationProviders is used during the collection to know what provider completions to collect
// when we are not at the root buffer (ie buffer state sent by the user with the completion request)
// We only collect completion that we can schedule
var prioritizedSpeculationProviders []pythonproviders.Provider

func init() {
	for _, p := range prioritizedProviders {
		if _, ok := speculationProviders[p]; ok {
			prioritizedSpeculationProviders = append(prioritizedSpeculationProviders, p)
		}
	}
}

var testedProviders = []pythonproviders.Provider{
	pythonproviders.Lexical{},
	pythonproviders.EmptyCalls{},
	pythonproviders.Attributes{},
	pythonproviders.Imports{},
	pythonproviders.CallModel{},
	pythonproviders.Names{},
	pythonproviders.Keywords{},
	pythonproviders.CallPatterns{},
	pythonproviders.KWArgs{},
	pythonproviders.DictKeys{},
	pythonproviders.AttributeModel{},
	pythonproviders.GGNNModelAccumulator{ForceUsePartialDecoder: true},
}

// TestProviders returns the providers for test cases
func TestProviders() []pythonproviders.Provider {
	return testedProviders
}

// UseTestProviders sets providers for test cases
func UseTestProviders() {
	prioritizedProviders = testedProviders
}

// NormalizedProviders ...
func NormalizedProviders() []pythonproviders.Provider {
	var normalized []pythonproviders.Provider
	for _, provider := range prioritizedProviders {
		if provider.Name() == data.PythonEmptyCallsProvider {
			continue
		}
		normalized = append(normalized, provider)
	}
	normalized = append(normalized, pythonproviders.GGNNModel{})
	return normalized
}
