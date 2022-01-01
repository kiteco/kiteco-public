// Package html renders an epytext AST into HTML.
package html

import (
	"fmt"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
	nethtml "golang.org/x/net/html"
)

// Render renders the epytext tree root into w. A complete HTML document
// is generated. The rendering uses basic HTML elements similar to
// what Sublime Text's minihtml supports:
// https://www.sublimetext.com/docs/3/minihtml.html.
// The root node must not be nil.
//
// It returns an error if the write to w fails.
func Render(root ast.Node, w io.Writer) error {
	var s stack
	ast.Walk(&s, root)

	if s.root == nil {
		return nil
	}
	fixupHTMLNode(s.root)
	return nethtml.Render(w, s.root)
}

// Subset of inline elements used in our rendering, to detect when an inline tag should
// be closed (when a block element is encountered).
// See https://developer.mozilla.org/en-US/docs/Web/HTML/Inline_elements#Elements
var inlineTags = map[string]bool{
	"a":      true,
	"b":      true,
	"code":   true,
	"i":      true,
	"span":   true,
	"strong": true,
}

// in-place post-processing of the HTML AST:
// - consecutive </ol><ol> are merged together
// - same for </ul><ul>
// - <hx> is closed when a block is encountered (e.g. a <p>, <ul>, <ol>, etc.)
func fixupHTMLNode(n *nethtml.Node) {
	// process depth-first, may add children to n
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		fixupHTMLNode(child)
	}

	// then fixup the immediate children of n
	isH := n.Type == nethtml.ElementNode && (n.Data == "h1" || n.Data == "h2" || n.Data == "h3")
	lastTag := ""
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == nethtml.ElementNode {
			if isH && !inlineTags[child.Data] {
				// inside a header and encountered a block element, close the header
				// and move the block (and subsequent children) as sibling of the header.
				// Cannot remove inside the loop because after the move, NextSibling won't
				// be the next child of n anymore.
				var toMove []*nethtml.Node
				for tag := child; tag != nil; tag = tag.NextSibling {
					toMove = append(toMove, tag)
				}
				before := n.NextSibling
				for _, tag := range toMove {
					n.RemoveChild(tag)
					n.Parent.InsertBefore(tag, before)
				}
				// no more children, so return
				return
			}

			if (lastTag == "ul" && child.Data == "ul") || (lastTag == "ol" && child.Data == "ol") {
				// consecutive lists, merge by moving all <li> children of child to
				// prevList.
				prevList := child.PrevSibling
				for li := child.FirstChild; li != nil; li = child.FirstChild {
					child.RemoveChild(li)
					prevList.AppendChild(li)
				}

				// once all children have been moved, remove the child node, a now unnecessary
				// <ul> or <ol>, and set child to prevList to be able to continue to loop
				// the children of n.
				child.Parent.RemoveChild(child)
				child = prevList
			}
			lastTag = child.Data
		}
	}
}

type stack struct {
	root  *nethtml.Node
	nodes []*nethtml.Node

	// field blocks are stored here when encountered, and rendering is deferred
	// until the closing of the <body>:
	// - map of normalized field name (e.g. "param", "type")
	//   - map of field definitions per "argument" (e.g. param name,
	//     type of param, may be empty if no argument)
	//     - slice of *nethtml.Node, the rendered content of each field-arg.
	fields fieldsDef

	// paramsOrder maintains the original ordering of the standard
	// (non-keyword) parameters.
	paramsOrder []string
}

// Appends n into parent, unwrapping n from its container node(s) if required.
// This is because when FieldBlocks are saved for deferred processing, their
// children are rendered in a <div> container. This <div> is removed when rendering
// the fields.
//
// Additionally, if inside the <div>, there is only a single <p> element as immediate
// child, that <p> is also unwrapped. This is so e.g. a @param and its @type are not rendered
// as two separate paragraphs inside the <li>.
func appendUnwrappedNode(parent *nethtml.Node, n *nethtml.Node) {
	var nodes []*nethtml.Node
	// unwrap the <div> parent
	if n.Type == nethtml.ElementNode && n.Data == "div" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			nodes = append(nodes, c)
		}
	}

	if len(nodes) == 1 && nodes[0].Type == nethtml.ElementNode && nodes[0].Data == "p" {
		// unwrap the single <p> container
		p := nodes[0]
		nodes = nodes[:0]
		for c := p.FirstChild; c != nil; c = c.NextSibling {
			nodes = append(nodes, c)
		}
	}

	// append under parent
	for _, n := range nodes {
		// n.Parent should never be nil, let it panic if this is the case
		n.Parent.RemoveChild(n)
		parent.AppendChild(n)
	}
}

// pushes n on top of the stack, so that future nodes are rendered
// under this parent.
func (s *stack) push(n *nethtml.Node) {
	if len(s.nodes) == 0 {
		parent := n
		for parent.Parent != nil {
			parent = parent.Parent
		}
		s.root = parent
	}
	s.nodes = append(s.nodes, n)
}

// pops the top of the stack, meaning that we've exited from this node.
func (s *stack) pop() *nethtml.Node {
	n := s.peek()
	if len(s.nodes) > 0 {
		s.nodes = s.nodes[:len(s.nodes)-1]
	}
	return n
}

// returns the top of the stack without popping it.
func (s *stack) peek() *nethtml.Node {
	// should never be called on an empty stack, let it panic if this is the case
	return s.nodes[len(s.nodes)-1]
}

// implements the Visitor interface for *stack so that it can be used
// to Walk the epytext AST.
func (s *stack) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		if tag := s.pop(); tag != nil && tag.Data == "body" {
			// about to exit from the body tag, render the fields in this node.
			s.renderFieldBlocks(tag)
		}
		return nil
	}

	// each case must push a single *nethtml.Node to s or panic.
	switch n := n.(type) {
	case *ast.DocBlock:
		hn := appendTree(nil, _html, body)
		// TODO: set a kite-specific id on <body> as recommended on minihtml?
		// https://www.sublimetext.com/docs/3/minihtml.html#best_practices
		s.push(hn.FirstChild) // <body>

	case *ast.SectionBlock:
		var el elem
		switch n.Header {
		case ast.H1:
			el = h1
		case ast.H2:
			el = h2
		case ast.H3:
			el = h3
		default:
			panic(fmt.Sprintf("invalid section header: %d", n.Header))
		}
		eln := appendTree(s.peek(), el)
		s.push(eln)

	case *ast.ParagraphBlock:
		pn := appendTree(s.peek(), p)
		s.push(pn)

	case *ast.ListBlock:
		var el elem
		switch n.ListType {
		case ast.OrderedList:
			el = ol
		case ast.UnorderedList:
			el = ul
		default:
			panic(fmt.Sprintf("invalid list type: %d", n.ListType))
		}
		eln := appendTree(s.peek(), el, li)
		s.push(eln.FirstChild) // li

	case *ast.FieldBlock:
		// NOTE: <dl>, <dt> and <dd> are not officially supported/styled by Sublime Text's
		// minihtml according to their docs.

		// add an HTML node that isn't added to the parent, i.e. a standalone
		// node to render the content of the field block. It will be added to
		// the document only at the end, during the processing of all field
		// blocks.
		div := &nethtml.Node{Type: nethtml.ElementNode, Data: "div"}
		s.saveFieldBlock(n, div)
		s.push(div)

	case *ast.DoctestBlock:
		pren := appendTree(s.peek(), pre, text(n.RawText))
		s.push(pren)

	case *ast.LiteralBlock:
		pren := appendTree(s.peek(), pre, code, text(n.RawText))
		s.push(pren)

	case *ast.BasicMarkup:
		var el elem
		switch n.Type {
		case ast.B:
			el = b
		case ast.C:
			el = code
		case ast.I, ast.X:
			// indexed terms show up as italics, but otherwise no other special rendering.
			el = i
		case ast.M:
			// math is ignored for now (no special markup), but its content is still rendered.
			el = none
		default:
			panic(fmt.Sprintf("invalid markup type: %d", n.Type))
		}

		if el == none {
			// for math, keep adding subsequent children under the same parent
			s.push(s.peek())
		} else {
			eln := appendTree(s.peek(), el)
			s.push(eln)
		}

	case *ast.URLMarkup:
		an := appendTree(s.peek(), a)
		an.Attr = append(an.Attr, nethtml.Attribute{Key: "href", Val: n.URL})
		s.push(an)

	case *ast.CrossRefMarkup:
		// for now we have no way to resolve cross-reference links,
		// so render the content without special markup.
		s.push(s.peek())

	case ast.Text:
		tn := appendTree(s.peek(), text(n))
		s.push(tn)

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
	return s
}
