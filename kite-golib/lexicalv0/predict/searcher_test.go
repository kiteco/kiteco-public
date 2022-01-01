package predict

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/stretchr/testify/assert"
)

func assertEqualCandidates(t *testing.T, expected, actual candidates) {
	if len(expected) != len(actual) {
		// use logging
		assert.Equal(t, expected, actual)
		return
	}

	for j, e := range expected {
		a := actual[j]
		assert.Equal(t, e.TokenIDs, a.TokenIDs)

		diff := math.Abs(float64(e.Prob) - float64(a.Prob))
		assert.True(t, diff < 1e-4, "%.4f != %.4f", e.Prob, a.Prob)
	}
}

func assertEqualBatchExts(t *testing.T, expected, actual batchExts) {
	if len(actual) != len(expected) {
		// use logging
		assert.Equal(t, expected, actual)
		return
	}

	for i, act := range actual {
		assertEqualCandidates(t, expected[i], act)
	}
}

type filterExtsTC struct {
	filter       func(candidates) candidates
	raw          batchExts
	rawBase      candidates
	expected     batchExts
	expectedBase candidates
}

func TestFilterExts(t *testing.T) {
	filter := func(cs candidates) candidates {
		switch len(cs) {
		case 0:
			return candidates{candidate{[]int64{1}, 0}}
		case 1:
			return nil
		case 2:
			return candidates{candidate{[]int64{-1}, 1}, candidate{[]int64{-2, -3}, .1}}
		default:
			return cs
		}
	}
	tcs := []filterExtsTC{
		filterExtsTC{
			// test reassign
			filter: filter,
			raw: batchExts{
				candidates{},
				candidates{
					{[]int64{1, 2}, .1},
					{[]int64{3, 4}, .2},
				},
			},
			rawBase: candidates{
				{[]int64{1, 2}, .1},
				{[]int64{3, 4}, .2},
			},
			expected: batchExts{
				candidates{candidate{[]int64{1}, 0}},
				candidates{candidate{[]int64{-1}, 1}, candidate{[]int64{-2, -3}, .1}},
			},
			expectedBase: candidates{
				{[]int64{1, 2}, .1},
				{[]int64{3, 4}, .2},
			},
		},
		filterExtsTC{
			// test everything filtered
			filter: filter,
			raw: batchExts{
				candidates{candidate{[]int64{3, 4}, .25}},
				candidates{candidate{[]int64{5, 6}, .05}},
			},
			rawBase: candidates{
				candidate{[]int64{1, 2}, .5},
				candidate{[]int64{3, 4}, .1},
			},
			expected:     nil,
			expectedBase: nil,
		},
		filterExtsTC{
			// test make sure base and exts stay in sync after filtering
			filter: filter,
			raw: batchExts{
				candidates{candidate{[]int64{3, 4}, .25}},
				candidates{
					candidate{[]int64{5, 6}, .05},
					candidate{[]int64{7, 8}, .1},
					candidate{[]int64{9, 10}, .1},
				},
			},
			rawBase: candidates{
				candidate{[]int64{1, 2}, .5},
				candidate{[]int64{3, 4}, .1},
			},
			expected: batchExts{
				candidates{
					candidate{[]int64{5, 6}, .05},
					candidate{[]int64{7, 8}, .1},
					candidate{[]int64{9, 10}, .1},
				},
			},
			expectedBase: candidates{
				candidate{[]int64{3, 4}, .1},
			},
		},
	}

	for _, tc := range tcs {
		actualBase, actualExts := searcher{strict: true}.filterExts(kitectx.Background(), tc.rawBase, tc.raw, tc.filter)
		assertEqualCandidates(t, tc.expectedBase, actualBase)
		assertEqualBatchExts(t, tc.expected, actualExts)
	}
}

type filterPredsTC struct {
	vocab    []string
	prefix   string
	raw      batchExts
	expected batchExts
}

func TestFilterPreds(t *testing.T) {
	tcs := []filterPredsTC{
		filterPredsTC{
			vocab:  []string{"abcdef", "abdefc", "cbadef", "abcjklm", "fn", "ab", "abc", "c"},
			prefix: "abc",
			raw: batchExts{
				candidates{
					candidate{[]int64{1, 2, 3}, 0.1},
					candidate{[]int64{0, 1, 6}, 0.2},
					candidate{[]int64{3, 2}, 0.4},
				},
				candidates{
					candidate{[]int64{7, 3}, 0.5},
					candidate{[]int64{5, 7, 3}, 0.3},
					candidate{[]int64{6}, 0.2},
					candidate{[]int64{4}, 0.1},
					candidate{[]int64{2, 6}, 0.4},
				},
			},
			expected: batchExts{
				candidates{
					candidate{[]int64{0, 1, 6}, 0.2},
					candidate{[]int64{3, 2}, 0.4},
				},
				candidates{
					candidate{[]int64{5, 7, 3}, 0.3},
					candidate{[]int64{6}, 0.2},
				},
			},
		},
		filterPredsTC{
			// test lowercase matching
			vocab:  []string{"abcdef", "abdefc", "cbadef", "abcjklm", "fn", "ab", "abc", "c"},
			prefix: "ABC",
			raw: batchExts{
				candidates{
					candidate{[]int64{1, 2, 3}, 0.1},
					candidate{[]int64{7, 0, 1, 6}, 0.2},
					candidate{[]int64{7, 3, 2}, 0.4},
				},
				candidates{
					candidate{[]int64{7, 3}, 0.5},
					candidate{[]int64{5, 7, 3}, 0.3},
					candidate{[]int64{6}, 0.2},
					candidate{[]int64{4}, 0.1},
					candidate{[]int64{2, 6}, 0.4},
				},
			},
			expected: batchExts{
				candidates{
					candidate{[]int64{5, 7, 3}, 0.3},
					candidate{[]int64{6}, 0.2},
				},
			},
		},
		filterPredsTC{
			vocab:  []string{"abcdef", "abdefc", "cbadef", "abcjklm", "fn", "ab", "abc", "c"},
			prefix: "",
			raw: batchExts{
				candidates{
					candidate{[]int64{1, 2, 3}, 0.1},
					candidate{[]int64{0, 1, 6}, 0.2},
					candidate{[]int64{3, 2}, 0.4},
				},
				candidates{
					candidate{[]int64{7, 3}, 0.5},
					candidate{[]int64{5, 7, 3}, 0.3},
				},
				candidates{
					candidate{[]int64{6}, 0.2},
					candidate{[]int64{4}, 0.1},
					candidate{[]int64{2, 6}, 0.4},
				},
			},
			expected: batchExts{
				candidates{
					candidate{[]int64{1, 2, 3}, 0.1},
					candidate{[]int64{0, 1, 6}, 0.2},
					candidate{[]int64{3, 2}, 0.4},
				},
				candidates{
					candidate{[]int64{7, 3}, 0.5},
					candidate{[]int64{5, 7, 3}, 0.3},
				},
				candidates{
					candidate{[]int64{6}, 0.2},
					candidate{[]int64{4}, 0.1},
					candidate{[]int64{2, 6}, 0.4},
				},
			},
		},
	}
	for _, tc := range tcs {
		s := searcher{
			enc: &lexicalv0.FileEncoder{
				IDToStringLower: toLower(tc.vocab),
			},
			prefix: tc.prefix,
			strict: true,
		}

		actualExts := s.filterByPrefix(kitectx.Background(), tc.raw)
		assert.Equal(t, tc.expected, actualExts)
	}
}

type normalizePredsTC struct {
	s         searcher
	prefixReg float32
	raw       batchExts
	expected  batchExts
}

func TestNormalizePreds(t *testing.T) {
	tcs := []normalizePredsTC{
		normalizePredsTC{
			prefixReg: 0,
			raw: batchExts{
				candidates{
					candidate{TokenIDs: []int64{3, 8, 2}, Prob: 1. / 100},
					candidate{TokenIDs: []int64{2, 4}, Prob: 2. / 100},
				},
				candidates{
					candidate{TokenIDs: []int64{9}, Prob: 25. / 100},
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: 25. / 100},
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: 50. / 100},
				},
				candidates{
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: 0.01},
				},
			},
			expected: batchExts{
				candidates{
					candidate{TokenIDs: []int64{3, 8, 2}, Prob: 1. / 3},
					candidate{TokenIDs: []int64{2, 4}, Prob: 2. / 3},
				},
				candidates{
					candidate{TokenIDs: []int64{9}, Prob: .25},
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: .25},
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: .50},
				},
				candidates{
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: 1.},
				},
			},
		},
		normalizePredsTC{
			prefixReg: 0.15,
			raw: batchExts{
				candidates{
					candidate{TokenIDs: []int64{3, 8, 2}, Prob: .25},
					candidate{TokenIDs: []int64{2, 4}, Prob: 0.3},
					candidate{TokenIDs: []int64{9}, Prob: 0.1},
				},
			},
			expected: batchExts{
				candidates{
					candidate{TokenIDs: []int64{3, 8, 2}, Prob: 0.3125},
					candidate{TokenIDs: []int64{2, 4}, Prob: 0.375},
					candidate{TokenIDs: []int64{9}, Prob: 0.125},
				},
			},
		},
		normalizePredsTC{
			prefixReg: 1.,
			raw: batchExts{
				candidates{
					candidate{TokenIDs: []int64{3, 8, 2}, Prob: 0.01},
					candidate{TokenIDs: []int64{2, 4}, Prob: 0.02},
					candidate{TokenIDs: []int64{9}, Prob: 0.04},
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: 0.01},
				},
			},
			expected: batchExts{
				candidates{
					candidate{TokenIDs: []int64{3, 8, 2}, Prob: 0.01},
					candidate{TokenIDs: []int64{2, 4}, Prob: 0.02},
					candidate{TokenIDs: []int64{9}, Prob: 0.04},
					candidate{TokenIDs: []int64{93, 12, 3, 3}, Prob: 0.01},
				},
			},
		},
	}
	for _, tc := range tcs {
		tc.raw.normalize(tc.prefixReg)
		assertEqualBatchExts(t, tc.expected, tc.raw)
	}
}

type softmaxTC struct {
	raw      batchExts
	expected batchExts
}

func TestSoftmax(t *testing.T) {
	tcs := []softmaxTC{
		softmaxTC{
			raw: batchExts{
				candidates{
					{[]int64{1, 3}, .5},
					{[]int64{2, 4}, .5},
				},
				candidates{
					{[]int64{5, 6}, 100},
					{[]int64{7, 8}, 100},
				},
			},
			expected: batchExts{
				candidates{
					{[]int64{1, 3}, .5},
					{[]int64{2, 4}, .5},
				},
				candidates{
					{[]int64{5, 6}, .5},
					{[]int64{7, 8}, .5},
				},
			},
		},
		softmaxTC{
			raw: batchExts{
				candidates{
					{[]int64{9, 10, 11}, 25},
					{[]int64{12}, 50},
					{[]int64{13, 14}, 25},
				},
			},
			expected: batchExts{
				candidates{
					{[]int64{9, 10, 11}, 0},
					{[]int64{12}, 1.},
					{[]int64{13, 14}, 0},
				},
			},
		},
		softmaxTC{
			raw: batchExts{
				candidates{
					{[]int64{9, 10, 11}, -1},
					{[]int64{12}, 1},
					{[]int64{13, 14}, 2},
				},
			},
			expected: batchExts{
				candidates{
					{[]int64{9, 10, 11}, 0.03511903},
					{[]int64{12}, 0.25949646},
					{[]int64{13, 14}, 0.70538451},
				},
			},
		},
	}

	for _, tc := range tcs {
		tc.raw.softmax()
		assertEqualBatchExts(t, tc.expected, tc.raw)
	}
}

type tempScaleTC struct {
	lexTemp   float32
	identTemp float32
	isLexical func(int) bool
	raw       batchExts
	expected  batchExts
}

func TestTempScale(t *testing.T) {
	tcs := []tempScaleTC{
		tempScaleTC{
			lexTemp:   1. / 3,
			identTemp: 2.,
			isLexical: func(c int) bool { return c > 10 },
			raw: batchExts{
				candidates{
					{[]int64{11, 1}, .5},
					{[]int64{1, 11}, 3},
				},
				candidates{
					{[]int64{2, 22}, 1},
				},
			},
			expected: batchExts{
				candidates{
					{[]int64{11, 1}, 1.5},
					{[]int64{1, 11}, 1.5},
				},
				candidates{
					{[]int64{2, 22}, .5},
				},
			},
		},
		tempScaleTC{
			lexTemp:   1. / 3,
			identTemp: 2.,
			isLexical: func(c int) bool { return c > 10 },
			raw:       batchExts{},
			expected:  batchExts{},
		},
		tempScaleTC{
			lexTemp:   1. / 3,
			identTemp: 2.,
			isLexical: func(c int) bool { return c > 10 },
			raw:       batchExts{candidates{}},
			expected:  batchExts{candidates{}},
		},
	}

	for _, tc := range tcs {
		tc.raw.temperatureScale(tc.isLexical, tc.lexTemp, tc.identTemp)
		assertEqualBatchExts(t, tc.expected, tc.raw)
	}
}

type newBatchExtsTC struct {
	scores   [][]float32
	expected batchExts
}

func TestNewBatchExts(t *testing.T) {
	tcs := []newBatchExtsTC{
		newBatchExtsTC{
			scores: [][]float32{
				{.1, .2, .4},
				{.1, .1},
			},
			expected: batchExts{
				candidates{
					{[]int64{0}, .1},
					{[]int64{1}, .2},
					{[]int64{2}, .4},
				},
				candidates{
					{[]int64{0}, .1},
					{[]int64{1}, .1},
				},
			},
		},
		newBatchExtsTC{
			scores:   [][]float32{},
			expected: batchExts{},
		},
		newBatchExtsTC{
			scores:   [][]float32{[]float32{}},
			expected: batchExts{candidates{}},
		},
	}

	for _, tc := range tcs {
		actual := newBatchExts(tc.scores)
		assertEqualBatchExts(t, tc.expected, actual)
	}
}

type allExpansionsTC struct {
	exts     batchExts
	base     candidates
	expected candidates
}

func TestAllExpansions(t *testing.T) {
	tcs := []allExpansionsTC{
		allExpansionsTC{
			// base case
			exts: batchExts{
				candidates{
					{[]int64{1}, .1},
					{[]int64{2}, .1},
					{[]int64{3}, .2},
				},
			},
			base: nil,
			expected: candidates{
				{[]int64{1}, .1},
				{[]int64{2}, .1},
				{[]int64{3}, .2},
			},
		},
		allExpansionsTC{
			// test underflow
			exts: batchExts{
				candidates{
					{[]int64{1}, .1},
					{[]int64{2}, 1e-10},
				},
				candidates{
					{[]int64{3}, .2},
					{[]int64{4}, .2},
				},
			},
			base: candidates{
				{[]int64{1, 2}, 1},
				{[]int64{3, 4}, .5},
			},
			expected: candidates{
				{[]int64{1, 2, 1}, .1},
				{[]int64{3, 4, 3}, .1},
				{[]int64{3, 4, 4}, .1},
			},
		},
		allExpansionsTC{
			// test underflow
			exts: batchExts{
				candidates{
					{[]int64{1}, .1},
					{[]int64{2}, .1},
				},
				candidates{
					{[]int64{3}, .2},
					{[]int64{4}, .2},
				},
			},
			base: candidates{
				{[]int64{1, 2}, 1},
				{[]int64{3, 4}, .5},
			},
			expected: candidates{
				{[]int64{1, 2, 1}, .1},
				{[]int64{1, 2, 2}, .1},
				{[]int64{3, 4, 3}, .1},
				{[]int64{3, 4, 4}, .1},
			},
		},
	}

	for _, tc := range tcs {
		exps := searcher{strict: true}.allExpansions(kitectx.Background(), tc.base, tc.exts)
		assertEqualCandidates(t, tc.expected, exps)
	}
}

type sampleTC struct {
	raw        candidates
	expected   candidates
	numSamples int
}

func TestSample(t *testing.T) {
	tcs := []sampleTC{
		sampleTC{
			// more samples than cands
			numSamples: 2,
			raw: candidates{
				{[]int64{1, 2}, .1},
				{[]int64{3, 4}, .4},
			},
			expected: candidates{
				{[]int64{1, 2}, .1},
				{[]int64{3, 4}, .4},
			},
		},
		sampleTC{
			numSamples: 3,
			raw: candidates{
				{[]int64{1, 2}, .01},
				{[]int64{3, 4}, .01},
				{[]int64{5, 6}, 1},
				{[]int64{7, 8}, 10},
				{[]int64{9, 10}, 100},
			},
			expected: candidates{
				{[]int64{9, 10}, 100},
				{[]int64{7, 8}, 10},
				{[]int64{5, 6}, 1},
			},
		},
	}

	for _, tc := range tcs {
		s := searcher{
			strict: true,
			rand:   rand.New(rand.NewSource(0)),
		}
		actual := s.sample(kitectx.Background(), tc.raw, tc.numSamples)
		assertEqualCandidates(t, tc.expected, actual)
	}
}

type drawSampleTC struct {
	raw      candidates
	expected int
}

func TestDrawSample(t *testing.T) {
	tcs := []drawSampleTC{
		drawSampleTC{
			raw: candidates{
				{Prob: .8},
				{Prob: .01},
			},
			expected: 0,
		},
		drawSampleTC{
			raw: candidates{
				{Prob: 0},
				{Prob: 0},
				{Prob: 1},
			},
			expected: 2,
		},
	}

	for _, tc := range tcs {
		s := searcher{
			strict: true,
			rand:   rand.New(rand.NewSource(0)),
		}
		assert.Equal(t, tc.expected, s.drawSample(tc.raw))
	}
}

type selectCandidatesTC struct {
	search   SearchConfig
	raw      candidates
	expected candidates
}

func TestSelectCandidates(t *testing.T) {
	// Note: selectCandidates is deterministic when BeamWidth >= TopK
	tcs := []selectCandidatesTC{
		selectCandidatesTC{
			// test top k
			search: SearchConfig{
				TopK:      2,
				MinP:      0.015,
				BeamWidth: 5,
			},
			raw: candidates{
				candidate{[]int64{9, 10}, 0.2},
				candidate{[]int64{3, 8, 2}, 0.4},
				candidate{[]int64{7, 1}, 0.3},
			},
			expected: candidates{
				candidate{[]int64{3, 8, 2}, 0.4},
				candidate{[]int64{7, 1}, 0.3},
			},
		},
		selectCandidatesTC{
			// test minp
			search: SearchConfig{
				TopK:      5,
				TopP:      0.5,
				MinP:      0.075,
				BeamWidth: 5,
			},
			raw: candidates{
				candidate{[]int64{9, 10}, 0.08},
				candidate{[]int64{3, 8, 2}, 0.03},
				candidate{[]int64{7, 1}, 0.04},
				candidate{[]int64{8, 3}, 0.07},
				candidate{[]int64{3, 1, 3, 5}, 0.06},
				candidate{[]int64{5, 2, 3, 5}, 0.09},
				candidate{[]int64{2, 3, 5}, 0.01},
				candidate{nil, 0.05},
			},
			expected: candidates{
				candidate{[]int64{9, 10}, 0.08},
				candidate{[]int64{5, 2, 3, 5}, 0.09},
			},
		},
		selectCandidatesTC{
			search: SearchConfig{
				TopK:      5,
				TopP:      0.5,
				MinP:      0.075,
				BeamWidth: 5,
			},
			raw:      nil,
			expected: candidates{},
		},
	}
	for _, tc := range tcs {
		s := searcher{
			strict: true,
			config: tc.search,
			rand:   rand.New(rand.NewSource(0)),
		}
		selected := s.selectCandidates(kitectx.Background(), tc.raw)
		assertEqualCandidates(t, tc.expected, selected)
	}
}

func toLower(str []string) []string {
	ret := make([]string, 0, len(str))
	for _, s := range str {
		ret = append(ret, strings.ToLower(s))
	}
	return ret
}

type idsMatchingPrefixTC struct {
	vocab    []string
	prefix   string
	expected map[int]bool
}

func TestIDsMatchingPrefix(t *testing.T) {
	tcs := []idsMatchingPrefixTC{
		idsMatchingPrefixTC{
			vocab:    []string{"abcdef", "abdefc", "cbadef", "abcjklm", "fn", "ab", "abc"},
			prefix:   "abc",
			expected: map[int]bool{0: true, 3: true, 5: true, 6: true},
		},
		idsMatchingPrefixTC{
			vocab:    []string{"abcdef", "abdefc", "cbadef", "abcjklm", "fn", "ab", "abc"},
			prefix:   "ABC",
			expected: map[int]bool{0: true, 3: true, 5: true, 6: true},
		},
		idsMatchingPrefixTC{
			vocab:    []string{"abcdef", "abdefc", "cbadef"},
			prefix:   "",
			expected: map[int]bool{0: true, 1: true, 2: true},
		},
	}
	for _, tc := range tcs {
		enc := &lexicalv0.FileEncoder{
			IDToStringLower: toLower(tc.vocab),
		}
		actual := idsMatchingPrefix(enc, tc.prefix)
		assert.Equal(t, tc.expected, actual)
	}
}

type idsMatchingPrefixSliceTC struct {
	vocab    []string
	prefix   string
	expected []int64
}

func TestIDsMatchingPrefixSlice(t *testing.T) {
	tcs := []idsMatchingPrefixSliceTC{
		idsMatchingPrefixSliceTC{
			vocab:    []string{"abcdef", "abdefc", "cbadef", "abcjklm", "fn", "ab", "abc"},
			prefix:   "abc",
			expected: []int64{0, 3, 5, 6},
		},
		idsMatchingPrefixSliceTC{
			vocab:    []string{"abcdef", "abdefc", "cbadef", "abcjklm", "fn", "ab", "abc"},
			prefix:   "ABC",
			expected: []int64{0, 3, 5, 6},
		},
		idsMatchingPrefixSliceTC{
			vocab:    []string{"abcdef", "abdefc", "cbadef"},
			prefix:   "",
			expected: []int64{0, 1, 2},
		},
	}
	for _, tc := range tcs {
		enc := &lexicalv0.FileEncoder{
			IDToStringLower: toLower(tc.vocab),
		}
		actual := idsMatchingPrefixSlice(enc, tc.prefix)
		assert.Equal(t, tc.expected, actual)
	}
}
