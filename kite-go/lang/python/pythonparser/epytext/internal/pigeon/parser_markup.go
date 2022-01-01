package pigeon

import (
	"bytes"
	"strings"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
)

func parseText(s string) []ast.Node {
	// create a dummy top-level NestingNode to avoid special-casing
	// nil as top-level.
	rs := []rune(s)
	root := &ast.DocBlock{}
	parseInto(root, rs, 0, false)
	return root.Nodes
}

func parseInto(parent ast.NestingNode, src []rune, startIndex int, insideMarkup bool) int {
	// helper function to create a text node
	createText := func(end int, isEscape bool) {
		if end > startIndex {
			text := ast.Text(src[startIndex:end])
			if len(text) > 0 {
				if isEscape {
					switch text {
					case "lb":
						text = ast.Text("{")
					case "rb":
						text = ast.Text("}")
					}
				}
				appendNodes(parent, text)
			}
		}
	}

	// helper function for the common code when parsing into a markup node
	parseIntoMarkup := func(markup ast.NestingNode, loopIndex *int) {
		next := parseInto(markup, src, (*loopIndex)+1, true)
		startIndex = next
		*loopIndex = next - 1 // to account for the increment on next iteration
		appendNodes(parent, markup)
	}

	for i := startIndex; i < len(src); i++ {
		var prev rune
		if i > 0 {
			prev = src[i-1]
		}
		cur := src[i]

		if cur == '{' && isMarkup(prev) {
			// create a Text for the prior text, if any
			createText(i-1, false)

			switch prev {
			case 'E':
				// pass a dummy parent, just to get the ending index of the
				// escaped text.
				dummy := &ast.BasicMarkup{Type: ast.Markup(prev)}
				end := parseInto(dummy, src, i+1, true)
				startIndex = i + 1
				createText(end-1, true) // end is 1 after the closing }

				i = end - 1 // to account for the i++
				startIndex = end

			case 'U':
				markup := &ast.URLMarkup{}
				parseIntoMarkup(markup, &i)
				markup.URL = extractLink(markup.Nodes)
			case 'L':
				markup := &ast.CrossRefMarkup{}
				parseIntoMarkup(markup, &i)
				markup.Object = extractLink(markup.Nodes)
			default:
				markup := &ast.BasicMarkup{Type: ast.Markup(prev)}
				parseIntoMarkup(markup, &i)
			}

		} else if cur == '}' && insideMarkup {
			// create a Text node and return
			createText(i, false)
			return i + 1 // continue after the '}'
		}
	}

	// create text node for the final part, if any
	createText(len(src), false)
	return len(src)
}

func extractLink(nodes []ast.Node) string {
	if len(nodes) == 0 {
		return ""
	}

	// if the last node is an ast.Text, look for text
	// within '<' and '>'.
	if last, ok := nodes[len(nodes)-1].(ast.Text); ok {
		s := strings.TrimRightFunc(string(last), unicode.IsSpace)
		if strings.HasSuffix(s, ">") {
			if start := strings.Index(s, "<"); start >= 0 {
				nodes[len(nodes)-1] = ast.Text(s[:start])
				return s[start+1 : len(s)-1]
			}
		}
	}

	// otherwise extract all text content and use it as link.
	var w textWriter
	for _, n := range nodes {
		ast.Walk(&w, n)
	}
	return w.String()
}

type textWriter struct {
	buf bytes.Buffer
}

func (w *textWriter) String() string {
	return w.buf.String()
}

func (w *textWriter) Visit(n ast.Node) ast.Visitor {
	if t, ok := n.(ast.Text); ok {
		w.buf.WriteString(string(t))
	}
	return w
}

func isMarkup(r rune) bool {
	return r == 'B' ||
		r == 'C' ||
		r == 'E' ||
		r == 'I' ||
		r == 'L' ||
		r == 'M' ||
		r == 'U' ||
		r == 'X'
}
