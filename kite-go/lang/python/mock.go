package python

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// ExcludeMockServices are constants for excluding specific
// objects within python.Services from being populated with method data.
type ExcludeMockServices int

const (
	// None exlcudes no services
	None ExcludeMockServices = 1 << iota
	// SignaturePatterns excludes pythoncode.SignaturePatterns from being populated.
	SignaturePatterns
	// Curation excludes pythoncuration.Searcher from being populated.
	Curation
)

// MockServicesFromMap returns a mock services object that can respond to requests
// for the provided methods.
func MockServicesFromMap(t *testing.T, methods map[string]pythonimports.Kind) *Services {
	var names []string
	for m := range methods {
		names = append(names, m)
	}

	graph := pythonimports.MockGraphFromMap(methods)
	manager := pythonresource.MockManager(t, pythonresource.InfosFromKinds(methods))
	return &Services{
		Curation:        pythoncuration.MockCurationFromMap(graph, methods),
		ImportGraph:     graph,
		Options:         &DefaultServiceOptions,
		ResourceManager: manager,
		Models:          pythonmodels.Mock(),
	}
}

// MockServicesWithReturns returns a mock services object that can
// respond to requests for the provided functions.
func MockServicesWithReturns(t *testing.T, methods map[string]pythonimports.Kind, returns map[string]string) *Services {
	var names []string
	for m := range methods {
		names = append(names, m)
	}

	graph := pythonimports.MockGraphFromMap(methods)
	manager := pythonresource.MockManager(t, pythonresource.InfosFromKinds(methods))

	return &Services{
		Curation:        pythoncuration.MockCurationFromMap(graph, methods),
		ImportGraph:     graph,
		Options:         &DefaultServiceOptions,
		ResourceManager: manager,
		Models:          pythonmodels.Mock(),
	}
}

// MockServices returns a mock services object that can respond to requests
// for the provided methods.
func MockServices(t testing.TB, opts *ServiceOptions, methods ...string) *Services {
	graph := pythonimports.MockGraph(methods...)
	manager := pythonresource.MockManager(t, nil, methods...)

	return &Services{
		Curation:        pythoncuration.MockCuration(graph, methods...),
		ImportGraph:     graph,
		ResourceManager: manager,
		Options:         &DefaultServiceOptions,
		Models:          pythonmodels.Mock(),
	}
}

// MockServicesExclude returns a mock services object that can respond to requests
// for the provided method, and excludes populating the objects specified by excludes.
func MockServicesExclude(t testing.TB, excludes ExcludeMockServices, methods ...string) *Services {
	graph := pythonimports.MockGraph(methods...)
	manager := pythonresource.MockManager(t, nil, methods...)

	var curation *pythoncuration.Searcher
	if excludes&Curation == Curation {
		curation = pythoncuration.MockCuration(graph)
	} else {
		curation = pythoncuration.MockCuration(graph, methods...)
	}

	return &Services{
		Curation:        curation,
		ImportGraph:     graph,
		Options:         &DefaultServiceOptions,
		ResourceManager: manager,
		githubPrior:     pythoncode.MockGithubPrior(),
		Models:          pythonmodels.Mock(),
	}
}
