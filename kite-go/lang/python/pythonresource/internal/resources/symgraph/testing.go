package symgraph

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/stringutil"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

func mustValue(t reflect.Type, rand *rand.Rand) interface{} {
	v, ok := quick.Value(t, rand)
	if !ok {
		panic("failed to generate random value")
	}
	return v.Interface()
}

// Generate implements quick.Generator for generating random Graphs for testing
func (*Graph) Generate(rand *rand.Rand, size int) reflect.Value {
	top := mustValue(reflect.TypeOf(""), rand).(string)

	g := make(Graph)
	g[top] = make([]Node, size)

	for _, n := range g[top] {
		parts := []string{top}
		parts = append(parts, mustValue(reflect.TypeOf(([]string)(nil)), rand).([]string)...)
		n.Canonical = CastDottedPath(pythonimports.NewPath(parts...))

		n.Children = make(map[uint64]NodeRef)
		numChildren := rand.Intn(size)
		for i := 0; i < numChildren; i++ {
			name := mustValue(reflect.TypeOf(""), rand).(string)

			var val NodeRef
			ty := rand.Intn(2)
			if ty == 0 { // Internal
				val.Internal = rand.Intn(size)
			} else { // External
				parts := mustValue(reflect.TypeOf(([]string)(nil)), rand).([]string)
				val.External = CastDottedPath(pythonimports.NewPath(parts...))
			}

			n.Children[stringutil.ToUint64(name)] = val
		}
	}

	return reflect.ValueOf(&g)
}

// MockGraph builds a mock symbol graph with the provided partial nodes (i.e. with no children filled in)
func MockGraph(t testing.TB, paths []pythonimports.DottedPath, infos map[pythonimports.Hash]keytypes.TypeInfo) *Graph {
	g := make(Graph)

	for _, path := range paths {
		tl := path.Head()
		index := g[tl]

		if len(index) == 0 {
			index = append(index, Node{
				Canonical: CastDottedPath(pythonimports.NewDottedPath(tl)),
				Children:  make(map[uint64]NodeRef),
			})
		}

		cur := &index[0]
		for i := 1; i < len(path.Parts); i++ {
			part := path.Parts[i]
			next, ok := cur.Children[stringutil.ToUint64(part)]
			if !ok {
				next.Internal = len(index)
				cur.Children[stringutil.ToUint64(part)] = NodeRef{Internal: next.Internal}
				index = append(index, Node{
					Canonical: CastDottedPath(pythonimports.NewPath(path.Parts[:i+1]...)),
					Children:  make(map[uint64]NodeRef),
				})
			}

			cur = &index[int(next.Internal)]
		}

		info := infos[path.Hash]
		cur.Kind = Kind(info.Kind)
		// all types & bases are encoded as an external reference for simplicity;
		// this works but is inefficient, and will not test the internal reference code path (but that is tested with child lookups, so it's ok)
		if !info.Type.Empty() {
			cur.Type = &NodeRef{External: CastDottedPath(info.Type)}
		}
		for _, basePath := range info.Bases {
			cur.Bases = append(cur.Bases, NodeRef{External: CastDottedPath(basePath)})
		}

		g[tl] = index
	}

	return &g
}
