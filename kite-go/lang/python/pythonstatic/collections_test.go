package pythonstatic

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

func TestAssembler_NamedTuple(t *testing.T) {
	src := `
from collections import namedtuple

T1 = namedtuple("T1", "t1 t2")
t1 = T1(1, "s")

T2 = namedtuple("T2", "t1, t2")
t2 = T2(1, "s")

T3 = namedtuple("T3", ("t1","t2"))
t3 = T3(1, "s")
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"collections":            keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"collections.namedtuple": keytypes.TypeInfo{Kind: keytypes.FunctionKind},
	})

	T1 := pythontype.NewNamedTupleType(pythontype.StrConstant("T1"), []string{"t1", "t2"}).(pythontype.NamedTupleType)
	t1 := pythontype.NewNamedTupleInstance(T1, []pythontype.Value{
		pythontype.IntConstant(1),
		pythontype.StrConstant("s"),
	})
	T2 := pythontype.NewNamedTupleType(pythontype.StrConstant("T2"), []string{"t1", "t2"}).(pythontype.NamedTupleType)
	t2 := pythontype.NewNamedTupleInstance(T2, []pythontype.Value{
		pythontype.IntConstant(1),
		pythontype.StrConstant("s"),
	})
	T3 := pythontype.NewNamedTupleType(pythontype.StrConstant("T3"), []string{"t1", "t2"}).(pythontype.NamedTupleType)
	t3 := pythontype.NewNamedTupleInstance(T3, []pythontype.Value{
		pythontype.IntConstant(1),
		pythontype.StrConstant("s"),
	})

	assertTypes(t, src, manager, map[string]pythontype.Value{
		"T1": T1,
		"t1": t1,
		"T2": T2,
		"t2": t2,
		"T3": T3,
		"t3": t3,
	})
}

func TestAssembler_Counter(t *testing.T) {
	src := `
from collections import Counter

a = Counter([1,"hello"])
a[True] = 1
a[1] = 1
b = Counter((1,"hello"))
c = Counter({1,"hello"})
d = Counter({1:0, 2:"hello"})
`
	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"collections":         keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"collections.Counter": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	strint := pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{})
	assertTypes(t, src, manager, map[string]pythontype.Value{
		"a": pythontype.UniteNoCtx(
			pythontype.NewCounter(strint, pythontype.IntInstance{}),
			pythontype.NewCounter(pythontype.UniteNoCtx(strint, pythontype.BoolInstance{}), pythontype.IntInstance{}),
		),
		"b": pythontype.NewCounter(strint, pythontype.IntInstance{}),
		"c": pythontype.NewCounter(strint, pythontype.IntInstance{}),
		"d": pythontype.NewCounter(pythontype.IntInstance{}, strint),
	})
}

func TestAssembler_OrderedDict(t *testing.T) {
	src := `
from collections import OrderedDict

a = OrderedDict([[1,"hello"]])
a[True] = 1
a[1] = 1
b = OrderedDict([(1,"hello")])
c = OrderedDict(([1,"hello"],))
d = OrderedDict(({1,"hello"},))
e = OrderedDict([{1,"hello"},])
f = OrderedDict({{1,"hello"},}) 
g = OrderedDict({1:0, 2:"hello"})
h = OrderedDict([{1:0, 2:"hello"}])
`
	strint := pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{})

	aKeyMap := make(map[pythontype.ConstantValue]pythontype.Value)
	aKeyMap[pythontype.IntConstant(1)] = pythontype.IntConstant(1)

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"collections":             keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"collections.OrderedDict": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	assertTypes(t, src, manager, map[string]pythontype.Value{
		"a": pythontype.UniteNoCtx(
			pythontype.NewOrderedDictWithMap(strint, strint, aKeyMap),
			pythontype.NewOrderedDict(pythontype.UniteNoCtx(strint, pythontype.BoolInstance{}), strint),
		),
		"b": pythontype.NewOrderedDict(pythontype.IntInstance{}, pythontype.StrInstance{}),
		"c": pythontype.NewOrderedDict(strint, strint),
		"d": pythontype.NewOrderedDict(strint, strint),
		"e": pythontype.NewOrderedDict(strint, strint),
		// NOTE: not allowed by python
		"f": pythontype.NewOrderedDict(strint, strint),
		"g": pythontype.NewOrderedDict(pythontype.IntInstance{}, strint),
		// NOTE: in python this would just contain 1 -> 2 as the only entry
		"h": pythontype.NewOrderedDict(strint, pythontype.IntInstance{}),
	})
}

func TestAssembler_DefaultDict(t *testing.T) {

	src := `
from collections import defaultdict

a = defaultdict(int, [[1,"hello"]])
a[True] = 1
a[1] = 1
b = defaultdict(int, [(1,"hello")])
c = defaultdict(int, ([1,"hello"],))
d = defaultdict(int, ({1,"hello"},))
e = defaultdict(int, [{1,"hello"},])
f = defaultdict(int, {{1,"hello"},})
g = defaultdict(int, {1:0, 2:"hello"})
h = defaultdict(int, [{1:0, 2:"hello"}])
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"collections":             keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"collections.defaultdict": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	strint := pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{})
	assertTypes(t, src, manager, map[string]pythontype.Value{
		"a": pythontype.UniteNoCtx(
			pythontype.NewDefaultDict(strint, strint, pythontype.Builtins.Int),
			pythontype.NewDefaultDict(pythontype.UniteNoCtx(strint, pythontype.BoolInstance{}), strint, pythontype.Builtins.Int),
		),
		"b": pythontype.NewDefaultDict(pythontype.IntInstance{}, pythontype.StrInstance{}, pythontype.Builtins.Int),
		"c": pythontype.NewDefaultDict(strint, strint, pythontype.Builtins.Int),
		"d": pythontype.NewDefaultDict(strint, strint, pythontype.Builtins.Int),
		"e": pythontype.NewDefaultDict(strint, strint, pythontype.Builtins.Int),
		// NOTE: not allowed by python
		"f": pythontype.NewDefaultDict(strint, strint, pythontype.Builtins.Int),
		"g": pythontype.NewDefaultDict(pythontype.IntInstance{}, strint, pythontype.Builtins.Int),
		// NOTE: in python this would just contain 1 -> 2 as the only entry
		"h": pythontype.NewDefaultDict(strint, pythontype.IntInstance{}, pythontype.Builtins.Int),
	})
}

func TestAssembler_Deque(t *testing.T) {
	src := `
from collections import deque

a = deque([1,"hello"])
a[1] = 1
b = deque([(1,"hello")])
c = deque(([1,"hello"],))
d = deque(({1,"hello"},))
e = deque([{1,"hello"},])
f = deque({{1,"hello"},}) 
g = deque({1:0, 2:"hello"})
h = deque([{1:0, 2:"hello"}])
`

	manager := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"collections":       keytypes.TypeInfo{Kind: keytypes.ModuleKind},
		"collections.deque": keytypes.TypeInfo{Kind: keytypes.TypeKind},
	})

	hKeyMap := make(map[pythontype.ConstantValue]pythontype.Value)
	hKeyMap[pythontype.IntConstant(1)] = pythontype.IntInstance{}
	hKeyMap[pythontype.IntConstant(2)] = pythontype.StrInstance{}

	strint := pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrInstance{})
	setstrint := pythontype.NewSet(strint)
	assertTypes(t, src, manager, map[string]pythontype.Value{
		"a": pythontype.UniteNoCtx(
			pythontype.NewDeque(pythontype.UniteNoCtx(pythontype.StrConstant("hello"), pythontype.IntConstant(1))),
		),
		"b": pythontype.NewDeque(pythontype.NewTuple(pythontype.IntConstant(1), pythontype.StrConstant("hello"))),
		"c": pythontype.NewDeque(pythontype.NewList(pythontype.UniteNoCtx(pythontype.StrConstant("hello"), pythontype.IntConstant(1)))),
		"d": pythontype.NewDeque(setstrint),
		"e": pythontype.NewDeque(setstrint),
		// TODO(juan): not allowed by python
		"f": pythontype.NewDeque(setstrint),
		"g": pythontype.NewDeque(pythontype.IntInstance{}),
		"h": pythontype.NewDeque(pythontype.NewDictWithMap(pythontype.IntInstance{}, strint, hKeyMap)),
	})
}
