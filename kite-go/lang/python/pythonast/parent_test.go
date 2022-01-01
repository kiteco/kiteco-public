package pythonast_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireAST(t *testing.T, src string) *pythonast.Module {
	mod, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err)
	require.NotNil(t, mod)

	return mod
}

func requireExpr(t *testing.T, node pythonast.Node, repr string) pythonast.Expr {
	var expr pythonast.Expr
	pythonast.Inspect(node, func(n pythonast.Node) bool {
		if n == nil {
			return false
		}

		if expr != nil {
			return false
		}

		if e, ok := n.(pythonast.Expr); ok {
			if pythonast.String(e) == repr {
				expr = e
			}
		}
		return true
	})
	require.False(t, pythonast.IsNil(expr), "unable to find node %s", repr)
	return expr
}

func TestStmtTable(t *testing.T) {
	src := `
def foo():
	bar
class car(): pass
`

	ast := requireAST(t, src)

	parents := pythonast.ConstructStmtTable(ast, 0)
	require.NotNil(t, parents)

	foo := requireExpr(t, ast, "NameExpr[foo]")
	parent := parents[foo]

	require.IsType(t, &pythonast.FunctionDefStmt{}, parent)
	assert.Equal(t, "foo", parent.(*pythonast.FunctionDefStmt).Name.Ident.Literal)

	bar := requireExpr(t, ast, "NameExpr[bar]")
	parent = parents[bar]

	require.IsType(t, &pythonast.ExprStmt{}, parent)
	assert.Equal(t, bar, parent.(*pythonast.ExprStmt).Value)

	car := requireExpr(t, ast, "NameExpr[car]")
	parent = parents[car]
	require.IsType(t, &pythonast.ClassDefStmt{}, parent)
	assert.Equal(t, "car", parent.(*pythonast.ClassDefStmt).Name.Ident.Literal)

}

func TestParentTable(t *testing.T) {
	src := `
def foo():
	bar
class car(): pass
	`

	ast := requireAST(t, src)

	parents := pythonast.ConstructParentTable(ast, 0)
	require.NotNil(t, parents)

	foo := requireExpr(t, ast, "NameExpr[foo]")
	parent := parents[foo]

	require.IsType(t, &pythonast.FunctionDefStmt{}, parent)
	assert.Equal(t, "foo", parent.(*pythonast.FunctionDefStmt).Name.Ident.Literal)

	bar := requireExpr(t, ast, "NameExpr[bar]")
	parent = parents[bar]

	require.IsType(t, &pythonast.ExprStmt{}, parent)
	assert.Equal(t, bar, parent.(*pythonast.ExprStmt).Value)

	parent = parents[parent]
	require.IsType(t, &pythonast.FunctionDefStmt{}, parent)
	assert.Equal(t, "foo", parent.(*pythonast.FunctionDefStmt).Name.Ident.Literal)

	car := requireExpr(t, ast, "NameExpr[car]")
	parent = parents[car]
	require.IsType(t, &pythonast.ClassDefStmt{}, parent)
	assert.Equal(t, "car", parent.(*pythonast.ClassDefStmt).Name.Ident.Literal)
}

func requireName(t *testing.T, ast pythonast.Node, name string) *pythonast.NameExpr {
	var ne *pythonast.NameExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if n == nil || ne != nil {
			return false
		}

		if nn, ok := n.(*pythonast.NameExpr); ok {
			if fmt.Sprintf("NameExpr[%s]", name) == pythonast.String(nn) {
				ne = nn
			}
		}

		return true
	})
	var buf bytes.Buffer
	pythonast.Print(ast, &buf, "\t")

	require.NotNil(t, ne, "unable to find name %s in ast:\n%s", name, buf.String())
	return ne
}

func TestScopeTable(t *testing.T) {
	src := `
@decorator
def foo(bar:far, l=f):
	car
	star
	def foo1():
		foobar
@lar
class mar(arr):
	x = 1
	@dec.orator
	def zer():
		lambda z: x + 1
[xx for xx in []]
`

	ast := requireAST(t, src)

	scopes := pythonast.ConstructScopeTable(ast)
	parent := pythonast.ConstructParentTable(ast, 0)

	// function name is resolved in parent scope
	foo := requireName(t, ast, "foo")
	assert.Equal(t, ast, scopes[foo])
	fn := parent[foo].(*pythonast.FunctionDefStmt)

	// names in function decorators are resolved in parent scope
	decorator := requireName(t, ast, "decorator")
	assert.Equal(t, ast, scopes[decorator])

	// function parameter names are resolved in function scope
	bar := requireName(t, ast, "bar")
	assert.Equal(t, fn, scopes[bar])
	l := requireName(t, ast, "l")
	assert.Equal(t, fn, scopes[l])

	// names in function parameter annotations are resolved in parent scope
	ann := parent[bar].(*pythonast.Parameter).Annotation.(*pythonast.NameExpr)
	assert.Equal(t, ast, scopes[ann])

	// names in function parameter defaults are resolved in parent scope
	def := parent[l].(*pythonast.Parameter).Default.(*pythonast.NameExpr)
	assert.Equal(t, ast, scopes[def])

	// names in funciton body resolved in function scope
	car := requireName(t, ast, "car")
	assert.Equal(t, fn, scopes[car])
	star := requireName(t, ast, "star")
	assert.Equal(t, fn, scopes[star])

	// names of nested function is resolved in parent function scope
	foo1 := requireName(t, ast, "foo1")
	assert.Equal(t, fn, scopes[foo1])
	foobar := requireName(t, ast, "foobar")
	fn = parent[foo1].(*pythonast.FunctionDefStmt)
	assert.Equal(t, fn, scopes[foobar])

	// names in class decorator are resolved in parent scope
	lar := requireName(t, ast, "lar")
	assert.Equal(t, ast, scopes[lar])

	// names in class arguments are resolved in parent scope
	arr := requireName(t, ast, "arr")
	assert.Equal(t, ast, scopes[arr])

	// class name is resolved in parent scope
	mar := requireName(t, ast, "mar")
	assert.Equal(t, ast, scopes[mar])

	cls := parent[mar].(*pythonast.ClassDefStmt)

	// names in class body are resolved in class scope
	x := requireName(t, ast, "x")
	assert.Equal(t, cls, scopes[x])

	// names in decorators for methods are resolved in class scope
	dec := requireName(t, ast, "dec")
	assert.Equal(t, cls, scopes[dec])

	// method names are resolved in class scope
	zer := requireName(t, ast, "zer")
	assert.Equal(t, cls, scopes[zer])

	// names in paramters for lambda functions are resolved in lambda scope
	z := requireName(t, ast, "z")
	lambda := parent[parent[z]].(*pythonast.LambdaExpr)
	assert.Equal(t, lambda, scopes[z])

	// names in comprehensions are resolved starting in the comprehension scope
	// NOTE: see comment for ConstructScopeTable, this matches the python 3 behavior.
	xx := requireName(t, ast, "xx")
	listComp := parent[xx].(*pythonast.ListComprehensionExpr)
	assert.Equal(t, listComp, scopes[xx])

}
