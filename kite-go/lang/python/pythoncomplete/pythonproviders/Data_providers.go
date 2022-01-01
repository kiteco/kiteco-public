package pythonproviders

import (
	"encoding/json"
	"reflect"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Providers lists all the providers by name
var Providers = map[data.ProviderName]Provider{
	data.PythonEmptyCallsProvider:     EmptyCalls{},
	data.PythonCallPatternsProvider:   CallPatterns{},
	data.PythonImportsProvider:        Imports{},
	data.PythonAttributesProvider:     Attributes{},
	data.PythonNamesProvider:          Names{},
	data.PythonKeywordsProvider:       Keywords{},
	data.PythonCallModelProvider:      CallModel{},
	data.PythonAttributeModelProvider: AttributeModel{},
	data.PythonExprModelProvider:      ExprModel{},
	data.PythonKWArgsProvider:         KWArgs{},
	data.PythonDictKeysProvider:       DictKeys{},
	data.PythonGGNNModelProvider:      GGNNModel{},
	data.PythonLexicalProvider:        Lexical{},
}

// SmartProviders lists all providers that can create "smart" completions
// This does not include PythonImportsProvider because
// Import is not smart but ImportAs is smart
var SmartProviders = map[data.ProviderName]bool{
	data.PythonCallPatternsProvider:   true,
	data.PythonCallModelProvider:      true,
	data.PythonAttributeModelProvider: true,
	data.PythonExprModelProvider:      true,
	data.PythonGGNNModelProvider:      true,
	data.PythonLexicalProvider:        true,
	// PythonImportsProvider:      special case (above comment)
}

// -

// OutputFunc is the callback for returning results from a Provider
type OutputFunc func(kitectx.Context, data.SelectedBuffer, MetaCompletion)

// Provider is a function that provides Completions for a SelectedBuffer by passing them to an OutputFunc
type Provider interface {
	Provide(kitectx.Context, Global, Inputs, OutputFunc) error
	Name() data.ProviderName
}

// Trivial is implemented by providers that only emit a single "trivial" completion,
// and should not be considered independent for mixing (TODO and metrics?)
type Trivial interface {
	trivial()
}

// CompletionPromoter ...
type CompletionPromoter interface {
	GetDistanceFromRoot(mc MetaCompletion) int
	promoted()
}

// CompletionComparator allows the provider to implement its own ordering logic. The return value indicates that comp1 has higher priority than comp2 at Mixing time
type CompletionComparator interface {
	// Sorting function used to order completion coming from the same provider
	// returning :
	// - true means comp1 should be first
	// - false means comp2 should be first
	CompareCompletions(comp1, comp2 MetaCompletion) bool
}

// - serdes

// ProviderJSON is a ser/des Provider type
type ProviderJSON struct {
	Provider
}

// MarshalJSON implements json.Marshaler
func (p ProviderJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name data.ProviderName `json:"type"`
		Provider
	}{
		Name:     p.Name(),
		Provider: p.Provider,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (p *ProviderJSON) UnmarshalJSON(b []byte) error {
	name := struct {
		Name data.ProviderName `json:"type"`
	}{}
	if err := json.Unmarshal(b, &name); err != nil {
		return err
	}

	provider := Providers[name.Name]
	if provider == nil {
		return errors.Errorf("no provider for name %d", name.Name)
	}

	ptr := reflect.New(reflect.TypeOf(provider))
	if err := json.Unmarshal(b, ptr.Interface()); err != nil {
		return err
	}
	p.Provider = ptr.Elem().Interface().(Provider)
	return nil
}
