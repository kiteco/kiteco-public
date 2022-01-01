package pythonast

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

func derefType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		return derefType(t.Elem())
	default:
		return t
	}
}

func typename(obj interface{}) string {
	return derefType(reflect.TypeOf(obj)).Name()
}

func litStr(word *pythonscanner.Word) string {
	if word == nil {
		return "<nil>"
	}
	return word.Literal
}

func tokStr(word *pythonscanner.Word) string {
	if word == nil {
		return "<nil>"
	}
	return word.Token.String()
}

func tokOrLitOnError(word *pythonscanner.Word) string {
	if word == nil {
		return "Nil"
	}
	switch word.Token {
	case pythonscanner.BadToken, pythonscanner.Cursor, pythonscanner.Illegal:
		return word.Token.String()
	default:
		return litStr(word)
	}
}

// String returns a short textual representation of a node
func String(n Node) string {
	if IsNil(n) {
		return "Nil"
	}
	out := typename(n)
	switch n := n.(type) {
	case *AttributeExpr:
		out += "[" + tokOrLitOnError(n.Attribute) + "]"
	case *NameExpr:
		out += "[" + tokOrLitOnError(n.Ident) + "]"
	case *NumberExpr:
		out += "[" + tokOrLitOnError(n.Number) + "]"
	case *StringExpr:
		var lits []string
		for _, s := range n.Strings {
			lits = append(lits, strings.Replace(tokOrLitOnError(s), "\n", "\\n", -1))
		}
		out += "[" + strings.Join(lits, " ") + "]"
	case *BinaryExpr:
		out += "[" + tokStr(n.Op) + "]"
	case *UnaryExpr:
		out += "[" + tokStr(n.Op) + "]"
	case *AugAssignStmt:
		out += "[" + tokStr(n.Op) + "]"
	}
	return out
}

type prettyPrinter struct {
	depth     int
	indent    string
	positions bool
	w         io.Writer
}

func (p *prettyPrinter) Visit(n Node) Visitor {
	if n == nil {
		p.depth--
	} else {
		prefix := strings.Repeat(p.indent, p.depth)
		if p.positions {
			prefix = fmt.Sprintf("[%4d...%4d]", n.Begin(), n.End()) + prefix
		}
		fmt.Fprintln(p.w, prefix+String(n))
		p.depth++
	}
	return p
}

// Print writes a textual representation of syntax tree to the given writer
func Print(root Node, w io.Writer, indent string) {
	printer := prettyPrinter{
		w:      w,
		indent: indent,
	}
	Walk(&printer, root)
}

// PrintPositions writes a textual representation of syntax tree to the given writer,
// including begin and end positions for each node.
func PrintPositions(root Node, w io.Writer, indent string) {
	printer := prettyPrinter{
		w:         w,
		indent:    indent,
		positions: true,
	}
	Walk(&printer, root)
}

// -

type debugPrinter struct {
	w     io.Writer
	depth int
}

func (h debugPrinter) VisitSlice(s NodeSliceRef) {
	VisitNodeSlice(h, s)
}

func (h debugPrinter) VisitNode(r NodeRef) {
	h.PrintAST(r.Lookup())
}

func (h debugPrinter) PrintAST(n Node) {
	fmt.Fprintf(h.w, "%sNode(%p): %s\n", strings.Repeat("  ", h.depth), n, String(n))
	if !IsNil(n) {
		n.Iterate(debugPrinter{w: h.w, depth: h.depth + 1})
	}
}

func (h debugPrinter) VisitWord(w **pythonscanner.Word) {
	var s string
	if *w == nil {
		s = "nil"
	} else {
		s = (**w).String()
	}
	fmt.Fprintf(h.w, "%sWord(%p): %s\n", strings.Repeat("  ", h.depth), *w, s)
}

// PrintDebug serializes the given AST as text, including contained Nodes, Words and pointer values thereof
func PrintDebug(root Node, w io.Writer) {
	debugPrinter{w: w}.PrintAST(root)
}
