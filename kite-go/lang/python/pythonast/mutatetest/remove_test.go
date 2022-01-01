package mutatetest

import (
	"go/token"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type removeCase struct {
	Begin   token.Pos
	End     token.Pos
	Full    string
	Removed string
}

func requireRemoveCase(t *testing.T, src string) removeCase {
	parts := strings.Split(src, "$")
	switch len(parts) {
	case 2:
		return removeCase{
			Begin:   token.Pos(len(parts[0])),
			End:     token.Pos(len(parts[0]) + len(parts[1])),
			Full:    strings.Join(parts, ""),
			Removed: parts[0],
		}
	case 3:
		return removeCase{
			Begin: token.Pos(len(parts[0])),
			End:   token.Pos(len(parts[0]) + len(parts[1])),
			Full:  strings.Join(parts, ""),
			Removed: strings.Join([]string{
				parts[0],
				parts[2],
			}, ""),
		}
	default:
		t.Errorf("invalid test source:\n%s\n", src)
		t.FailNow()
		return removeCase{}
	}
}

func assertRemoveNode(t *testing.T, src string) {
	c := requireRemoveCase(t, src)

	t.Log("test case")
	t.Logf("begin: %d, end: %d\n", c.Begin, c.End)
	t.Logf("full source:\n%s\n", c.Full)
	t.Logf("removed source:\n%s\n", c.Removed)

	// parse full source
	fullMod := requireParsed(t, c.Full)

	// find deepest node to remove
	var nodes []pythonast.Node
	pythonast.Inspect(fullMod, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}

		if c.Begin >= n.Begin() && c.End <= n.End() {
			nodes = append(nodes, n)
		}
		return true
	})

	if len(nodes) == 0 {
		t.Error("unable to find node to remove")
		return
	}

	// TODO: pretty hacky, pick the deepest node that
	// exactly covers all nodes that are deeper than it
	var remove pythonast.Node
	for i := len(nodes) - 2; i > -1; i-- {
		if nodes[i].Begin() == nodes[i+1].Begin() && nodes[i].End() == nodes[i+1].End() {
			continue
		}
		remove = nodes[i+1]
		break
	}

	t.Logf("removing node:\n%s\n", printNode(remove))

	// remove the node
	require.NoError(t, pythonast.Replace(fullMod, remove, nil))

	// parse the removed source code
	removedMod := requireParsed(t, c.Removed)

	assertAST(t, printNode(removedMod), printNode(fullMod))
}

func TestRemoveArg(t *testing.T) {
	src := `foo($bar$)`

	assertRemoveNode(t, src)
}

func requireCallByName(t *testing.T, ast *pythonast.Module, fn string) *pythonast.CallExpr {
	var c *pythonast.CallExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if c != nil || pythonast.IsNil(n) {
			return false
		}

		call, ok := n.(*pythonast.CallExpr)
		if !ok {
			return true
		}

		if name, ok := call.Func.(*pythonast.NameExpr); ok {
			if fn == name.Ident.Literal {
				c = call
			}
		}

		return true
	})
	require.NotNil(t, c, "unable to find call %s", fn)
	return c
}

func TestRemoveCallArgs(t *testing.T) {
	src := `foo($bar,car,star$)`

	c := requireRemoveCase(t, src)

	ast := requireParsed(t, c.Full)

	call := requireCallByName(t, ast, "foo")

	pythonast.RemoveArgs(ast, call)

	assertAST(t, printNode(requireParsed(t, c.Removed)), printNode(ast))

	assert.Equal(t, token.Pos(3), call.LeftParen.Begin)
	assert.Equal(t, token.Pos(4), call.LeftParen.End)
	assert.Equal(t, token.Pos(4), call.RightParen.Begin)
	assert.Equal(t, token.Pos(5), call.RightParen.End)
}
