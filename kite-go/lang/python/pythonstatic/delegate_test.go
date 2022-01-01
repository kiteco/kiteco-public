package pythonstatic

import (
	"bytes"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type delegate struct {
	exprs  map[pythonast.Expr]pythontype.Value
	active bool
}

func (d *delegate) Pass(cur, total int) {
	if cur == total-1 {
		d.active = true
	}
}

func (d *delegate) Resolved(expr pythonast.Expr, val pythontype.Value) {
	if !d.active {
		return
	}
	d.exprs[expr] = val
}

func (d *delegate) MissingName(*pythonast.NameExpr) {}

func (d *delegate) MissingAttr(*pythonast.AttributeExpr, pythontype.Value) {}

func (d *delegate) MissingCall(*pythonast.CallExpr) {}

var (
	testUID = int64(3)
	testMID = "machine"
)

func requireName(t *testing.T, top pythonast.Node, name string) *pythonast.NameExpr {
	var ne *pythonast.NameExpr
	pythonast.Inspect(top, func(node pythonast.Node) bool {
		if ne != nil || node == nil {
			return false
		}

		toTry, ok := node.(*pythonast.NameExpr)
		if ok {
			if toTry.Ident.Literal == name {
				ne = toTry
			}
		}

		return true
	})
	require.NotNil(t, ne)
	return ne
}

func assertDelegate(t *testing.T, src string, expected []string) {
	opts := DefaultOptions
	opts.UseCapabilities = false

	delegate := &delegate{
		exprs: make(map[pythonast.Expr]pythontype.Value),
	}

	ai := AssemblerInputs{
		User:     testUID,
		Machine:  testMID,
		Graph:    pythonresource.MockManager(t, nil),
		Delegate: delegate,
	}
	assembler := NewAssembler(kitectx.Background(), ai, opts)

	ast, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err)
	assembler.AddSource(ASTBundle{AST: ast, Path: "/test.py", Imports: FindImports(kitectx.Background(), "/test.py", ast)})

	var buf bytes.Buffer
	assembler.SetTrace(&buf)
	_, err = assembler.Build(kitectx.Background())
	require.NoError(t, err)

	if buf.Len() > 0 {
		t.Log(buf.String())
	}

	for _, e := range expected {
		name := requireName(t, ast, e)

		assert.Contains(t, delegate.exprs, name, "unable to find name %s", e)
	}
}

func TestDelegateArgument(t *testing.T) {
	src := `
foo(bar=car)
	`

	assertDelegate(t, src, []string{
		"foo",
		"bar",
		"car",
	})
}

func TestDelegateGeneratorFilter(t *testing.T) {
	src := `
(foo for foo in bar if car)
	`
	assertDelegate(t, src, []string{
		"foo",
		"bar",
		"car",
	})
}

func TestDelegateAnnotationStmt(t *testing.T) {
	src := `
foo: bar
`
	assertDelegate(t, src, []string{
		"foo",
		"bar",
	})
}

func TestDelegateAnnotationAssign(t *testing.T) {
	src := `
foo: bar = baz
`
	assertDelegate(t, src, []string{
		"foo",
		"bar",
		"baz",
	})
}

func TestDelegateAnnotationVarargs(t *testing.T) {
	src := `
def foo(*args: bar, **kwargs: baz):
	pass
`
	assertDelegate(t, src, []string{
		"foo",
		"bar",
		"baz",
	})
}
