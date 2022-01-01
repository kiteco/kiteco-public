package pythonmetrics

import (
	"bytes"
	"go/token"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireCursorAndSource(t *testing.T, src string) (int64, []byte) {
	parts := strings.Split(src, "$")
	require.Len(t, parts, 2)

	return int64(len(parts[0])), []byte(strings.Join(parts, ""))
}

func assertASTNodeTrace(t *testing.T, tstSrc string, expected string) {
	cursor, src := requireCursorAndSource(t, tstSrc)

	ast, _ := pythonparser.Parse(kitectx.Background(), src, pythonparser.Options{
		ErrorMode:   pythonparser.Recover,
		Approximate: true,
		ScanOptions: pythonscanner.Options{
			ScanComments: true,
			ScanNewLines: true,
			Label:        "test.py",
		},
	})

	require.NotNil(t, ast)

	trace := newASTTrace(ASTTraceInputs{
		AST:    ast,
		Cursor: cursor,
	})

	assert.Equal(t, token.Pos(cursor), trace.Cursor)

	var buf bytes.Buffer
	trace.Print(&buf, false)
	actual := buf.String()

	// trim header
	idx := strings.Index(actual, "\n")
	actual = actual[idx+1:]

	actual = strings.TrimSpace(actual)

	expected = strings.TrimSpace(expected)

	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")

	m := len(actualLines)
	if m > len(expectedLines) {
		m = len(expectedLines)
	}

	for i := 0; i < m; i++ {
		al := strings.TrimSpace(actualLines[i])
		el := strings.TrimSpace(expectedLines[i])
		if al != el {
			t.Errorf("Mismatch on line %d:\nExpected: %s\nActual: %s\n", i, el, al)
		}
	}

	if len(expectedLines) > len(actualLines) {
		t.Error("Missing lines:")
		for i := m; i < len(expectedLines); i++ {
			t.Log(strings.TrimSpace(expectedLines[i]))
		}
	}

	if len(actualLines) > len(expectedLines) {
		t.Error("Got Extra lines:")
		for i := m; i < len(actualLines); i++ {
			t.Log(strings.TrimSpace(actualLines[i]))
		}
	}

}

func TestTraceName(t *testing.T) {
	src := `f$oo`

	expected := `
Module
- Body ->
ExprStmt
- Value ->
NameExpr
`

	assertASTNodeTrace(t, src, expected)
}

func TestTracePartialAttr(t *testing.T) {
	src := `foo.$`

	expected := `
Module
- Body ->
BadStmt
- Approximation ->
ExprStmt
- Value ->
AttributeExpr
`

	assertASTNodeTrace(t, src, expected)
}

func TestTraceAttr(t *testing.T) {
	src := `foo.$bar`

	expected := `
Module
- Body ->
ExprStmt
- Value ->
AttributeExpr
`

	assertASTNodeTrace(t, src, expected)
}

func TestTracePartialCall(t *testing.T) {
	src := `foo(bar=$`

	expected := `
Module
- Body ->
BadStmt
- Approximation ->
ExprStmt
- Value ->
CallExpr
- Args ->
Argument
- Value ->
BadExpr
	`

	assertASTNodeTrace(t, src, expected)
}

func TestTracePartialCallKeywordArg(t *testing.T) {
	src := `foo(bar=b$`

	expected := `
Module
- Body ->
BadStmt
- Approximation ->
ExprStmt
- Value ->
CallExpr
- Args ->
Argument
- Value ->
NameExpr
	`

	assertASTNodeTrace(t, src, expected)
}

func TestTraceCall(t *testing.T) {
	src := `foo(ba$)`

	expected := `
Module
- Body ->
ExprStmt
- Value ->
CallExpr
- Args ->
Argument
- Value ->
NameExpr
	`

	assertASTNodeTrace(t, src, expected)
}

func TestTraceEmptyCall(t *testing.T) {
	src := `foo($)`

	expected := `
Module
- Body ->
ExprStmt
- Value ->
CallExpr
`

	assertASTNodeTrace(t, src, expected)
}
