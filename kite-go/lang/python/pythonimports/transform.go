package pythonimports

// A NodeTransformer transforms nodes in the import graph
type NodeTransformer interface {
	// Transform maps nodes to nodes
	Transform(*Node) *Node
}

// Transform performs a recursive transform on an import graph. It first executes
// the transformer on the input node, then replaces each referenced node with
// the result of calling the transformer on them, and so on recursively. Eventually
// the transformer will be applied to all ndoes reachable from the input node. The
// transformer is only called once per node, even for nodes that are referenced in
// multiple places within the graph.
//
// The second return parameter is a map from nodes to the result of executing the
// transformer on them.
func Transform(root *Node, transformer NodeTransformer) (*Node, map[*Node]*Node) {
	cache := make(map[*Node]*Node)
	cache[nil] = nil
	return transform(root, transformer, cache), cache
}

func transform(x *Node, t NodeTransformer, cache map[*Node]*Node) *Node {
	if y, found := cache[x]; found {
		return y
	}

	y := t.Transform(x)
	cache[x] = y

	if y != nil {
		y.Type = transform(y.Type, t, cache)
		for attr, child := range y.Members {
			y.Members[attr] = transform(child, t, cache)
		}
		for i, base := range y.Bases {
			y.Bases[i] = transform(base, t, cache)
		}
	}
	return y
}
