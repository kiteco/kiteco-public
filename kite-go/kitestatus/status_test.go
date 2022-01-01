package kitestatus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Counter(t *testing.T) {
	defer clear()
	c1 := GetCounter("counter1")
	c2 := GetCounter("counter2")

	c1.Incr(1)
	c2.Incr(2)

	require.Equal(t, makeMap("counter1", int64(1)), c1.Value())
	require.Equal(t, makeMap("counter2", int64(2)), c2.Value())

	ks := Get()
	require.Equal(t,
		map[string]interface{}{
			"counter1": int64(1),
			"counter2": int64(2),
		},
		ks,
	)

	Reset()

	require.Equal(t, makeMap("counter1", int64(0)), c1.Value())
	require.Equal(t, makeMap("counter2", int64(0)), c2.Value())

	ks = Get()
	require.Equal(t,
		map[string]interface{}{
			"counter1": int64(0),
			"counter2": int64(0),
		},
		ks,
	)
}

func Test_Boolean(t *testing.T) {
	defer clear()
	b1 := GetBooleanDefault("bool1", true)
	b2 := GetBooleanDefault("bool2", false)

	b1.SetBool(false)
	b2.SetBool(true)

	require.Equal(t, makeMap("bool1", false), b1.Value())
	require.Equal(t, makeMap("bool2", true), b2.Value())

	ks := Get()
	require.Equal(t,
		map[string]interface{}{
			"bool1": false,
			"bool2": true,
		},
		ks,
	)

	Reset()

	require.Equal(t, makeMap("bool1", true), b1.Value())
	require.Equal(t, makeMap("bool2", false), b2.Value())

	ks = Get()
	require.Equal(t,
		map[string]interface{}{
			"bool1": true,
			"bool2": false,
		},
		ks,
	)
}

type testMetric struct {
	val1 int
	val2 string
}

func (t *testMetric) Value() map[string]interface{} {
	return map[string]interface{}{
		"val1": t.val1,
		"val2": t.val2,
	}
}

func (t *testMetric) Reset() {
	t.val1 = 0
	t.val2 = ""
}

func Test_Metric(t *testing.T) {
	defer clear()
	tm := &testMetric{}
	m := GetMetric("metric1", tm)

	tm.val1 = 42
	tm.val2 = "hello world"

	require.Equal(t,
		map[string]interface{}{
			"val1": 42,
			"val2": "hello world",
		},
		m.Value(),
	)

	ks := Get()
	require.Equal(t,
		map[string]interface{}{
			"val1": 42,
			"val2": "hello world",
		},
		ks,
	)

	Reset()

	require.Equal(t,
		map[string]interface{}{
			"val1": 0,
			"val2": "",
		},
		m.Value(),
	)

	ks = Get()
	require.Equal(t,
		map[string]interface{}{
			"val1": 0,
			"val2": "",
		},
		ks,
	)
}

// --

func makeMap(name string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		name: value,
	}
}

func clear() {
	metrics.Range(func(key, value interface{}) bool {
		metrics.Delete(key)
		return true
	})
}
