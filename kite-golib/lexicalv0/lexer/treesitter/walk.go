package treesitter

import (
	"fmt"
	"io"
	"strings"

	sitter "github.com/kiteco/go-tree-sitter"
)

// Visitor defines the method to implement to walk a tree-sitter AST.
type Visitor interface {
	Visit(n *sitter.Node) Visitor
}

// Walk walks a tree-sitter AST using Visitor.
func Walk(v Visitor, n *sitter.Node) {
	if v = v.Visit(n); v == nil {
		return
	}

	for i := 0; i < int(n.ChildCount()); i++ {
		Walk(v, n.Child(i))
	}
	v.Visit(nil)
}

type inspector func(*sitter.Node) bool

func (f inspector) Visit(node *sitter.Node) Visitor {
	if f(node) {
		return f
	}
	return nil
}

// Inspect traverses an AST in depth-first order: It starts by calling
// f(node); node must not be nil. If f returns true, Inspect invokes f
// recursively for each of the non-nil children of node, followed by a
// call of f(nil).
func Inspect(node *sitter.Node, f func(*sitter.Node) bool) {
	Walk(inspector(f), node)
}

// Print the provided ast to the specified writer
func Print(root *sitter.Node, w io.Writer, indent string) {
	var depth int
	Inspect(root, func(n *sitter.Node) bool {
		if n == nil {
			depth--
			return false
		}
		space := strings.Repeat(indent, depth)
		fmt.Fprintln(w, fmt.Sprintf("%s%s", space, n.String()))
		depth++
		return true
	})
}
