package calls

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/stretchr/testify/require"
)

func assertParseStmt(t *testing.T, expected string, src string) pythonast.Stmt {
	t.Log(src)
	node, err := newParser([]byte(src), MaxLines(testMaxLines)).parseStmt()
	require.NoError(t, err)
	require.NotNil(t, node)
	offsets := strings.HasPrefix(strings.TrimLeft(expected, "\n"), "[")
	assertAST(t, expected, node, offsets)
	return node
}

func TestSimpleClass(t *testing.T) {
	src := `class X`
	expected := `
[   0...   7]ClassDefStmt
[   6...   7]	NameExpr[X]
[   7...   7]	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestSimpleClassWithArgs(t *testing.T) {
	src := `class X(Y) :`
	expected := `
ClassDefStmt
	NameExpr[X]
	Argument
		NameExpr[Y]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestClassMultipleInheritance(t *testing.T) {
	src := `class X(mod.A, mod.B, C) :`
	expected := `
ClassDefStmt
	NameExpr[X]
	Argument
		AttributeExpr[A]
			NameExpr[mod]
	Argument
		AttributeExpr[B]
			NameExpr[mod]
	Argument
		NameExpr[C]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestClassPartialArgs1(t *testing.T) {
	src := `class X (A, ,  :`
	expected := `
ClassDefStmt
	NameExpr[X]
	Argument
		NameExpr[A]
	Argument
		BadExpr
	Argument
		BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestClassPartialArgs2(t *testing.T) {
	src := `class X (a=1,`
	expected := `
ClassDefStmt
	NameExpr[X]
	Argument
		NameExpr[a]
		NumberExpr[1]
	Argument
		BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestClassIgnoresBody(t *testing.T) {
	// For now, we only read the class without body
	src := `class X:
	a`
	expected := `
ClassDefStmt
	NameExpr[X]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestClassIgnoreBodyNoColon(t *testing.T) {
	src := `class X
	a`
	expected := `
ClassDefStmt
	NameExpr[X]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestClassRequireWhitespace(t *testing.T) {
	src := `classX`
	_, err := ParseStmt([]byte(src), MaxLines(testMaxLines))
	t.Log(src)
	assertParserErrorContains(t, err, "no match found")
}
