package ast

// Inspect traverses an AST in depth-first order: It starts by calling
// f(node); node must not be nil. If f returns true, Inspect invokes f
// recursively for each of the non-nil children of node, followed by a
// call of f(nil).
func Inspect(node *Node, f func(*Node) bool) {
	if f(node) {
		for _, child := range node.Children {
			Inspect(child, f)
		}
		f(nil)
	}
}
