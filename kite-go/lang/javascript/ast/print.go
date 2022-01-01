package ast

import (
	"fmt"
	"io"
	"strings"
)

func print(node *Node, w io.Writer, indent string, printPositions bool) {
	var depth int
	Inspect(node, func(n *Node) bool {
		if n == nil {
			depth--
			return true
		}

		prefix := strings.Repeat(indent, depth)
		var pos string
		if printPositions {
			pos = fmt.Sprintf("[%d...%d]", n.Begin, n.End)
		}
		fmt.Fprintln(w, fmt.Sprintf("%s%v%s", prefix, n, pos))
		depth++
		return true
	})
}

// Print the AST to the provided writer with the specified indent.
func Print(node *Node, w io.Writer, indent string) {
	print(node, w, indent, false)
}

// PrintPositions prints the AST to the provided writer with
// the specified index and node positions.
func PrintPositions(node *Node, w io.Writer, indent string) {
	print(node, w, indent, true)
}
