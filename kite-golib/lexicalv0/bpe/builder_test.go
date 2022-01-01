package bpe

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SplitWord(t *testing.T) {
	type tc struct {
		desc     string
		w        string
		useBytes bool
		expected []string
	}

	// ⌘ == "\u2318" == "\xe2\x8c\x98"
	tcs := []tc{
		{
			desc:     "ascii by bytes",
			w:        "foo",
			useBytes: true,
			expected: []string{"f", "o", "o"},
		},
		{
			desc:     "ascii by unicode",
			w:        "foo",
			useBytes: false,
			expected: []string{"f", "o", "o"},
		},
		{
			desc:     "unicode by unicode",
			w:        "fo\u2318o",
			useBytes: false,
			expected: []string{"f", "o", "\u2318", "o"},
		},
		{
			desc:     "unicode by bytes",
			w:        "fo\u2318o",
			useBytes: true,
			expected: []string{"f", "o", "\xe2", "\x8c", "\x98", "o"},
		},
	}

	for i, tc := range tcs {
		actual := splitWord(tc.w, tc.useBytes)
		assert.Equal(t, tc.expected, actual, "\ncase %d: %s", i, tc.desc)
	}
}

func Test_BuilderSerdes(t *testing.T) {
	for _, useBytes := range []bool{true, false} {
		// ⌘ == "\u2318" == "\xe2\x8c\x98"
		words := []string{"fo\u2318o", "fo\u2318o", "fo\u2318o", "boo", "doo"}

		b := NewBuilder(useBytes)
		b.Add(words)
		b.Merge(MergeOptions{Logging: true})

		require.True(t, len(b.CurrentVocab()) > 0, "use bytes %v", useBytes)

		tmpfile, err := ioutil.TempFile("", "test-builder")
		require.NoError(t, err, "use bytes %v", useBytes)

		_, err = b.WriteTo(tmpfile)
		require.NoError(t, err, "use bytes %v", useBytes)

		require.NoError(t, tmpfile.Close(), "use bytes %v", useBytes)

		b2, err := NewBuilderWithVocab(tmpfile.Name())
		require.NoError(t, err, "use bytes %v", useBytes)

		require.Equal(t, vocabToSlice(b.CurrentVocab()), vocabToSlice(b2.CurrentVocab()), "use bytes %v", useBytes)
	}
}

func Test_BuilderWithUnicode(t *testing.T) {
	// ⌘ == "\u2318" == "\xe2\x8c\x98"
	words := []string{"fo\u2318o", "fo\u2318o", "fo\u2318o", "boo", "doo"}
	counts := map[string]int{
		"fo\u2318o": 3,
		"oo":        2,
		"b":         1,
		"d":         1,
	}
	expectedVocab := makeVocab([]string{"\u2318o", "o\u2318o", "fo\u2318o", "oo", "\u2318", "f", "o", "b", "d"})

	// 1) basic add words and merge
	b := NewBuilder(false)
	b.Add(words)
	b.Merge(MergeOptions{Logging: true})
	require.Equal(t, counts, b.CurrentTokens())
	require.Equal(t, expectedVocab, vocabToSlice(b.CurrentVocab()))

	tmpfile, err := ioutil.TempFile("", "test-builder")
	require.NoError(t, err)

	_, err = b.WriteTo(tmpfile)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	// 2) load build from vocab, add same words, make sure vocab is the same
	b2, err := NewBuilderWithVocab(tmpfile.Name())
	require.NoError(t, err)

	b2.Add(words)
	require.Equal(t, counts, b2.CurrentTokens())

	require.Equal(t, expectedVocab, vocabToSlice(b2.CurrentVocab()))

	tmpdir, err := ioutil.TempDir("", "test-builder")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	b.saveWords(tmpdir)
	wordsPath := filepath.Join(tmpdir, "wordcounts.json")

	// 3) build vocab from word counts, ensure same result
	b3 := NewBuilder(false)
	err = b3.LoadWords(wordsPath, LoadOptions{})
	require.NoError(t, err)
	b3.Merge(MergeOptions{Logging: true})

	require.Equal(t, counts, b2.CurrentTokens())
	require.Equal(t, expectedVocab, vocabToSlice(b3.CurrentVocab()))
}

func Test_BuilderWithBytes(t *testing.T) {
	// ⌘ == "\u2318" == "\xe2\x8c\x98"
	// fo⌘o == "\x66\x6f\xe2\x8c\x98\x6f"
	words := []string{"fo\xe2\x8c\x98o", "fo\xe2\x8c\x98o", "fo\xe2\x8c\x98o", "boo", "doo"}
	counts := map[string]int{
		"fo\xe2\x8c\x98o": 3,
		"oo":              2,
		"b":               1,
		"d":               1,
	}
	expectedVocab := makeVocab([]string{"f", "o", "b", "d", "\xe2", "\x8c", "\x98", "oo", "\xe2\x8c", "\xe2\x8c\x98", "\xe2\x8c\x98o", "o\xe2\x8c\x98o", "fo\xe2\x8c\x98o"})

	// 1) basic add words and merge
	b := NewBuilder(true)
	b.Add(words)
	b.Merge(MergeOptions{Logging: true})
	require.Equal(t, counts, b.CurrentTokens())
	require.Equal(t, expectedVocab, vocabToSlice(b.CurrentVocab()))

	tmpfile, err := ioutil.TempFile("", "test-builder")
	require.NoError(t, err)

	_, err = b.WriteTo(tmpfile)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	// 2) load build from vocab, add same words, make sure vocab is the same
	b2, err := NewBuilderWithVocab(tmpfile.Name())
	require.NoError(t, err)

	b2.Add(words)
	require.Equal(t, counts, b2.CurrentTokens())

	require.Equal(t, expectedVocab, vocabToSlice(b2.CurrentVocab()))

	tmpdir, err := ioutil.TempDir("", "test-builder")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	b.saveWords(tmpdir)
	wordsPath := filepath.Join(tmpdir, "wordcounts.json")

	// 3) build vocab from word counts, ensure same result
	b3 := NewBuilder(true)
	err = b3.LoadWords(wordsPath, LoadOptions{})
	require.NoError(t, err)
	b3.Merge(MergeOptions{Logging: true})

	require.Equal(t, counts, b2.CurrentTokens())
	require.Equal(t, expectedVocab, vocabToSlice(b3.CurrentVocab()))
}

func Test_VocabWithBytes(t *testing.T) {
	// ⌘ == "\u2318" == "\xe2\x8c\x98"
	// fo⌘o == "\x66\x6f\xe2\x8c\x98\x6f"
	words := []string{"fo\xe2\x8c\x98o", "fo\xe2\x8c\x98o", "fo\xe2\x8c\x98o", "boo", "doo"}
	counts := map[string]int{
		"fo\xe2\x8c\x98o": 3,
		"oo":              2,
		"b":               1,
		"d":               1,
	}
	expectedVocab := makeVocab([]string{"f", "o", "b", "d", "\xe2", "\x8c", "\x98", "oo", "\xe2\x8c", "\xe2\x8c\x98", "\xe2\x8c\x98o", "o\xe2\x8c\x98o", "fo\xe2\x8c\x98o"})

	// 1) basic add words and merge
	b := NewBuilder(true)
	b.Add(words)
	b.Merge(MergeOptions{Logging: true})
	require.Equal(t, counts, b.CurrentTokens())
	require.Equal(t, expectedVocab, vocabToSlice(b.CurrentVocab()))

	tmpfile, err := ioutil.TempFile("", "test-builder")
	require.NoError(t, err)

	_, err = b.WriteTo(tmpfile)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	//

	enc1, err := NewEncoder(tmpfile.Name())
	require.NoError(t, err)

	b2 := NewBuilderFromEncoder(enc1)
	enc2 := NewEncoderFromVocab(b2.Vocab())

	require.True(t, enc1.useBytes)
	require.True(t, enc2.useBytes)
	require.Equal(t, enc1.Vocab(), enc2.Vocab())
}

func Test_TokenizedWord(t *testing.T) {
	type testCase struct {
		word                string
		initialTokenization string
		pairs               []MergedPair
		newTokenization     string
	}

	testCases := []testCase{
		{"hello$", "h,e,l,l,o,$", []MergedPair{{"e", "l", "el"}}, "h,el,l,o,$"},                              // middle of word
		{"hello$", "h,e,l,l,o,$", []MergedPair{{"h", "e", "he"}}, "he,l,l,o,$"},                              // beginning of word
		{"hello$", "h,e,l,l,o,$", []MergedPair{{"o", "$", "o$"}}, "h,e,l,l,o$"},                              // end of word
		{"hello$", "h,e,l,l,o,$", []MergedPair{{"x", "y", "xy"}}, "h,e,l,l,o,$"},                             // no-op
		{"hello$", "h,e,l,l,o,$", []MergedPair{{"h", "e", "he"}, {"he", "l", "hel"}}, "hel,l,o,$"},           // multiple merges
		{"foobarfoobaz", "f,o,o,b,a,r,f,o,o,b,a,z", []MergedPair{{"o", "o", "oo"}}, "f,oo,b,a,r,f,oo,b,a,z"}, // multiple pairs
		{"oooo", "o,o,o,o", []MergedPair{{"o", "o", "oo"}}, "oo,oo"},                                         // overlapping pairs
		{"ooooo", "o,o,o,o,o", []MergedPair{{"o", "o", "oo"}}, "oo,oo,o"},                                    // overlapping pairs
	}

	for _, ts := range testCases {
		tw := newTokenizedWord(strings.Split(ts.word, ""))
		tokenization := strings.Join(tw.tokens(), ",")
		assert.Equal(t, ts.initialTokenization, tokenization)
		for _, pair := range ts.pairs {
			tw.mergePair(pair)
		}
		tokenization = strings.Join(tw.tokens(), ",")
		assert.Equal(t, ts.newTokenization, tokenization)
	}
}

func makeVocab(tokens []string) []string {
	sort.Strings(tokens)
	return tokens
}

func vocabToSlice(m map[string]struct{}) []string {
	var ret []string
	for k := range m {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}
