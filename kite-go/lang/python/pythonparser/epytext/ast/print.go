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
	case *DocBlock:
		fmt.Fprintf(p.w, "%sDoc\n", strings.Repeat(p.indent, p.depth-1))

	case *SectionBlock:
		fmt.Fprintf(p.w, "%sSection[%c]\n",
			strings.Repeat(p.indent, p.depth-1), n.Header)

	case *ParagraphBlock:
		fmt.Fprintf(p.w, "%sParagraph\n",
			strings.Repeat(p.indent, p.depth-1))

	case *ListBlock:
		fmt.Fprintf(p.w, "%sList[%s (%d)]\n",
			strings.Repeat(p.indent, p.depth-1), n.Bullet, n.ListType)

	case *FieldBlock:
		fmt.Fprintf(p.w, "%sField[%s (%s)]\n",
			strings.Repeat(p.indent, p.depth-1), n.Name, n.Arg)

	case *DoctestBlock:
		fmt.Fprintf(p.w, "%sDoctest[%s]\n",
			strings.Repeat(p.indent, p.depth-1), n.RawText)

	case *LiteralBlock:
		fmt.Fprintf(p.w, "%sLiteral[%s]\n",
			strings.Repeat(p.indent, p.depth-1), n.RawText)

	case *BasicMarkup:
		fmt.Fprintf(p.w, "%sBasicMarkup[%c]\n",
			strings.Repeat(p.indent, p.depth-1), n.Type)

	case *URLMarkup:
		fmt.Fprintf(p.w, "%sURLMarkup[%s]\n",
			strings.Repeat(p.indent, p.depth-1), n.URL)

	case *CrossRefMarkup:
		fmt.Fprintf(p.w, "%sCrossRefMarkup[%s]\n",
			strings.Repeat(p.indent, p.depth-1), n.Object)

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
