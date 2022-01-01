package pythonresource

import (
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var errResponse = errors.New("errorManager error response")

// errorManager is for testing and returns error where possible
type errorManager struct {
}

func (m *errorManager) Close() error {
	return nil
}

func (m *errorManager) Reset() {
}

func (m *errorManager) Distributions() []keytypes.Distribution {
	return nil
}

func (m *errorManager) DistLoaded(dist keytypes.Distribution) bool {
	return false
}

func (m *errorManager) ArgSpec(sym Symbol) *pythonimports.ArgSpec {
	return nil
}

func (m *errorManager) PopularSignatures(sym Symbol) []*editorapi.Signature {
	return nil
}

func (m *errorManager) CumulativeNumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	return 1, false
}

func (m *errorManager) KeywordArgFrequency(sym Symbol, arg string) (int, bool) {
	return 1, false
}

func (m *errorManager) NumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	return 1, false
}

func (m *errorManager) Documentation(sym Symbol) *docs.Entity {
	return nil
}

func (m *errorManager) SymbolCounts(sym Symbol) *symbolcounts.Counts {
	return nil
}

func (m *errorManager) Kwargs(sym Symbol) *KeywordArgs {
	return nil
}

func (m *errorManager) TruthyReturnTypes(sym Symbol) []TruthySymbol {
	return nil
}

func (m *errorManager) ReturnTypes(sym Symbol) []Symbol {
	return nil
}

func (m *errorManager) PathSymbol(path pythonimports.DottedPath) (Symbol, error) {
	return Symbol{}, errResponse
}

func (m *errorManager) PathSymbols(ctx kitectx.Context, path pythonimports.DottedPath) ([]Symbol, error) {
	return nil, errResponse
}

func (m *errorManager) NewSymbol(dist keytypes.Distribution, path pythonimports.DottedPath) (Symbol, error) {
	return Symbol{}, errResponse
}

func (m *errorManager) Kind(s Symbol) keytypes.Kind {
	return keytypes.NoneKind
}

func (m *errorManager) Type(s Symbol) (Symbol, error) {
	return Symbol{}, errResponse
}

func (m *errorManager) Bases(s Symbol) []Symbol {
	return nil
}

func (m *errorManager) Children(s Symbol) ([]string, error) {
	return nil, errResponse
}

func (m *errorManager) ChildSymbol(s Symbol, c string) (Symbol, error) {
	return Symbol{}, errResponse
}

func (m *errorManager) CanonicalSymbols(dist keytypes.Distribution) ([]Symbol, error) {
	return nil, errResponse
}

func (m *errorManager) TopLevels(dist keytypes.Distribution) ([]string, error) {
	return nil, errResponse
}

func (m *errorManager) Pkgs() []string {
	return nil
}

func (m *errorManager) DistsForPkg(pkg string) []keytypes.Distribution {
	return nil
}

func (m *errorManager) SigStats(sym Symbol) *SigStats {
	return nil
}
