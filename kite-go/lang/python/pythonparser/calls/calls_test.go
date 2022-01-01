package calls

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/calls/internal/pigeon"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/testparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// for tests to be isolated from changes to the defaultMaxLines.
const testMaxLines = 4

type wordValidator struct {
	t *testing.T
}

func (v wordValidator) VisitNode(r pythonast.NodeRef)       { pythonast.Iterate(v, r.Lookup()) }
func (v wordValidator) VisitSlice(r pythonast.NodeSliceRef) { pythonast.VisitNodeSlice(v, r) }
func (v wordValidator) VisitWord(w **pythonscanner.Word) {
	if *w == nil {
		return
	}
	require.True(v.t, (*w).Valid(), "invalid word %s", **w)
}

func assertParse(t *testing.T, expected string, src string) *pythonast.CallExpr {
	return assertParseWithOptions(t, expected, src, MaxLines(testMaxLines))
}

func assertParseWithOptions(t *testing.T, expected string, src string, opts ...Option) *pythonast.CallExpr {
	t.Log(src)
	node, err := Parse([]byte(src), opts...)
	require.NoError(t, err)
	require.NotNil(t, node)
	assertAST(t, expected, node, false)
	return node
}

func assertParseWithOffsetsAndOptions(t *testing.T, expected string, src string, opts ...Option) *pythonast.CallExpr {
	t.Log(src)
	node, err := Parse([]byte(src), opts...)
	require.NoError(t, err)
	require.NotNil(t, node)
	assertAST(t, expected, node, true)
	return node
}

func assertParseWithOffsets(t *testing.T, expected string, src string) *pythonast.CallExpr {
	return assertParseWithOffsetsAndOptions(t, expected, src, MaxLines(testMaxLines))
}

func assertAST(t *testing.T, expected string, node pythonast.Node, includeOffsets bool) {
	pythonast.Iterate(wordValidator{t}, node)

	var buf bytes.Buffer
	if includeOffsets {
		pythonast.PrintPositions(node, &buf, "\t")
	} else {
		pythonast.Print(node, &buf, "\t")
	}
	actual := buf.String()

	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	if actual != expected {
		expectedLines := strings.Split(expected, "\n")
		actualLines := strings.Split(actual, "\n")

		n := len(expectedLines)
		if len(actualLines) > n {
			n = len(actualLines)
		}

		errorLine := -1
		sidebyside := fmt.Sprintf("      | %-40s | %-40s |\n", "EXPECTED", "ACTUAL")
		var errorExpected, errorActual string
		for i := 0; i < n; i++ {
			var expectedLine, actualLine string
			if i < len(expectedLines) {
				expectedLine = strings.Replace(expectedLines[i], "\t", "    ", -1)
			}
			if i < len(actualLines) {
				actualLine = strings.Replace(actualLines[i], "\t", "    ", -1)
			}
			symbol := "   "
			if actualLine != expectedLine {
				symbol = "***"
				if errorLine == -1 {
					errorLine = i
					errorExpected = strings.TrimSpace(expectedLine)
					errorActual = strings.TrimSpace(actualLine)
				}
			}
			sidebyside += fmt.Sprintf("%-6s| %-40s | %-40s |\n", symbol, expectedLine, actualLine)
		}

		t.Errorf("expected %s but got %s (line %d):\n%s", errorExpected, errorActual, errorLine, sidebyside)
	}

	t.Log("\n" + actual)
}

func assertParserErrorContains(t *testing.T, err error, msg string) {
	if msg == "" {
		require.Nil(t, err)
		return
	}

	require.Error(t, err)
	require.Contains(t, err.Error(), msg)
}

func TestGeneratedParserUpToDate(t *testing.T) {
	testparser.ParserUpToDate(t, "internal/pigeon/parser.peg")
}

func TestNoCallExpr(t *testing.T) {
	cases := []string{
		"",
		" \t\r\n ",
		"foo", // this parser requires an opening parenthesis, signaling a CallExpr
		"foo\nbar",
		"# foo.bar()\n",
		"foo.bar\n",
		"'hello' 'world'\n",
		"3.14\n",
		"()",
		"( )",
		"...",
	}
	for _, c := range cases {
		_, err := Parse([]byte(c), MaxLines(testMaxLines))
		t.Log(c)
		assertParserErrorContains(t, err, pigeon.ErrNoCallExpr.Error())
	}
}

func TestCallExpression(t *testing.T) {
	// this case may seem surprising, but it returns a *CallExpr
	// because there is an atom (empty tuple) followed by a DotTrailer
	// where the MaybeID is missing (nothing after the dot), and then
	// a CallTrailer (the parentheses without arguments).
	src := `().()`
	expected := `
CallExpr
	AttributeExpr[]
		TupleExpr
`
	assertParse(t, expected, src)
}

func TestTooManyLines(t *testing.T) {
	src := []byte(`foo(
		a,
		b,
		c,
		d,
		e,
		f,
		g,
		h,
		i,
		j,
		k)
`)
	_, err := Parse(src, MaxLines(testMaxLines))
	assert.Equal(t, errors.ErrorReason(err), errors.TooManyLines)
}

func TestEdgeCases(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"a(.)", true},    // parses dot as bad expr
		{"a(..)", true},   // same
		{"a(...)", true},  // ellipsis argument
		{"a(....)", true}, // parses ellipsis and empty attribute
		{"a(.(.))", true}, // paren matching makes this parse completely
		{"a(=)", true},    // parses as BadExpr
		{"a(===)", true},  // same
		{"a(,)", true},    // empty arg allowed
		{"a(,,,)", true},
		{"a(=,=,=,=)", true},         // parses as multiple BadExpr
		{"a(.=.,.=.,.=.,.=.)", true}, // same, see TestMultipleBadArgs
		{"a(((())))", true},          // argument is an empty tuple
		{"a((((", true},              // right paren always optional, so this parses fine
		{"a(((()", true},             // same here
		{"a(?", true},                // parses as BadExpr
		{"a( ?", true},               // same
		{"a( # comment ", true},
	}

	for _, c := range cases {
		// TestAttr entrypoint forces the AtomExpr to be followed by EOF, so it tries
		// to parse the whole input as part of the call.
		_, err := pigeon.Parse("", []byte(c.in), pigeon.Entrypoint("TestAttr"))
		if c.ok {
			assert.NoError(t, err, c.in)
		} else {
			assert.Error(t, err, c.in)
		}
	}
}

func TestBadArgUnclosed(t *testing.T) {
	src := "a(?"
	expected := `
[   0...   3]CallExpr
[   0...   1]	NameExpr[a]
[   2...   3]	Argument
[   2...   3]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestMultipleBadArgs(t *testing.T) {
	src := "a(.=.,.=., .=.  ,.=.)"
	expected := `
[   0...  21]CallExpr
[   0...   1]	NameExpr[a]
[   2...   5]	Argument
[   2...   5]		BadExpr
[   6...   9]	Argument
[   6...   9]		BadExpr
[  11...  16]	Argument
[  11...  16]		BadExpr
[  17...  20]	Argument
[  17...  20]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestBadAtomCallExpr(t *testing.T) {
	// NOTE: this parses ".(.)" as BadExpr (the argument),
	// balancing the parens.
	src := `a(.(.))`
	expected := `
[   0...   7]CallExpr
[   0...   1]	NameExpr[a]
[   2...   6]	Argument
[   2...   6]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestSingleID(t *testing.T) {
	src := `foo()`
	expected := `
CallExpr
	NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestSingleIDSingleArg(t *testing.T) {
	src := `foo(bar)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
`
	assertParse(t, expected, src)
}

func TestSingleIDMultiArg(t *testing.T) {
	src := `foo(bar, baz)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		NameExpr[baz]
`
	assertParse(t, expected, src)
}

func TestSingleAttribute(t *testing.T) {
	src := `foo.bar()`
	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestSingleAttributeKeywords(t *testing.T) {
	src := `foo.bar(arg1 , kw=arg2)`
	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		NameExpr[arg1]
	Argument
		NameExpr[kw]
		NameExpr[arg2]
`
	assertParse(t, expected, src)
}

func TestSingleAttributeSingleArg(t *testing.T) {
	src := `foo.bar(baz)`
	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		NameExpr[baz]
`
	assertParse(t, expected, src)
}

func TestSingleAttributeMultiArg(t *testing.T) {
	src := `foo.bar(baz, qux, whatev )`
	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		NameExpr[baz]
	Argument
		NameExpr[qux]
	Argument
		NameExpr[whatev]
`
	assertParse(t, expected, src)
}

func TestMultiAttribute(t *testing.T) {
	src := `foo.bar.baz()`
	expected := `
CallExpr
	AttributeExpr[baz]
		AttributeExpr[bar]
			NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestMultiAttributeSingleArg(t *testing.T) {
	src := `foo.bar.baz(qux)`
	expected := `
CallExpr
	AttributeExpr[baz]
		AttributeExpr[bar]
			NameExpr[foo]
	Argument
		NameExpr[qux]
`
	assertParse(t, expected, src)
}

func TestMultiAttributeMultiArg(t *testing.T) {
	src := `foo.bar.baz(qux, zuk, blah)`
	expected := `
CallExpr
	AttributeExpr[baz]
		AttributeExpr[bar]
			NameExpr[foo]
	Argument
		NameExpr[qux]
	Argument
		NameExpr[zuk]
	Argument
		NameExpr[blah]
`
	assertParse(t, expected, src)
}

func TestMultiline(t *testing.T) {
	src := `foo(bar,
		baz,
		qux)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		NameExpr[baz]
	Argument
		NameExpr[qux]
`
	assertParse(t, expected, src)
}

func TestMultilineWindows(t *testing.T) {
	src := "a\r\n(b,\r\nc,\r\nd,\r\ne)"
	expected := `
CallExpr
	NameExpr[a]
	Argument
		NameExpr[b]
	Argument
		NameExpr[c]
	Argument
		NameExpr[d]
	Argument
		BadExpr
`
	node, err := Parse([]byte(src), MaxLines(4))
	assert.Error(t, err)
	assert.Equal(t, errors.ErrorReason(err), errors.TooManyLines)
	assertAST(t, expected, node, false)
}

func TestMultipleCalls(t *testing.T) {
	src := `
foo()

bar()
`
	expected := `
CallExpr
	NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestMissingParenthesisNoArg(t *testing.T) {
	src := `foo(`
	expected := `
CallExpr
	NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestMissingParenthesis(t *testing.T) {
	src := `foo(bar`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
`
	assertParse(t, expected, src)
}

func TestIncompleteSingleKeyword(t *testing.T) {
	src := `foo(bar=`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
		BadExpr
`
	assertParse(t, expected, src)
}

func TestUnclosedSingleKeyword(t *testing.T) {
	src := `foo(bar = val`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
		NameExpr[val]
`
	assertParse(t, expected, src)
}

func TestIncompleteMultipleKeywords(t *testing.T) {
	src := `foo.method(arg, , some = val ,
  bar = , baz=`

	expected := `
CallExpr
	AttributeExpr[method]
		NameExpr[foo]
	Argument
		NameExpr[arg]
	Argument
		BadExpr
	Argument
		NameExpr[some]
		NameExpr[val]
	Argument
		NameExpr[bar]
		BadExpr
	Argument
		NameExpr[baz]
		BadExpr
`
	assertParse(t, expected, src)
}

func TestUnclosedMultiline(t *testing.T) {
	src := `foo(bar,
		car,
		star`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		NameExpr[car]
	Argument
		NameExpr[star]
`
	assertParse(t, expected, src)
}

func TestOpenParenthesisOnNextLine(t *testing.T) {
	// this is invalid python (would require explicit line continuation),
	// but in the context of this parser, probably makes sense to support.
	src := `foo
    (bar,
`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestCommentNewLine(t *testing.T) {
	src := `foo # blah
    (bar
`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
`
	assertParse(t, expected, src)
}

func TestExplicitLineContinuation(t *testing.T) {
	src := `foo \
  (bar, \
  baz=
`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		NameExpr[baz]
		BadExpr
`
	assertParse(t, expected, src)
}

func TestMissingArgumentNoOffsets(t *testing.T) {
	src := `foo(bar,,car`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
	Argument
		BadExpr
	Argument
		NameExpr[car]
`
	assertParse(t, expected, src)
}

func TestMissingArgument(t *testing.T) {
	src := `foo(bar,,car`
	expected := `
[   0...  12]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   7]	Argument
[   4...   7]		NameExpr[bar]
[   8...   8]	Argument
[   8...   8]		BadExpr
[   9...  12]	Argument
[   9...  12]		NameExpr[car]
`
	assertParseWithOffsets(t, expected, src)
}

func TestIncompleteMultipleKeywordsOffsets(t *testing.T) {
	src := `foo.method(arg, , some = val ,
  bar = , baz=`

	expected := `
[   0...  45]CallExpr
[   0...  10]	AttributeExpr[method]
[   0...   3]		NameExpr[foo]
[  11...  14]	Argument
[  11...  14]		NameExpr[arg]
[  16...  16]	Argument
[  16...  16]		BadExpr
[  18...  28]	Argument
[  18...  22]		NameExpr[some]
[  25...  28]		NameExpr[val]
[  33...  39]	Argument
[  33...  36]		NameExpr[bar]
[  39...  39]		BadExpr
[  41...  45]	Argument
[  41...  44]		NameExpr[baz]
[  45...  45]		BadExpr
`
	assertParseWithOffsets(t, expected, src)
}

func TestAttributeArguments(t *testing.T) {
	src := `foo.bar(qux.zune.leaf, arg2.leaf2)`

	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		AttributeExpr[leaf]
			AttributeExpr[zune]
				NameExpr[qux]
	Argument
		AttributeExpr[leaf2]
			NameExpr[arg2]
`
	assertParse(t, expected, src)
}

func TestAttributeKeywordArguments(t *testing.T) {
	src := `foo.bar(kw1=qux.zune.leaf)`

	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		NameExpr[kw1]
		AttributeExpr[leaf]
			AttributeExpr[zune]
				NameExpr[qux]
`
	assertParse(t, expected, src)
}

func TestStringArguments(t *testing.T) {
	src := `foo.bar("hello", b'literals!')`

	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		StringExpr["hello"]
	Argument
		StringExpr[b'literals!']
`
	assertParse(t, expected, src)
}

func TestStringArguments2(t *testing.T) {
	src := `foo.bar(fr"hello"		u'''various''', r"""String""" b'literals!')`

	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		StringExpr[fr"hello" u'''various''']
	Argument
		StringExpr[r"""String""" b'literals!']
`
	assertParse(t, expected, src)
}

func TestEscapeStringArgument(t *testing.T) {
	src := `foo.bar("this \
is \t some \xff string")`

	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		StringExpr["this \\nis \t some \xff string"]
`
	assertParse(t, expected, src)
}

func TestStringMethod(t *testing.T) {
	src := `'hello'.upper()`

	expected := `
CallExpr
	AttributeExpr[upper]
		StringExpr['hello']
`
	assertParse(t, expected, src)
}

func TestStringMethodEscapedQuoteArg(t *testing.T) {
	src := `'hello'.foobar(key='\'"'"\"'")`

	expected := `
CallExpr
	AttributeExpr[foobar]
		StringExpr['hello']
	Argument
		NameExpr[key]
		StringExpr['\'"' "\"'"]
`
	assertParse(t, expected, src)
}

func TestNumberArguments(t *testing.T) {
	src := `foo.bar(0,.1, 4012L,2.34j, 5.67e-10)`

	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		NumberExpr[0]
	Argument
		NumberExpr[.1]
	Argument
		NumberExpr[4012L]
	Argument
		NumberExpr[2.34j]
	Argument
		NumberExpr[5.67e-10]
`
	assertParse(t, expected, src)
}

func TestFloatMethod(t *testing.T) {
	src := `3.14.is_integer()`

	expected := `
CallExpr
	AttributeExpr[is_integer]
		NumberExpr[3.14]
`
	assertParse(t, expected, src)
}

func TestFloatMethod2(t *testing.T) {
	src := `.123.is_integer()`

	expected := `
CallExpr
	AttributeExpr[is_integer]
		NumberExpr[.123]
`
	assertParse(t, expected, src)
}

func TestFloatMethod3(t *testing.T) {
	src := `1..is_integer()`

	expected := `
CallExpr
	AttributeExpr[is_integer]
		NumberExpr[1.]
`
	assertParse(t, expected, src)
}

func TestImaginaryMethod(t *testing.T) {
	src := `1j.is_integer()`

	expected := `
CallExpr
	AttributeExpr[is_integer]
		NumberExpr[1j]
`
	assertParse(t, expected, src)
}

func TestImaginaryMethod2(t *testing.T) {
	src := `1e4j.is_integer()`

	expected := `
CallExpr
	AttributeExpr[is_integer]
		NumberExpr[1e4j]
`
	assertParse(t, expected, src)
}

func TestIntegerMethod(t *testing.T) {
	src := `(1).bit_length()`

	expected := `
CallExpr
	AttributeExpr[bit_length]
		NumberExpr[1]
`
	assertParse(t, expected, src)
}

func TestNestedCall(t *testing.T) {
	src := `foo.bar('hello', kw=baz.qux.split(r'abc', 3))`

	expected := `
CallExpr
	AttributeExpr[bar]
		NameExpr[foo]
	Argument
		StringExpr['hello']
	Argument
		NameExpr[kw]
		CallExpr
			AttributeExpr[split]
				AttributeExpr[qux]
					NameExpr[baz]
			Argument
				StringExpr[r'abc']
			Argument
				NumberExpr[3]
`
	assertParse(t, expected, src)
}

func TestIncompleteStringArg(t *testing.T) {
	src := `
foo (
  bar = rf"hello`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
		StringExpr[rf"hello]
`
	assertParse(t, expected, src)
}

func TestIncompleteStringArg2(t *testing.T) {
	src := `
foo
	(
  bar = rf"hello
	,3 ,,
 , b'''world
!`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[bar]
		StringExpr[rf"hello]
	Argument
		NumberExpr[3]
	Argument
		BadExpr
	Argument
		BadExpr
	Argument
		StringExpr[b'''world\n!]
`

	assertParseWithOptions(t, expected, src, MaxLines(8))
}

func TestIncompleteStringArgOffset(t *testing.T) {
	src := `foo
	( bar = rf"hello
	,3 ,, zig =
 , b'''world
!`

	expected := `
[   0...  49]CallExpr
[   0...   3]	NameExpr[foo]
[   7...  21]	Argument
[   7...  10]		NameExpr[bar]
[  13...  21]		StringExpr[rf"hello]
[  24...  25]	Argument
[  24...  25]		NumberExpr[3]
[  27...  27]	Argument
[  27...  27]		BadExpr
[  29...  36]	Argument
[  29...  32]		NameExpr[zig]
[  36...  36]		BadExpr
[  38...  49]	Argument
[  38...  49]		StringExpr[b'''world\n!]
`
	assertParseWithOffsetsAndOptions(t, expected, src, MaxLines(6))
}

func TestIncompleteNestedCalls(t *testing.T) {
	src := `U"blah".upper(
		3.14.is_integer( `

	expected := `
CallExpr
	AttributeExpr[upper]
		StringExpr[U"blah"]
	Argument
		CallExpr
			AttributeExpr[is_integer]
				NumberExpr[3.14]
`
	assertParse(t, expected, src)
}

func TestIncompleteNestedCallsIncompleteAttr(t *testing.T) {
	src := `rf"blah".upper(
		a.b.c(x = b'\xab'.`

	expected := `
CallExpr
	AttributeExpr[upper]
		StringExpr[rf"blah"]
	Argument
		CallExpr
			AttributeExpr[c]
				AttributeExpr[b]
					NameExpr[a]
			Argument
				NameExpr[x]
				AttributeExpr[]
					StringExpr[b'\xab']
`
	assertParse(t, expected, src)
}

func TestMultiParens(t *testing.T) {
	src := `(( (
    (( 'hello' """
	world
""" )))
) ).method()`

	expected := `
CallExpr
	AttributeExpr[method]
		StringExpr['hello' """\n	world\n"""]
`
	assertParseWithOptions(t, expected, src, MaxLines(6))
}

func TestAwaitCall(t *testing.T) {
	src := `await foo(kw=(await bar()))`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[kw]
		CallExpr
			NameExpr[bar]
`
	assertParse(t, expected, src)
}

func TestMissingAttributeParts(t *testing.T) {
	src := `foo(a...b)`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		AttributeExpr[b]
			AttributeExpr[]
				AttributeExpr[]
					NameExpr[a]
`
	assertParse(t, expected, src)
}

func TestMissingAttributePartsUnclosed(t *testing.T) {
	src := `foo(a....`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		AttributeExpr[]
			AttributeExpr[]
				AttributeExpr[]
					AttributeExpr[]
						NameExpr[a]
`
	assertParse(t, expected, src)
}

func TestDottedCalls(t *testing.T) {
	src := `a().b.c(d).e(f=g, h)`

	expected := `
CallExpr
	AttributeExpr[e]
		CallExpr
			AttributeExpr[c]
				AttributeExpr[b]
					CallExpr
						NameExpr[a]
			Argument
				NameExpr[d]
	Argument
		NameExpr[f]
		NameExpr[g]
	Argument
		NameExpr[h]
`
	assertParse(t, expected, src)
}

func TestTooManyLinesReturnsPartialCall(t *testing.T) {
	src := `a(
		b,
		c,
		d,
		e,
)`

	// it parses b, c, d and an empty arg because before the 4th newline
	// there's `d` followed by a comma (so the parser knows there's another
	// argument).
	expected := `
CallExpr
	NameExpr[a]
	Argument
		NameExpr[b]
	Argument
		NameExpr[c]
	Argument
		NameExpr[d]
	Argument
		BadExpr
`
	node, err := Parse([]byte(src), MaxLines(4))
	assert.Error(t, err)
	assert.Equal(t, errors.ErrorReason(err), errors.TooManyLines)
	assertAST(t, expected, node, false)
}

func TestCallExprAsArgument(t *testing.T) {
	src := `foo(bar(), baz())`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		CallExpr
			NameExpr[bar]
	Argument
		CallExpr
			NameExpr[baz]
`
	assertParse(t, expected, src)
}

func TestMixedCallExprAsArgument(t *testing.T) {
	src := `foo("abc", b=c(), 123, 4.56, bar().baz.qux(z))`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		StringExpr["abc"]
	Argument
		NameExpr[b]
		CallExpr
			NameExpr[c]
	Argument
		NumberExpr[123]
	Argument
		NumberExpr[4.56]
	Argument
		CallExpr
			AttributeExpr[qux]
				AttributeExpr[baz]
					CallExpr
						NameExpr[bar]
			Argument
				NameExpr[z]
`
	assertParse(t, expected, src)
}

func TestPartialCallExprAsArgument(t *testing.T) {
	src := `foo(bar().)`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		AttributeExpr[]
			CallExpr
				NameExpr[bar]
`
	assertParse(t, expected, src)
}

func TestMaxExpressions(t *testing.T) {
	src := `foo(bar,star,123)`

	ce, err := Parse([]byte(src), MaxExpressions(3))
	assert.Nil(t, ce)
	assert.Equal(t, errors.MaxExpressionsLimit, errors.ErrorReason(err))
}

func TestTupleSyntax(t *testing.T) {
	src := `foo((1, 2, "a",))`

	expected := `
CallExpr
	NameExpr[foo]
	Argument
		TupleExpr
			NumberExpr[1]
			NumberExpr[2]
			StringExpr["a"]
`
	assertParse(t, expected, src)
}

func TestVararg(t *testing.T) {
	src := `foo(*args)`
	expected := `
CallExpr
	NameExpr[foo]
	NameExpr[args]
`
	assertParse(t, expected, src)
}

func TestKwarg(t *testing.T) {
	src := `foo(**kwargs)`
	expected := `
CallExpr
	NameExpr[foo]
	NameExpr[kwargs]
`
	assertParse(t, expected, src)
}

func TestCombineVarargKwarg(t *testing.T) {
	src := `foo(*args, **kwargs)`
	expected := `
CallExpr
	NameExpr[foo]
	NameExpr[args]
	NameExpr[kwargs]
`
	assertParse(t, expected, src)
}

func TestCombineArgsVarargKwarg(t *testing.T) {
	src := `foo(, b, *args, **kwargs)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	Argument
		NameExpr[b]
	NameExpr[args]
	NameExpr[kwargs]
`
	assertParse(t, expected, src)
}

func TestVarargLiteral(t *testing.T) {
	src := `foo(*["a", "b"])`
	expected := `
CallExpr
	NameExpr[foo]
	ListExpr
		StringExpr["a"]
		StringExpr["b"]
`
	assertParse(t, expected, src)
}

func TestKwargLiteral(t *testing.T) {
	src := `foo(**{"a": 1, "b": 2})`
	expected := `
CallExpr
	NameExpr[foo]
	DictExpr
		KeyValuePair
			StringExpr["a"]
			NumberExpr[1]
		KeyValuePair
			StringExpr["b"]
			NumberExpr[2]
`
	assertParse(t, expected, src)
}

func TestVarargKwargFunc(t *testing.T) {
	src := `foo("abc", *bar.baz(1, true), **get_kw(fn(x, y), z))`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		StringExpr["abc"]
	CallExpr
		AttributeExpr[baz]
			NameExpr[bar]
		Argument
			NumberExpr[1]
		Argument
			NameExpr[true]
	CallExpr
		NameExpr[get_kw]
		Argument
			CallExpr
				NameExpr[fn]
				Argument
					NameExpr[x]
				Argument
					NameExpr[y]
		Argument
			NameExpr[z]
`
	assertParse(t, expected, src)
}

func TestVarargKwargIncomplete(t *testing.T) {
	src := `foo(*args, **kwarg`
	expected := `
CallExpr
	NameExpr[foo]
	NameExpr[args]
	NameExpr[kwarg]
`
	assertParse(t, expected, src)
}

func TestVarargKwargMissingArg(t *testing.T) {
	src := `foo(,*args, **kwarg`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	NameExpr[args]
	NameExpr[kwarg]
`
	assertParse(t, expected, src)
}

func TestInvalidVararg(t *testing.T) {
	src := `foo(*args, b)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
			UnaryExpr[*]
				NameExpr[args]
	Argument
		NameExpr[b]
`
	assertParse(t, expected, src)
}

func TestInvalidKwarg(t *testing.T) {
	src := `foo(**kwarg, b)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
			UnaryExpr[**]
				NameExpr[kwarg]
	Argument
		NameExpr[b]
`
	assertParse(t, expected, src)
}

func TestInvalidVarargKwarg(t *testing.T) {
	src := `foo(**kwarg, *args, b)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
			UnaryExpr[**]
				NameExpr[kwarg]
	Argument
		BadExpr
			UnaryExpr[*]
				NameExpr[args]
	Argument
		NameExpr[b]
`
	assertParse(t, expected, src)
}

func TestValidAndInvalidVarargKwarg(t *testing.T) {
	src := `foo(**x, *y, b, *vararg, **kwarg)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
			UnaryExpr[**]
				NameExpr[x]
	Argument
		BadExpr
			UnaryExpr[*]
				NameExpr[y]
	Argument
		NameExpr[b]
	NameExpr[vararg]
	NameExpr[kwarg]
`
	assertParse(t, expected, src)
}

func TestEllipsis1(t *testing.T) {
	src := `foo(.)`
	expected := `
[   0...   6]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   5]	Argument
[   4...   5]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis2(t *testing.T) {
	src := `foo(..)`
	expected := `
[   0...   7]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   6]	Argument
[   4...   6]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis3(t *testing.T) {
	src := `foo(...)`
	expected := `
[   0...   8]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   7]	Argument
[   4...   7]		EllipsisExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis4(t *testing.T) {
	src := `foo(....)`
	expected := `
[   0...   9]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   8]	Argument
[   4...   8]		AttributeExpr[]
[   4...   7]			EllipsisExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis1ID(t *testing.T) {
	src := `foo(.id)`
	expected := `
[   0...   8]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   7]	Argument
[   4...   7]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis2ID(t *testing.T) {
	src := `foo(..id)`
	expected := `
[   0...   9]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   8]	Argument
[   4...   8]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis3ID(t *testing.T) {
	src := `foo(...id)`
	expected := `
[   0...  10]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   9]	Argument
[   4...   9]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis4ID(t *testing.T) {
	// NOTE: this parses as an ellipsis, followed by a dot and
	// an `id` attribute.

	src := `foo(....id)`
	expected := `
[   0...  11]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  10]	Argument
[   4...  10]		AttributeExpr[id]
[   4...   7]			EllipsisExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsisID1(t *testing.T) {
	src := `foo(id.)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		AttributeExpr[]
			NameExpr[id]
`
	assertParse(t, expected, src)
}

func TestEllipsisID2(t *testing.T) {
	src := `foo(id..)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		AttributeExpr[]
			AttributeExpr[]
				NameExpr[id]
`
	assertParse(t, expected, src)
}

func TestEllipsisID3(t *testing.T) {
	src := `foo(id...)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		AttributeExpr[]
			AttributeExpr[]
				AttributeExpr[]
					NameExpr[id]
`
	assertParse(t, expected, src)
}

func TestEllipsisID4(t *testing.T) {
	src := `foo(id....)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		AttributeExpr[]
			AttributeExpr[]
				AttributeExpr[]
					AttributeExpr[]
						NameExpr[id]
`
	assertParse(t, expected, src)
}

func TestEllipsis1Eq(t *testing.T) {
	src := `foo(.=)`
	expected := `
[   0...   7]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   6]	Argument
[   4...   6]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis2Eq(t *testing.T) {
	src := `foo(..=)`
	expected := `
[   0...   8]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   7]	Argument
[   4...   7]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis3Eq(t *testing.T) {
	src := `foo(...=)`
	expected := `
[   0...   9]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   8]	Argument
[   4...   8]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsis4Eq(t *testing.T) {
	src := `foo(....=)`
	expected := `
[   0...  10]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   9]	Argument
[   4...   9]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestEllipsisWithArg(t *testing.T) {
	src := `foo(a, b=c, ..., d)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		NameExpr[b]
		NameExpr[c]
	Argument
		EllipsisExpr
	Argument
		NameExpr[d]
`
	assertParse(t, expected, src)
}

func TestEllipsisUnclosedCall(t *testing.T) {
	src := `foo(a, ...`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		EllipsisExpr
`
	assertParse(t, expected, src)
}

func TestMultipleEllipsis(t *testing.T) {
	src := `foo(a, ..., b, ...)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		EllipsisExpr
	Argument
		NameExpr[b]
	Argument
		EllipsisExpr
`
	assertParse(t, expected, src)
}

func TestEllipsisValue(t *testing.T) {
	src := `foo(a = ...)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
		EllipsisExpr
`
	assertParse(t, expected, src)
}

func TestNestedEllipsis(t *testing.T) {
	src := `foo(a, ..., bar(x, ...))`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		EllipsisExpr
	Argument
		CallExpr
			NameExpr[bar]
			Argument
				NameExpr[x]
			Argument
				EllipsisExpr
`
	assertParse(t, expected, src)
}

func TestEmptyArgNone(t *testing.T) {
	src := `foo(`
	expected := `
CallExpr
	NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestEmptyArg1Comma(t *testing.T) {
	src := `foo(,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestEmptyArg2Commas(t *testing.T) {
	src := `foo(,,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	Argument
		BadExpr
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestEmptyArg3Commas(t *testing.T) {
	src := `foo(,,,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	Argument
		BadExpr
	Argument
		BadExpr
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestEmptyArgNoneClosed(t *testing.T) {
	src := `foo()`
	expected := `
CallExpr
	NameExpr[foo]
`
	assertParse(t, expected, src)
}

func TestEmptyArg1CommaClosed(t *testing.T) {
	src := `foo(,)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestEmptyArg2CommasClosed(t *testing.T) {
	src := `foo(,,)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestEmptyArg3CommasClosed(t *testing.T) {
	src := `foo(,,,)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
	Argument
		BadExpr
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestArgEmptyArgClosed(t *testing.T) {
	src := `foo(a, )`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
`
	assertParse(t, expected, src)
}

func TestArgsEmptyArgClosed(t *testing.T) {
	src := `foo(a, b, )`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		NameExpr[b]
`
	assertParse(t, expected, src)
}

func TestArg2EmptyArgsClosed(t *testing.T) {
	src := `foo(a, ,)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestArgEmptyArgUnclosed(t *testing.T) {
	src := `foo(a, `
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestArgsEmptyArgUnclosed(t *testing.T) {
	src := `foo(a, b, `
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		NameExpr[b]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestArg2EmptyArgsUnclosed(t *testing.T) {
	src := `foo(a, ,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		NameExpr[a]
	Argument
		BadExpr
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestKwargTrailingCommaClosed(t *testing.T) {
	src := `foo(**kwarg,)`
	expected := `
CallExpr
	NameExpr[foo]
	NameExpr[kwarg]
`
	assertParse(t, expected, src)
}

func TestKwargTrailingCommaUnclosed(t *testing.T) {
	// NOTE: this parses kwarg as a BadExpr because the comma + unclosed paren
	// implies an extra (empty) parameter, making the **kwarg invalid at this
	// position.
	src := `foo(**kwarg,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
			UnaryExpr[**]
				NameExpr[kwarg]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestVarargTrailingCommaClosed(t *testing.T) {
	src := `foo(*args,)`
	expected := `
CallExpr
	NameExpr[foo]
	NameExpr[args]
`
	assertParse(t, expected, src)
}

func TestVarargTrailingCommaUnclosed(t *testing.T) {
	// NOTE: this parses args as a BadExpr because the comma + unclosed paren
	// implies an extra (empty) parameter, making the *args invalid at this
	// position.
	src := `foo(*args,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
			UnaryExpr[*]
				NameExpr[args]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestVarargKwargTrailingCommaClosed(t *testing.T) {
	src := `foo(*args, **kwarg,)`
	expected := `
CallExpr
	NameExpr[foo]
	NameExpr[args]
	NameExpr[kwarg]
`
	assertParse(t, expected, src)
}

func TestVarargKwargTrailingCommaUnclosed(t *testing.T) {
	// NOTE: this parses args and kwarg as a BadExpr because the comma + unclosed paren
	// implies an extra (empty) parameter, making the star args invalid at this
	// position.
	src := `foo(*args, **kwarg,`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		BadExpr
			UnaryExpr[*]
				NameExpr[args]
	Argument
		BadExpr
			UnaryExpr[**]
				NameExpr[kwarg]
	Argument
		BadExpr
`
	assertParse(t, expected, src)
}

func TestInvalidArgs1(t *testing.T) {
	src := `foo(bar=*baz)`
	expected := `
[   0...  13]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  12]	Argument
[   4...   7]		NameExpr[bar]
[   8...  12]		UnaryExpr[*]
[   9...  12]			NameExpr[baz]
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs2(t *testing.T) {
	src := `foo(bar=1-, baz=0)`
	expected := `
[   0...  18]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  10]	Argument
[   4...  10]		BadExpr
[  12...  17]	Argument
[  12...  15]		NameExpr[baz]
[  16...  17]		NumberExpr[0]
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs3(t *testing.T) {
	src := `foo(bar ==)`
	expected := `
[   0...  11]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  10]	Argument
[   4...  10]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs4(t *testing.T) {
	src := `foo(bar for bar )`
	expected := `
[   0...  17]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  16]	Argument
[   4...  16]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs5(t *testing.T) {
	src := `foo(bar[], baz)`
	expected := `
[   0...  15]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   9]	Argument
[   4...   9]		BadExpr
[  11...  14]	Argument
[  11...  14]		NameExpr[baz]
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs6(t *testing.T) {
	src := `foo(bar[1//3], (baz.))`
	expected := `
[   0...  22]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  13]	Argument
[   4...  13]		BadExpr
[  16...  20]	Argument
[  16...  20]		AttributeExpr[]
[  16...  19]			NameExpr[baz]
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs7(t *testing.T) {
	src := `foo(bar, car baz)`
	expected := `
[   0...  17]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   7]	Argument
[   4...   7]		NameExpr[bar]
[   9...  16]	Argument
[   9...  16]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs8(t *testing.T) {
	src := `foo(bar=car baz)`
	expected := `
[   0...  16]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  15]	Argument
[   4...  15]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs9(t *testing.T) {
	// NOTE: in this case we stop at the newline and don't look at
	// the rest, as is the current proposal in the github issue.
	src := `foo(a b c
d, e)`
	expected := `
[   0...  15]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  11]	Argument
[   4...  11]		BadExpr
[  13...  14]	Argument
[  13...  14]		NameExpr[e]
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs10(t *testing.T) {
	// here the boundary is EOF
	src := `foo(a b c`
	expected := `
[   0...   9]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   9]	Argument
[   4...   9]		BadExpr
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs11(t *testing.T) {
	// with a comma after the invalid argument, the parser is back in
	// sync and can continue parsing.
	src := `foo(a b c,
d, e)`
	expected := `
[   0...  16]CallExpr
[   0...   3]	NameExpr[foo]
[   4...   9]	Argument
[   4...   9]		BadExpr
[  11...  12]	Argument
[  11...  12]		NameExpr[d]
[  14...  15]	Argument
[  14...  15]		NameExpr[e]
`
	assertParseWithOffsetsAndOptions(t, expected, src)
}

func TestInvalidArgs12(t *testing.T) {
	src := `foo(bar(x y, z), baz)`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		CallExpr
			NameExpr[bar]
			Argument
				BadExpr
			Argument
				NameExpr[z]
	Argument
		NameExpr[baz]
`
	assertParse(t, expected, src)
}

func TestInvalidStar1(t *testing.T) {
	// parens required around *foo otherwise it would be a UnaryExpr where the
	// Value is the CallExpr, and this (correctly) returns an error because the top-level
	// expression is not a CallExpr.
	src := `(*foo)(x)`
	expected := `
CallExpr
	UnaryExpr[*]
		NameExpr[foo]
	Argument
		NameExpr[x]
`
	assertParse(t, expected, src)
}

func TestInvalidArgsBalancedParens(t *testing.T) {
	src := `foo((bar, baz-))`
	expected := `
[   0...  16]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  15]	Argument
[   4...  15]		BadExpr
	`
	call := assertParseWithOffsets(t, expected, src)
	assert.Equal(t, "foo((bar, baz-))", src[call.Begin():call.End()])
	require.Len(t, call.Args, 1)
	arg := call.Args[0]
	assert.Equal(t, "(bar, baz-)", src[arg.Begin():arg.End()])
}

func TestMatchingParen1(t *testing.T) {
	src := `foo(bar(y)+)`
	expected := `
[   0...  12]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  11]	Argument
[   4...  11]		BadExpr
`
	assertParseWithOffsets(t, expected, src)
}

func TestMatchingParen2(t *testing.T) {
	src := `foo(((---)))`
	expected := `
[   0...  12]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  11]	Argument
[   4...  11]		BadExpr
`
	assertParseWithOffsets(t, expected, src)
}

func TestMatchingParen3(t *testing.T) {
	src := `range(len(arr)-)`
	expected := `
[   0...  16]CallExpr
[   0...   5]	NameExpr[range]
[   6...  15]	Argument
[   6...  15]		BadExpr
`
	assertParseWithOffsets(t, expected, src)
}

func TestMatchingParen4(t *testing.T) {
	// NOTE: this is the same as TestMatchingParen5, but with the missing comma
	// added after the tuple.
	src := `foo((bar, baz), car)`
	expected := `
[   0...  20]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  14]	Argument
[   4...  14]		TupleExpr
[   5...   8]			NameExpr[bar]
[  10...  13]			NameExpr[baz]
[  16...  19]	Argument
[  16...  19]		NameExpr[car]
`
	assertParseWithOffsets(t, expected, src)
}

func TestMatchingParen5(t *testing.T) {
	src := `foo((bar, baz) car)`
	expected := `
[   0...  19]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  18]	Argument
[   4...  18]		BadExpr
`
	assertParseWithOffsets(t, expected, src)
}

func TestMatchingParen6(t *testing.T) {
	// This is like TestMatchingParen5, but the invalid argument is nested in the
	// bar CallExpr, an argument to the foo CallExpr (to test that the parsing works
	// recursively).
	src := `foo(bar((bar, baz) car), qux)`
	expected := `
[   0...  29]CallExpr
[   0...   3]	NameExpr[foo]
[   4...  23]	Argument
[   4...  23]		CallExpr
[   4...   7]			NameExpr[bar]
[   8...  22]			Argument
[   8...  22]				BadExpr
[  25...  28]	Argument
[  25...  28]		NameExpr[qux]
`
	assertParseWithOffsets(t, expected, src)
}

func TestNestedCalls(t *testing.T) {
	src := `foo(bar(baz(qux(a, b), c), d))`
	expected := `
CallExpr
	NameExpr[foo]
	Argument
		CallExpr
			NameExpr[bar]
			Argument
				CallExpr
					NameExpr[baz]
					Argument
						CallExpr
							NameExpr[qux]
							Argument
								NameExpr[a]
							Argument
								NameExpr[b]
					Argument
						NameExpr[c]
			Argument
				NameExpr[d]
`
	assertParse(t, expected, src)
}
