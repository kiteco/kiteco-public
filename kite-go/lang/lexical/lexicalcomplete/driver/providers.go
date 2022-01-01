package driver

import "github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"

var allProviders = map[lexicalproviders.Provider]struct{}{
	lexicalproviders.Text{}: struct{}{},
}

// blockingProviders will block Driver.Update for at most a duration of api.CompleteOptions.BlockTimeout,
// if this is 0 then they will block Driver.Update until all of them complete their work.
var blockingProviders = map[lexicalproviders.Provider]struct{}{
	lexicalproviders.Text{}: struct{}{},
}

var speculationProviders = map[lexicalproviders.Provider]struct{}{
	lexicalproviders.Text{}: struct{}{},
}

// prioritizedProviders is the provider priority used for mixing:
// - dedupe during collection
// - sorting
// Earlier in the list indicates a higher priority.
var prioritizedProviders = []lexicalproviders.Provider{
	lexicalproviders.Text{},
}

// prioritizedSpeculationProviders is used during the collection to know what provider completions to collect
// when we are not at the root buffer (ie buffer state sent by the user with the completion request)
// We only collect completion that we can schedule
var prioritizedSpeculationProviders []lexicalproviders.Provider

func init() {
	for _, p := range prioritizedProviders {
		if _, ok := speculationProviders[p]; ok {
			prioritizedSpeculationProviders = append(prioritizedSpeculationProviders, p)
		}
	}
}

// TestProviders returns the providers for test cases
func TestProviders() []lexicalproviders.Provider {
	return prioritizedProviders
}
