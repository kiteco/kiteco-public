package predict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mergeContextsTC struct {
	curated            []editorEvent
	natural            []int
	window             int
	naturalWindow      int
	minNatural         int
	expectedMerged     []int
	expectedNumCurated int
}

func TestMergeContexts(t *testing.T) {
	tcs := []mergeContextsTC{
		mergeContextsTC{
			curated: []editorEvent{
				{[]int{1, 2, 3}},
				{[]int{8, 2}},
			},
			natural:            []int{10, 4, 6, 4, 32},
			window:             6,
			naturalWindow:      4,
			minNatural:         3,
			expectedMerged:     []int{8, 2, 4, 6, 4, 32},
			expectedNumCurated: 2,
		},
		mergeContextsTC{
			curated: []editorEvent{
				{[]int{1, 2, 3}},
				{[]int{3, 4}},
				{[]int{8, 2}},
			},
			natural:            []int{6, 4, 32},
			window:             6,
			naturalWindow:      4,
			minNatural:         3,
			expectedMerged:     []int{8, 2, 6, 4, 32},
			expectedNumCurated: 2,
		},
		mergeContextsTC{
			curated: []editorEvent{
				{[]int{1, 2, 3}},
				{[]int{3, 4}},
				{[]int{8, 2}},
			},
			natural:            []int{6, 4, 32},
			window:             7,
			naturalWindow:      4,
			minNatural:         3,
			expectedMerged:     []int{3, 4, 8, 2, 6, 4, 32},
			expectedNumCurated: 4,
		},
		mergeContextsTC{
			curated: []editorEvent{
				{[]int{1, 2, 3}},
				{[]int{3, 4}},
				{[]int{8, 2}},
			},
			natural:            []int{6, 4, 32},
			window:             8,
			naturalWindow:      4,
			minNatural:         3,
			expectedMerged:     []int{3, 4, 8, 2, 6, 4, 32},
			expectedNumCurated: 4,
		},
		mergeContextsTC{
			curated: []editorEvent{
				{[]int{1, 2, 3}},
				{[]int{8, 2}},
			},
			natural:            nil,
			window:             6,
			naturalWindow:      4,
			minNatural:         3,
			expectedMerged:     nil,
			expectedNumCurated: 0,
		},
	}
	for _, tc := range tcs {
		merged, numCurated := mergeContexts(tc.curated, tc.natural, tc.window, tc.naturalWindow, tc.minNatural)
		assert.Equal(t, tc.expectedMerged, merged)
		assert.Equal(t, tc.expectedNumCurated, numCurated)
	}
}

type curatedContextUsedTC struct {
	curatedTokens map[int]bool
	tokens        []int
	expected      bool
}

func TestCuratedContextUsed(t *testing.T) {
	tcs := []curatedContextUsedTC{
		curatedContextUsedTC{
			curatedTokens: map[int]bool{
				3: true,
				8: true,
			},
			tokens:   []int{5, 10},
			expected: false,
		},
		curatedContextUsedTC{
			curatedTokens: map[int]bool{
				3: true,
				8: true,
			},
			tokens:   []int{5, 3, 10},
			expected: true,
		},
	}
	for _, tc := range tcs {
		used := curatedContextUsed(tc.curatedTokens, tc.tokens)
		assert.Equal(t, tc.expected, used)
	}
}
