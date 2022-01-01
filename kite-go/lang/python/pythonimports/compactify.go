package pythonimports

import "fmt"

// CompactifyNodes transforms a slice of node pointers to a slice
// of nodes, updating internal pointers as necessary. It returns
// an error if there are any internal pointers to nodes that are
// not in the list.
func CompactifyNodes(ptrs []*Node) ([]Node, map[*Node]*Node) {
	// construct a map from old to new pointers
	nodes := make([]Node, len(ptrs))
	ptrmap := make(map[*Node]*Node)
	for i, ptr := range ptrs {
		if ptr == nil {
			panic(fmt.Errorf("nodes[%d] was nil", i))
		}
		// only copy NodeInfo, not the members, bases, etc, since maps and slices
		// are reference types and we would then clobber the original data later
		ptrmap[ptr] = &nodes[i]
	}

	// construct the forwarding function
	forward := func(old *Node) *Node {
		if n, found := ptrmap[old]; found {
			return n
		}
		// return the original node pointer for nodes outside the local graph
		return old
	}
	for i, ptr := range ptrs {
		nodes[i].NodeInfo = ptr.NodeInfo
		nodes[i].Type = forward(ptr.Type)
		for _, base := range ptr.Bases {
			nodes[i].Bases = append(nodes[i].Bases, forward(base))
		}
		nodes[i].Members = make(map[string]*Node, len(ptr.Members))
		for attr, child := range ptr.Members {
			nodes[i].Members[attr] = forward(child)
		}
	}
	return nodes, ptrmap
}
