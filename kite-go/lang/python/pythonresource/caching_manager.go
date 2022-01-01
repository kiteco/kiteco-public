package pythonresource

import (
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// NewCachingManager wraps another manager and caches the results of that manager
func NewCachingManager(mgr Manager) Manager {
	cache, err := ristretto.NewCache(&ristretto.Config{
		MaxCost:     10000,
		NumCounters: 100000,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}

	return &cachingManager{
		mgr:          mgr,
		callCache:    cache,
		callCacheTTL: 10 * time.Minute,
	}
}

// cachingManager wraps a pythonresource.Manager and caches the responses of this manager.
type cachingManager struct {
	mgr          Manager
	callCache    *ristretto.Cache
	callCacheTTL time.Duration
}

func (m *cachingManager) cacheKey(name string, argString string) string {
	if argString == "" {
		return name
	}
	return name + "_" + argString
}

// caching returns a cached result, if it already exists. Otherwise valueProvider is called, the result is cached and then returned
func (m *cachingManager) caching(name string, argString string, valueProvider func() interface{}) interface{} {
	key := m.cacheKey(name, argString)
	if v, ok := m.callCache.Get(key); ok {
		return v
	}

	v := valueProvider()
	m.callCache.SetWithTTL(key, v, 1, m.callCacheTTL)
	return v
}

// cachingError is like caching, but handled a valueProvider, which returns result and an error.
// Errors returned by valueProvider are not cached.
func (m *cachingManager) cachingError(name string, argString string, valueProvider func() (interface{}, error)) (interface{}, error) {
	key := m.cacheKey(name, argString)
	if v, ok := m.callCache.Get(key); ok {
		return v, nil
	}

	v, err := valueProvider()
	if err != nil {
		return v, err
	}

	m.callCache.SetWithTTL(key, v, 1, m.callCacheTTL)
	return v, nil
}

func (m *cachingManager) Close() error {
	err := m.mgr.Close()
	m.callCache.Close()
	return err
}

func (m *cachingManager) Reset() {
	m.mgr.Reset()
	m.callCache.Clear()
}

func (m *cachingManager) Distributions() []keytypes.Distribution {
	return m.caching("Distributions", "", func() interface{} {
		return m.mgr.Distributions()
	}).([]keytypes.Distribution)
}

func (m *cachingManager) DistLoaded(dist keytypes.Distribution) bool {
	return m.caching("DistLoaded", dist.String(), func() interface{} {
		return m.mgr.DistLoaded(dist)
	}).(bool)
}

func (m *cachingManager) ArgSpec(sym Symbol) *pythonimports.ArgSpec {
	return m.caching("ArgSpec", sym.String(), func() interface{} {
		return m.mgr.ArgSpec(sym)
	}).(*pythonimports.ArgSpec)
}

func (m *cachingManager) PopularSignatures(sym Symbol) []*editorapi.Signature {
	return m.caching("PopularSignatures", sym.String(), func() interface{} {
		return m.mgr.PopularSignatures(sym)
	}).([]*editorapi.Signature)
}

func (m *cachingManager) CumulativeNumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	type response struct {
		f float64
		b bool
	}

	result := m.caching("CumulativeNumArgsFrequency", sym.String()+"_"+strconv.Itoa(numArgs), func() interface{} {
		f, b := m.mgr.CumulativeNumArgsFrequency(sym, numArgs)
		return response{f: f, b: b}
	}).(response)
	return result.f, result.b
}

func (m *cachingManager) KeywordArgFrequency(sym Symbol, arg string) (int, bool) {
	type response struct {
		i int
		b bool
	}

	result := m.caching("KeywordArgFrequency", sym.String()+"_"+arg, func() interface{} {
		i, b := m.mgr.KeywordArgFrequency(sym, arg)
		return response{i: i, b: b}
	}).(response)
	return result.i, result.b
}

func (m *cachingManager) NumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	type response struct {
		f float64
		b bool
	}

	result := m.caching("NumArgsFrequency", sym.String()+"_"+strconv.Itoa(numArgs), func() interface{} {
		f, b := m.mgr.NumArgsFrequency(sym, numArgs)
		return response{f: f, b: b}
	}).(response)
	return result.f, result.b
}

func (m *cachingManager) Documentation(sym Symbol) *docs.Entity {
	return m.caching("Documentation", sym.String(), func() interface{} {
		return m.mgr.Documentation(sym)
	}).(*docs.Entity)
}

func (m *cachingManager) SymbolCounts(sym Symbol) *symbolcounts.Counts {
	return m.caching("SymbolCounts", sym.String(), func() interface{} {
		return m.mgr.SymbolCounts(sym)
	}).(*symbolcounts.Counts)
}

func (m *cachingManager) Kwargs(sym Symbol) *KeywordArgs {
	return m.caching("Kwargs", sym.String(), func() interface{} {
		return m.mgr.Kwargs(sym)
	}).(*KeywordArgs)
}

func (m *cachingManager) TruthyReturnTypes(sym Symbol) []TruthySymbol {
	return m.caching("TruthyReturnTypes", sym.String(), func() interface{} {
		return m.mgr.TruthyReturnTypes(sym)
	}).([]TruthySymbol)
}

func (m *cachingManager) ReturnTypes(sym Symbol) []Symbol {
	return m.caching("ReturnTypes", sym.String(), func() interface{} {
		return m.mgr.ReturnTypes(sym)
	}).([]Symbol)
}

func (m *cachingManager) PathSymbol(path pythonimports.DottedPath) (Symbol, error) {
	v, err := m.cachingError("PathSymbol", path.String(), func() (interface{}, error) {
		return m.mgr.PathSymbol(path)
	})
	if err != nil {
		return Symbol{}, err
	}
	return v.(Symbol), nil
}

func (m *cachingManager) PathSymbols(ctx kitectx.Context, path pythonimports.DottedPath) ([]Symbol, error) {
	ctx.CheckAbort()

	v, err := m.cachingError("PathSymbols", path.String(), func() (interface{}, error) {
		return m.mgr.PathSymbols(ctx, path)
	})
	if err != nil {
		return nil, err
	}
	return v.([]Symbol), nil
}

func (m *cachingManager) NewSymbol(dist keytypes.Distribution, path pythonimports.DottedPath) (Symbol, error) {
	v, err := m.cachingError("NewSymbol", dist.String()+"_"+path.String(), func() (interface{}, error) {
		return m.mgr.NewSymbol(dist, path)
	})
	if err != nil {
		return Symbol{}, err
	}
	return v.(Symbol), nil
}

func (m *cachingManager) Kind(s Symbol) keytypes.Kind {
	return m.caching("Kind", s.String(), func() interface{} {
		return m.mgr.Kind(s)
	}).(keytypes.Kind)
}

func (m *cachingManager) Type(s Symbol) (Symbol, error) {
	v, err := m.cachingError("Type", s.String(), func() (interface{}, error) {
		return m.mgr.Type(s)
	})
	if err != nil {
		return Symbol{}, err
	}
	return v.(Symbol), nil
}

func (m *cachingManager) Bases(s Symbol) []Symbol {
	return m.caching("Bases", s.String(), func() interface{} {
		return m.mgr.Bases(s)
	}).([]Symbol)
}

func (m *cachingManager) Children(s Symbol) ([]string, error) {
	v, err := m.cachingError("Children", s.String(), func() (interface{}, error) {
		return m.mgr.Children(s)
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (m *cachingManager) ChildSymbol(s Symbol, c string) (Symbol, error) {
	v, err := m.cachingError("ChildSymbol", strings.Join([]string{s.String(), c}, "_"), func() (interface{}, error) {
		return m.mgr.ChildSymbol(s, c)
	})
	if err != nil {
		return Symbol{}, err
	}
	return v.(Symbol), err
}

func (m *cachingManager) CanonicalSymbols(dist keytypes.Distribution) ([]Symbol, error) {
	v, err := m.cachingError("CanonicalSymbols", dist.String(), func() (interface{}, error) {
		return m.mgr.CanonicalSymbols(dist)
	})
	if err != nil {
		return nil, err
	}
	return v.([]Symbol), nil
}

func (m *cachingManager) TopLevels(dist keytypes.Distribution) ([]string, error) {
	v, err := m.cachingError("TopLevels", dist.String(), func() (interface{}, error) {
		return m.mgr.TopLevels(dist)
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}

func (m *cachingManager) Pkgs() []string {
	return m.caching("Pkgs", "", func() interface{} {
		return m.mgr.Pkgs()
	}).([]string)
}

func (m *cachingManager) DistsForPkg(pkg string) []keytypes.Distribution {
	return m.caching("DistsForPkg", pkg, func() interface{} {
		return m.mgr.DistsForPkg(pkg)
	}).([]keytypes.Distribution)
}

func (m *cachingManager) SigStats(sym Symbol) *SigStats {
	return m.caching("SigStats", sym.String(), func() interface{} {
		return m.mgr.SigStats(sym)
	}).(*SigStats)
}
