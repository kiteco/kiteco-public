package mutatetest

import (
	"go/token"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type replaceCase struct {
	Begin    token.Pos
	End      token.Pos
	Insert   string
	Original string
	Replaced string
}

func requireReplaceCase(t *testing.T, src string, insert string) replaceCase {
	parts := strings.Split(src, "$")
	if len(parts) != 3 {
		t.Errorf("invalid test source:\n%s\n", src)
		t.FailNow()
		return replaceCase{}
	}

	return replaceCase{
		Begin:    token.Pos(len(parts[0])),
		End:      token.Pos(len(parts[0]) + len(parts[1])),
		Insert:   insert,
		Original: strings.Join(parts, ""),
		Replaced: strings.Join([]string{
			parts[0],
			insert,
			parts[2],
		}, ""),
	}

}

func requireNodeToReplace(t *testing.T, begin, end token.Pos, mod *pythonast.Module) pythonast.Node {
	// find deepest node to remove
	var replace pythonast.Node
	pythonast.Inspect(mod, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}

		if begin >= n.Begin() && end <= n.End() {
			replace = n
		}
		return true
	})

	if pythonast.IsNil(replace) {
		t.Error("unable to find node to replace")
		t.FailNow()
		return nil
	}

	return replace
}

func requireExprToInsert(t *testing.T, insert string) pythonast.Expr {
	mod := requireParsed(t, insert)
	var expr pythonast.Expr
	pythonast.Inspect(mod, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !pythonast.IsNil(expr) {
			return false
		}

		if e, ok := n.(pythonast.Expr); ok {
			expr = e
		}

		return true
	})

	if pythonast.IsNil(expr) {
		t.Error("unable to find expression to replace in insert text")
		t.FailNow()
		return nil
	}
	return expr
}

func assertReplaceNodeExpr(t *testing.T, src string, insert string) {
	c := requireReplaceCase(t, src, insert)

	t.Log("test case")
	t.Logf("begin: %d, end: %d\n", c.Begin, c.End)
	t.Logf("original source:\n%s\n", c.Original)
	t.Logf("replaced source:\n%s\n", c.Replaced)
	t.Logf("inserting:\n%s\n", c.Insert)

	// parse original source
	actualMod := requireParsed(t, c.Original)

	replace := requireNodeToReplace(t, c.Begin, c.End, actualMod)

	t.Logf("replacing node:\n%s\n", printNode(replace))

	// parse the insert code and pick the shallowest expression
	newNode := requireExprToInsert(t, c.Insert)

	// replace the node
	require.NoError(t, pythonast.Replace(actualMod, replace, newNode))

	// parse the replaced source code
	expectedMod := requireParsed(t, c.Replaced)

	assertAST(t, printNode(expectedMod), printNode(actualMod))
}

func TestReplaceArg(t *testing.T) {
	src := `foo(bar,$baz$,bang)`

	insert := `1 + 2`

	assertReplaceNodeExpr(t, src, insert)
}

func TestInvalidReplace(t *testing.T) {
	src := `class $foo$(): pass`

	c := requireReplaceCase(t, src, `1 + 2`)

	mod := requireParsed(t, c.Original)

	replace := requireNodeToReplace(t, c.Begin, c.End, mod)

	insert := requireExprToInsert(t, c.Insert)

	assert.Error(t, pythonast.Replace(mod, replace, insert))
}
