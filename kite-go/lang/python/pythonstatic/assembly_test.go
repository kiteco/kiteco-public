package pythonstatic

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mockFilename = "/code/src.py"

func assertEqual(t *testing.T, a, b pythontype.Value) {
	if !pythontype.EqualNoCtx(a, b) {
		t.Errorf("expected %v == %v", a, b)
	}
}

func assertNotEqual(t *testing.T, a, b pythontype.Value) {
	if pythontype.EqualNoCtx(a, b) {
		t.Errorf("expected %v != %v", a, b)
	}
}

func nav(manager pythonresource.Manager, path string) pythontype.Value {
	sym, err := manager.PathSymbol(pythonimports.NewDottedPath(path))
	if err != nil {
		panic(fmt.Sprintf("could not find %s in manager", path))
	}
	return pythontype.TranslateExternal(sym, manager)
}

type byName []*pythontype.Symbol

func (xs byName) Len() int           { return len(xs) }
func (xs byName) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byName) Less(i, j int) bool { return xs[i].Name.Path.String() < xs[j].Name.Path.String() }

func assertAssemblerBatchWithMissing(
	t *testing.T,
	srcs map[string]string,
	assembler *Assembler,
	expected map[string]pythontype.Value,
	expectedMissing []string) map[string]*pythontype.Symbol {

	for name, src := range srcs {
		ast, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
		require.NoError(t, err)
		assembler.AddSource(ASTBundle{AST: ast, Path: name, Imports: FindImports(kitectx.Background(), name, ast)})
	}

	var buf bytes.Buffer
	assembler.SetTrace(&buf)
	result, err := assembler.Build(kitectx.Background())
	require.NoError(t, err, "got error '%v', expected none", err)

	if buf.Len() > 0 {
		t.Log(buf.String())
	}

	actual := make(map[string]*pythontype.Symbol)
	symlists := make(map[string][]*pythontype.Symbol)
	result.WalkSymbols(func(s *pythontype.Symbol) {
		symlists[s.Name.File] = append(symlists[s.Name.File], s)
		actual[s.Name.Path.String()] = s
	})

	for filename, symlist := range symlists {
		t.Log(filename)
		sort.Sort(byName(symlist))
		for _, s := range symlist {
			key := s.Name.Path.String()
			if _, isbuiltin := pythontype.BuiltinSymbols[key]; !isbuiltin && !strings.Contains(key, "__") {
				t.Logf("%-30s := %v", key, s.Value)
			}
		}
	}

	for key, expectedType := range expected {
		sym, found := actual[key]
		if !found {
			t.Errorf("expected %s to be %v but no such symbol found", key, expectedType)
			continue
		}

		if !pythontype.EqualNoCtx(sym.Value, expectedType) {
			t.Errorf("expected %s to have type %v but got %v", key, expectedType, sym.Value)
		}
	}

	for _, key := range expectedMissing {
		_, found := actual[key]
		assert.False(t, found, "found symbol %s but expected no such symbol", key)
	}

	return actual
}

func assertAssemblerBatch(
	t *testing.T,
	srcs map[string]string,
	assembler *Assembler,
	expected map[string]pythontype.Value) map[string]*pythontype.Symbol {
	return assertAssemblerBatchWithMissing(t, srcs, assembler, expected, nil)
}

func assertBatchTypesWithMissing(t *testing.T,
	srcs map[string]string,
	rm pythonresource.Manager,
	expected map[string]pythontype.Value,
	expectedMissing []string) map[string]*pythontype.Symbol {

	opts := DefaultOptions
	opts.UseCapabilities = false
	ai := AssemblerInputs{
		Graph: rm,
	}

	assembler := NewAssembler(kitectx.Background(), ai, opts)
	return assertAssemblerBatchWithMissing(t, srcs, assembler, expected, expectedMissing)
}

func assertBatchTypes(t *testing.T,
	srcs map[string]string,
	rm pythonresource.Manager,
	expected map[string]pythontype.Value) map[string]*pythontype.Symbol {
	return assertBatchTypesWithMissing(t, srcs, rm, expected, nil)
}

func assertTypesWithMissing(t *testing.T, src string, rm pythonresource.Manager,
	expected map[string]pythontype.Value, expectedMissing []string) map[string]*pythontype.Symbol {
	return assertBatchTypesWithMissing(t, map[string]string{mockFilename: src}, rm, expected, expectedMissing)
}

func assertTypes(t *testing.T, src string, rm pythonresource.Manager,
	expected map[string]pythontype.Value) map[string]*pythontype.Symbol {
	return assertTypesWithMissing(t, src, rm, expected, nil)
}

func TestAssembler_ScalarLiterals(t *testing.T) {
	src := `
a = 1
b = 33L
c = 1.2
d = 6j
e = "foo"
f = None
g = True
h = False
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.IntConstant(1),
		"b": pythontype.IntInstance{},
		"c": pythontype.FloatInstance{},
		"d": pythontype.ComplexInstance{},
		"e": pythontype.StrConstant("foo"),
		"f": pythontype.NoneConstant{},
		"g": pythontype.BoolConstant(true),
		"h": pythontype.BoolConstant(false),
	})
}

func TestAssembler_ScalarConstructors(t *testing.T) {
	src := `
a = int(3.4)
b = int(unknown)
c = float(4, base=8)
d = complex()
e = str(something)
f = bool(a)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.IntInstance{},
		"b": pythontype.IntInstance{},
		"c": pythontype.FloatInstance{},
		"d": pythontype.ComplexInstance{},
		"e": pythontype.StrInstance{},
		"f": pythontype.BoolInstance{},
	})
}

func TestAssembler_CompoundLiterals(t *testing.T) {
	src := `
a = [-1]
b = {"foo": 33L, "bar": "bar"}
c = (-15, 1.2, 3j)
d = {-1, -2, -3}
e = [{"foo": [-1, -2, -3]}, {"bar": [-4, -5, -6]}]
f = [-15, "foo"]
g = {-5: "foo", 6L: "bar"}
`

	bMap := make(map[pythontype.ConstantValue]pythontype.Value)
	bMap[pythontype.StrConstant("bar")] = pythontype.StrInstance{}
	bMap[pythontype.StrConstant("foo")] = pythontype.IntInstance{}

	eMap := make(map[pythontype.ConstantValue]pythontype.Value)
	eMap[pythontype.StrConstant("foo")] = pythontype.NewList(pythontype.IntInstance{})
	eMap[pythontype.StrConstant("bar")] = pythontype.NewList(pythontype.IntInstance{})

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.NewList(pythontype.IntInstance{}),
		"b": pythontype.NewDictWithMap(pythontype.StrInstance{}, pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{}), bMap),
		"c": pythontype.NewTuple(pythontype.IntInstance{}, pythontype.FloatInstance{}, pythontype.ComplexInstance{}),
		"d": pythontype.NewSet(pythontype.IntInstance{}),
		"e": pythontype.NewList(pythontype.NewDictWithMap(pythontype.StrInstance{}, pythontype.NewList(pythontype.IntInstance{}), eMap)),
		"f": pythontype.NewList(pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrConstant("foo"))),
		"g": pythontype.NewDict(pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.IntInstance{}), pythontype.StrInstance{}),
	})
}

func TestAssembler_CompoundConstructors(t *testing.T) {
	src := `
a = list([-1])
b = dict({"foo": 33L})
c = tuple((-15, 1.2, 3j))
d = set({-1, -2, -3})
e = list([-15, "foo"])
f = list(unknown)
g = list(b)
h = dict([{1:3, 2:"bar"}])
i = dict({ {1,"hello"},})
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.NewList(pythontype.IntInstance{}),
		"b": pythontype.NewDict(pythontype.StrInstance{}, pythontype.IntInstance{}),
		"c": pythontype.NewTuple(pythontype.IntInstance{}, pythontype.FloatInstance{}, pythontype.ComplexInstance{}),
		"d": pythontype.NewSet(pythontype.IntInstance{}),
		"e": pythontype.NewList(pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrConstant("foo"))),
		"f": pythontype.NewList(nil),
		"g": pythontype.NewList(pythontype.StrInstance{}),
		// NOTE: in python this would just be a dict mapping from 1 -> 2.
		"h": pythontype.NewDict(
			pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{}), pythontype.IntInstance{}),
		// NOTE: not allowed in python
		"i": pythontype.NewDict(
			pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{}),
			pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{})),
	})
}

func TestAssembler_Comprehensions(t *testing.T) {
	src := `
seq = [-1, -2, -3]
a = [x for x in seq]
b = {"%d" % i : i for i in seq}
c = {(x, x+1) for x in seq}
`

	assertTypesWithMissing(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.NewList(pythontype.IntInstance{}),
		"b": pythontype.NewDict(pythontype.StrInstance{}, pythontype.IntInstance{}),
		"c": pythontype.NewSet(pythontype.NewTuple(pythontype.IntInstance{}, pythontype.IntInstance{})),
	}, []string{"x"})
}

func TestAssembler_Parameters_Simple(t *testing.T) {
	src := `
a = -123
def f(x=a): pass
f("foo")
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"f.x": pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrConstant("foo")),
	})
}

func TestAssembler_Parameters_Nested(t *testing.T) {
	src := `
def h(x): return "xyz"
def f(x): return g(x)
def g(x): return h(x)
out = f(-123)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"h.x":        pythontype.IntInstance{},
		"h.[return]": pythontype.StrConstant("xyz"),
		"out":        pythontype.StrConstant("xyz"),
	})
}

func TestAssembler_Parameters_KeywordOnly(t *testing.T) {
	src := `
def foo(a, b): pass

foo(b="foo", a=1)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"foo.a": pythontype.IntConstant(1),
		"foo.b": pythontype.StrConstant("foo"),
	})
}

func TestAssembler_Parameters_Mixed(t *testing.T) {
	src := `
def foo(a, b, c, d): pass

foo(1, 2, d=4, c=3)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"foo.a": pythontype.IntConstant(1),
		"foo.b": pythontype.IntConstant(2),
		"foo.c": pythontype.IntConstant(3),
		"foo.d": pythontype.IntConstant(4),
	})
}

func TestAssembler_Parameters_Varargs(t *testing.T) {
	src := `
def foo(a, *args): pass

foo(-1, -2, -3)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"foo.a":    pythontype.IntInstance{},
		"foo.args": pythontype.NewList(pythontype.IntInstance{}),
	})
}

func TestAssembler_Parameters_VarargsWithReceiver(t *testing.T) {
	src := `
class C:
	def foo(self, a, *args): pass

C().foo(1, "x", True)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"C.foo.a": pythontype.IntConstant(1),
		"C.foo.args": pythontype.NewList(pythontype.UniteNoCtx(
			pythontype.StrConstant("x"), pythontype.BoolConstant(true))),
	})
}

func TestAssembler_Parameters_StarVarargs(t *testing.T) {
	src := `
def foo(a, *args): pass
data = [str(), str(), str()]
foo(-1, *data)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"foo.a":    pythontype.IntInstance{},
		"foo.args": pythontype.NewList(pythontype.StrInstance{}),
	})
}

func TestAssembler_Parameters_Kwargs(t *testing.T) {
	src := `
def foo(**kwargs):
	x = kwargs["x"]
	y = kwargs["y"]

foo(x=1, y="foo")
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"foo.x": pythontype.IntConstant(1),
		"foo.y": pythontype.StrConstant("foo"),
	})
}

func TestAssembler_Parameters_NestedKwargs(t *testing.T) {
	src := `
def foo(**kwargs):
	bar(**kwargs)

def bar(**kwargs):
	x = kwargs["x"]
	y = kwargs["y"]

foo(x=1, y="foo")
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"bar.x": pythontype.IntConstant(1),
		"bar.y": pythontype.StrConstant("foo"),
	})
}

func TestAssembler_Parameters_UnpackKwargs(t *testing.T) {
	src := `
def foo(**kwargs):
	bar(**kwargs)

def bar(x, y): pass

foo(x=1, y="foo")
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"bar.x": pythontype.IntConstant(1),
		"bar.y": pythontype.StrConstant("foo"),
	})
}

func TestAssembler_Parameters_KwargsMethods(t *testing.T) {
	src := `
def foo(**kwargs):
	a = kwargs.get("x", [])
	b = kwargs.pop("y", [])
	c = kwargs.setdefault("z", [])
	d = kwargs.copy().get("x")

foo(x=1, y="foo", z=True)
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"foo.a": pythontype.UniteNoCtx(pythontype.IntConstant(1), pythontype.NewList(nil)),
		"foo.b": pythontype.UniteNoCtx(pythontype.StrConstant("foo"), pythontype.NewList(nil)),
		"foo.c": pythontype.UniteNoCtx(pythontype.BoolConstant(true), pythontype.NewList(nil)),
		"foo.d": pythontype.IntConstant(1),
	})
}

func TestAssembler_Classes_Simple(t *testing.T) {
	src := `
a = -123

class Foo(object):
	x = a

Foo.x = "xyz"
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"Foo.x": pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrConstant("xyz")),
	})
}

func TestAssembler_Classes_Constructor(t *testing.T) {
	src := `
class Foo(object):
	def __init__(self, value=None):
		self.x = value

f = Foo(-123)
out = f.x
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out": pythontype.UniteNoCtx(pythontype.NoneConstant{}, pythontype.IntInstance{}),
	})
}

func TestAssembler_NestedScopes(t *testing.T) {
	src := `
a = -123
def foo():
	a = 2.3
	def bar():
		out = a
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a":           pythontype.IntInstance{},
		"foo.a":       pythontype.FloatInstance{},
		"foo.bar.out": pythontype.FloatInstance{},
	})
}

func TestAssembler_ChangingScope(t *testing.T) {
	src := `
class Base():
	x = -123

class Derived(Base):
	def a(self):
		out = self.x

	def b(self):
		self.x = "abc"
`

	t.Log(src)
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"Derived.a.out": pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrConstant("abc")),
	})
}

func TestAssembler_ClassScopesNotInherited(t *testing.T) {
	src := `
var = -123
class C():
	var = "foo"
	def bar():
		out = var   # should bind to the outer "var" _not_ the class member
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"C.bar.out": pythontype.IntInstance{},
	})
}

func TestAssembler_ClassScopesNotInherited2(t *testing.T) {
	src := `
var = -123
class C():
	var = "foo"
	class D():
		out = var   # should bind to the outer "var" _not_ the class member
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"C.D.out": pythontype.IntInstance{},
	})
}

func TestAssembler_ResolveBuiltins(t *testing.T) {
	src := `out = object`
	t.Log(src)
	manager := pythonresource.MockManager(t, nil)
	assertTypes(t, src, manager, map[string]pythontype.Value{
		"out": nav(manager, "builtins.object"),
	})
}

func TestAssembler_ShaddowBuiltins(t *testing.T) {
	src := `
object = -123
out = object
`

	t.Log(src)
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out": pythontype.IntInstance{},
	})
}

func TestAssembler_DefaultModuleMembers(t *testing.T) {
	src := `
file = __file__
name = __name__
doc = __doc__
package = __package__
builtins = __builtins__
`

	t.Log(src)
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"file":     pythontype.StrInstance{},
		"name":     pythontype.StrInstance{},
		"doc":      pythontype.StrInstance{},
		"package":  pythontype.StrInstance{},
		"builtins": pythontype.BuiltinModule,
	})
}

func TestAssembler_AutomaticClassAttrs(t *testing.T) {
	src := `
class C(object): pass
c = C()

out1 = C.__name__
out2 = c.__class__
out3 = c.__module__
`

	syms := assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out1": pythontype.StrInstance{},
		"out3": pythontype.StrInstance{},
	})

	assertEqual(t, syms["out2"].Value, syms["C"].Value)
}

func TestAssembler_DerivedClassMembers(t *testing.T) {
	src := `
class Base(object):
	data = 1234

	def getdata(self): return self.data

	@classmethod
	def getdata_classmethod(cls): return cls.data

	@staticmethod
	def getdata_staticmethod(self): return Base.data

class Derived1(Base):
	data = "xyz"

class Derived2(Base):
	data = 0.1

out1 = Derived1().getdata()
out2 = Derived1.getdata_classmethod()
out3 = Derived1.getdata_staticmethod()
`

	// when a member exists on both a base class and a derived class, it is the
	// runtime type of "self" or "cls" by which the member is resolved, even when
	// accessed from a base class

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out1": pythontype.UniteNoCtx(pythontype.StrConstant("xyz"), pythontype.IntInstance{}, pythontype.FloatInstance{}),
		"out2": pythontype.UniteNoCtx(pythontype.StrConstant("xyz"), pythontype.IntInstance{}, pythontype.FloatInstance{}),
		"out3": pythontype.IntInstance{},
	})
}

func TestAssembler_AssignToUnionAttr(t *testing.T) {
	src := `
class A(object): pass
class B(object): pass

c = A() if foo else B()
c.data = 1234
`

	// when a member exists on both a base class and a derived class, it is the
	// runtime type of "self" or "cls" by which the member is resolved, even when
	// accessed from a base class

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"A.data": pythontype.IntInstance{},
		"B.data": pythontype.IntInstance{},
	})
}

func TestAssembler_CreateSymbolsWithUnknownType(t *testing.T) {
	src := `
class C():
	def __init__(self):
		self.y = not_defined

x = not_defined
`

	t.Log(src)
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"x":   nil,
		"C.y": nil,
	})
}

func TestAssembler_TypeInduction(t *testing.T) {
	src := `
import requests
out = requests.get("http://example.com")
`
	methods := map[string]pythonimports.Kind{
		"requests.get":      pythonimports.Function,
		"requests.Response": pythonimports.Type,
	}
	manager := pythonresource.MockManager(t, pythonresource.InfosFromKinds(methods))
	manager.MockReturnType(t, "requests.get", "requests.Response")

	sym, err := manager.PathSymbol(pythonimports.NewPath("requests", "Response"))
	require.NoError(t, err)

	t.Log(src)
	assertTypes(t, src, manager, map[string]pythontype.Value{
		"out": pythontype.ExternalInstance{TypeExternal: pythontype.NewExternal(sym, manager)},
	})
}

func TestAssembler_MultipleBaseClasses(t *testing.T) {
	src := `
class Base1(object):
	foo = -123

class Base2(object):
	bar = "bar"

class C(Base1, Base2):
	pass

c = C()
out1 = c.foo
out2 = c.bar
`

	t.Log(src)
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out1": pythontype.IntInstance{},
		"out2": pythontype.StrConstant("bar"),
	})
}

func TestAssembler_BaseClassLoop(t *testing.T) {
	src := `
class Foo(Bar):
	pass
class Bar(Foo):
	pass
class Car(Car):
	pass
`
	symbs := assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{})

	assert.NotNil(t, symbs["Foo"])
	res1, _ := pythontype.AttrNoCtx(symbs["Foo"].Value, "hello")
	assert.False(t, res1.Found())

	assert.NotNil(t, symbs["Bar"])
	res2, _ := pythontype.AttrNoCtx(symbs["Bar"].Value, "hello")
	assert.False(t, res2.Found())

	assert.NotNil(t, symbs["Car"])
	res3, _ := pythontype.AttrNoCtx(symbs["Car"].Value, "hello")
	assert.False(t, res3.Found())
}

func TestAssembler_BaseClassLoopUnion(t *testing.T) {
	src := `
class Foo(Bar):
	pass
class Bar(Foo,x):
	pass

def foo():
	return Foo if True else Bar

x = foo()
`
	symbs := assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{})

	assert.NotNil(t, symbs["Bar"])
	res1, err := pythontype.AttrNoCtx(symbs["Bar"].Value, "hello")
	assert.False(t, res1.Found())
	assert.Equal(t, kitectx.ContextExpiredError{Err: kitectx.CallLimitError{}}, err)

	assert.NotNil(t, symbs["Foo"])
	res2, err := pythontype.AttrNoCtx(symbs["Foo"].Value, "hello")
	assert.False(t, res2.Found())
	assert.Equal(t, kitectx.ContextExpiredError{Err: kitectx.CallLimitError{}}, err)
}

func TestAssembler_ExternalBaseClasses(t *testing.T) {
	src := `
import requests
class C(requests.Response):
	pass

c = C()
out = c.status_code
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"requests.Response":             keytypes.TypeInfo{Kind: keytypes.TypeKind},
		"requests.Response.status_code": keytypes.TypeInfo{Kind: keytypes.ObjectKind},
	})

	t.Log(src)
	assertTypes(t, src, manager, map[string]pythontype.Value{
		"out": nav(manager, "requests.Response.status_code"),
	})
}

func TestAssembler_BuiltinInstances(t *testing.T) {
	src := `
import foo
a = foo.some_bool
b = foo.some_int
c = foo.some_float
d = foo.some_complex
e = foo.some_str
f = foo.some_list
g = foo.some_set
h = foo.some_dict
i = foo.some_tuple
`

	typenames := []string{"bool", "int", "float", "complex", "str", "list", "set", "dict", "tuple"}
	infos := make(map[string]keytypes.TypeInfo)
	for _, s := range typenames {
		infos["foo.some_"+s] = keytypes.TypeInfo{Kind: keytypes.ObjectKind, Type: pythonimports.NewPath("builtins", s)}
	}

	t.Log(src)
	assertTypes(t, src, pythonresource.MockManager(t, infos), map[string]pythontype.Value{
		"a": pythontype.BoolInstance{},
		"b": pythontype.IntInstance{},
		"c": pythontype.FloatInstance{},
		"d": pythontype.ComplexInstance{},
		"e": pythontype.StrInstance{},
		"f": pythontype.NewList(nil),
		"g": pythontype.NewSet(nil),
		"h": pythontype.NewDict(nil, nil),
		"i": pythontype.NewTuple(),
	})
}

func TestAssembler_Imports(t *testing.T) {
	src := `
import foo
import foo.bar
from foo.bar import car
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"foo":     keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"foo.bar": keytypes.TypeInfo{Kind: keytypes.ModuleKind},
	})

	sym, err := manager.PathSymbol(pythonimports.NewPath("foo"))
	require.NoError(t, err)

	t.Log(src)
	assertTypes(t, src, manager, map[string]pythontype.Value{
		"foo": pythontype.NewExternal(sym, manager),
	})
}

func TestAssembler_IndexDict(t *testing.T) {
	src := `
a = {"a":1, 1:2}
b = a[1]
`
	aKeyMap := make(map[pythontype.ConstantValue]pythontype.Value)
	aKeyMap[pythontype.IntConstant(1)] = pythontype.IntInstance{}
	aKeyMap[pythontype.StrConstant("a")] = pythontype.IntInstance{}

	k := pythontype.UniteNoCtx(pythontype.StrInstance{}, pythontype.IntInstance{})
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.NewDictWithMap(k, pythontype.IntInstance{}, aKeyMap),
		"b": pythontype.IntInstance{},
	})
}

func TestAssembler_Destructure(t *testing.T) {
	src := `
a,b = [1,2]
c,d = ("3",4)
e,f = {"e","f"}
g,h = {1:"g","h":"h"}
`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.UniteNoCtx(pythontype.IntConstant(1), pythontype.IntConstant(2)),
		"b": pythontype.UniteNoCtx(pythontype.IntConstant(1), pythontype.IntConstant(2)),
		"c": pythontype.StrConstant("3"),
		"d": pythontype.IntConstant(4),
		"e": pythontype.StrInstance{},
		"f": pythontype.StrInstance{},
		"g": pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{}),
		"h": pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{}),
	})
}

func TestAssembler_AssignIndexExpr(t *testing.T) {
	src := `
a[0] = 1234
b["foo"] = [1., 2., 3.]
c[5.] = 1.23
c["xxx"] = 1.23
d[unknown] = None
e = [1]
e[1] = "hello"
f = {"hello":2}
f[1] = "hello"
g = (1,2)
g[1] = "hello"

`

	bKeyMap := make(map[pythontype.ConstantValue]pythontype.Value)
	bKeyMap[pythontype.StrConstant("foo")] = pythontype.NewList(pythontype.FloatInstance{})

	// Bug here, we should also have 5. in the keyMap but the setIndex receives a FloatInstance instead of the FloatConstant
	// Also the support for dict key currently is explcitely for StrConstant and IntConstant, so even if a FloatConstant
	// was sent to SetIndex, it wouldn't be stored in the keymap
	cKeyMap := make(map[pythontype.ConstantValue]pythontype.Value)
	cKeyMap[pythontype.StrConstant("xxx")] = pythontype.FloatInstance{}

	fKeyMap := make(map[pythontype.ConstantValue]pythontype.Value)
	fKeyMap[pythontype.IntConstant(1)] = pythontype.StrConstant("hello")
	fKeyMap[pythontype.StrConstant("hello")] = pythontype.IntInstance{}

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"a": pythontype.NewList(pythontype.IntInstance{}),
		"b": pythontype.NewDictWithMap(pythontype.StrInstance{}, pythontype.NewList(pythontype.FloatInstance{}), bKeyMap),
		"c": pythontype.UniteNoCtx(
			pythontype.NewDictWithMap(pythontype.FloatInstance{}, pythontype.FloatInstance{}, cKeyMap),
			pythontype.NewDict(pythontype.UniteNoCtx(pythontype.StrInstance{}, pythontype.FloatInstance{}), pythontype.FloatInstance{}),
		),
		"d": pythontype.UniteNoCtx(
			pythontype.NewList(pythontype.NoneConstant{}),
			pythontype.NewDict(nil, pythontype.NoneConstant{}),
		),
		"e": pythontype.UniteNoCtx(
			pythontype.NewList(pythontype.IntConstant(1)),
			pythontype.NewList(pythontype.UniteNoCtx(pythontype.StrConstant("hello"), pythontype.IntConstant(1))),
		),
		"f": pythontype.UniteNoCtx(
			pythontype.NewDictWithMap(pythontype.StrInstance{}, pythontype.IntInstance{}, fKeyMap),
			pythontype.NewDict(pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{}),
				pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{})),
		),
		"g": pythontype.UniteNoCtx(
			pythontype.NewTuple(pythontype.IntConstant(1), pythontype.IntConstant(2)),
		),
	})
}

func TestAssembler_StringSlice(t *testing.T) {
	// this tests indexing and slicing of strings
	src := `
a = "abc"
b = a[0]
c = a[1:]
for d in a: pass
`
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"b": pythontype.StrInstance{},
		"c": pythontype.StrInstance{},
		"d": pythontype.StrInstance{},
	})
}

func TestAssembler_StarImportsModule(t *testing.T) {
	src1 := `
def Foo():
	return 1

class Bar():
	xyz = "hello"
`
	src2 := `
from src1 import *
x = Foo()
y = Bar().xyz
	`

	srcs := map[string]string{
		"/code/src1.py": src1,
		"/code/src2.py": src2,
	}

	assertBatchTypes(t, srcs, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"x": pythontype.IntConstant(1),
		"y": pythontype.StrConstant("hello"),
	})
}

func TestAssembler_StarImportsPackage(t *testing.T) {
	src1 := `
def Foo():
	return 1

class Bar():
	xyz = "hello"
`

	init := `from src1 import *`

	src2 := `
from pkg import *
x = Foo()
y = Bar().xyz
	`

	srcs := map[string]string{
		"/code/pkg/src1.py":     src1,
		"/code/pkg/__init__.py": init,
		"/code/src2.py":         src2,
	}

	assertBatchTypes(t, srcs, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"x": pythontype.IntConstant(1),
		"y": pythontype.StrConstant("hello"),
	})
}

func TestAssembler_CallBuiltinWithNilConstructor(t *testing.T) {
	// this tests calling a builtin type with no constructor
	src := `
object()
object.__call__()
`
	assertTypes(t, src, pythonresource.MockManager(t, nil), nil)
}

func TestAssembler_EvalString(t *testing.T) {
	// this tests calling a builtin type with no constructor
	src := `
aye = "a"
bee = "b"
plus = "+"
equals = "="
one = "1"
two = "2"

eval(bee + equals + two)
eval(aye + equals + bee + plus + one)

out = eval(aye)
`
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out": pythontype.IntInstance{},
	})
}

func TestAssembler_Generator(t *testing.T) {
	src := `
def foo():
	yield 1

x = foo()
y = (yy for yy in [1])
`
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"x": pythontype.NewGenerator(pythontype.IntConstant(1)),
		"y": pythontype.NewGenerator(pythontype.IntInstance{}),
	})
}

func TestAssembler_ImportSubDir(t *testing.T) {
	src1 := `
from scratch.scratch import Bar
c1 = Bar.car
`

	src2 := `
from scratch.scratch import Bar
c2 = Bar.car
`

	scratch := `
class Bar():
	car = "hello!"
`

	srcs := map[string]string{
		"/scratch/src1.py":            src1,
		"/scratch/src2.py":            src2,
		"/scratch/scratch/scratch.py": scratch,
	}

	assertBatchTypes(t, srcs, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"c1": pythontype.StrConstant("hello!"),
		"c2": pythontype.StrConstant("hello!"),
	})
}

func TestAssembler_VarargKwargName(t *testing.T) {
	src := `def foo(a, b=3, *args, **kwargs): pass`
	syms := assertTypes(t, src, pythonresource.MockManager(t, nil), nil)

	foosym := syms["foo"]
	require.NotNil(t, foosym)
	require.NotNil(t, foosym.Value)

	foo := foosym.Value.(*pythontype.SourceFunction)
	require.NotNil(t, foo.Vararg)
	require.NotNil(t, foo.Kwarg)
	assert.Equal(t, "args", foo.Vararg.Name)
	assert.Equal(t, "kwargs", foo.Kwarg.Name)
}

func TestAssembler_Cancel(t *testing.T) {
	ai := AssemblerInputs{
		Graph: pythonresource.MockManager(t, nil),
	}
	assembler := NewAssembler(kitectx.Background(), ai, DefaultOptions)

	src := `
def foo():
	x = 1
`

	ast, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err)
	assembler.AddSource(ASTBundle{AST: ast, Path: "/test.py", Imports: FindImports(kitectx.Background(), "/test.py", ast)})

	err = kitectx.Background().WithCancel(func(ctx kitectx.Context, cancel kitectx.CancelFunc) error {
		cancel()
		ctx.WaitExpiry(t)
		assembler.Build(ctx)
		return nil
	})
	require.Error(t, err)
}

func TestAssembler_LocalAndGlobalPackage_SameName(t *testing.T) {
	init := `
b = 1
	`

	src := `
import tkinter

c = tkinter.b
`

	srcs := map[string]string{
		"/tkinter/__init__.py": init,
		"/tkinter/src.py":      src,
	}

	mgr := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"tkinter":   keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"tkinter.b": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	})

	ps, err := mgr.PathSymbol(pythonimports.NewDottedPath("tkinter.b"))
	require.NoError(t, err)

	ext := pythontype.TranslateExternal(ps, mgr)

	assertBatchTypes(t, srcs, mgr, map[string]pythontype.Value{
		"c": pythontype.UniteNoCtx(
			pythontype.IntConstant(1),
			ext,
		),
	})

}

func TestAssembler_Property(t *testing.T) {
	src := `
def set_bar(self):
    pass

def get_bar(self):
	return "bar"

class Foo(object):
	@property
	def foo(self):
		return self._foo

	@foo.setter
	def foo(self, value):
		self._foo = value

	bar = property(get_bar, set_bar)

foo_inst = Foo()

foo_inst.foo = 123
foo_val = foo_inst.foo

foo_inst.bar = 123
bar_val = foo_inst.bar

`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"foo_val": pythontype.IntConstant(123),
		"bar_val": pythontype.UniteNoCtx(pythontype.StrConstant("bar"), pythontype.IntConstant(123)),
	})
}
