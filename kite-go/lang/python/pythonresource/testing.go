package pythonresource

import (
	"strings"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/returntypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symgraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/require"
)

func transformKind(k pythonimports.Kind) keytypes.Kind {
	if k == pythonimports.Root {
		return keytypes.NoneKind
	}
	return keytypes.Kind(k)
}

var (
	python3MockBuiltins map[string]keytypes.TypeInfo
)

func init() {
	python3MockBuiltins = map[string]keytypes.TypeInfo{
		"builtins": keytypes.TypeInfo{Kind: keytypes.ModuleKind},
	}

	for attr, ti := range builtinAttrs {
		attr3 := attr
		if !strings.HasPrefix(attr, "types") {
			attr3 = "builtins." + attr
		}

		python3MockBuiltins[attr3] = ti
	}
}

// for now, duplicate of pythonimports/builtincache.go:mockBuiltins
// once the import graph goes away, this will be the source of truth
var builtinAttrs = map[string]keytypes.TypeInfo{
	"bool":       keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"int":        keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"float":      keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"complex":    keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"str":        keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"list":       keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"tuple":      keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"dict":       keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"set":        keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"print":      keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"type":       keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"object":     keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"map":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"super":      keytypes.TypeInfo{Kind: keytypes.TypeKind},
	"all":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"any":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"bin":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"memoryview": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"callable":   keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"chr":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"compile":    keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"credits":    keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"delattr":    keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"dir":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"eval":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"exit":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"format":     keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"globals":    keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"hash":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"hex":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"id":         keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"input":      keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"isinstance": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"issubclass": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"len":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"license":    keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"oct":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"open":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"ord":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"quit":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"range":      keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"repr":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"round":      keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"setattr":    keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"vars":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"divmod":     keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"enumerate":  keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"getattr":    keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"max":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"min":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"next":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"iter":       keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"pow":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"reversed":   keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"sorted":     keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"sum":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"filter":     keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	"zip":        keytypes.TypeInfo{Kind: keytypes.FunctionKind},
}

var defaultTestManager *manager

// DefaultTestManager returns a test manager, using bindata-embedded production data (datadeps) for a handful of distributions
func DefaultTestManager(t testing.TB) *manager {
	if defaultTestManager == nil {
		require.NoError(t, datadeps.UseAssetFileMap())

		// load resource manager with only the following distributions
		opts := DefaultLocalOptions
		opts.CacheSize = 0
		opts.Dists = []keytypes.Distribution{
			keytypes.BuiltinDistribution3,
			keytypes.AlembicDistribution,
			keytypes.RequestsDistribution,
		}
		rm, errc := NewManager(opts)
		require.NoError(t, <-errc)

		defaultTestManager = rm.(*manager)
	}
	return defaultTestManager
}

// InfosFromKinds translates an argument to pythonimports.MockGraphFromMap to an argument to MockManager
func InfosFromKinds(kinds map[string]pythonimports.Kind) map[string]keytypes.TypeInfo {
	out := make(map[string]keytypes.TypeInfo)
	for p, k := range kinds {
		switch k {
		case pythonimports.Function:
			out[p] = keytypes.TypeInfo{Kind: keytypes.FunctionKind}
		case pythonimports.Type:
			out[p] = keytypes.TypeInfo{Kind: keytypes.TypeKind}
		case pythonimports.Module:
			out[p] = keytypes.TypeInfo{Kind: keytypes.ModuleKind}
		case pythonimports.Descriptor:
			out[p] = keytypes.TypeInfo{Kind: keytypes.DescriptorKind}
		case pythonimports.Object:
			out[p] = keytypes.TypeInfo{Kind: keytypes.ObjectKind}
		// ignore these cases:
		case pythonimports.Root:
		case pythonimports.None:
		}
	}
	return out
}

// MockManager returns a mock Manager populated with a symbol graph containing the provided paths;
// paths provided in pathsWithInfo will be associated with the corresponding keytypes.Kind, while
// paths provided in the variable pathStrs argument will be associated with keytypes.NoneKind
func MockManager(t testing.TB, pathsWithInfo map[string]keytypes.TypeInfo, pathStrs ...string) *manager {
	pathSet := make(map[pythonimports.Hash]struct{})
	var paths []pythonimports.DottedPath
	addPath := func(p pythonimports.DottedPath) {
		if p.Empty() {
			return
		}
		if _, ok := pathSet[p.Hash]; ok {
			return
		}
		paths = append(paths, p)
		pathSet[p.Hash] = struct{}{}
	}

	for _, pathStr := range pathStrs {
		addPath(pythonimports.NewDottedPath(pathStr))
	}

	infos := make(map[pythonimports.Hash]keytypes.TypeInfo)
	for _, m := range []map[string]keytypes.TypeInfo{python3MockBuiltins, pathsWithInfo} {
		for pathStr, info := range m {
			path := pythonimports.NewDottedPath(pathStr)
			addPath(path)
			for _, basePath := range info.Bases {
				addPath(basePath)
			}
			addPath(info.Type)

			infos[path.Hash] = info
		}
	}

	shardedPaths := make(map[keytypes.Distribution][]pythonimports.DottedPath)
	for _, path := range paths {
		dist := keytypes.Distribution{Name: path.Head()}
		shardedPaths[dist] = append(shardedPaths[dist], path)
	}

	cache, err := lru.New(50)
	if err != nil {
		panic(err)
	}

	var dists []keytypes.Distribution
	index := make(distidx.Index)
	manifest := make(manifest.Manifest)

	now := time.Now()
	for dist, paths := range shardedPaths {
		dists = append(dists, dist)
		manifest[dist] = make(resources.LocatorGroup)

		tl := dist.Name
		index[tl] = append(index[tl], dist)

		rg := resources.EmptyGroup()
		rg.SymbolGraph = symgraph.MockGraph(t, paths, infos)
		cache.Add(dist, dynamicDistribution{
			ResourceGroup: rg,
			LoadedAt:      now,
		})
	}

	return &manager{
		index:    index,
		cache:    cache,
		manifest: manifest,
	}
}

// MockReturnType adds a return type (retStr) to the given symbol (pathStr).
// It should be used with a Manager created via MockManager.
func (rm *manager) MockReturnType(t testing.TB, pathStr string, retStr string) {
	path := pythonimports.NewDottedPath(pathStr)
	dd, ok := rm.cache.Get(keytypes.Distribution{Name: path.Head()})
	require.True(t, ok)

	types := dd.(dynamicDistribution).ResourceGroup.ReturnTypes
	typeSet := types[uint64(path.Hash)]
	if typeSet == nil {
		typeSet = make(returntypes.Entity)
		types[uint64(path.Hash)] = typeSet
	}
	typeSet[retStr] = keytypes.StubTruthiness
}

// MockSymbolCounts associates counts with the given symbol path for a "mock" manager.
func (rm *manager) MockSymbolCounts(t testing.TB, pathStr string, counts symbolcounts.Counts) {
	path := pythonimports.NewDottedPath(pathStr)
	dist := keytypes.Distribution{Name: path.Head()}
	dd, ok := rm.cache.Get(dist)
	require.True(t, ok)
	dd.(dynamicDistribution).ResourceGroup.SymbolCounts[pathStr] = counts
}
