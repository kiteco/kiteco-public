package pythontype

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnion(t *testing.T) {
	a := UniteNoCtx(IntConstant(123), StrInstance{}, nil)
	b := UniteNoCtx(IntConstant(123), nil, StrInstance{})
	c := UniteNoCtx(IntInstance{}, FloatInstance{})

	assert.True(t, EqualNoCtx(a, b))
	assert.False(t, EqualNoCtx(a, c))
	assert.False(t, EqualNoCtx(b, c))
}

func assertEqual(t *testing.T, a, b Value) {
	if !EqualNoCtx(a, b) {
		t.Errorf("expected %v == %v", a, b)
	}
}

func assertNotEqual(t *testing.T, a, b Value) {
	if EqualNoCtx(a, b) {
		t.Errorf("expected %v != %v", a, b)
	}
}

func TestEqual(t *testing.T) {
	// lists
	assertEqual(t, IntInstance{}, IntInstance{})
	assertEqual(t, FloatInstance{}, FloatInstance{})
	assertEqual(t, nil, nil)
	assertNotEqual(t, StrInstance{}, IntInstance{})
	assertNotEqual(t, StrInstance{}, nil)
	assertNotEqual(t, nil, StrInstance{})

	// lists
	assertEqual(t, NewList(IntInstance{}), NewList(IntInstance{}))
	assertEqual(t, NewList(nil), NewList(nil))
	assertEqual(t, NewList(NewList(StrInstance{})), NewList(NewList(StrInstance{})))
	assertEqual(t, NewList(UniteNoCtx(IntInstance{}, StrInstance{})), NewList(UniteNoCtx(IntInstance{}, StrInstance{})))
	assertNotEqual(t, NewList(IntInstance{}), NewList(UniteNoCtx(IntInstance{}, StrInstance{})))
	assertNotEqual(t, NewList(nil), NewList(StrInstance{}))

	// dicts
	assertEqual(t, NewDict(IntInstance{}, StrInstance{}), NewDict(IntInstance{}, StrInstance{}))
	assertEqual(t, NewDict(FloatInstance{}, nil), NewDict(FloatInstance{}, nil))
	assertEqual(t,
		NewDict(FloatInstance{}, NewDict(IntInstance{}, StrInstance{})),
		NewDict(FloatInstance{}, NewDict(IntInstance{}, StrInstance{})))
	assertNotEqual(t,
		NewDict(IntInstance{}, StrInstance{}),
		NewDict(UniteNoCtx(IntInstance{}, StrInstance{}), StrInstance{}))
	assertNotEqual(t, NewDict(nil, IntInstance{}), NewDict(StrInstance{}, IntInstance{}))

	// tuples
	assertEqual(t, NewTuple(IntInstance{}, StrInstance{}), NewTuple(IntInstance{}, StrInstance{}))
	assertEqual(t, NewTuple(nil), NewTuple(nil))
	assertEqual(t, NewTuple(), NewTuple())
	assertEqual(t, NewTuple(UniteNoCtx(IntInstance{}, StrInstance{})), NewTuple(UniteNoCtx(StrInstance{}, IntInstance{})))
	assertNotEqual(t, NewTuple(nil), NewTuple(nil, nil))
	assertNotEqual(t, NewTuple(IntInstance{}, StrInstance{}), NewTuple(StrInstance{}, IntInstance{}))

	// unions
	assertEqual(t, UniteNoCtx(), UniteNoCtx())
	assertEqual(t, UniteNoCtx(IntInstance{}), UniteNoCtx(IntInstance{}))
	assertEqual(t, UniteNoCtx(IntInstance{}, StrInstance{}), UniteNoCtx(IntInstance{}, StrInstance{}))
	assertEqual(t, UniteNoCtx(IntInstance{}, StrInstance{}), UniteNoCtx(StrInstance{}, IntInstance{}))
	assertEqual(t, UniteNoCtx(IntInstance{}, StrInstance{}), UniteNoCtx(StrInstance{}, IntInstance{}, IntInstance{}))
	assertEqual(t, IntInstance{}, UniteNoCtx(IntInstance{}, UniteNoCtx(IntInstance{}, IntInstance{})))
	assertEqual(t,
		UniteNoCtx(FloatInstance{}, IntInstance{}, NewTuple(StrInstance{}, IntInstance{})),
		UniteNoCtx(NewTuple(StrInstance{}, IntInstance{}), UniteNoCtx(FloatInstance{}, IntInstance{})))
	assertNotEqual(t, UniteNoCtx(), UniteNoCtx(ComplexInstance{}))
	assertNotEqual(t, UniteNoCtx(IntInstance{}), UniteNoCtx(IntInstance{}, ComplexInstance{}))
	assertNotEqual(t, UniteNoCtx(IntInstance{}, StrInstance{}, ComplexInstance{}), UniteNoCtx(IntInstance{}, StrInstance{}))
	assertNotEqual(t, UniteNoCtx(IntInstance{}, StrInstance{}), StrInstance{})
	assertNotEqual(t, IntInstance{}, UniteNoCtx(StrInstance{}, IntInstance{}, IntInstance{}))
	assertNotEqual(t, IntInstance{}, UniteNoCtx(IntInstance{}, UniteNoCtx(ComplexInstance{}, IntInstance{})))
	assertNotEqual(t,
		UniteNoCtx(FloatInstance{}, IntInstance{}, NewTuple(StrInstance{}, IntInstance{})),
		UniteNoCtx(NewTuple(StrInstance{}, IntInstance{}), UniteNoCtx(FloatInstance{}, StrInstance{})))

	// unknowns
	assertEqual(t, UniteNoCtx(StrInstance{}), UniteNoCtx(nil, StrInstance{}))
}
