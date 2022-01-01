package pythontype

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	a := NewList(FloatInstance{})
	b := NewList(FloatInstance{})
	c := NewList(NewTuple(StrConstant("foo"), IntInstance{}))
	assert.True(t, EqualNoCtx(a, b))
	assert.False(t, EqualNoCtx(a, c))
	assert.False(t, EqualNoCtx(b, c))

	res, _ := AttrNoCtx(a, "count")
	countFunc := res.Single.Value
	count := countFunc.(Callable).Call(Args{})
	assert.True(t, EqualNoCtx(count, IntInstance{}))

	res, _ = AttrNoCtx(c, "pop")
	popFunc := res.Single.Value
	popped := popFunc.(Callable).Call(Args{})
	elt := popped.(Indexable).Index(IntConstant(0), false)
	assert.Equal(t, string(elt.(StrConstant)), "foo")
}
