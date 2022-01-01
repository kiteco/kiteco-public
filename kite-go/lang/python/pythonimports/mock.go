package pythonimports

import (
	"sort"
	"strings"
)

type graphBuilder struct {
	nodeByName map[string]*FlatNode
}

func newGraphBuilder() *graphBuilder {
	return &graphBuilder{
		nodeByName: make(map[string]*FlatNode),
	}
}

func (b *graphBuilder) add(name string, kind Kind) *FlatNode {
	node := b.nodeByName[name]
	if node != nil {
		return node
	}

	node = mockNode(len(b.nodeByName), name, kind)
	b.nodeByName[name] = node

	pos := strings.LastIndex(name, ".")
	if pos != -1 {
		parent := b.add(name[:pos], Module)
		parent.Members = append(parent.Members, FlatMember{
			Attr:   name[pos+1:],
			NodeID: node.ID,
		})
	}
	return node
}

func (b *graphBuilder) build() *Graph {
	var nodes []*FlatNode
	for name, node := range b.nodeByName {
		// set types for builtin nodes to be self reference.
		if _, found := mockBuiltins[name]; found {
			node.TypeID = b.nodeByName[name].ID
		}
		nodes = append(nodes, node)
	}
	return NewGraphFromNodes(nodes)
}

func mockNode(id int, name string, kind Kind) *FlatNode {
	return &FlatNode{
		TypeID: -1,
		NodeInfo: NodeInfo{
			ID:             int64(id),
			CanonicalName:  NewDottedPath(name),
			Classification: kind,
		},
	}
}

// MockGraph returns a mock graph containing the nodes specified in names.
func MockGraph(names ...string) *Graph {
	b := newGraphBuilder()
	for name, kind := range mockBuiltins {
		b.add(name, kind)
	}
	for _, name := range names {
		b.add(name, Object)
	}
	return b.build()
}

// MockGraphFromMap returns a mock graph containing the nodes specified in names.
func MockGraphFromMap(kinds map[string]Kind) *Graph {
	b := newGraphBuilder()
	for name, kind := range mockBuiltins {
		b.add(name, kind)
	}
	// sort for determinism
	var names []string
	for name := range kinds {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b.add(name, kinds[name])
	}
	return b.build()
}
