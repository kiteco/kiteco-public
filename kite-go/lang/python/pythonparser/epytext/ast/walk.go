package ast

// Visitor defines a Visit method invoked for each node encountered
// by Walk. If the result visitor w is not nil, Walk visits each of
// the children of node with the visitor w, followed by a call of
// w.Visit(nil).
type Visitor interface {
	Visit(n Node) (w Visitor)
}

// Walk traverses an AST in depth-first order: It starts by calling
// v.Visit(n); n must not be nil. If the visitor w returned by
// v.Visit(n) is not nil, Walk is invoked recursively with visitor
// w for each of the non-nil children of n, followed by a call of
// w.Visit(nil).
func Walk(v Visitor, n Node) {
	if v = v.Visit(n); v == nil {
		return
	}

	if nn, ok := n.(NestingNode); ok {
		for _, child := range nn.children() {
			Walk(v, child)
		}
	}
	v.Visit(nil)
}
