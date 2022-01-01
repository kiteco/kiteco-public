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

	switch n := n.(type) {
	case *Doc:
		for _, child := range n.Content {
			Walk(v, child)
		}
	case *Section:
		for _, child := range n.Content {
			Walk(v, child)
		}
	case *Directive:
		for _, child := range n.Content {
			Walk(v, child)
		}
	case *Paragraph:
		for _, child := range n.Content {
			Walk(v, child)
		}
	case *Definition:
		for _, child := range n.Subject {
			Walk(v, child)
		}
		for _, child := range n.Type {
			Walk(v, child)
		}
		for _, child := range n.Content {
			Walk(v, child)
		}
	}
	v.Visit(nil)
}
