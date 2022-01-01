package minihtml

import (
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func spanToDiv(node *html.Node) ([]*html.Node, error) {
	if node.DataAtom == atom.Span {
		next := &html.Node{
			Type:      node.Type,
			Data:      "div",
			DataAtom:  atom.Div,
			Namespace: node.Namespace,
			Attr:      node.Attr,
		}
		return []*html.Node{next}, nil
	}
	return cp(node)
}

func spanToDivs(node *html.Node) ([]*html.Node, error) {
	if node.DataAtom == atom.Span {
		next := []*html.Node{}
		for i := 0; i < 2; i++ {
			next = append(next, &html.Node{
				Type:      node.Type,
				Data:      "div",
				DataAtom:  atom.Div,
				Namespace: node.Namespace,
				Attr:      node.Attr,
			})
		}
		return next, nil
	}
	return cp(node)
}

func dropSpan(node *html.Node) ([]*html.Node, error) {
	if node.DataAtom == atom.Span {
		return []*html.Node{}, nil
	}
	return cp(node)
}

var identity = map[html.NodeType]ConversionFunc{
	html.DocumentNode: cp,
	html.TextNode:     cp,
	html.ElementNode:  cp,
}

func assertNodesEqual(t *testing.T, expected, actual *html.Node) {
	log.Printf("====\nexpected: %v\nactual: %v\n\n", *expected, *actual)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Data, actual.Data)
	assert.Equal(t, expected.DataAtom, actual.DataAtom)
	assert.Equal(t, expected.Namespace, actual.Namespace)
	assert.Equal(t, true, reflect.DeepEqual(expected.Attr, actual.Attr))
	e, a := expected.FirstChild, actual.FirstChild
	for e != nil && a != nil {
		assertNodesEqual(t, e, a)
		e = e.NextSibling
		a = a.NextSibling
	}
	require.Nil(t, e)
	require.Nil(t, a)
}

func TestConvertInner(t *testing.T) {
	var raw string
	var input []*html.Node
	var expected, actual []*html.Node
	var err error

	conv := map[html.NodeType]ConversionFunc{
		html.DocumentNode: cp,
		html.TextNode:     cp,
		html.ElementNode:  spanToDiv,
	}

	raw = `
<div>
  <div>Foo</div>
</div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	expected, err = ConvertInner(input[0], identity)
	require.NoError(t, err)
	assert.Equal(t, 1, len(expected))

	raw = `
<div>
  <span>Foo</span>
</div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	actual, err = ConvertInner(input[0], conv)
	require.NoError(t, err)
	assert.Equal(t, len(expected), len(actual))
	assertNodesEqual(t, expected[0], actual[0])
}

func TestConvertInnerMultipleChildren(t *testing.T) {
	var raw string
	var input []*html.Node
	var expected, actual []*html.Node
	var err error

	conv := map[html.NodeType]ConversionFunc{
		html.DocumentNode: cp,
		html.TextNode:     cp,
		html.ElementNode:  spanToDivs,
	}

	raw = `
<div>
  <div></div><div></div>
</div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	expected, err = ConvertInner(input[0], identity)
	require.NoError(t, err)
	assert.Equal(t, 1, len(expected))

	raw = `
<div>
  <span></span>
</div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	actual, err = ConvertInner(input[0], conv)
	require.NoError(t, err)
	assert.Equal(t, len(expected), len(actual))
	assertNodesEqual(t, expected[0], actual[0])
}

func TestConvertInnerMultipleChildrenWithError(t *testing.T) {
	var raw string
	var input []*html.Node
	var err error

	conv := map[html.NodeType]ConversionFunc{
		html.DocumentNode: cp,
		html.TextNode:     cp,
		html.ElementNode:  spanToDivs,
	}

	raw = `
<div>
  <span>Foo</span>
</div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	_, err = ConvertInner(input[0], conv)
	require.Error(t, err)
}

func TestConvertInnerDrop(t *testing.T) {
	var raw string
	var input []*html.Node
	var expected, actual []*html.Node
	var err error

	conv := map[html.NodeType]ConversionFunc{
		html.DocumentNode: cp,
		html.TextNode:     cp,
		html.ElementNode:  dropSpan,
	}

	raw = `<div></div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	expected, err = ConvertInner(input[0], identity)
	require.NoError(t, err)
	assert.Equal(t, 1, len(expected))

	raw = `<div><span></span></div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	actual, err = ConvertInner(input[0], conv)
	require.NoError(t, err)
	assert.Equal(t, len(expected), len(actual))
	assertNodesEqual(t, expected[0], actual[0])

	raw = `<div>Foo</div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	expected, err = ConvertInner(input[0], identity)
	require.NoError(t, err)
	assert.Equal(t, 1, len(expected))

	raw = `<div><span>Foo</span></div>`
	input, _ = html.ParseFragment(strings.NewReader(raw), nil)
	actual, err = ConvertInner(input[0], conv)
	require.NoError(t, err)
	assert.Equal(t, len(expected), len(actual))
	assertNodesEqual(t, expected[0], actual[0])
}

func TestConvert(t *testing.T) {
	var raw string
	var input *html.Node
	var expected, actual *html.Node
	var err error

	conv := map[html.NodeType]ConversionFunc{
		html.DocumentNode: cp,
		html.TextNode:     cp,
		html.ElementNode:  spanToDiv,
	}

	raw = `
<div>
  <div>Foo</div>
</div>`
	input, _ = html.Parse(strings.NewReader(raw))
	expected, err = Convert(input, identity)
	require.NoError(t, err)

	raw = `
<div>
  <span>Foo</span>
</div>`
	input, _ = html.Parse(strings.NewReader(raw))
	actual, err = Convert(input, conv)
	require.NoError(t, err)
	assertNodesEqual(t, expected, actual)
}

func TestConvertWithError(t *testing.T) {
	var raw string
	var input *html.Node
	var err error

	conv := map[html.NodeType]ConversionFunc{
		html.DocumentNode: func(node *html.Node) ([]*html.Node, error) {
			a, _ := cp(node)
			b, _ := cp(node)
			return []*html.Node{a[0], b[0]}, nil
		},
		html.TextNode:    cp,
		html.ElementNode: cp,
	}

	raw = `<div></div>`
	input, _ = html.Parse(strings.NewReader(raw))
	_, err = Convert(input, conv)
	require.Error(t, err)
}
