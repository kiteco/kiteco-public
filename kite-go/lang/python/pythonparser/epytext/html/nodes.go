package html

import (
	nethtml "golang.org/x/net/html"
)

type elem string

// list of element tags used to render the AST as HTML.
const (
	a      elem = "a"
	b      elem = "b"
	body   elem = "body"
	code   elem = "code"
	dd     elem = "dd"
	div    elem = "div"
	dl     elem = "dl"
	dt     elem = "dt"
	h1     elem = "h1"
	h2     elem = "h2"
	h3     elem = "h3"
	_html  elem = "html"
	i      elem = "i"
	li     elem = "li"
	ol     elem = "ol"
	p      elem = "p"
	pre    elem = "pre"
	span   elem = "span"
	strong elem = "strong"
	ul     elem = "ul"

	none elem = ""
)

type textOrElem interface {
	node() *nethtml.Node
}

func (e elem) node() *nethtml.Node {
	return &nethtml.Node{Type: nethtml.ElementNode, Data: string(e)}
}

type text string

func (t text) node() *nethtml.Node {
	return &nethtml.Node{Type: nethtml.TextNode, Data: string(t)}
}

// helper function to build a tree of HTML nodes, pass either a
// string or an elem in the variadic tree parameter and it will be
// created as either a TextNode (string) or an ElementNode (elem),
// with each value the parent of the next. If parent is not nil,
// the first value in tree is appended as a child to parent.
//
// It returns the first node of tree.
func appendTree(parent *nethtml.Node, tree ...textOrElem) *nethtml.Node {
	var first *nethtml.Node
	container := parent
	for _, v := range tree {
		n := v.node()
		if container != nil {
			container.AppendChild(n)
		}
		if first == nil {
			first = n
		}
		container = n
	}
	return first
}
