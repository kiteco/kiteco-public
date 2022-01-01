package calls

import (
	"testing"
)

func TestSimpleFunction(t *testing.T) {
	src := `def x`
	expected := `
FunctionDefStmt
	NameExpr[x]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionIgnoreBody(t *testing.T) {
	src := `def x :
a`
	expected := `
[   0...   7]FunctionDefStmt
[   4...   5]	NameExpr[x]
[   7...   7]	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionIgnoreBodyNoColon(t *testing.T) {
	src := `def x 
a`
	expected := `
[   0...   7]FunctionDefStmt
[   4...   5]	NameExpr[x]
[   7...   7]	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionParens(t *testing.T) {
	src := `def x():`
	expected := `
FunctionDefStmt
	NameExpr[x]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionArgs(t *testing.T) {
	src := `def x(y, z):`
	expected := `
[   0...  12]FunctionDefStmt
[   4...   5]	NameExpr[x]
[   6...   7]	Parameter
[   6...   7]		NameExpr[y]
[   9...  10]	Parameter
[   9...  10]		NameExpr[z]
[  12...  12]	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionArgsDefaults(t *testing.T) {
	src := `def x(y = "a", z = 3):`
	expected := `
FunctionDefStmt
	NameExpr[x]
	Parameter
		NameExpr[y]
		StringExpr["a"]
	Parameter
		NameExpr[z]
		NumberExpr[3]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionArgsVararg(t *testing.T) {
	src := `def x(y = "a", z = 3, *v)`
	expected := `
FunctionDefStmt
	NameExpr[x]
	Parameter
		NameExpr[y]
		StringExpr["a"]
	Parameter
		NameExpr[z]
		NumberExpr[3]
	ArgsParameter
		NameExpr[v]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionArgsKwarg(t *testing.T) {
	src := `def x(y = "a", z = 3, **k)`
	expected := `
FunctionDefStmt
	NameExpr[x]
	Parameter
		NameExpr[y]
		StringExpr["a"]
	Parameter
		NameExpr[z]
		NumberExpr[3]
	ArgsParameter
		NameExpr[k]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionArgsPartial(t *testing.T) {
	src := `def x(y, , z=, *v`
	expected := `
FunctionDefStmt
	NameExpr[x]
	Parameter
		NameExpr[y]
	Parameter
		BadExpr
	Parameter
		NameExpr[z]
		BadExpr
	ArgsParameter
		NameExpr[v]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionArgsBadExpr(t *testing.T) {
	src := `def x(22=x, **"s", *3=4)`
	expected := `
FunctionDefStmt
	NameExpr[x]
	Parameter
		BadExpr
	Parameter
		BadExpr
			UnaryExpr[**]
				StringExpr["s"]
	Parameter
		BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestFunctionRequireWhitespace(t *testing.T) {
	src := `defx`
	_, err := ParseStmt([]byte(src), MaxLines(testMaxLines))
	t.Log(src)
	assertParserErrorContains(t, err, "no match found")
}
