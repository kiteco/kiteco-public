package calls

import "testing"

func TestSimpleIf(t *testing.T) {
	src := `if a`
	expected := `
IfStmt
	Branch
		NameExpr[a]
		BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestIfColon(t *testing.T) {
	src := `if a :`
	expected := `
[   0...   6]IfStmt
[   3...   6]	Branch
[   3...   4]		NameExpr[a]
[   6...   6]		BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestIfCond(t *testing.T) {
	src := `if (a.b(c)):`
	expected := `
IfStmt
	Branch
		CallExpr
			AttributeExpr[b]
				NameExpr[a]
			Argument
				NameExpr[c]
		BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestIfNoCond(t *testing.T) {
	src := `if :`
	expected := `
IfStmt
	Branch
		BadExpr
		BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestIfNoCondNoColon(t *testing.T) {
	src := `if`
	expected := `
IfStmt
	Branch
		BadExpr
		BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestIfPartialCond(t *testing.T) {
	src := `if a.b(,c`
	expected := `
IfStmt
	Branch
		CallExpr
			AttributeExpr[b]
				NameExpr[a]
			Argument
				BadExpr
			Argument
				NameExpr[c]
		BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestIfRequireWhitespace(t *testing.T) {
	src := `ifx:`
	_, err := ParseStmt([]byte(src), MaxLines(testMaxLines))
	t.Log(src)
	assertParserErrorContains(t, err, "no match found")
}
