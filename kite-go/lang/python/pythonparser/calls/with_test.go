package calls

import "testing"

func TestSimpleWith(t *testing.T) {
	src := `with a`
	expected := `
WithStmt
	WithItem
		NameExpr[a]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWithColon(t *testing.T) {
	src := `with a :`
	expected := `
[   0...   8]WithStmt
[   5...   6]	WithItem
[   5...   6]		NameExpr[a]
[   8...   8]	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWithItemAs(t *testing.T) {
	src := `with x as y:`
	expected := `
WithStmt
	WithItem
		NameExpr[x]
		NameExpr[y]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWithNoItem(t *testing.T) {
	src := `with :`
	expected := `
WithStmt
	WithItem
		BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWithNoItemNoColon(t *testing.T) {
	src := `with`
	expected := `
WithStmt
	WithItem
		BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWithMultiItems(t *testing.T) {
	src := `with a as b, c() as d, e(3, "a") as f:`
	expected := `
WithStmt
	WithItem
		NameExpr[a]
		NameExpr[b]
	WithItem
		CallExpr
			NameExpr[c]
		NameExpr[d]
	WithItem
		CallExpr
			NameExpr[e]
			Argument
				NumberExpr[3]
			Argument
				StringExpr["a"]
		NameExpr[f]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWithPartialExprItem(t *testing.T) {
	src := `with f(a, :`
	expected := `
WithStmt
	WithItem
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

func TestWithRequireWhitespace(t *testing.T) {
	src := `withf as g:`
	_, err := ParseStmt([]byte(src), MaxLines(testMaxLines))
	t.Log(src)
	assertParserErrorContains(t, err, "no match found")
}
