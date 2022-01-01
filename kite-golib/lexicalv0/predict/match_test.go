package predict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MatchSuffix(t *testing.T) {
	type testCase struct {
		embedded   []int
		given      []int
		match      bool
		unembedded []int
	}

	tcs := []testCase{
		// larger embedding, match somewhere in the middle should fail
		{
			embedded: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			given:    []int{4, 5, 6},
			match:    false,
		},
		// larger embedding, match at the front should fail
		{
			embedded: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			given:    []int{1, 2, 3, 4},
			match:    false,
		},
		// larger embedding, matches at the end should work with at least minOverlap
		{
			embedded:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			given:      []int{10, 1, 2, 3},
			match:      true,
			unembedded: []int{1, 2, 3},
		},
		// larger embedding, no overlap, no match
		{
			embedded: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			given:    []int{11, 12, 13},
			match:    false,
		},
		// larger context, match somewhere in the middle should fail
		{
			embedded: []int{1, 2, 3, 4, 5},
			given:    []int{0, 1, 2, 3, 4, 5, 6, 7},
			match:    false,
		},
		// larger context, match at the front should succeed
		{
			embedded:   []int{1, 2, 3, 4, 5},
			given:      []int{1, 2, 3, 4, 5, 6, 7, 8},
			match:      true,
			unembedded: []int{6, 7, 8},
		},
		// larger context, matches at the end should work with at least minOverlap
		{
			embedded:   []int{1, 2, 3, 4, 5},
			given:      []int{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			match:      true,
			unembedded: []int{6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		},
		// larger context, no overlap, no match
		{
			embedded: []int{1, 2, 3, 4, 5},
			given:    []int{11, 12, 13},
			match:    false,
		},
		// exact match
		{
			embedded:   []int{1, 2, 3, 4, 5},
			given:      []int{1, 2, 3, 4, 5},
			match:      true,
			unembedded: nil,
		},
	}

	for idx, tc := range tcs {
		emb, unemb, match := matchSuffix(toInt64(tc.embedded), toInt64(tc.given))
		assert.Equal(t, tc.match, match, "%d", idx)
		assert.Equal(t, toInt64(tc.embedded), emb, "%d", idx)
		assert.Equal(t, toInt64(tc.unembedded), unemb, "%d", idx)
	}
}

func Test_MatchPrefix(t *testing.T) {
	type testCase struct {
		embedded []int64
		given    []int64
		match    int
	}

	tcs := []testCase{
		{
			embedded: nil,
			given:    nil,
			match:    0,
		},
		{
			embedded: nil,
			given:    []int64{1},
			match:    0,
		},
		{
			embedded: []int64{1},
			given:    nil,
			match:    0,
		},
		{
			embedded: []int64{1, 2, 3},
			given:    []int64{4, 5},
			match:    0,
		},
		{
			embedded: []int64{1, 2, 3},
			given:    []int64{1, 2},
			match:    2,
		},
		{
			embedded: []int64{1, 2},
			given:    []int64{1, 2, 3},
			match:    2,
		},
		{
			embedded: []int64{1, 2, 3},
			given:    []int64{3, 4, 5},
			match:    0,
		},
		{
			embedded: []int64{1, 2, 3, 4, 5},
			given:    []int64{1, 2, 3, 4, 5},
			match:    5,
		},
	}

	for idx, tc := range tcs {
		match := matchPrefix(tc.embedded, tc.given)
		assert.Equal(t, tc.match, match, "for case %d", idx)
	}
}

func Test_PrefixSuffixPartialRunMatch(t *testing.T) {
	type testCase struct {
		slots                         int
		numAfterTokensRequiredToMatch int
		before                        []int64
		after                         []int64
		givenBefore                   []int64
		givenAfter                    []int64
		embedded                      []int64
		unembedded                    []int64
		match                         bool
	}

	tcs := []testCase{
		// start with after, no after given, fail
		{
			after: []int64{1, 2},
			match: false,
		},
		// no after to start with, given after, fail
		{
			givenAfter: []int64{1, 2},
			match:      false,
		},
		// not enough overlap in after, fail
		{
			after:                         []int64{1, 2, 3, 4},
			givenAfter:                    []int64{1, 2, 5, 6, 7, 8},
			numAfterTokensRequiredToMatch: 4,
			match:                         false,
		},
		// larger embedding in before, match somewhere in the middle should fail
		{
			before:      []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			givenBefore: []int64{4, 5, 6},
			match:       false,
		},
		// larger embedding in before, match at the front should fail
		{
			before:      []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			givenBefore: []int64{1, 2, 3, 4},
			match:       false,
		},
		// exact match in before
		{
			before:      []int64{1, 2, 3, 4, 5},
			givenBefore: []int64{1, 2, 3, 4, 5},
			match:       true,
			unembedded:  nil,
			embedded:    []int64{1, 2, 3, 4, 5},
		},
		// exact match after, but len less than min number of tokens, pass
		{
			after:       []int64{1, 2, 3},
			givenAfter:  []int64{1, 2, 3},
			before:      []int64{1, 2, 3, 4, 5},
			givenBefore: []int64{1, 2, 3, 4, 5},
			match:       true,
			unembedded:  nil,
			embedded:    []int64{1, 2, 3, 4, 5},
		},
	}

	// not enough slots
	var ctx []int64
	for i := 0; i < 512; i++ {
		ctx = append(ctx, int64(i))
	}
	tcs = append(tcs, testCase{
		before:      ctx,
		givenBefore: []int64{254, 255},
		match:       false,
		slots:       3,
	})

	for idx, tc := range tcs {
		m := PrefixSuffixPartialRunModel{
			hp: HParams{
				ContextSize: 1024,
			},
			initIn: PrefixSuffixInputs{
				After:  tc.after,
				Before: tc.before,
			},
			numAfterTokensRequiredToMatch: tc.numAfterTokensRequiredToMatch,
		}

		if tc.numAfterTokensRequiredToMatch == 0 {
			m.numAfterTokensRequiredToMatch = numAfterTokensRequiredToMatch
		}

		embedded, unembedded, match := m.Match(tc.givenBefore, tc.givenAfter, tc.slots, SearchConfig{})
		assert.Equal(t, tc.match, match, "match case %d", idx)
		assert.Equal(t, tc.embedded, embedded, "embedded case %d", idx)
		assert.Equal(t, tc.unembedded, unembedded, "unembedded case %d", idx)
	}
}
