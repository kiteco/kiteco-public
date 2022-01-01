package pythongraph

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

func (b *graphBuilder) RemoveEdge(from, to *Node, t EdgeType) {
	delete(from.outgoing, to)
	delete(to.outgoing, from)

	edges := make([]*Edge, 0, len(b.edges))
	for _, edge := range b.edges {
		if edge.Type == t && edge.from == from && edge.to == to {
			continue
		}

		if edge.Type == t && edge.to == from && edge.from == to {
			continue
		}

		edges = append(edges, edge)
	}

	b.edges = edges
}

// TODO:
// - delete old next token edge -- hard because reverting needs to add back this edge
// - add new next token edge -- hard because reverting the graph needs to update node.outgoing
// new terminal node
func (b *graphBuilder) AddWordTerminal(prevWord, parent *Node, word pythonscanner.Word) *Node {
	node := &Node{
		ID:   NodeID(len(b.nodes)),
		Type: ASTTerminalNode,
		Attrs: Attributes{
			Literal: traindata.WordLiteral(word),
			Types:   []string{traindata.NAType},
			Token:   word.Token,
		},
		outgoing: make(nodeSet),
	}

	b.nodes = append(b.nodes, node)

	b.edges = append(b.edges, makeEdges(prevWord, node, NextToken)...)
	b.edges = append(b.edges, makeEdges(parent, node, ASTChild)...)

	return node
}

// TODO:
// - delete old next token edge -- hard because reverting needs to add back this edge
// - add new next token edge -- hard because reverting the graph needs to update node.outgoing
// new terminal node
func (b *graphBuilder) AddIdentTerminal(prevWord, parent *Node, lit string, ts ...string) *Node {
	node := &Node{
		ID:   NodeID(len(b.nodes)),
		Type: ASTTerminalNode,
		Attrs: Attributes{
			Literal: lit,
			Types:   ts,
			Token:   pythonscanner.Ident,
		},
		outgoing: make(nodeSet),
	}

	b.nodes = append(b.nodes, node)

	b.edges = append(b.edges, makeEdges(prevWord, node, NextToken)...)
	b.edges = append(b.edges, makeEdges(parent, node, ASTChild)...)

	return node
}
