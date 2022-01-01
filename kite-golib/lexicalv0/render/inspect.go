package render

import (
	"strings"
	"unicode"

	"github.com/go-errors/errors"
	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/javascript"
	"github.com/kiteco/go-tree-sitter/python"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
)

// IndentCheckFn returns true if IndentInspect should extract the indentation
// level from the beginning of the provided node.
type IndentCheckFn func(*sitter.Node) bool

// IndentInspect takes source code and the current cursor position,
// and returns the indent string of current file and the depth of current line
func IndentInspect(buf []byte, pos int, l lang.Language, checker IndentCheckFn) (indent string, depth int, err error) {
	p := sitter.NewParser()
	defer p.Close()
	switch l {
	case lang.Python:
		p.SetLanguage(python.GetLanguage())
	case lang.JavaScript, lang.JSX, lang.Vue:
		p.SetLanguage(javascript.GetLanguage())
	default:
		return "", 0, errors.Errorf("unsupported lang %v", l)
	}

	tree := p.Parse(buf)
	defer tree.Close()

	root := tree.RootNode()
	ir := &indentRetriever{buf: buf, checker: checker}
	treesitter.Walk(ir, root)

	if ir.err != nil {
		// If there's ERROR node in AST tree, we will get it manually
		indent = FindIndentationFromSource(string(buf))
	} else {
		indent = ir.indent
	}

	if indent == "" {
		return "", 0, nil
	}

	pos = findStartOfLine(buf, pos)
	dr := &depthRetriever{pos: pos, indent: indent}
	treesitter.Walk(dr, root)

	// User's indentation is not consistent
	if dr.err != nil {
		return "", 0, errors.New("inconsistent indentation")
	}

	return indent, dr.depth, nil
}

// find the start of the line:
// 1) find end byte of the first newline that occurs before (inclusive) the cursor
// 2) move forward from that newline until we hit a non whitespace character or the cursor
func findStartOfLine(buf []byte, pos int) int {
	b := data.NewBuffer(string(buf))

	newLineEnd := b.RangeReverse(pos, func(i int, r rune) bool {
		if r == '\n' {
			return false
		}
		return true
	})

	lineStart := newLineEnd
	b.Range(newLineEnd, func(i int, r rune) bool {
		lineStart = i
		if i == pos || !unicode.IsSpace(r) {
			return false
		}
		return true
	})

	// hacky way to account for the case when the cursor is at the end of the file
	// and the start of a line, in this case the range loop above will stop early
	if lineStart < pos && unicode.IsSpace(b.RuneAt(lineStart)) {
		return pos
	}

	return lineStart
}

// FindIndentationFromSource determines the indentation symbol from the first
// line that has leading whitespace
func FindIndentationFromSource(src string) string {
	l := len(src)
	var indent string
	for i, s := range src {
		// Find the first line that starts with whitespaces, call that our indent
		if s == '\n' && i < l-1 {
			k := i + 1
			for p := 1; k+p <= l && !strings.Contains(src[k:k+p], "\n") &&
				!strings.Contains(src[k:k+p], "\r") &&
				strings.TrimSpace(src[k:k+p]) == ""; p++ {
				indent = src[k : k+p]
			}
			if indent != "" {
				break
			}
		}
	}
	return indent
}

type indentRetriever struct {
	checker IndentCheckFn
	buf     []byte
	indent  string
	err     error
}

func (ir *indentRetriever) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		return nil
	}

	if n.HasError() {
		ir.err = errors.New("ERROR node in AST")
		return nil
	}
	if ir.checker(n) {
		pos := n.StartByte()
		ir.indent = string(ir.buf[pos-n.StartPoint().Column : pos])
		return nil
	}
	return ir
}

type depthRetriever struct {
	pos    int
	indent string
	depth  int
	err    error
}

func (dr *depthRetriever) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		return nil
	}
	if int(n.StartByte()) == dr.pos {
		if len(dr.indent) == 0 {
			dr.depth = 0
			return nil
		}
		if int(n.StartPoint().Column)%len(dr.indent) != 0 {
			dr.err = errors.New("fail to retrieve depth of current line")
			return nil
		}
		dr.depth = int(n.StartPoint().Column) / len(dr.indent)
		return nil
	}
	return dr
}
