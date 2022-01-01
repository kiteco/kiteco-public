package main

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/stretchr/testify/assert"
)

func newFlatNode(id int64, cn string, members ...pythonimports.FlatMember) *node {
	return &node{FlatNode: pythonimports.FlatNode{
		NodeInfo: pythonimports.NodeInfo{
			ID:            id,
			CanonicalName: pythonimports.NewDottedPath(cn),
		},
		Members: members,
	}}
}

func TestVerifyCanonicalNames(t *testing.T) {
	n1 := newFlatNode(1, "a.b")
	n2 := newFlatNode(2, "a.b")
	n3 := newFlatNode(3, "a",
		pythonimports.FlatMember{
			Attr:   "b",
			NodeID: 2,
		},
	)

	graph := map[int64]*node{
		1: n1,
		2: n2,
		3: n3,
	}

	numDups := verifyCanonicalNames(graph)
	assert.Len(t, graph, 3) // should have one node removed
	assert.Equal(t, 1, numDups)

	assert.Equal(t, "", n1.CanonicalName.String())
}
