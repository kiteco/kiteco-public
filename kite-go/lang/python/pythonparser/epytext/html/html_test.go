package html

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	nethtml "golang.org/x/net/html"
)

type assertMode int

const (
	assertStrict assertMode = iota
	assertLooseWhitespace
)

func assertRender(t *testing.T, expected string, src ast.Node) bool {
	return assertRenderMode(t, expected, src, assertStrict)
}

func assertRenderMode(t *testing.T, expected string, src ast.Node, mode assertMode) bool {
	var buf bytes.Buffer
	if !assert.NoError(t, Render(src, &buf)) {
		return false
	}

	got := buf.String()
	switch mode {
	case assertStrict:
		if got != expected {
			t.Errorf("want:\n%s\ngot:\n%s\n", expected, got)
			return false
		}

	case assertLooseWhitespace:
		// parse both html strings and compare the nodes, trimming whitespace
		expNode, err := nethtml.Parse(strings.NewReader(expected))
		if !assert.NoError(t, err) {
			return false
		}
		gotNode, err := nethtml.Parse(strings.NewReader(got))
		if !assert.NoError(t, err) {
			return false
		}
		if !looseWhitespaceCompare(expNode, gotNode) {
			t.Errorf("want:\n%s\ngot:\n%s\n", expected, got)
			return false
		}

	default:
		t.Errorf("unknown assert mode: %d", mode)
		return false
	}
	return true
}

func looseWhitespaceCompare(exp, got *nethtml.Node) bool {
	if exp.Type != got.Type {
		return false
	}
	if exp.Namespace != got.Namespace {
		return false
	}
	if exp.DataAtom != got.DataAtom {
		return false
	}
	// ignore variations in whitespace, so if a string is "a \r\n\t b", normalize
	// it as "a b".
	fields := strings.Fields(exp.Data)
	expStr := strings.Join(fields, " ")
	fields = strings.Fields(got.Data)
	gotStr := strings.Join(fields, " ")
	if expStr != gotStr {
		return false
	}
	if len(exp.Attr) != len(got.Attr) {
		return false
	}
	for i := 0; i < len(exp.Attr); i++ {
		if exp.Attr[i] != got.Attr[i] {
			return false
		}
	}
	var expChildren, gotChildren []*nethtml.Node
	for c := exp.FirstChild; c != nil; c = c.NextSibling {
		// skip nodes that are purely whitespace
		if c.Type == nethtml.TextNode && strings.TrimSpace(c.Data) == "" {
			continue
		}
		expChildren = append(expChildren, c)
	}
	for c := got.FirstChild; c != nil; c = c.NextSibling {
		// skip nodes that are purely whitespace
		if c.Type == nethtml.TextNode && strings.TrimSpace(c.Data) == "" {
			continue
		}
		gotChildren = append(gotChildren, c)
	}
	if len(expChildren) != len(gotChildren) {
		return false
	}
	for i := 0; i < len(expChildren); i++ {
		if !looseWhitespaceCompare(expChildren[i], gotChildren[i]) {
			return false
		}
	}
	return true
}

// errWriter is an error and an io.Writer that fails to write
// by returning itself as error.
type errWriter string

func (w errWriter) Error() string               { return string(w) }
func (w errWriter) Write(p []byte) (int, error) { return 0, w }

func TestWriteErr(t *testing.T) {
	w := errWriter("fail")

	n := &ast.DocBlock{
		Nodes: []ast.Node{
			&ast.ParagraphBlock{
				Nodes: []ast.Node{ast.Text("hello")},
			},
		},
	}
	err := Render(n, w)
	assert.Equal(t, w, err)
}

func TestEmptyDocBlock(t *testing.T) {
	n := &ast.DocBlock{}
	expected := `<html><body></body></html>`
	assertRender(t, expected, n)
}

func TestSectionBlock(t *testing.T) {
	src := `
Header
======
	`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<h1>Header</h1>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestNestedSectionBlocks(t *testing.T) {
	src := `
Header
======

   Sub
   ---
      Final
      ~~~~~

   Sub2
   ----
H2
==
	`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<h1>Header</h1>
		<h2>Sub</h2>
		<h3>Final</h3>
		<h2>Sub2</h2>
		<h1>H2</h1>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestParagraphBlock(t *testing.T) {
	src := `
paragraph
	`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>paragraph</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestSingleListBlock(t *testing.T) {
	src := `
	- item
	`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<ul>
			<li>
				<p>item</p>
			</li>
		</ul>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestMultiListBlock(t *testing.T) {
	src := `
	- item 1
	- item 2
	- item 3
	`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<ul>
			<li>
				<p>item 1</p>
			</li>
			<li>
				<p>item 2</p>
			</li>
			<li>
				<p>item 3</p>
			</li>
		</ul>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestComplexMultiListBlock(t *testing.T) {
	src := `
	- item 1
	- item 2
    1. ordered 1
    2. ordered 2
paragraph
      - item x
      - item y
    3. ordered 3
  - item 3
  - item 4
	`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<ul>
			<li>
				<p>item 1</p>
			</li>
			<li>
				<p>item 2</p>
			</li>
		</ul>
		<ol>
			<li>
				<p>ordered 1</p>
			</li>
			<li>
				<p>ordered 2</p>
			</li>
		</ol>
		<p>paragraph
			<ul>
				<li>
					<p>item x</p>
				</li>
				<li>
					<p>item y</p>
				</li>
			</ul>
			<ol>
				<li>
					<p>ordered 3</p>
				</li>
			</ol>
			<ul>
				<li>
					<p>item 3</p>
				</li>
				<li>
					<p>item 4</p>
				</li>
			</ul>
		</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestLiteralBlock(t *testing.T) {
	src := `
paragraph::

  literal
    block
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `<html><body><p>paragraph:<pre><code>
  literal
    block</code></pre></p></body></html>`
	assertRender(t, expected, n)
}

func TestDoctestBlock(t *testing.T) {
	src := `
paragraph:

  >>> doctest
  ...
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `<html><body><p>paragraph:<pre>&gt;&gt;&gt; doctest
...</pre></p></body></html>`
	assertRender(t, expected, n)
}

func TestBoldMarkup(t *testing.T) {
	src := `
some B{bold} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some <b>bold</b> text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestCodeMarkup(t *testing.T) {
	src := `
some C{code} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some <code>code</code> text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestItalicsMarkup(t *testing.T) {
	src := `
some I{italics} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some <i>italics</i> text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestIndexedMarkup(t *testing.T) {
	src := `
some X{indexed} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some <i>indexed</i> text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestMathMarkup(t *testing.T) {
	src := `
some M{math} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some math text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestURLMarkup(t *testing.T) {
	src := `
some U{www.python.org} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some <a href="www.python.org">www.python.org</a> text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestURLMarkup2(t *testing.T) {
	src := `
some U{python<www.python.org>} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some <a href="www.python.org">python</a> text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestCrossRefMarkup(t *testing.T) {
	src := `
some L{cross-ref} text.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some cross-ref text.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestEscapeMarkup(t *testing.T) {
	src := `
E{-} paragraph.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>- paragraph.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestNestedMarkup(t *testing.T) {
	src := `
some B{I{nested} markup} U{B{even I{inside}} urls<www.python.org>}.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>some <b><i>nested</i> markup</b> <a href="www.python.org"><b>even <i>inside</i></b> urls</a>.</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFullDocument(t *testing.T) {
	src := `
Header
======

p1

p2
  - list
p3

Sub
---
p4
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<h1>Header</h1>
		<p>p1</p>
		<p>p2
			<ul>
				<li>
					<p>list</p>
				</li>
			</ul>
		</p>
		<p>p3</p>
		<h2>Sub</h2>
		<p>p4</p>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}
