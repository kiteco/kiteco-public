package minihtml

import (
	"fmt"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Tags allowed in the minihtml spec
// See: https://www.sublimetext.com/docs/3/minihtml.html
var tags = map[atom.Atom]bool{
	atom.Html:   true,
	atom.Head:   true,
	atom.Style:  true,
	atom.Body:   true,
	atom.H1:     true,
	atom.H2:     true,
	atom.H3:     true,
	atom.H4:     true,
	atom.H5:     true,
	atom.H6:     true,
	atom.Div:    true,
	atom.P:      true,
	atom.Ul:     true,
	atom.Ol:     true,
	atom.Li:     true,
	atom.B:      true,
	atom.Strong: true,
	atom.I:      true,
	atom.Em:     true,
	atom.U:      true,
	atom.Big:    true,
	atom.Small:  true,
	atom.A:      true,
	atom.Code:   true,
	atom.Var:    true,
	atom.Tt:     true,
}

// ConversionFunc is a function that outputs a set of nodes that should
// replace the input node in the new minihtml tree. It may also return
// an error.
type ConversionFunc func(node *html.Node) ([]*html.Node, error)

// Conversion maps a node's type to its appropriate ConversionFunc. It
// is user-defined and should contain all the logic necessary to convert
// the html tree to the minihtml version.
type Conversion map[html.NodeType]ConversionFunc

// ConvertInner performs a depth-first iteration to convert an html node
// to its minihtml counterpart. This function may return more than one
// node since it is meant to be used on an arbitrary node in the tree.
//
// Users should provide a Conversion map that produces a new tree as
// opposed to modifying in place since this function will handle the
// binding of parents and siblings automatically. In fact, this function
// will error if a ConversionFunc returns a node that already has its
// relatives set.
func ConvertInner(node *html.Node, conv Conversion) ([]*html.Node, error) {
	fn, ok := conv[node.Type]
	if !ok {
		return nil, fmt.Errorf("conversion mapping does not support %v", node.Type)
	}
	next, err := fn(node)
	if err != nil {
		return nil, err
	}
	for _, n := range next {
		if n.Type == html.ElementNode && !IsValidTag(n.DataAtom) {
			return nil, fmt.Errorf("invalid tag returned: %s", n.Data)
		}
	}

	num := len(next)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		children, err := ConvertInner(child, conv)
		if err != nil {
			return nil, err
		}
		if num == 0 {
			next = append(next, children...)
			continue
		}
		if num > 1 && len(children) > 0 {
			return nil, fmt.Errorf("cannot append multiple children to multiple parents")
		}
		for _, c := range children {
			next[0].AppendChild(c)
		}
	}

	return next, nil
}

// Convert converts a root html node to its minihtml counterpart. It does
// so by using the ConvertInner function defined above.
func Convert(root *html.Node, conv Conversion) (*html.Node, error) {
	next, err := ConvertInner(root, conv)
	if err != nil {
		return nil, err
	}
	if len(next) != 1 {
		return nil, fmt.Errorf("more than one root returned")
	}
	return next[0], nil
}

// GetValidTags returns a copy of the tags allowed in minihtml.
func GetValidTags() []atom.Atom {
	ret := []atom.Atom{}
	for t := range tags {
		ret = append(ret, t)
	}
	return ret
}

// IsValidTag returns whether or not a tag is valid minihtml.
func IsValidTag(t atom.Atom) bool {
	return tags[t]
}
