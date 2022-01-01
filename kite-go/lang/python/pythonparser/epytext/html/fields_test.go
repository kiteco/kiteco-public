package html

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext"
	"github.com/stretchr/testify/require"
)

func TestDeduplicateKeepLast(t *testing.T) {
	cases := []struct {
		in, out []string
	}{
		{nil, nil},
		{[]string{"a"}, []string{"a"}},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"a", "b", "a"}, []string{"b", "a"}},
		{[]string{"a", "b", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"a", "c", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"a", "c", "c", "c"}, []string{"a", "c"}},
		{[]string{"c", "c", "c", "c"}, []string{"c"}},
	}
	for _, c := range cases {
		got := deduplicateKeepLast(c.in)
		require.Equal(t, c.out, got)
	}
}

func TestFieldBlockParam(t *testing.T) {
	src := `
@param x: desc for x
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
  <body>
    <dl>
      <dt>Parameters:</dt>
      <dd>
        <ul>
          <li><strong><code>x</code></strong> - desc for x</li>
        </ul>
      </dd>
    </dl>
  </body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockType(t *testing.T) {
	src := `
@type x: type for x
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Parameters:</dt>
			<dd>
				<ul>
					<li><strong><code>x</code></strong> (type for x)</li>
				</ul>
			</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockParamAndType(t *testing.T) {
	src := `
@param x: desc for x
@type x: type for x
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Parameters:</dt>
			<dd>
				<ul>
					<li><strong><code>x</code></strong> (type for x) - desc for x</li>
				</ul>
			</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockReturn(t *testing.T) {
	src := `
@return: return desc
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Returns: </dt>
			<dd>return desc</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockRtype(t *testing.T) {
	src := `
@rtype: return type
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Returns: return type</dt>
			<dd></dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockReturnAndRtype(t *testing.T) {
	src := `
@rtype: return type
@return: return desc
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Returns: return type</dt>
			<dd>return desc</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockRaise(t *testing.T) {
	src := `
@raise e: exception
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Raises:</dt>
			<dd>
				<ul>
					<li><strong><code>e</code></strong> - exception</li>
				</ul>
			</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockRaiseMany(t *testing.T) {
	src := `
@raise y: exception y
@raise e: exception e1
@raise x: exception x
@raise e: exception e2
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Raises:</dt>
			<dd>
				<ul>
					<li><strong><code>e</code></strong> - exception e2</li>
					<li><strong><code>x</code></strong> - exception x</li>
					<li><strong><code>y</code></strong> - exception y</li>
				</ul>
			</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockNote(t *testing.T) {
	src := `
@note: some note
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<div>
			<p><strong>Note:</strong>some note</p>
		</div>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockMultipleNotes(t *testing.T) {
	src := `
@note: some note
@note: some other note
@note: some final note
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<div>
			<strong>Notes:</strong>
			<ul>
				<li>some note</li>
				<li>some other note</li>
				<li>some final note</li>
			</ul>
		</div>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockTodos(t *testing.T) {
	src := `
@todo v2: test v2
@todo: add 1
@todo: add 2
@todo v1: test v1
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<div>
			<strong>To Do:</strong>
			<ul>
				<li>add 1</li>
				<li>add 2</li>
			</ul>
			<p><strong>To Do (v1):</strong>test v1</p>
			<p><strong>To Do (v2):</strong>test v2</p>
		</div>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockUnknown(t *testing.T) {
	src := `
@unknown x: some text
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<div>
			<p><strong>unknown (x):</strong>some text</p>
		</div>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockMultipleUnknown(t *testing.T) {
	src := `
@unknown x: some text
@unknown x: other text
@unknown b: b def
@unknown: no arg
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<div>
			<p><strong>unknown:</strong>no arg</p>
			<p><strong>unknown (b):</strong>b def</p>
			<strong>unknown (x):</strong>
			<ul>
				<li>some text</li>
				<li>other text</li>
			</ul>
		</div>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockWithinFieldBlock(t *testing.T) {
	src := `
@parent: p
  @child: c
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	// field blocks cannot be parents of field blocks, so this should
	// be parsed as two distinct fields.
	expected := `
<html>
	<body>
		<div>
			<p><strong>child:</strong>c</p>
			<p><strong>parent:</strong>p</p>
		</div>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockComplexDef(t *testing.T) {
	src := `
@param x: parameter C{x} specifies the foo
	of the bar::
		x.foo != bar
	x can have the following values:
		- 11
		- 22
@type x: C{B{I}I{nteger}}
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<dl>
			<dt>Parameters:</dt>
			<dd>
				<ul>
					<li><strong><code>x</code></strong> (<code><b>I</b><i>nteger</i></code>) -
						<p>parameter <code>x</code> specifies the foo of the bar:
							<pre>
								<code>x.foo != bar</code>
							</pre>
						</p>
						<p>x can have the following values:
							<ul>
								<li><p>11</p></li>
								<li><p>22</p></li>
							</ul>
						</p>
					</li>
				</ul>
			</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestFieldBlockCompleteDoc(t *testing.T) {
	src := `
Sends a GET request.

@param url: URL for the new L{Request} object.
@type url: string
@param params: (optional) Dictionary or bytes to be sent in the query string for the L{Request}.
@param kwargs: Optional arguments that C{request} takes.
@kwarg user: Optional username.
@return: L{Response <Response>} object
@rtype: requests.Response
@summary: A request.
`
	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
		<p>Sends a GET request.</p>
		<dl>
			<dt>Parameters:</dt>
			<dd>
				<ul>
					<li><strong><code>url</code></strong> (string) - URL for the new Request object.</li>
					<li><strong><code>params</code></strong> - (optional) Dictionary or bytes to be sent in the query string for the Request.</li>
					<li><strong><code>kwargs</code></strong> - Optional arguments that <code>request</code> takes.</li>
					<li><strong><code>user</code></strong> - Optional username.</li>
				</ul>
			</dd>
			<dt>Returns: requests.Response</dt>
			<dd>
				Response object
			</dd>
		</dl>
    <div>
      <p><strong>Summary:</strong>A request.</p>
    </div>
	</body>
</html>
`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}

func TestTrailingWhitespace(t *testing.T) {
	src := `
example code

@param x  :   hello x.
@type x :  int  `

	n, err := epytext.Parse([]byte(src))
	require.NoError(t, err)

	expected := `
<html>
	<body>
    <p>example code</p>
		<dl>
			<dt>Parameters:</dt>
			<dd>
				<ul>
					<li><strong><code>x</code></strong> (int) - hello x.</li>
				</ul>
			</dd>
		</dl>
	</body>
</html>`
	assertRenderMode(t, expected, n, assertLooseWhitespace)
}
