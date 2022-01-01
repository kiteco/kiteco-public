package pythongraph

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
)

// NodeLabels map a node id to a label
type NodeLabels map[NodeID]string

// Prepend ...
func (n NodeLabels) Prepend(nn NodeLabels) NodeLabels {
	if n == nil {
		n = make(NodeLabels)
	}

	for id, l := range nn {
		if old := n[id]; old != "" {
			n[id] = l + "::" + old
		} else {
			n[id] = l
		}
	}
	return n
}

// SavedNode is a wrapper around a node
type SavedNode struct {
	Node *Node

	// Level for hierarchical layout, -1 if the
	// graphing engine should determine the hierarchical structure
	Level int

	// Hover is the string to display when the user hovers over the ndoe
	Hover string
}

func (s *SavedNode) deepCopy() *SavedNode {
	return &SavedNode{
		Node:  s.Node.deepCopy(),
		Level: s.Level,
		Hover: s.Hover,
	}
}

// SavedEdge between SavedNodes
type SavedEdge struct {
	From    *SavedNode
	To      *SavedNode
	Type    EdgeType
	Forward bool
}

// SavedGraph is a graph with debug info
type SavedGraph struct {
	Nodes []*SavedNode
	Edges []*SavedEdge
}

func (sg *SavedGraph) deepCopy() (*SavedGraph, map[*SavedNode]*SavedNode) {
	oldToNew := make(map[*SavedNode]*SavedNode)
	var newNodes []*SavedNode
	for _, old := range sg.Nodes {
		new := old.deepCopy()
		oldToNew[old] = new
		newNodes = append(newNodes, new)
	}

	var newEdges []*SavedEdge
	for _, old := range sg.Edges {
		newEdges = append(newEdges, &SavedEdge{
			From:    oldToNew[old.From],
			To:      oldToNew[old.To],
			Type:    old.Type,
			Forward: old.Forward,
		})
	}

	return &SavedGraph{
		Nodes: newNodes,
		Edges: newEdges,
	}, oldToNew
}

// SavedBundle of debugging data
type SavedBundle struct {
	builder    *graphBuilder
	Label      string
	Graph      *SavedGraph
	NodeLabels NodeLabels
	EdgeValues map[string]EdgeValue
	Weights    ContextWeights
	Buffer     []byte
	AST        pythonast.Node

	// Used for saving bundles as part of beam search
	// TODO: clean this up a bit
	Entries []SavedBundle

	Children []SavedBundle

	Prob float32
}

// Saver stores debuggging data when generating a training sample or running inference
type Saver interface {
	Save(SavedBundle)
}

func bufferBundle(label string, buffer []byte) SavedBundle {
	return SavedBundle{
		Label:  label,
		Buffer: buffer,
	}
}

func nodeLabels(n *Node, label string) map[NodeID]string {
	return map[NodeID]string{n.ID: label}
}

func save(s Saver, sb SavedBundle) {
	saveScope(s, sb, nil)
}

func saveScope(s Saver, sb SavedBundle, scope scope) {
	if s == nil {
		return
	}

	if sb.NodeLabels == nil {
		sb.NodeLabels = make(map[NodeID]string)
	}

	if sb.builder != nil {
		sb.Graph = newSavedGraphFromBuilder(sb.builder)

		if sb.AST == nil {
			sb.AST = sb.builder.a.RAST.Root
		}
	}

	if sb.AST != nil {
		sb.AST = pythonast.DeepCopy(sb.AST)[sb.AST]
	}
	sb.Buffer = append([]byte{}, sb.Buffer...)

	if sb.Graph != nil {
		sb.Graph, _ = sb.Graph.deepCopy()
	}

	s.Save(sb)
}

func newSavedGraphFromBuilder(builder *graphBuilder) *SavedGraph {
	// make new saved nodes
	var nodes []*SavedNode
	origToSaved := make(map[*Node]*SavedNode)
	for _, n := range builder.nodes {
		ns := &SavedNode{
			Node: n,
		}
		nodes = append(nodes, ns)
		origToSaved[n] = ns
	}

	// Layout rough idea
	// - level 0 (top) -- module
	// - level 1 -- sof
	// - level 2 -- first statement
	// - level 2 + children of fist statement  -- second statement
	// TODO: within a statement level it would be awesome if we could go from left to right
	// based on the position in the line
	// start offset 1 so SOF can go at level 1
	offset := 1
	for _, stmt := range builder.a.RAST.Root.Body {
		var maxDepth int
		depth := 1
		pythonast.InspectEdges(stmt, func(p, c pythonast.Node, _ string) bool {
			if depth > maxDepth {
				maxDepth = depth
			}

			if pythonast.IsNil(c) {
				// done with parent
				if _, ok := p.(pythonast.Stmt); ok {
					depth--
				}
				return false
			}

			gn := builder.astNodes[c]
			if s := origToSaved[gn]; s != nil {
				// need nil check in case node was pruned (since pruning does not update astNodes map)
				s.Level = depth + offset
			}

			// handle words for node
			words := builder.wordsForNodes[c]
			// move word nodes attached to a statement to be one level down
			// so they are on the same level as the rest of the children of the stmt
			level := depth + offset
			if _, ok := c.(pythonast.Stmt); ok {
				level++
			}
			for _, w := range words {
				if n := builder.wordNodes[w]; n != nil {
					if s := origToSaved[n]; s != nil {
						// need nil check in case node was pruned (since pruning does not update wordNodes map)
						s.Level = level
					}
				}
			}

			if _, ok := c.(pythonast.Stmt); ok {
				depth++
			}
			return true
		})
		offset += maxDepth
	}

	// handle SOF and EOF nodes
	for _, n := range builder.nodes {
		if n.Attrs.Literal == traindata.EOFMarker {
			// put eof in the bottom
			s := origToSaved[n]
			s.Level = offset + 1
		}
		if n.Attrs.Literal == traindata.SOFMarker {
			// put sof at level 1
			s := origToSaved[n]
			s.Level = 1
		}
	}

	// make edges
	var edges []*SavedEdge
	for _, edge := range builder.edges {
		edges = append(edges, &SavedEdge{
			From:    origToSaved[edge.from],
			To:      origToSaved[edge.to],
			Type:    edge.Type,
			Forward: edge.Forward,
		})
	}

	return &SavedGraph{
		Nodes: nodes,
		Edges: edges,
	}
}
