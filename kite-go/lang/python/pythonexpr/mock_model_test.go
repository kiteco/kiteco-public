package pythonexpr

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type mockModel struct {
	path    string
	opts    Options
	callMap map[string]int
}

func newMockModel(path string, opts Options) (*mockModel, error) {
	return &mockModel{
		path:    path,
		opts:    opts,
		callMap: make(map[string]int),
	}, nil
}

func (m *mockModel) called(name string) {
	if _, ok := m.callMap[name]; !ok {
		m.callMap[name] = 0
	}
	m.callMap[name]++
}

func (m *mockModel) getCalledCount(name string) int {
	if _, ok := m.callMap[name]; !ok {
		return 0
	}
	return m.callMap[name]
}

// --

// Load the model
func (m *mockModel) Load() error {
	m.called("Load")
	return nil
}

// Reset unloads all shards
func (m *mockModel) Reset() {
	m.called("Reset")
}

// IsLoaded returns true if the underlying model was successfully loaded.
func (m *mockModel) IsLoaded() bool {
	m.called("IsLoaded")
	return true
}

// AttrSupported returns nil if the Model is able to provide completions for the
// specified parent.
func (m *mockModel) AttrSupported(rm pythonresource.Manager, parent pythonresource.Symbol) error {
	m.called("AttrSupported")
	return nil
}

// AttrCandidates for the specified parent symbol
func (m *mockModel) AttrCandidates(rm pythonresource.Manager, parent pythonresource.Symbol) ([]int32, []pythonresource.Symbol, error) {
	m.called("AttrCandidates")
	return nil, nil, nil
}

// CallSupported returns nil if the model is able to provide call completions for the
// specified symbol.
func (m *mockModel) CallSupported(rm pythonresource.Manager, sym pythonresource.Symbol) error {
	m.called("CallSupported")
	return nil
}

// Dir returns the directory from which the model was loaded.
func (m *mockModel) Dir() string {
	m.called("Dir")
	return m.path
}

// FuncInfo gets all the needed info for call completion
func (m *mockModel) FuncInfo(rm pythonresource.Manager, sym pythonresource.Symbol) (*pythongraph.FuncInfo, error) {
	m.called("FuncInfo")
	return nil, nil
}

// Predict an expression completion
func (m *mockModel) Predict(ctx kitectx.Context, in Input) (*GGNNResults, error) {
	m.called("Predict")
	return nil, nil
}

// MetaInfo for the model
func (m *mockModel) MetaInfo() MetaInfo {
	m.called("MetaInfo")
	return MetaInfo{}
}
