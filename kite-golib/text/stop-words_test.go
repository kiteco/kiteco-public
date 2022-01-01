package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStopWords(t *testing.T) {
	testWords := []string{"i", "he", "has", "weren't"}
	stopWords := StopWords()
	for _, word := range testWords {

		_, exists := stopWords[word]
		assert.Equal(t, true, exists)
	}
}
