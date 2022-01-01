package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderedMap(t *testing.T) {
	om := NewOrderedMap(0)

	// test set return values
	require.True(t, om.Set(3, 30))
	require.True(t, om.Set(2, 20))
	require.True(t, om.Set(1, 10))
	require.False(t, om.Set(2, 21))

	require.Equal(t, 3, om.Len())

	// test get
	val, ok := om.Get(1)
	require.True(t, ok)
	require.Equal(t, 10, val)
	val, ok = om.Get(2)
	require.True(t, ok)
	require.Equal(t, 21, val)
	val, ok = om.Get(3)
	require.True(t, ok)
	require.Equal(t, 30, val)

	// test get non-existent key
	_, ok = om.Get(4)
	require.False(t, ok)

	// test RangeInc
	numIters := 0
	om.RangeInc(func(k, _ interface{}) bool {
		require.Equal(t, 3-numIters, k.(int))
		numIters++
		return true
	})
	require.Equal(t, 3, numIters)

	// test RangeDec
	numIters = 0
	om.RangeDec(func(k, _ interface{}) bool {
		numIters++
		require.Equal(t, numIters, k.(int))
		return true
	})
	require.Equal(t, 3, numIters)

	// test RangeInc return early
	numIters = 0
	om.RangeInc(func(_, _ interface{}) bool {
		numIters++
		if numIters == 2 {
			return false
		}
		return true
	})
	require.Equal(t, 2, numIters)

	// test RangeDec with delete inside
	om.RangeDec(func(k, _ interface{}) bool {
		om.Delete(k)
		return true
	})
	require.Equal(t, 0, om.Len())
}
