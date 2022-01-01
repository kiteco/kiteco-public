package pythonkeyword

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/stretchr/testify/assert"
)

func TestKeywordCatTokenMapping(t *testing.T) {
	toks := pythonscanner.KeywordTokens
	for _, tok := range toks {
		cat := KeywordTokenToCat(tok)
		if cat >= 0 {
			tokFromCat := KeywordCatToToken(cat)
			assert.Equal(t, tok, tokFromCat)
		}
	}
}

func TestKeywordCatAreConsecutives(t *testing.T) {
	// This test ensure that all categories are unique and consecutive
	// If this test fail, make sure that there's no holes in the category in the file kite-go/lang/python/pythoncompletions/keywords.go
	catMap := make(map[int]bool)
	toks := pythonscanner.KeywordTokens
	for _, tok := range toks {
		cat := KeywordTokenToCat(tok)
		if cat >= 0 {
			catMap[cat] = true
		}
	}
	assert.Equal(t, uint(len(catMap)), NumKeywords())
	for i := uint(1); i <= NumKeywords(); i++ {
		_, ok := catMap[int(i)]
		assert.True(t, ok, "No keyword is associated to the category ", i)
	}
}
