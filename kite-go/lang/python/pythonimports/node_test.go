package pythonimports

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenNodes(t *testing.T) {
	nodes := make([]Node, 3)
	nodes[0].Type = &nodes[1]
	nodes[0].Members = map[string]*Node{"x": &nodes[2]}

	flat, _ := FlattenNodes(nodes)
	require.Len(t, flat, 3)
	assert.EqualValues(t, flat[0].NodeInfo, nodes[0].NodeInfo)
	assert.EqualValues(t, 1, flat[0].TypeID)
	require.Len(t, flat[0].Members, 1)
	assert.Equal(t, "x", flat[0].Members[0].Attr)
}

func TestBaseLoopAttrs(t *testing.T) {
	node := &Node{
		Members: make(map[string]*Node),
	}
	base := &Node{
		Members: make(map[string]*Node),
		Bases:   []*Node{node},
	}
	node.Bases = append(node.Bases, base)

	require.Len(t, node.Attrs(), 0)
}

func TestBaseLoopAttrsByKind(t *testing.T) {
	node := &Node{
		NodeInfo: NodeInfo{
			Classification: Object,
		},
		Members: make(map[string]*Node),
	}
	base := &Node{
		NodeInfo: NodeInfo{
			Classification: Object,
		},
		Members: make(map[string]*Node),
		Bases:   []*Node{node},
	}
	node.Bases = append(node.Bases, base)

	require.Len(t, node.AttrsByKind(), 0)
}
