package pythonast

import "sync"

// An EdgeVisitor is a function that is invoked for each edge in an AST. If the
// result is not nil, WalkEdges visits each of the children of node with the
// EdgeVisitor w, followed by a call of w.VisitEdge(node, nil, "").
//
// Each node in the AST will be the CHILD exactly once, including the root node,
// for which PARENT will be nil and FIELD will be the empty string.
//
// After processing each child for a node, VisitEdge will be called with CHILD set to
// nil to indicate that the parent node is done.
type EdgeVisitor interface {
	VisitEdge(parent, child Node, field string) (w EdgeVisitor)
}

// WalkEdges traverses each edge in an AST. An edge is a parent node together with
// one of its direct descendants. The root node will not be visited as a descendant
// of any node.
func WalkEdges(v EdgeVisitor, node Node) {
	walkEdge(v, nil, node, "")
}

// Helper functions for common node lists.

func walkNameList(v EdgeVisitor, parent Node, list []*NameExpr, field string) {
	for _, child := range list {
		walkEdge(v, parent, child, field)
	}
}

func walkExprList(v EdgeVisitor, parent Node, list []Expr, field string) {
	for _, child := range list {
		walkEdge(v, parent, child, field)
	}
}

func walkStmtList(v EdgeVisitor, parent Node, list []Stmt, field string) {
	for _, child := range list {
		walkEdge(v, parent, child, field)
	}
}

// walkEdge traverses the (parent, child) edges of an AST in depth-first order, starting
// with the edge described by (parent, child, field). For each node N in the AST, f.VisitEdge
// will be called with parameters (parent(N), N, field(N)) where parent(N) is the immediate
// parent of N and field(N) is the field of the parent node that contains N.
func walkEdge(v EdgeVisitor, parent, node Node, field string) {
	v = v.VisitEdge(parent, node, field)
	if v != nil {
		node.walk(v)
		v.VisitEdge(node, nil, "")
	}
}

// A Visitor is a function that is invoked for each node encountered by Walk.
// If the result visitor w is not nil, Walk visits each of the children
// of node with the visitor w, followed by a call of w.Visit(nil).
type Visitor interface {
	Visit(node Node) (w Visitor)
}

// Walk traverses an AST in depth-first order: It starts by calling
// v.Visit(node); node must not be nil. If the visitor w returned by
// v.Visit(node) is not nil, Walk is invoked recursively with visitor
// w for each of the non-nil children of node, followed by a call of
// w.Visit(nil).
func Walk(v Visitor, node Node) {
	vA := visitorAdapters.Get().(*visitorAdapter)
	*vA = visitorAdapter{v}
	WalkEdges(vA, node)
	visitorAdapters.Put(vA)
}

var visitorAdapters = sync.Pool{New: func() interface{} {
	return &visitorAdapter{}
}}

// A visitorAdapter adapts a Visitor to an EdgeVisitor
type visitorAdapter struct {
	Visitor
}

// VisitEdge calls Visit on the underlying visitor
func (v *visitorAdapter) VisitEdge(parent, child Node, field string) EdgeVisitor {
	w := v.Visit(child)
	if child == nil {
		// v is no longer needed, so free it for re-use
		visitorAdapters.Put(v)
		return nil
	}
	if w == nil {
		return nil
	}

	wA := visitorAdapters.Get().(*visitorAdapter)
	*wA = visitorAdapter{w}
	return wA
}

// inspector wraps an inspection function so that it can be passed to Visit instead
type inspector func(Node) bool

// Visit delegates to the underlying inspect function
func (f inspector) Visit(node Node) Visitor {
	if f(node) {
		return f
	}
	return nil
}

// Inspect traverses an AST in depth-first order: It starts by calling
// f(node); node must not be nil. If f returns true, Inspect invokes f
// recursively for each of the non-nil children of node, followed by a
// call of f(nil).
func Inspect(node Node, f inspector) {
	InspectEdges(node, func(parent, child Node, field string) bool { return f(child) })
}

// edgeInspector wraps an inspection function so that it can be passed to VisitEdges instead
type edgeInspector func(parent, child Node, field string) bool

// VisitEdge delegates to the underlying inspect function
func (f edgeInspector) VisitEdge(parent, child Node, field string) EdgeVisitor {
	if f(parent, child, field) {
		return f
	}
	return nil
}

// InspectEdges traverses the (parent, child) edges of an AST in depth-first order. For each
// node N in the AST, f will be called with parameters (parent(N), N, field(N))
// where parent(N) is the immediate parent of N and field(N) is the field of the parent node
// that contains N. If f returns false then f will not be called for the children of N.
//
// NOTES:
// When N is the root node, parent(N) is nil and field(N) is the empty string.
// Hence the first call made by InspectEdges will always be f.VisitEdge(nil, node, "").
//
// Each node in the AST will be the CHILD exactly once.
//
// After processing each child for a node, VisitEdge will be called with CHILD set to
// nil to indicate that the parent node is done.
func InspectEdges(node Node, f edgeInspector) {
	WalkEdges(f, node)
}
