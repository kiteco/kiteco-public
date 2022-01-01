package pythontype

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var manager pythonresource.Manager

func getManager(t *testing.T) pythonresource.Manager {
	if manager == nil {
		manager = pythonresource.MockManager(t, nil, "foo.bar")
	}
	return manager
}

func serdes(t testing.TB, in interface{}, out interface{}) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	require.NoError(t, enc.Encode(in))
	dec := gob.NewDecoder(&buf)
	require.NoError(t, dec.Decode(out))
}

func assertFlattenInflate(t *testing.T, v Value) {
	t.Logf("%# v", pretty.Formatter(v))

	fs, err := FlattenValues(kitectx.Background(), []Value{v})
	require.NoError(t, err)

	var fsOut []*FlatValue
	serdes(t, fs, &fsOut)

	vs, err := InflateValues(fsOut, getManager(t))
	require.NoError(t, err)

	require.NotEmpty(t, vs)
	u := vs[MustHash(v)]

	if !reflect.DeepEqual(u, v) {
		t.Errorf("%v after flattening and then inflating became %v", v, u)
		t.Logf("before: %# v", pretty.Formatter(v))
		t.Logf("after: %# v", pretty.Formatter(u))
		t.Logf("flat representation of %v was %# v", v, pretty.Formatter(fs[0]))
	}
}

func assertFlattenInflateAlt(t *testing.T, v Value) {
	fs, err := FlattenValues(kitectx.Background(), []Value{v})
	require.NoError(t, err)

	var fsOut []*FlatValue
	serdes(t, fs, &fsOut)

	vs, err := InflateValues(fsOut, getManager(t))
	require.NoError(t, err)

	require.NotEmpty(t, vs)
	u := vs[MustHash(v)]

	if !EqualNoCtx(u, v) {
		t.Errorf("%v after flattening and then inflating became %v", v, u)
		t.Logf("flat representation of %v was %# v", v, pretty.Formatter(fs[0]))
	}
}

var (
	boolean = BoolInstance{}
	intg    = IntInstance{}
	flt     = FloatInstance{}
	cmplx   = ComplexInstance{}
	str     = StrInstance{}
)

func TestFlattenInternal(t *testing.T) {
	assertFlattenInflate(t, NoneConstant{})
	assertFlattenInflate(t, BoolInstance{})
	assertFlattenInflate(t, BoolConstant(true))
	assertFlattenInflate(t, BoolConstant(false))
	assertFlattenInflate(t, IntInstance{})
	assertFlattenInflate(t, IntConstant(3))
	assertFlattenInflate(t, FloatInstance{})
	assertFlattenInflate(t, FloatConstant(3.5))
	assertFlattenInflate(t, ComplexInstance{})
	assertFlattenInflate(t, ComplexConstant(1+3i))
	assertFlattenInflate(t, StrInstance{})
	assertFlattenInflate(t, StrConstant("foo"))

	assertFlattenInflate(t, NewList(intg))
	assertFlattenInflate(t, NewList(nil))

	assertFlattenInflate(t, NewDict(intg, str))
	assertFlattenInflate(t, NewDict(intg, nil))

	assertFlattenInflate(t, NewSet(cmplx))
	assertFlattenInflate(t, NewSet(nil))

	assertFlattenInflate(t, NewTuple())
	assertFlattenInflate(t, NewTuple(intg))
	assertFlattenInflate(t, NewTuple(intg, intg, str))
	assertFlattenInflate(t, NewTuple(intg, nil, nil, intg, str))

	assertFlattenInflate(t, UniteNoCtx(intg, intg, str))
	assertFlattenInflate(t, UniteNoCtx(intg, intg, UniteNoCtx(intg, intg, str)))
}

func TestFlattenExternal(t *testing.T) {
	manager := getManager(t)
	bar, err := manager.PathSymbol(pythonimports.NewDottedPath("foo.bar"))
	require.NoError(t, err)

	ext := NewExternal(bar, manager)
	assertFlattenInflateAlt(t, ext)
	assertFlattenInflateAlt(t, ExternalInstance{ext})
}

func symbolTable(path string, members map[string]Value) *SymbolTable {
	addr := SplitAddress(path)
	table := NewSymbolTable(addr, nil)
	for n, v := range members {
		table.Put(n, v)
	}
	return table
}

func TestFlattenSourceFunc(t *testing.T) {
	locals := symbolTable("test.func", map[string]Value{
		"foo": flt,
		"bar": intg,
	})
	fun := &SourceFunction{

		Return:      &Symbol{Name: locals.Name.WithTail("[return]"), Value: intg},
		HasReceiver: true,
		Parameters: []Parameter{
			Parameter{Name: "foo", Default: flt, Symbol: locals.Find("foo")},
		},
		Locals: locals,
	}

	assertFlattenInflate(t, fun)
}

func TestFlattenSourceClass(t *testing.T) {
	bt := symbolTable("test.base", map[string]Value{
		"xyz": str,
	})

	base := &SourceClass{
		Members: bt,
	}

	ct := symbolTable("test.cls", map[string]Value{
		"abc": boolean,
	})
	cls := &SourceClass{
		Bases:   []Value{base},
		Members: ct,
	}

	assertFlattenInflate(t, cls)
}

func TestFlattenSourceModule(t *testing.T) {
	table := symbolTable("test.mod", map[string]Value{
		"abc": boolean,
	})
	mod := &SourceModule{
		Members: table,
	}

	assertFlattenInflate(t, mod)
}

func TestFlattenSourcePackage(t *testing.T) {
	init := &SourceModule{
		Members: symbolTable("test.__init__", map[string]Value{
			"abc": boolean,
		}),
	}

	mod := &SourceModule{
		Members: symbolTable("test.foo", map[string]Value{
			"foo": intg,
		}),
	}

	pkg := &SourcePackage{
		LowerCase: true,
		Init:      init,
		DirEntries: symbolTable("test", map[string]Value{
			"foo": mod,
		}),
	}

	assertFlattenInflate(t, pkg)
}

func TestFlattenCircularSource(t *testing.T) {
	ft := symbolTable("test.func", map[string]Value{
		"foo": flt,
	})

	fun := &SourceFunction{
		HasReceiver: true,
		Parameters: []Parameter{
			Parameter{Name: "foo", Default: flt, Symbol: ft.Find("foo")},
		},
		Locals: ft,
	}

	ct := symbolTable("test.cls", map[string]Value{
		"f": fun,
	})
	cls := &SourceClass{
		Members: ct,
	}

	fun.Return = &Symbol{Name: ft.Name.WithTail("[return]"), Value: SourceInstance{cls}}

	assertFlattenInflate(t, cls)
}

func TestFlattenExplicit(t *testing.T) {
	assertFlattenInflateAlt(t, BuiltinSymbols["map"])
	assertFlattenInflateAlt(t, Builtins.Super)
	assertFlattenInflateAlt(t, Builtins.List)
	assertFlattenInflateAlt(t, BuiltinModule)
}

func TestFlattenKwargDict(t *testing.T) {
	v := NewKwargDict()
	v.Add("foo", IntConstant(1))
	v.Add("bar", StrInstance{})
	assertFlattenInflate(t, v)
}

func TestFlattenCancel(t *testing.T) {
	err := kitectx.Background().WithCancel(func(ctx kitectx.Context, cancel kitectx.CancelFunc) error {
		cancel()
		ctx.WaitExpiry(t)
		FlattenValues(ctx, []Value{IntConstant(1)})
		return nil
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "context canceled"))
}
