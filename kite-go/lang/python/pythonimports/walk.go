package pythonimports

// WalkFunc is the type passed to pythonimports.Walk
type WalkFunc func(*Node) bool

// Walk executes a function on each node reachable from a given start node.
func Walk(root *Node, f WalkFunc) {
	walk(root, f, make(map[*Node]bool))
}

func walk(node *Node, f WalkFunc, seen map[*Node]bool) {
	if node == nil {
		return
	}
	if _, found := seen[node]; found {
		return
	}

	seen[node] = true
	if !f(node) {
		return
	}

	walk(node.Type, f, seen)
	for _, child := range node.Members {
		walk(child, f, seen)
	}
	for _, base := range node.Bases {
		walk(base, f, seen)
	}
}
