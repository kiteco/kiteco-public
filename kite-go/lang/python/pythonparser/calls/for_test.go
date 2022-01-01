package calls

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/stretchr/testify/require"
)

func TestSimpleFor(t *testing.T) {
	src := `for x in y`
	expected := `
ForStmt
	NameExpr[x]
	NameExpr[y]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForColon(t *testing.T) {
	src := `for x in y:`
	expected := `
[   0...  11]ForStmt
[   4...   5]	NameExpr[x]
[   9...  10]	NameExpr[y]
[  11...  11]	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForWithoutIn(t *testing.T) {
	src := `for x:`
	expected := `
ForStmt
	NameExpr[x]
	BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForEmptyIn(t *testing.T) {
	src := `for x in :`
	expected := `
ForStmt
	NameExpr[x]
	BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForNoCondNoIn(t *testing.T) {
	src := `for`
	// MaybeID returns an empty NameExpr in that case
	expected := `
ForStmt
	NameExpr[]
	BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForMultiList(t *testing.T) {
	src := `for x, y, z in fn():`
	expected := `
ForStmt
	NameExpr[x]
	NameExpr[y]
	NameExpr[z]
	CallExpr
		NameExpr[fn]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForPartialIn(t *testing.T) {
	src := `for x in f(a, :`
	expected := `
ForStmt
	NameExpr[x]
	CallExpr
		NameExpr[f]
		Argument
			NameExpr[a]
		Argument
			BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForMultiIn(t *testing.T) {
	src := `for a, b, c in x, y, z:`
	expected := `
ForStmt
	NameExpr[a]
	NameExpr[b]
	NameExpr[c]
	TupleExpr
		NameExpr[x]
		NameExpr[y]
		NameExpr[z]
	BadStmt
`
	node := assertParseStmt(t, expected, src)
	tuple := node.(*pythonast.ForStmt).Iterable.(*pythonast.TupleExpr)
	require.Len(t, tuple.Commas, 2)
}

func TestForEmptyTargetsWithIn(t *testing.T) {
	src := `for in x`
	expected := `
ForStmt
	NameExpr[]
	NameExpr[x]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestForRequireWhitespace(t *testing.T) {
	src := `forx in y`
	_, err := ParseStmt([]byte(src), MaxLines(testMaxLines))
	t.Log(src)
	assertParserErrorContains(t, err, "no match found")
}
