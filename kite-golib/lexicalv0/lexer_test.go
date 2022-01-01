package lexicalv0

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexerForMetrics(t *testing.T) {
	for _, l := range AllLangsGroup.Langs {
		_, err := NewLexerForMetrics(l)
		assert.NoError(t, err)
	}
}
