package pythontype

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTuple(t *testing.T) {
	a := NewTuple(IntConstant(123), StrInstance{}, nil)
	b := NewTuple(IntConstant(123), StrInstance{}, nil)
	c := NewTuple(IntConstant(123), StrInstance{})
	assert.True(t, EqualNoCtx(a, b))
	assert.False(t, EqualNoCtx(a, c))
	assert.False(t, EqualNoCtx(b, c))

	countFunc, _ := AttrNoCtx(a, "count")
	count := countFunc.Value().(Callable).Call(Args{})
	assert.EqualValues(t, 3, count)

	indexFunc, _ := AttrNoCtx(a, "index")
	index := indexFunc.Value().(Callable).Call(Args{})
	assert.True(t, EqualNoCtx(index, IntInstance{}))
}
