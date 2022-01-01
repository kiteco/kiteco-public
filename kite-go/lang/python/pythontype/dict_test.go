package pythontype

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDict(t *testing.T) {
	a := NewDict(StrInstance{}, FloatInstance{})
	b := NewDict(StrInstance{}, FloatInstance{})
	c := NewDict(IntInstance{}, IntInstance{})
	assert.True(t, EqualNoCtx(a, b))
	assert.False(t, EqualNoCtx(a, c))
	assert.False(t, EqualNoCtx(b, c))

	// simulate "for x in a.items(): ..."
	itemsFunc, _ := AttrNoCtx(a, "items")
	items := itemsFunc.Value().(Callable).Call(Args{})
	tup := items.(Iterable).Elem()
	assert.True(t, EqualNoCtx(tup, NewTuple(StrInstance{}, FloatInstance{})))

	// simulate "x = a.setdefault(1, None): ..."
	setdefaultFunc, _ := AttrNoCtx(a, "setdefault")
	ret := setdefaultFunc.Value().(Callable).Call(Positional(StrInstance{}, Builtins.None))
	assert.True(t, EqualNoCtx(ret, FloatInstance{}))
}

func Test_uniteDictsKeyMap(t *testing.T) {

	a := NewDictWithMap(StrInstance{}, IntInstance{}, make(map[ConstantValue]Value)).(DictInstance)
	a.TrackedKeys = make(map[ConstantValue]Value)
	a.TrackedKeys[StrConstant("babar")] = IntConstant(18)

	b := NewDictWithMap(StrInstance{}, StrInstance{}, make(map[ConstantValue]Value)).(DictInstance)
	b.TrackedKeys = make(map[ConstantValue]Value)
	b.TrackedKeys[StrConstant("hector")] = StrConstant("est content")

	c := UniteNoCtx(a, b).(DictInstance)
	assert.Len(t, c.TrackedKeys, 2)
	assert.NotEqual(t, &a.TrackedKeys, &c.TrackedKeys)
	assert.NotEqual(t, &b.TrackedKeys, &c.TrackedKeys)
	v, ok := c.TrackedKeys[StrConstant("babar")]
	assert.True(t, ok)
	assert.Equal(t, IntConstant(18), v)
	v, ok = c.TrackedKeys[StrConstant("hector")]
	assert.True(t, ok)
	assert.Equal(t, StrConstant("est content"), v)

}
