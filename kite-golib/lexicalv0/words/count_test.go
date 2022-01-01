package words

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type wordCountDataTC struct {
	Word  string
	Count int
	Ext   string
}

func requireNewWordCounts(t *testing.T, data []wordCountDataTC) Counts {
	cs := make(Counts)
	for _, d := range data {
		cs.Hit(d.Word, d.Ext, d.Count)
	}
	return cs
}

func TestNormalizeCounts(t *testing.T) {
	type tc struct {
		Desc     string
		Data     []wordCountDataTC
		MinCount int
		Expected map[string]int
	}

	tcs := []tc{
		{
			Desc: "basic counting single extension",
			Data: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 2,
				},
				{
					Word:  "bar",
					Count: 3,
				},
				{
					Word:  "car",
					Count: 10,
				},
			},
			MinCount: 2,
			Expected: map[string]int{
				"foo": 2,
				"bar": 3,
				"car": 10,
			},
		},
		{
			Desc: "basic counting single extension, with min",
			Data: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 2,
				},
				{
					Word:  "bar",
					Count: 3,
				},
				{
					Word:  "car",
					Count: 10,
				},
			},
			MinCount: 3,
			Expected: map[string]int{
				"bar": 3,
				"car": 10,
			},
		},
		{
			Desc: "basic counting multiple extensions",
			Data: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 2,
					Ext:   ".js",
				},
				{
					Word:  "foo",
					Count: 4,
					Ext:   ".py",
				},
				{
					Word:  "bar",
					Count: 3,
					Ext:   ".js",
				},
				{
					Word:  "bar",
					Count: 3,
					Ext:   ".py",
				},
				{
					Word:  "car",
					Count: 10,
					Ext:   ".js",
				},
			},
			MinCount: 2,
			Expected: map[string]int{
				"foo": 10,
				"bar": 8,
				"car": 10,
			},
		},
		{
			Desc: "basic counting multiple extensions with min",
			Data: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 1,
					Ext:   ".js",
				},
				{
					Word:  "foo",
					Count: 4,
					Ext:   ".py",
				},
				{
					Word:  "bar",
					Count: 3,
					Ext:   ".js",
				},
				{
					Word:  "bar",
					Count: 3,
					Ext:   ".py",
				},
				{
					Word:  "car",
					Count: 10,
					Ext:   ".js",
				},
			},
			MinCount: 6,
			Expected: map[string]int{
				"bar": 10,
				"car": 10,
			},
		},
	}

	for i, tc := range tcs {
		cs := requireNewWordCounts(t, tc.Data)

		actual := cs.Normalized(tc.MinCount)
		assert.Equal(t, tc.Expected, actual, "test case %d: %s", i, tc.Desc)
	}
}

func TestAdd(t *testing.T) {
	type tc struct {
		Desc         string
		Data1, Data2 []wordCountDataTC
		Expected     []wordCountDataTC
	}

	tcs := []tc{
		{
			Desc: "single extension",
			Data1: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 10,
				},
				{
					Word:  "bar",
					Count: 100,
				},
			},
			Data2: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 20,
				},
				{
					Word:  "car",
					Count: 1000,
				},
			},
			Expected: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 30,
				},
				{
					Word:  "bar",
					Count: 100,
				},
				{
					Word:  "car",
					Count: 1000,
				},
			},
		},
		{
			Desc: "multiple extensions",
			Data1: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 10,
					Ext:   ".js",
				},
				{
					Word:  "foo",
					Count: 20,
					Ext:   ".py",
				},
				{
					Word:  "bar",
					Count: 30,
					Ext:   ".js",
				},
				{
					Word:  "car",
					Count: 40,
					Ext:   ".js",
				},
			},
			Data2: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 15,
					Ext:   ".py",
				},
				{
					Word:  "foo",
					Count: 20,
					Ext:   ".js",
				},
				{
					Word:  "bar",
					Count: 30,
					Ext:   ".js",
				},
				{
					Word:  "star",
					Count: 50,
					Ext:   ".py",
				},
			},
			Expected: []wordCountDataTC{
				{
					Word:  "foo",
					Count: 30,
					Ext:   ".js",
				},
				{
					Word:  "foo",
					Count: 35,
					Ext:   ".py",
				},
				{
					Word:  "bar",
					Count: 60,
					Ext:   ".js",
				},
				{
					Word:  "car",
					Count: 40,
					Ext:   ".js",
				},
				{
					Word:  "star",
					Count: 50,
					Ext:   ".py",
				},
			},
		},
	}

	for i, tc := range tcs {
		a := requireNewWordCounts(t, tc.Data1)
		b := requireNewWordCounts(t, tc.Data2)

		expected := requireNewWordCounts(t, tc.Expected)

		a.Add(b)

		assert.Equal(t, expected, a, "test case %d: %s, a into b", i, tc.Desc)

		a = requireNewWordCounts(t, tc.Data1)
		b = requireNewWordCounts(t, tc.Data2)

		b.Add(a)

		assert.Equal(t, expected, b, "test case %d: %s, b into a", i, tc.Desc)
	}
}
