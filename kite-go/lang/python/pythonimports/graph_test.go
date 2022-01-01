package pythonimports

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphFind(t *testing.T) {
	graph := MockGraph("os.path.join")

	n, err := graph.Find("os.path")
	require.NoError(t, err)
	assert.Equal(t, "os.path", n.CanonicalName.String())

	n, err = graph.Find("os.path.join")
	require.NoError(t, err)
	assert.Equal(t, "os.path.join", n.CanonicalName.String())

	n, err = graph.Find("sys")
	require.Error(t, err)
}

func TestGraphWalk(t *testing.T) {
	type expectedWalk struct {
		prefix string
		walk   []string
		err    bool
	}
	testCases := []expectedWalk{
		expectedWalk{"os", []string{"os", "os.path", "os.path.join"}, false},
		expectedWalk{"os.path", []string{"os.path", "os.path.join"}, false},
		expectedWalk{"os.path.join", []string{"os.path.join"}, false},
		expectedWalk{"sys", nil, true},
	}

	graph := MockGraph("os.path.join")
	for _, tc := range testCases {
		var names []string
		err := graph.Walk(tc.prefix, func(name string, node *Node) bool {
			names = append(names, name)
			return true
		})
		if tc.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		assert.Equal(t, tc.walk, names)
	}
}

// --

var (
	graph     *Graph
	graphOnce sync.Once
)

func loadGraph() (*Graph, error) {
	var err error
	graphOnce.Do(func() {
		graph, err = NewGraph(DefaultImportGraph)
	})
	return graph, err
}

func BenchmarkFind(b *testing.B) {
	g, err := loadGraph()
	if err != nil {
		b.Fatalf("error loading graph: %v", err)
	}

	// collect some canonical names to use for random access into the graph
	var queries []string
	for i := range g.Nodes {
		node := &g.Nodes[i]
		if !node.CanonicalName.Empty() {
			if _, err := g.Find(node.CanonicalName.String()); err == nil {
				queries = append(queries, node.CanonicalName.String())
			}
		}
	}

	if len(queries) == 0 {
		b.Fatalf("did not find any nodes with valid canonical names")
	}

	// shuffle the list
	rand.Seed(0)
	for i := range queries {
		j := rand.Intn(i + 1)
		queries[i], queries[j] = queries[j], queries[i]
	}

	// perform a fixed number of queries
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Find(queries[i%len(queries)])
	}
}

func TestGraphRoot(t *testing.T) {
	var nodes []*FlatNode
	pkgs := []string{"foo", "bar"}
	for i, pkg := range pkgs {
		node := &FlatNode{
			NodeInfo: NodeInfo{
				ID:             int64(i),
				CanonicalName:  NewDottedPath(pkg),
				Classification: Module,
			},
		}
		nodes = append(nodes, node)
	}

	graph := NewGraphFromNodes(nodes)

	root := graph.Root
	require.NotNil(t, root)
	require.Len(t, root.Members, 2)
	_, haveFoo := root.Members["foo"]
	assert.True(t, haveFoo)
	_, haveBar := root.Members["bar"]
	assert.True(t, haveBar)
}
