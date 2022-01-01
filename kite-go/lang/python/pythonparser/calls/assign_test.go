package calls

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

func TestSimpleAssign(t *testing.T) {
	src := `x = 1`
	expected := `
[   0...   5]AssignStmt
[   0...   1]	NameExpr[x]
[   4...   5]	NumberExpr[1]
`
	assertParseStmt(t, expected, src)
}

func TestDottedAssign(t *testing.T) {
	src := `x.y.z = "a"`
	expected := `
[   0...  11]AssignStmt
[   0...   5]	AttributeExpr[z]
[   0...   3]		AttributeExpr[y]
[   0...   1]			NameExpr[x]
[   8...  11]	StringExpr["a"]
`
	assertParseStmt(t, expected, src)
}

func TestAugAssignOps(t *testing.T) {
	src := `x%s1`
	expected := `
AugAssignStmt[%s]
	NameExpr[x]
	NumberExpr[1]
`

	// ignore comparison ops that end with "=", those are not assignments
	comparisonOps := map[string]bool{
		pythonscanner.Eq.String(): true,
		pythonscanner.Le.String(): true,
		pythonscanner.Ge.String(): true,
		pythonscanner.Ne.String(): true,
	}
	for _, tok := range pythonscanner.OperatorTokens {
		op := tok.String()
		if len(op) > 1 && op[len(op)-1] == '=' && !comparisonOps[op] {
			t.Run(op, func(t *testing.T) {
				assertParseStmt(t, fmt.Sprintf(expected, op), fmt.Sprintf(src, op))
			})
		}
	}
}

func TestAssignMissingRHS(t *testing.T) {
	src := `x.y = `
	expected := `
AssignStmt
	AttributeExpr[y]
		NameExpr[x]
	BadExpr
`
	assertParseStmt(t, expected, src)
}

func TestAssignPartialRHS(t *testing.T) {
	src := `x.y = fn(a,,3`
	expected := `
AssignStmt
	AttributeExpr[y]
		NameExpr[x]
	CallExpr
		NameExpr[fn]
		Argument
			NameExpr[a]
		Argument
			BadExpr
		Argument
			NumberExpr[3]
`
	assertParseStmt(t, expected, src)
}
