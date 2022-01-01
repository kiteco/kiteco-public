package calls

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertParseArgs(t *testing.T, expected string, src string, commas []int64) {
	t.Log(src)
	node, err := newParser([]byte(src), MaxLines(testMaxLines)).parseArguments()
	require.NoError(t, err)
	require.NotNil(t, node)
	assertAST(t, expected, node, false)

	if commas != nil {
		assertCommas(t, commas, node.Commas)
	}
}

func assertCommas(t *testing.T, expected []int64, actual []*pythonscanner.Word) {
	if len(expected) != len(actual) {
		t.Errorf("expected %d commas but got %d\n", len(expected), len(actual))
		return
	}

	for i, comma := range actual {
		expectedBegin := expected[i]
		assert.Equal(t, expectedBegin, int64(comma.Begin), "expected comma %d to begin at %d, got %d", i, expectedBegin, comma.Begin)
		assert.Equal(t, expectedBegin+1, int64(comma.End), "expected comma %d to end at %d, got %d", i, expectedBegin+1, comma.End)
	}
}

func assertParseArgsWithOffsets(t *testing.T, expected string, src string, commas []int64) {
	t.Log(src)
	node, err := newParser([]byte(src), MaxLines(testMaxLines)).parseArguments()
	require.NoError(t, err)
	require.NotNil(t, node)
	assertAST(t, expected, node, true)
	if commas != nil {
		assertCommas(t, commas, node.Commas)
	}
}

func TestArgsNoMatchFound(t *testing.T) {
	// those cases fail because the arguments-only parser expects an input
	// that starts with the left parenthesis.
	cases := []string{
		"",
		"\n",
		"a",
		"1",
		".()",
		"...",
	}
	for _, c := range cases {
		_, err := newParser([]byte(c), MaxLines(testMaxLines)).parseArguments()
		t.Log(c)
		assertParserErrorContains(t, err, "no match found")
	}
}

func TestEmpty(t *testing.T) {
	src := `()`
	expected := `
CallExpr
	NameExpr[]
`
	assertParseArgs(t, expected, src, nil)
}

func TestSingleArg(t *testing.T) {
	src := `(a)`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NameExpr[a]
`
	assertParseArgs(t, expected, src, nil)
}

func TestSingleArgWithOffsets(t *testing.T) {
	src := `( a )`
	expected := `
[   0...   5]CallExpr
[   0...   0]	NameExpr[]
[   2...   3]	Argument
[   2...   3]		NameExpr[a]
`
	assertParseArgsWithOffsets(t, expected, src, nil)
}

func TestMultipleArgs(t *testing.T) {
	src := `(a, b = c, d)`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NameExpr[a]
	Argument
		NameExpr[b]
		NameExpr[c]
	Argument
		NameExpr[d]
`
	assertParseArgs(t, expected, src, []int64{2, 9})
}

func TestUnclosedArgs(t *testing.T) {
	src := `(a`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NameExpr[a]
`
	assertParseArgs(t, expected, src, nil)
}

func TestMultilineArgs(t *testing.T) {
	src := `(a ,
b =
c)`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NameExpr[a]
	Argument
		NameExpr[b]
		NameExpr[c]
`
	assertParseArgs(t, expected, src, []int64{3})
}

func TestUnclosedMultilineArgs(t *testing.T) {
	src := `(a ,
b =
c`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NameExpr[a]
	Argument
		NameExpr[b]
		NameExpr[c]
`
	assertParseArgs(t, expected, src, []int64{3})
}

func TestLiteralArgs(t *testing.T) {
	src := `(1 , 2.3, true, "hi")`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NumberExpr[1]
	Argument
		NumberExpr[2.3]
	Argument
		NameExpr[true]
	Argument
		StringExpr["hi"]
`
	assertParseArgs(t, expected, src, []int64{3, 8, 14})
}

func TestUnclosedStringArg(t *testing.T) {
	src := `("hello`
	expected := `
CallExpr
	NameExpr[]
	Argument
		StringExpr["hello]
`
	assertParseArgs(t, expected, src, nil)
}

func TestMissingArgs(t *testing.T) {
	src := `(a ,, b=,c)`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NameExpr[a]
	Argument
		BadExpr
	Argument
		NameExpr[b]
		BadExpr
	Argument
		NameExpr[c]
`
	assertParseArgs(t, expected, src, []int64{3, 4, 8})
}

func TestArgumentsStructResult(t *testing.T) {
	src := `(a ,, b=,c)`
	args, err := ParseArguments([]byte(src))
	require.NoError(t, err)
	require.Equal(t, len(args.Args), 4)

	expected := []string{
		"Argument\n\tNameExpr[a]\n",
		"Argument\n\tBadExpr\n",
		"Argument\n\tNameExpr[b]\n\tBadExpr\n",
		"Argument\n\tNameExpr[c]\n",
	}
	for i, arg := range args.Args {
		assertAST(t, expected[i], arg, false)
	}
}

func TestArgsVararg(t *testing.T) {
	src := `(*args)`
	expected := `
CallExpr
	NameExpr[]
	NameExpr[args]
`
	assertParseArgs(t, expected, src, nil)
}

func TestArgsKwarg(t *testing.T) {
	src := `(**kwargs)`
	expected := `
CallExpr
	NameExpr[]
	NameExpr[kwargs]
`
	assertParseArgs(t, expected, src, nil)
}

func TestCombineArgVarargKwarg(t *testing.T) {
	src := `(x, *args, **kwargs)`
	expected := `
CallExpr
	NameExpr[]
	Argument
		NameExpr[x]
	NameExpr[args]
	NameExpr[kwargs]
`
	assertParseArgs(t, expected, src, []int64{2, 9})
}

func TestArgsEllipsis(t *testing.T) {
	src := `(...)`
	expected := `
CallExpr
	NameExpr[]
	Argument
		EllipsisExpr
`
	assertParseArgs(t, expected, src, nil)
}

func TestArgsEllipsisWithArg(t *testing.T) {
	src := `(a, b=c, ..., d)`
	expected := `
CallExpr
	NameExpr[]
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
	assertParseArgs(t, expected, src, []int64{2, 7, 12})
}

func TestArgsInvalidArgs1(t *testing.T) {
	src := `(bar=*baz)`
	expected := `
[   0...  10]CallExpr
[   0...   0]	NameExpr[]
[   1...   9]	Argument
[   1...   4]		NameExpr[bar]
[   5...   9]		UnaryExpr[*]
[   6...   9]			NameExpr[baz]
`
	assertParseArgsWithOffsets(t, expected, src, nil)
}

func TestArgsInvalidArgs2(t *testing.T) {
	src := `(a b,
c`
	expected := `
[   0...   7]CallExpr
[   0...   0]	NameExpr[]
[   1...   4]	Argument
[   1...   4]		BadExpr
[   6...   7]	Argument
[   6...   7]		NameExpr[c]
`
	assertParseArgsWithOffsets(t, expected, src, nil)
}

func TestArgsInvalidArgsBalancedParens(t *testing.T) {
	src := `((bar, baz-))`
	expected := `
[   0...  13]CallExpr
[   0...   0]	NameExpr[]
[   1...  12]	Argument
[   1...  12]		BadExpr
	`
	assertParseArgsWithOffsets(t, expected, src, nil)
}
