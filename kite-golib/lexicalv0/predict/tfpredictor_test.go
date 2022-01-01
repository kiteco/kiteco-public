package predict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type makeKeyTC struct {
	embedded []int64
	contexts [][]int64
	expected string
}

func TestMakeKey(t *testing.T) {
	tcs := []makeKeyTC{
		makeKeyTC{
			embedded: []int64{8, 12, 4},
			contexts: [][]int64{
				{9, 3, 10, 4},
				{9, 3, 17, 8},
				{9, 3, 3, 7, 2},
			},
			expected: "8,12,4,9,3,10,4:8,12,4,9,3,17,8:8,12,4,9,3,3,7,2",
		},
		makeKeyTC{
			embedded: []int64{8, 12, 4},
			contexts: [][]int64{
				[]int64{10, 4},
				[]int64{17, 8},
				[]int64{3, 7, 2},
			},
			expected: "8,12,4,10,4:8,12,4,17,8:8,12,4,3,7,2",
		},
		makeKeyTC{
			embedded: nil,
			contexts: [][]int64{
				{8, 12, 4, 10, 4},
				{8, 12, 4, 17, 8},
				{8, 12, 4, 3, 7, 2},
			},
			expected: "8,12,4,10,4:8,12,4,17,8:8,12,4,3,7,2",
		},
		makeKeyTC{
			expected: "",
		},
		makeKeyTC{
			embedded: []int64{8, 12, 4},
			contexts: [][]int64{
				{9, 3},
			},
			expected: "8,12,4,9,3",
		},
	}
	for _, tc := range tcs {
		p := &TFPredictor{}
		key := p.predCacheKey(tc.embedded, tc.contexts)
		assert.Equal(t, key, tc.expected)
	}
}
