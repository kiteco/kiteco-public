package pythonproviders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportAs(t *testing.T) {
	tc := requireTestCaseWithModelAndSymbols(t, `import matplotlib a$`)
	completions := requireCompletions(t, tc, Imports{})
	require.Len(t, completions, 1, "only one instance of completion should be returned")
	comp := completions[0]
	assert.Equal(t, "as mpl", comp.Snippet.Text)
}
