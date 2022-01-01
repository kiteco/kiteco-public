package calls

import "testing"

func TestSimpleWhile(t *testing.T) {
	src := `while a`
	expected := `
WhileStmt
	NameExpr[a]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWhileColon(t *testing.T) {
	src := `while a :`
	expected := `
[   0...   9]WhileStmt
[   6...   7]	NameExpr[a]
[   9...   9]	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWhileCond(t *testing.T) {
	src := `while (a.b(c)):`
	expected := `
WhileStmt
	CallExpr
		AttributeExpr[b]
			NameExpr[a]
		Argument
			NameExpr[c]
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWhileNoCond(t *testing.T) {
	src := `while :`
	expected := `
WhileStmt
	BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWhileNoCondNoColon(t *testing.T) {
	src := `while`
	expected := `
WhileStmt
	BadExpr
	BadStmt
`
	assertParseStmt(t, expected, src)
}

func TestWhilePartialCond(t *testing.T) {
	src := `while a.b(,c`
	expected := `
WhileStmt
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

func TestWhileRequireWhitespace(t *testing.T) {
	src := `whilex :`
	_, err := ParseStmt([]byte(src), MaxLines(testMaxLines))
	t.Log(src)
	assertParserErrorContains(t, err, "no match found")
}
