package pythonproviders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeywordsReturnNoSpace(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
def myFavoriteFunction():
	print("Life is beautiful!")
	ret$`)

	completions := requireCompletions(t, tc, Keywords{})
	require.NotEmpty(t, completions)
	comp := completions[0]
	assert.Equal(t, "return", comp.Snippet.Text, "There shouldn't be a space added after the return keyword")
}

func TestKeywordsPartialMatch(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `
im$bar`)
	completions := requireCompletions(t, tc, Keywords{})
	require.NotEmpty(t, completions)
	comp := completions[0]
	assert.Equal(t, "import ", comp.Snippet.Text, "partial match should be suggested")
}
