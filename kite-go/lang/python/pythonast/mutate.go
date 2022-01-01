package pythonast

import (
	"fmt"
	"go/token"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

// - destructively add offset to Words

type offseter token.Pos

func (h offseter) VisitSlice(s NodeSliceRef) {
	VisitNodeSlice(h, s)
}

func (h offseter) VisitNode(r NodeRef) {
	n := r.Lookup()
	if IsNil(n) {
		return
	}

	n.Iterate(h)

	// handle special case of BadExpr/BadStmt
	switch n := n.(type) {
	case *BadExpr:
		n.From += token.Pos(h)
		n.To += token.Pos(h)
	case *BadStmt:
		n.From += token.Pos(h)
		n.To += token.Pos(h)
	}
}

func (h offseter) VisitWord(r **pythonscanner.Word) {
	w := *r
	if w == nil {
		return
	}

	w.Begin += token.Pos(h)
	w.End += token.Pos(h)
}

// - replace a node with another

type replacer struct {
	// old.Begin() == new.Begin()
	old Node
	new Node
	off token.Pos // == new.End() - old.End()

	parent Node // for the error message
	err    error
}

func (h *replacer) VisitSlice(s NodeSliceRef) {
	if h.err != nil {
		return
	}

	// we do two passes to avoid allocating an extra slice in the
	// typical case in which no nodes are changed
	var remove bool
	for i := 0; i < s.Len(); i++ {
		ref := s.Get(i)
		if ref.Lookup() == h.old && IsNil(h.new) {
			remove = true
		} else {
			h.VisitNode(ref)
		}
	}

	if remove {
		nodes := make([]Node, 0, s.Len()-1)
		for i := 0; i < s.Len(); i++ {
			if n := s.Get(i).Lookup(); n != h.old {
				nodes = append(nodes, n)
			}
		}

		// we removed a node from the slice so we need to update
		// the slice
		if !s.Assign(nodes) {
			h.err = fmt.Errorf("error deleting node %v from %v", h.old, s)
		}
	}
}

func (h *replacer) VisitNode(r NodeRef) {
	if h.err != nil {
		return
	}

	n := r.Lookup()
	if IsNil(n) {
		return
	}

	if n == h.old {
		// do the replacement
		if !r.Assign(h.new) {
			h.err = fmt.Errorf("assigning incompatible Node type %T to NodeRef type %T of parent Node type %T", h.new, r, h.parent)
		}
		return
	}

	if n.End() < h.old.Begin() {
		// if the visited Node is strictly before the replacement, don't recurse
		// since the Word locations are already correct
		return
	}

	defer func(m Node) { h.parent = m }(h.parent)
	h.parent = n

	n.Iterate(h)
}

func (h *replacer) VisitWord(r **pythonscanner.Word) {
	if h.err != nil {
		return
	}

	w := *r
	if w == nil {
		return
	}

	if w.Begin >= h.old.End() {
		w.Begin += h.off
	}
	if w.End >= h.old.End() {
		w.End += h.off
	}
}

// Replace replaces old with new under the given root and updates all necessary Word positions.
// It may mutate the provided new Node without copying, as well as the provided root AST.
// If the type of the new Node is not compatible with the location of old, an error is returned. The AST may still be mutated.
// If new is nil then the specified old node is removed
func Replace(root Node, old Node, new Node) error {
	if !IsNil(new) {
		if new.Begin() != old.Begin() {
			// offset words to make new.Begin() == old.Begin()
			new.Iterate(offseter(old.Begin() - new.Begin()))
		}
	}

	var off token.Pos
	if !IsNil(new) {
		off = new.End() - old.End()
	} else {
		off = old.Begin() - old.End()
	}

	r := &replacer{
		old: old,
		new: new,
		off: off,
	}
	root.Iterate(r)
	return r.err
}

type sliceRemover struct {
	old   NodeSliceRef
	begin token.Pos
	end   token.Pos
	off   token.Pos
}

func (h *sliceRemover) VisitSlice(s NodeSliceRef) {
	if s.Equal(h.old) {
		s.Assign(nil)
		return
	}

	VisitNodeSlice(h, s)
}

func (h *sliceRemover) VisitNode(r NodeRef) {
	n := r.Lookup()
	if IsNil(n) {
		return
	}

	if n.End() < h.begin {
		// if the visited Node is strictly before the replacement, don't recurse
		// since the Word locations are already correct
		return
	}

	n.Iterate(h)
}

func (h *sliceRemover) VisitWord(r **pythonscanner.Word) {
	w := *r
	if w == nil {
		return
	}

	if w.Begin >= h.end {
		w.Begin += h.off
	}
	if w.End >= h.end {
		w.End += h.off
	}
}

// RemoveArgs from the provided call
func RemoveArgs(root Node, call *CallExpr) {
	h := &sliceRemover{
		old:   argumentSlice{&call.Args},
		begin: call.LeftParen.End,
		end:   call.RightParen.Begin,
		off:   call.LeftParen.End - call.RightParen.Begin,
	}

	root.Iterate(h)

	call.Commas = nil
}
