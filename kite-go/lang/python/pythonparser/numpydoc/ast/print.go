package ast

import (
	"fmt"
	"io"
	"strings"
)

type printer struct {
	depth  int
	indent string
	w      io.Writer
}

func (p *printer) Visit(n Node) Visitor {
	if n == nil {
		p.depth--
		return nil
	}

	p.depth++
	switch n := n.(type) {
	case *Doc:
		fmt.Fprintf(p.w, "%sDoc\n", strings.Repeat(p.indent, p.depth-1))

	case *Section:
		fmt.Fprintf(p.w, "%sSection[%s]\n",
			strings.Repeat(p.indent, p.depth-1), n.Header)

	case *Directive:
		fmt.Fprintf(p.w, "%sDirective[%s]\n",
			strings.Repeat(p.indent, p.depth-1), n.Name)

	case *Paragraph:
		fmt.Fprintf(p.w, "%sParagraph\n",
			strings.Repeat(p.indent, p.depth-1))

	case *Definition:
		fmt.Fprintf(p.w, "%sDefinition\n",
			strings.Repeat(p.indent, p.depth-1))

	case *Doctest:
		fmt.Fprintf(p.w, "%sDoctest[%q]\n",
			strings.Repeat(p.indent, p.depth-1), n.Text)

	case *Inline:
		fmt.Fprintf(p.w, "%sInline[%c %q]\n",
			strings.Repeat(p.indent, p.depth-1), n.Markup, n.Text)

	case Text:
		fmt.Fprintf(p.w, "%sText[%s]\n",
			strings.Repeat(p.indent, p.depth-1), n)

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
	return p
}

// Print writes a textual representation of the syntax tree to the given writer.
func Print(root Node, w io.Writer, indent string) {
	p := &printer{
		w:      w,
		indent: indent,
	}
	Walk(p, root)
}
