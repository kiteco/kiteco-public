package pythondocs

import (
	"bytes"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/minihtml"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func cp(node *html.Node) ([]*html.Node, error) {
	next := &html.Node{
		Type:      node.Type,
		Data:      node.Data,
		DataAtom:  node.DataAtom,
		Namespace: node.Namespace,
		Attr:      node.Attr,
	}
	return []*html.Node{next}, nil
}

func drop(node *html.Node) ([]*html.Node, error) {
	return []*html.Node{}, nil
}

func fallback(node *html.Node) ([]*html.Node, error) {
	if node.DataAtom == atom.Head || node.DataAtom == atom.Style {
		return drop(node)
	}
	if minihtml.IsValidTag(node.DataAtom) {
		return cp(node)
	}
	next := &html.Node{
		Type:      node.Type,
		Data:      "div",
		DataAtom:  atom.Div,
		Namespace: node.Namespace,
	}
	copy(next.Attr, node.Attr)
	next.Attr = append(next.Attr, html.Attribute{
		Key: "otag",
		Val: node.Data,
	})
	return []*html.Node{next}, nil
}

var conv = map[html.NodeType]minihtml.ConversionFunc{
	html.ErrorNode:    drop,
	html.DocumentNode: cp,
	html.TextNode:     cp,
	html.ElementNode:  fallback,
	html.CommentNode:  drop,
	html.DoctypeNode:  drop,
}

// MiniHTML takes a string representating an HTML node and returns
// the string represtation of the minihtml version.
func MiniHTML(body string) (string, error) {
	node, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", err
	}
	mini, err := minihtml.Convert(node, conv)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = html.Render(buf, mini)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
