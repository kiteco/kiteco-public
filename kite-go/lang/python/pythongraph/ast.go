package pythongraph

import (
	"bytes"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

func nodeForWord(module *pythonast.Module, word pythonscanner.Word) pythonast.Node {
	var res pythonast.Node = module
	// find the deepest node overlapping word's boundaries
	pythonast.Inspect(module, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		if word.End <= n.Begin() || word.Begin >= n.End() {
			return false
		}
		res = n
		return true
	})
	return res
}

func wordsForNodes(module *pythonast.Module, words []pythonscanner.Word) map[pythonast.Node][]pythonscanner.Word {
	res := make(map[pythonast.Node][]pythonscanner.Word)
	for _, word := range words {
		// skipped words get assigned to the module by default
		n := pythonast.Node(module)
		if !skipWord(word) {
			n = nodeForWord(module, word)
		}
		res[n] = append(res[n], word)
	}
	// consider relevant "synthetic" words
	pythonast.Inspect(module, func(n pythonast.Node) bool {
		if n, _ := n.(*pythonast.AttributeExpr); n != nil {
			switch {
			// these can come from the approx & partial parsers
			case n.Attribute.Token == pythonscanner.Ident && n.Attribute.Literal == "":
				fallthrough
			// these come from the primary parser
			case n.Attribute.Token == pythonscanner.Cursor:
				res[n] = append(res[n], *n.Attribute)
			}
		}
		return true
	})

	for _, selected := range res {
		sort.Slice(selected, func(i, j int) bool {
			si, sj := selected[i], selected[j]
			if si.Begin == sj.Begin {
				if si.End == sj.End {
					if sj.Token == pythonscanner.EOF {
						return true
					} else if si.Token == pythonscanner.EOF {
						return false
					}
				}
				return si.End < sj.End
			}
			return si.Begin < sj.Begin
		})
	}

	return res
}

func skipWord(w pythonscanner.Word) bool {
	switch {
	case w.Token.IsKeyword():
		return false
	case w.Token.IsLiteral():
		return false
	case w.Token.IsOperator():
		return false
	case w.Token == pythonscanner.EOF:
		return false
	case w.Token == pythonscanner.Cursor:
		return false
	default:
		return true
	}
}

func printNode(n pythonast.Node) string {
	if pythonast.IsNil(n) {
		return "<nil>"
	}
	var b bytes.Buffer
	pythonast.PrintPositions(n, &b, "\t")
	return b.String()
}

func unpackParamName(param *pythonast.Parameter) []*pythonast.NameExpr {
	var names []*pythonast.NameExpr
	pythonast.Inspect(param.Name, func(n pythonast.Node) bool {
		if name, ok := n.(*pythonast.NameExpr); ok {
			names = append(names, name)
		}
		return true
	})
	return names
}

func namesFromExprs(exprs ...pythonast.Expr) []*pythonast.NameExpr {
	var names []*pythonast.NameExpr
	for _, expr := range exprs {
		if pythonast.IsNil(expr) {
			continue
		}

		pythonast.Inspect(expr, func(n pythonast.Node) bool {
			if name, ok := n.(*pythonast.NameExpr); ok {
				names = append(names, name)
			}
			return true
		})
	}

	return names
}

// TODO: kind of expensive, could probably just get away with checking node begin/end
func isChildOf(parent pythonast.Node, child pythonast.Node) bool {
	if pythonast.IsNil(parent) {
		return false
	}

	var found bool
	pythonast.Inspect(parent, func(n pythonast.Node) bool {
		if n == child {
			found = true
		}
		if found {
			return false
		}
		return true
	})

	return found
}

// findAndReplaceAST finds and replaces the given nodes inside of the given AST root.
// It does not update word positions, so the resulting AST will be inconsistent. To update word positions, use pythonast.Replace.
// TODO: test this
func findAndReplaceAST(root pythonast.Node, find pythonast.Node, replace pythonast.Node) (pythonast.Node, func()) {
	h := &astReplacer{
		find:      find,
		replace:   replace,
		curParent: root,
	}
	root.Iterate(h)
	return h.parent, h.undo
}

type astReplacer struct {
	// inputs
	find    pythonast.Node
	replace pythonast.Node
	// state
	curParent pythonast.Node
	// results
	found  bool
	undo   func()
	parent pythonast.Node
}

func (h *astReplacer) VisitSlice(s pythonast.NodeSliceRef) {
	pythonast.VisitNodeSlice(h, s)
}

func (h *astReplacer) withCurParent(n pythonast.Node) func() {
	curParent := h.curParent
	h.curParent = n
	return func() { h.curParent = curParent }
}
func (h *astReplacer) VisitNode(r pythonast.NodeRef) {
	if h.found {
		return
	}

	n := r.Lookup()
	if pythonast.IsNil(n) {
		return
	}

	if n == h.find {
		h.found = true

		// do the replacement
		if r.Assign(h.replace) {
			h.undo = func() { r.Assign(n) }
			h.parent = h.curParent
		}
		return
	}

	// ideally we'd return if n.End() < h.find.Begin() || n.Begin() > h.find.End()
	// but the provided AST might have inconsistent begin/end positions

	defer h.withCurParent(n)()
	n.Iterate(h)
}
func (h *astReplacer) VisitWord(r **pythonscanner.Word) {}
