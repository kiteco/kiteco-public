package bpe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type encodeWordTC struct {
	word       string
	vocabWords []string
	expected   []string
}

func TestEncodeWord(t *testing.T) {
	tcs := []encodeWordTC{
		encodeWordTC{
			word:       "aabab",
			vocabWords: []string{"ab", "aa", "b", "a"},
			expected:   []string{"a", "ab", "ab"},
		},
		encodeWordTC{
			word:       "aabab",
			vocabWords: []string{"ab", "aa", "b", "a"},
			expected:   []string{"a", "ab", "ab"},
		},
		encodeWordTC{
			word: "abcdefghi$",
			vocabWords: []string{
				"a", "b", "c", "d", "e", "f", "g", "h", "i", "$",
				"ab", "de", "i$", "def", "abc", "hi$", "ghi$",
			},
			expected: []string{"abc", "def", "ghi$"},
		},
		encodeWordTC{
			word: "aceghfdb$",
			vocabWords: []string{
				"a", "b", "c", "d", "e", "f", "g", "h", "$",
				"ac", "ce", "eg", "gh", "hf", "fd", "db", "b$",
			},
			expected: []string{"a", "ce", "gh", "fd", "b$"},
		},
		encodeWordTC{
			word: "bachfegd$",
			vocabWords: []string{
				"a", "b", "c", "d", "e", "f", "g", "h", "$",
				"ba", "ac", "ch", "hf", "fe", "eg", "gd", "d$",
			},
			expected: []string{"b", "ac", "hf", "eg", "d$"},
		},
		encodeWordTC{
			word: "abcdefgh",
			vocabWords: []string{
				"a", "b", "c", "d", "e", "f", "g", "h",
				"ab", "cd", "ef",
				"abc", "cde", "efg",
				"abcd", "cdef", "efgh",
				"cdefg",
			},
			expected: []string{"abcd", "efgh"},
		},
	}
	for i, tc := range tcs {
		var vocab []Entry
		for _, word := range tc.vocabWords {
			vocab = append(vocab, Entry{BytePair: word, BytePairBytes: nil, Count: 0})
		}
		enc := NewEncoderFromVocab(vocab)
		actual := enc.encodeWord(tc.word)
		assert.Equal(t, tc.expected, actual, "test case %d", i)
	}
}
