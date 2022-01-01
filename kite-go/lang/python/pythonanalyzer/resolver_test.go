package pythonanalyzer

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var emptyManager pythonresource.Manager

func getEmptyManager(t *testing.T) pythonresource.Manager {
	if emptyManager == nil {
		emptyManager = pythonresource.MockManager(t, nil)
	}
	return emptyManager
}

// Find a node in an AST given the source for the node
func findExpr(root pythonast.Node, s string, orig string) pythonast.Expr {
	var ret pythonast.Expr
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		// ignore DottedExpr since they can make us fail to find the NameExpr inside
		expr, isexpr := node.(pythonast.Expr)
		_, isdotted := node.(*pythonast.DottedExpr)
		if isexpr && !isdotted && orig[expr.Begin():expr.End()] == s {
			ret = expr
		}
		return ret == nil
	})
	return ret
}

// Find a name in an AST
func findName(root pythonast.Node, name string) *pythonast.NameExpr {
	var ret *pythonast.NameExpr
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		expr, ok := node.(*pythonast.NameExpr)
		if ok && expr.Ident.Literal == name {
			ret = expr
		}
		return ret == nil
	})
	return ret
}

// Find a function def in an AST
func findFunctionDef(root pythonast.Node, name string) *pythonast.FunctionDefStmt {
	var ret *pythonast.FunctionDefStmt
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		funcdef, isfunc := node.(*pythonast.FunctionDefStmt)
		if isfunc && funcdef.Name.Ident.Literal == name {
			ret = funcdef
		}
		return ret == nil
	})
	return ret
}

// Find a class def in an AST
func findClassDef(root pythonast.Node, name string) *pythonast.ClassDefStmt {
	var ret *pythonast.ClassDefStmt
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		classdef, isclass := node.(*pythonast.ClassDefStmt)
		if isclass && classdef.Name.Ident.Literal == name {
			ret = classdef
		}
		return ret == nil
	})
	return ret
}

type reference struct {
	Value      pythontype.Value
	Expression pythonast.Expr
}

type byPosition []*reference

func (xs byPosition) Len() int           { return len(xs) }
func (xs byPosition) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byPosition) Less(i, j int) bool { return xs[i].Expression.Begin() < xs[j].Expression.Begin() }

func valueString(v pythontype.Value) string {
	if v == nil {
		return "<nil>"
	}
	vs := fmt.Sprintf("%v", v)
	if strings.HasPrefix(vs, "generic:") {
		return strings.TrimPrefix(vs, "generic:")
	}
	return vs
}

func assertResolve(t *testing.T, src string, graph pythonresource.Manager, expected map[string]string) *ResolvedAST {
	t.Helper()
	for i, line := range strings.Split(src, "\n") {
		t.Logf("%3d  %s", i+1, line)
	}

	var parseOpts pythonparser.Options
	mod, err := pythonparser.Parse(kitectx.Background(), []byte(src), parseOpts)
	require.NoError(t, err)

	var trace bytes.Buffer
	r := NewResolver(graph, Options{
		Path:  "/__main__.py",
		Trace: &trace,
	})

	result, err := r.Resolve(mod)
	require.NoError(t, err)

	t.Log(trace.String())

	file := pythonscanner.File([]byte(src))
	require.NotNil(t, file)

	// Print references
	var refs []*reference
	for expr, value := range result.References {
		refs = append(refs, &reference{
			Value:      value,
			Expression: expr,
		})
	}
	sort.Sort(byPosition(refs))

	for _, ref := range refs {
		s := pythonast.String(ref.Expression)
		require.NotNil(t, ref.Expression)
		line := file.Line(ref.Expression.Begin())
		require.NotNil(t, ref, "nil value found in references (for %s)", s)
		require.NotNil(t, ref.Expression, "nil expression found in references (for %s)", s)
		t.Logf("%25s (line %2d) -> %-40s", s, line, valueString(ref.Value))
	}

	// Check that the types match their expected values
	for exprStr, expectedName := range expected {
		expr := findExpr(mod, exprStr, src)
		require.NotNil(t, expr, "could not find AST node for '%s'", exprStr)
		line := file.Line(expr.Begin())
		ref, hasref := result.References[expr]

		t.Logf("resolving %s (line %d)", pythonast.String(expr), line)
		if !assert.True(t, hasref, "expected %s to resolve to %s but no reference found", exprStr, expectedName) {
			continue
		}
		assertValue(t, expectedName, ref, exprStr)
	}

	// check that every expression appears in the list of references
	pythonast.Inspect(mod, func(n pythonast.Node) bool {
		switch n := n.(type) {
		case *pythonast.DottedExpr:
			// ignore
			return true
		case pythonast.Expr:
			if _, found := result.References[n]; !found {
				t.Errorf("No reference for %s (%d...%d)", pythonast.String(n), n.Begin(), n.End())
			}
		}
		return true
	})

	return result
}

func assertValue(t *testing.T, expectedName string, v pythontype.Value, expr string) {
	if expectedName == "unknown" {
		assert.Nil(t, v, "expected node for %s to be nil but got %v", expr, v)
		return
	}
	if !assert.NotNil(t, v, "expected %s to resolve to %s but node was nil", expr, expectedName) {
		return
	}
	if strings.HasPrefix(expectedName, "instanceof ") {
		if !assert.NotNil(t, v.Type(),
			"expected %s to resolve to %s but got %s", expr, expectedName, v) {
			return
		}
		expectedType := strings.TrimPrefix(expectedName, "instanceof ")
		assert.Equal(t, expectedType, valueString(v.Type()),
			"expected %s to resolve to %s but got instanceof %s", expr, expectedName, valueString(v.Type()))
	} else {
		assert.Equal(t, expectedName, valueString(v),
			"expected %s to resolve to %s but got %s", expr, expectedName, valueString(v))
	}
}

func assertHasMemeber(t *testing.T, v pythontype.Value, name, attr, cn string) pythontype.Value {
	if !assert.NotNil(t, v) {
		return nil
	}
	child, _ := pythontype.AttrNoCtx(v, attr)
	if !assert.NotNil(t, child.Value()) {
		return nil
	}
	assertValue(t, cn, child.Value(), name+"."+attr)
	return child.Value()
}

func TestComprehension(t *testing.T) {
	src := `(ch for ch in [' ', '\n', '\r'])`

	assertResolve(t, src, getEmptyManager(t), map[string]string{})
}

func TestBuiltin(t *testing.T) {
	src := `x = map(str, [1, 2, 3])`

	graph := pythonresource.MockManager(t, nil, "builtins.map", "builtins.str")
	assertResolve(t, src, graph, map[string]string{
		"map": "builtins.map",
		"str": "builtins.str",
	})
}

func TestOverrideBuiltin(t *testing.T) {
	src := `
def foo():
	str = 123
	x = str

y = str`

	graph := pythonresource.MockManager(t, nil, "builtins.str")
	assertResolve(t, src, graph, map[string]string{
		"x": "instanceof builtins.int",
		"y": "builtins.str",
	})
}

func TestImport(t *testing.T) {
	src := `
import foo
foo.bar`
	graph := pythonresource.MockManager(t, nil, "foo", "foo.bar")
	assertResolve(t, src, graph, map[string]string{
		"foo.bar": "external:foo:foo.bar",
	})
}

// test multi part external name resolution.
func TestImportAs(t *testing.T) {
	src := `
import foo.bar as car
car.star`
	graph := pythonresource.MockManager(t, nil, "foo", "foo.bar", "foo.bar.star")
	assertResolve(t, src, graph, map[string]string{
		"car.star": "external:foo:foo.bar.star",
	})
}

func TestImportFromWildcard(t *testing.T) {
	src := `
from foo import *

bar
bar.x
`

	graph := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"foo":       keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"foo.bar":   keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"foo.bar.x": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	})
	assertResolve(t, src, graph, map[string]string{
		"bar.x": "external:foo:foo.bar.x",
		"bar":   "external:foo:foo.bar",
	})
}

func TestImportFrom(t *testing.T) {
	src := `
from foo import bar
bar.x`
	graph := pythonresource.MockManager(t, nil, "foo", "foo.bar", "foo.bar.x")
	assertResolve(t, src, graph, map[string]string{
		"bar.x": "external:foo:foo.bar.x",
	})
}

func TestImportFromAs(t *testing.T) {
	src := `
from foo import bar as car
car.star
	`
	graph := pythonresource.MockManager(t, nil, "foo", "foo.bar", "foo.bar.star")
	assertResolve(t, src, graph, map[string]string{
		"car.star": "external:foo:foo.bar.star",
	})
}

func TestAssignment(t *testing.T) {
	src := `
import foo, bar
foo = bar
foo.x`
	graph := pythonresource.MockManager(t, nil, "foo", "bar.x")
	assertResolve(t, src, graph, map[string]string{
		"foo.x": "external:bar:bar.x",
	})
}

func TestAttrAssignment(t *testing.T) {
	src := `
class Foo(): pass
f = Foo()
f.abc = 1
out = f.abc`
	graph := pythonresource.MockManager(t, nil, "foo", "bar.x")
	assertResolve(t, src, graph, map[string]string{
		"out": "instanceof builtins.int",
	})
}

func TestDestructureAssignment(t *testing.T) {
	src := `
[a] = (1,)
(b,) = (2,)
(c) = (3,)
d = (4,)
([(e)]) = (5,)
(f, (g, h), i) = (1, ("foo", 12.34), None)
`
	graph := pythonresource.MockManager(t, nil, "foo", "bar.x")
	assertResolve(t, src, graph, map[string]string{
		"a": "instanceof builtins.int",
		"b": "instanceof builtins.int",
		"c": "instanceof builtins.tuple",
		"d": "instanceof builtins.tuple",
		"e": "instanceof builtins.int",
		"f": "instanceof builtins.int",
		"g": "instanceof builtins.str",
		"h": "instanceof builtins.float",
		"i": "instanceof builtins.None.__class__",
	})
}

func TestAssignUnknownToAttr(t *testing.T) {
	// This tests what happens if we assign an unresolved expression to an attribute.
	// It should
	src := `
class Foo(): pass
f = Foo()
f.abc = unknown
out = f.abc`
	graph := pythonresource.MockManager(t, nil, "foo", "bar.x")
	assertResolve(t, src, graph, nil)
}

func TestStringLiteral(t *testing.T) {
	src := `
foo = "123"
foo.find("1")
`
	graph := pythonresource.MockManager(t, nil, "builtins.str.find")
	assertResolve(t, src, graph, map[string]string{
		"foo.find": "builtins.str.find",
	})
}

func TestIntLiteral(t *testing.T) {
	src := `
foo = 123
foo.bit_length
`
	graph := pythonresource.MockManager(t, nil, "builtins.int.bit_length")
	assertResolve(t, src, graph, map[string]string{
		"foo.bit_length": "builtins.int.bit_length",
	})
}

func TestListLiteral(t *testing.T) {
	src := `
foo = [1, 2, 3]
foo.append(4)
`
	graph := pythonresource.MockManager(t, nil, "builtins.list.append")
	assertResolve(t, src, graph, map[string]string{
		"foo.append": "boundmethod:builtins.list.append",
	})
}

func TestDictLiteral(t *testing.T) {
	src := `
foo = {x:x+1 for x in range(5)}
foo.get(4)
`
	graph := pythonresource.MockManager(t, nil, "builtins.dict.get")
	assertResolve(t, src, graph, map[string]string{
		"foo.get": "boundmethod:builtins.dict.get",
	})
}

func TestDefaultArgs(t *testing.T) {
	src := `
def foo(x=1):
	x.bit_length
`
	graph := pythonresource.MockManager(t, nil, "builtins.int.bit_length")
	assertResolve(t, src, graph, map[string]string{
		"x.bit_length": "builtins.int.bit_length",
	})
}

func TestVarargs(t *testing.T) {
	src := `
def foo(*args):
	observed = args
`
	graph := pythonresource.MockManager(t, nil, "builtins.int.bit_length")
	assertResolve(t, src, graph, map[string]string{
		"observed": "instanceof builtins.list",
	})
}

func TestKwargs(t *testing.T) {
	src := `
def foo(**kwargs):
	observed = kwargs
`
	graph := pythonresource.MockManager(t, nil, "builtins.int.bit_length")
	assertResolve(t, src, graph, map[string]string{
		"observed": "instanceof builtins.dict",
	})
}

func TestWithStmt(t *testing.T) {
	src := `
foo = "abc"
with foo as bar:
	bar.split
`
	graph := pythonresource.MockManager(t, nil, "builtins.str.split")
	assertResolve(t, src, graph, map[string]string{
		"bar.split": "builtins.str.split",
	})
}

func TestTryStmt(t *testing.T) {
	src := `
try:
	a = 123
except IOError as ex:
	exc = ex
`
	graph := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"builtins.IOError": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})
	assertResolve(t, src, graph, map[string]string{
		"exc": "instanceof external:builtins:builtins.IOError",
	})
}

func TestTryStmt_NoType(t *testing.T) {
	src := `
try:
	a = 123
except SomeRandomException as ex:
	pass
except:
	pass
`
	graph := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"builtins.IOError": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})
	assertResolve(t, src, graph, map[string]string{})
}

func TestWhileStmt(t *testing.T) {
	src := `
while True:
	a = 123
`
	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"a": "instanceof builtins.int",
	})
}

func TestListComprehension(t *testing.T) {
	src := `
matrix = [[1, 2, 3], [4, 5, 6]]
x = [[y * 2 for y in vector]
		for vector in matrix]
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"matrix": "instanceof builtins.list",
		"x":      "instanceof builtins.list",
	})
}

func TestFunctionParams(t *testing.T) {
	src := `
def foo(a, b=1, (c, d), *e, **f):
	out_a = a
	out_b = b
	out_c = c
	out_d = d
	out_e = e
	out_f = f
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out_b": "instanceof builtins.int",
		"out_e": "instanceof builtins.list",
		"out_f": "instanceof builtins.dict",
	})
}

func TestLambdaExpr(t *testing.T) {
	src := `
g = lambda x: x*2
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"g": "instanceof types.FunctionType",
	})
}

func TestFunctionDecorators(t *testing.T) {
	src := `
@some_decorator(123)
def foo(): pass
`
	graph := pythonresource.MockManager(t, nil, "builtins.some_decorator")

	assertResolve(t, src, graph, map[string]string{
		"some_decorator": "external:builtins:builtins.some_decorator",
		"123":            "instanceof builtins.int",
	})
}

func TestTrueFalseNone(t *testing.T) {
	src := `
a = True
b = False
c = None
`
	graph := pythonresource.MockManager(t, nil, "builtins.some_decorator")

	assertResolve(t, src, graph, map[string]string{
		"a": "instanceof builtins.bool",
		"b": "instanceof builtins.bool",
		"c": "instanceof builtins.None.__class__",
	})
}

func TestUnaryOps(t *testing.T) {
	src := `
a = +123
b = -123
c = ~123
`
	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"a": "instanceof builtins.int",
		"b": "instanceof builtins.int",
		"c": "instanceof builtins.int",
	})
}

func TestBinaryOps(t *testing.T) {
	src := `
a = "abc" + "def"
b = 1 - 2
c = 1.1 * 2.2
d = None or {}
e = None or False
f = fmt % name
`
	graph := getEmptyManager(t)

	assertResolve(t, src, graph, map[string]string{
		"a": "instanceof builtins.str",
		"b": "instanceof builtins.int",
		"c": "instanceof builtins.float",
		"d": "instanceof (generic:builtins.None.__class__ | generic:builtins.dict)",
		"e": "instanceof (generic:builtins.None.__class__ | generic:builtins.bool)",
		"f": "instanceof builtins.str",
	})
}

func TestIndexDict(t *testing.T) {
	src := `
a = {"l": 1, 1:2}
b = a[1]
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"b": "instanceof builtins.int",
	})
}

func TestAssignDestructure(t *testing.T) {
	src := `
a,b = [1,2]
c,d = {1,2}
e,f = {1:"a", 2:"b"}
g,h = (1,2)
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"a": "instanceof builtins.int",
		"b": "instanceof builtins.int",
		"c": "instanceof builtins.int",
		"d": "instanceof builtins.int",
		"e": "instanceof builtins.int",
		"f": "instanceof builtins.int",
		"g": "instanceof builtins.int",
		"h": "instanceof builtins.int",
	})
}

func TestSuper_Python3(t *testing.T) {
	src := `
class Foo(object):
	def __init__(self): pass

class Bar(Foo):
	def __init__(self):
		super().__init__()
`
	graph := getEmptyManager(t)

	assertResolve(t, src, graph, map[string]string{
		"super().__init__": "src-func:__main__.py:Foo.__init__",
	})
}

func TestSuper_MultipleBases(t *testing.T) {
	src := `
class Base1(object):
	def x(self): pass

class Base2(object):
	def y(self): pass

class C(Base1, Base2):
	def __init__(self):
		super().x
		super().y
`
	graph := getEmptyManager(t)

	assertResolve(t, src, graph, map[string]string{
		"super().x": "src-func:__main__.py:Base1.x",
		"super().y": "src-func:__main__.py:Base2.y",
	})
}

func TestDictOfResponses(t *testing.T) {
	src := `
import requests
urls = ["http://example.com"]
a = {url: float(len(url)) for url in urls}
b = [(i, x[0], x[1]) for i, x in enumerate(a.items())]
c = b.pop()
c1 = c[0]
c2 = c[1]
c3 = c[2]`

	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"c1": "instanceof builtins.int",
		"c2": "instanceof builtins.str",
		"c3": "instanceof builtins.float",
	})
}

func TestMapList(t *testing.T) {
	src := `
sizes = map(len, [str(n) for n in range(100)])
for i in sizes:
	xyz = i`

	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"sizes": "instanceof builtins.list",
		"xyz":   "instanceof builtins.int",
	})
}

func TestClassInstance(t *testing.T) {
	src := `
class C(object): pass
c = C()`

	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"C": "src-class:__main__.py:C",
		"c": "instanceof src-class:__main__.py:C",
	})
}

func TestUnionBaseClasses(t *testing.T) {
	src := `
class BaseA(object):
	def a(self): pass

class BaseB(object):
	def b(self): pass

cls = BaseA if foo else BaseB

class Derived(cls):
	pass

obj = Derived()
obj.a()
obj.b()
`

	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"obj.a": "src-func:__main__.py:BaseA.a",
		"obj.b": "src-func:__main__.py:BaseB.b",
	})
}

func TestUnionOfTypes(t *testing.T) {
	src := `
class Foo(object):
   pass

class Bar(object):
   def __init__(self, n=0): pass
   def bar_func(): pass

f = Foo()

cls = Foo if name == "Foo" else Bar
obj = cls()
obj.bar_func()
`

	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"obj":          "instanceof (src-class:__main__.py:Foo | src-class:__main__.py:Bar)",
		"obj.bar_func": "src-func:__main__.py:Bar.bar_func",
	})
}

func TestDataMember(t *testing.T) {
	src := `
class C(object):
	def __init__(self):
		self.n = -1

c = C()
c.n
`

	graph := getEmptyManager(t)
	assertResolve(t, src, graph, map[string]string{
		"c.n": "instanceof builtins.int",
	})
}

func TestClassDef(t *testing.T) {
	src := `
import foo

class A(object):
	speed = 5
	def __init__(self):
		self.x = 1
		self.y = B()
		x = 3
		self.z = self.speed


class B(object):
	def __init__(self):
		self.x = 1

a = A()
x = a.speed
b = a.y

bx = a.y.x
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"A":     "src-class:__main__.py:A",
		"B":     "src-class:__main__.py:B",
		"speed": "instanceof builtins.int",
		"a":     "instanceof src-class:__main__.py:A",
		"x":     "instanceof builtins.int",
		"b":     "instanceof src-class:__main__.py:B",
		"bx":    "instanceof builtins.int",
	})
}

func TestLocalScope(t *testing.T) {
	src := `
def test():
	for x in [1, 3, 4]:
		print(x)

print(x)
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"test": "src-func:__main__.py:test",
	})
}

func TestCallExprLHS(t *testing.T) {
	src := `foo() = 123`
	// just check that we don't panic
	assertResolve(t, src, getEmptyManager(t), map[string]string{})
}

func TestBuiltinRef(t *testing.T) {
	src := `
__file__
__name__
__doc__
__package__
__builtins__
	`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"__file__":     "instanceof builtins.str",
		"__name__":     "instanceof builtins.str",
		"__doc__":      "instanceof builtins.str",
		"__package__":  "instanceof builtins.str",
		"__builtins__": "builtins",
	})
}

func traceAST(node pythonast.Node) string {
	var buf bytes.Buffer
	pythonast.Print(node, &buf, "\t")
	return string(buf.Bytes())
}

func isBadNode(root pythonast.Node) bool {
	var bad bool
	pythonast.Inspect(root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}

		// check if node is a BadStmt
		_, bad = node.(*pythonast.BadStmt)
		if bad {
			return false
		}

		// check if node is a BadExpr
		_, bad = node.(*pythonast.BadExpr)
		if bad {
			return false
		}

		// check if node contains any BadTokens
		n := reflect.ValueOf(node).Elem()
		for i := 0; i < n.NumField(); i++ {
			if word, ok := n.Field(i).Interface().(*pythonscanner.Word); ok {
				if word != nil {
					bad = word.Token == pythonscanner.BadToken
					if bad {
						return false
					}
				}
			}
		}
		return true
	})
	return bad
}

func TestBadNodes(t *testing.T) {
	src := `
def foo():
	bar = "bar"
	print(bar)
	# BadExpr
	x =
	# BadStmt
	<<
`

	mod, _ := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{
		Approximate: true,
	})

	require.NotNil(t, mod)

	graph := getEmptyManager(t)
	resolver := NewResolver(graph, Options{
		Path: "/__main__.py",
	})

	resolvedAST, err := resolver.Resolve(mod)
	require.NoError(t, err)

	// make sure we still resolve the good nodes
	expected := map[string]string{
		"foo":   "src-func:__main__.py:foo",
		"bar":   "instanceof builtins.str",
		"print": "external:builtins:builtins.print",
	}

	for exprStr, expectedName := range expected {
		expr := findExpr(mod, exprStr, src)
		require.NotNil(t, expr, "could not find AST node for '%s'", exprStr)
		ref, ok := resolvedAST.References[expr]
		require.True(t, ok, "node for '%s' was not present in References", exprStr)
		t.Logf("for '%s' (%s)", exprStr, pythonast.String(expr))
		if assert.NotNil(t, ref) {
			assertValue(t, expectedName, ref, exprStr)
		}
	}
}

func TestImports(t *testing.T) {
	src := `
import numpy as np, missing as whatever
from kite import ranking, test as foo, nothing as bar
from kite.test import util, blah
from a import *
b
`
	infos := pythonresource.InfosFromKinds(map[string]pythonimports.Kind{
		"numpy": pythonimports.Module,
		"kite":  pythonimports.Module,
		"a":     pythonimports.Module,
	})

	graph := pythonresource.MockManager(t, infos, "numpy.ndarray", "kite.ranking", "kite.test.util", "a.b", "a.c")
	assertResolve(t, src, graph, map[string]string{
		"numpy":    "external:numpy:numpy",
		"np":       "external:numpy:numpy",
		"missing":  "unknown",
		"whatever": "unknown",
		"kite":     "external:kite:kite",
		"ranking":  "external:kite:kite.ranking",
		"test":     "external:kite:kite.test",
		"foo":      "external:kite:kite.test",
		"nothing":  "unknown",
		"bar":      "unknown",
		"util":     "external:kite:kite.test.util",
		"blah":     "unknown",
		"a":        "external:a:a",
		"b":        "external:a:a.b",
	})
}

func TestModuleValue(t *testing.T) {
	src := `
class A(object):
	def __init__(self):
		print("test")

def test():
	print("test")
`
	resolved := assertResolve(t, src, getEmptyManager(t), nil)
	require.NotNil(t, resolved.Module, "module should not be nil")
	assertHasMemeber(t, resolved.Module, "", "A", "src-class:__main__.py:A")
	assertHasMemeber(t, resolved.Module, "", "test", "src-func:__main__.py:test")
}

func TestNoPanicRelativeImportNoLocal(t *testing.T) {
	src := `from . import bar`
	resolved := assertResolve(t, src, getEmptyManager(t), nil)
	assert.Len(t, resolved.References, 1)
}

func TestFunctionReturnTypes_PassThrough(t *testing.T) {
	src := `
def foo(x): return x
out = foo("abc")
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.str",
	})
}

func TestFunctionReturnTypes_Implied(t *testing.T) {
	src := `
def foo(x): return str(x)
out = foo()
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.str",
	})
}

func TestFunctionReturnTypes_Defaults(t *testing.T) {
	src := `
def foo(x="abc"): return x
out = foo()
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.str",
	})
}

func TestFunctionReturnTypes_ParamAnnotation(t *testing.T) {
	src := `
def foo(x:str): return x
out = foo()
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.str",
	})
}

func TestFunctionReturnTypes_ReturnAnnotation(t *testing.T) {
	src := `
def foo(x) -> str: pass
out = foo()
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.str",
	})
}

func TestFunctionReturnTypes_Vararg(t *testing.T) {
	src := `
def foo(*args): return args
out = foo()
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.list",
	})
}

func TestFunctionReturnTypes_Kwarg(t *testing.T) {
	src := `
def foo(**kwargs): return kwargs
out = foo()
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.dict",
	})
}

func TestClassScopeRules(t *testing.T) {
	src := `
x = 1
y = C()
class C(object):
	something = x      # classes can access symbols from parent scope
	y.z = "something"  # classes can write to symbols from parent scope

out = C().something
out2 = y.z
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out":  "instanceof builtins.int",
		"out2": "instanceof builtins.str",
	})
}

func TestReceiver_Method(t *testing.T) {
	src := `
class C:
	def foo(self, a): pass

C().foo(123)
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"self": "instanceof src-class:__main__.py:C",
		"a":    "instanceof builtins.int",
	})
}

func TestReceiver_Classmethod(t *testing.T) {
	src := `
class C:
	@classmethod
	def foo(cls, a): pass

C.foo(123)
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"cls": "src-class:__main__.py:C",
		"a":   "instanceof builtins.int",
	})
}

func TestReceiver_Staticmethod(t *testing.T) {
	src := `
class C:
	@staticmethod
	def foo(a): pass

C.foo(123)
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"a": "instanceof builtins.int",
	})
}

func TestReceiver_MethodSubclass(t *testing.T) {
	src := `
class C:
	def foo(self, a):
		self.bar

class D(C):
	def bar(self): pass
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"self.bar": "src-func:__main__.py:D.bar",
	})
}

func TestReceiver_ClassmethodSublcass(t *testing.T) {
	src := `
class C:
	@classmethod
	def foo(cls, a):
		cls.bar

class D(C):
	def bar(self): pass
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"cls.bar": "src-func:__main__.py:D.bar",
	})
}

func TestReturnYieldOutsideFunc(t *testing.T) {
	// this is valid syntax but nonsensical... just make sure we don't panic
	src := `
return 123
yield "abc"
class C:
	return 123
	yield "abc"`
	assertResolve(t, src, getEmptyManager(t), nil)
}

func TestAssertIsInstance(t *testing.T) {
	src := `
def foo(x):
	assert isinstance(x, str)
	out = x
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out": "instanceof builtins.str",
	})
}

func TestAssertIsInstanceMulti(t *testing.T) {
	src := `
def foo(x):
	assert isinstance(x, (str, list))
	out1 = x.pop
	out2 = x.isalpha
`
	graph := pythonresource.MockManager(t, nil, "builtins.str.isalpha", "builtins.list.pop")
	assertResolve(t, src, graph, map[string]string{
		"out1": "boundmethod:builtins.list.pop",
		"out2": "builtins.str.isalpha",
	})
}

func TestAssertIsSubclass(t *testing.T) {
	src := `
class C: pass

class D(C):
	def foo(self): pass

class E(C):
	def bar(self): pass

def foo(cls):
	assert issubclass(cls, C)
	out1 = cls.foo
	out2 = cls.bar
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"out1": "src-func:__main__.py:D.foo",
		"out2": "src-func:__main__.py:E.bar",
	})
}

func TestParametersOfUnknownFunction(t *testing.T) {
	// this test checks that the parameters to an unknown function appear in the resolved map
	src := `
a = 123
b = "xyz"
unknown(a, b)
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"a": "instanceof builtins.int",
		"b": "instanceof builtins.str",
	})
}

func TestDestructuredParameters(t *testing.T) {
	src := `
def f(a, (b, c)):
	pass

f("abc", (123, True))
`
	assertResolve(t, src, getEmptyManager(t), map[string]string{
		"a": "instanceof builtins.str",
		"b": "instanceof builtins.int",
		"c": "instanceof builtins.bool",
	})
}

func TestVarargKwargName(t *testing.T) {
	src := `def f(a, *args, **kwargs): pass`
	ast := assertResolve(t, src, getEmptyManager(t), nil)

	fexpr := findName(ast.Root, "f")
	require.NotNil(t, fexpr)

	fval := ast.References[fexpr]
	require.NotNil(t, fval)

	f, ok := fval.(*pythontype.SourceFunction)
	require.True(t, ok)

	require.NotNil(t, f.Vararg)
	require.NotNil(t, f.Kwarg)
	assert.Equal(t, "args", f.Vararg.Name)
	assert.Equal(t, "kwargs", f.Kwarg.Name)
}
