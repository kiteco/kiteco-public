package pythonindex

import (
	"index/suffixarray"
	"strings"
)

type suffixArray struct {
	array         *suffixarray.Index
	indexedTokens string
}

func newSuffixArray(tokens []string) *suffixArray {
	joined := "\x00" + strings.Join(tokens, "\x00") + "\x00"
	return &suffixArray{
		array:         suffixarray.New([]byte(joined)),
		indexedTokens: joined,
	}
}

func (c *suffixArray) prefixedBy(t string) []string {
	if t == "" {
		return nil
	}
	var prefixed []string

	// The query must be the prefix of the string, which can be changed later.
	indices := c.array.Lookup([]byte("\x00"+t), -1)
	for _, start := range indices {
		end := strings.Index(c.indexedTokens[start+1:], "\x00") + start + 1
		prefixed = append(prefixed, c.indexedTokens[start+1:end])

	}
	return prefixed
}
