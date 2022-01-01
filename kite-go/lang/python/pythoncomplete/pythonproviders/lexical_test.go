package pythonproviders

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

type filterTC struct {
	buffer     string
	completion string
	expected   LexicalFiltersMeta
}

func process(t *testing.T, buffer string) data.SelectedBuffer {
	parts := strings.Split(buffer, "$")
	require.True(t, len(parts) == 2 || len(parts) == 3)
	content := strings.Join(parts, "")
	if len(parts) == 2 {
		return data.Buffer(content).Select(data.Cursor(len(parts[0])))
	}
	return data.Buffer(content).Select(data.Selection{
		Begin: len(parts[0]),
		End:   len(parts[0]) + len(parts[1]),
	})
}

func TestValid(t *testing.T) {
	// add a test case with a placeholder in the completion
	tcs := []filterTC{
		filterTC{
			buffer:     "alpha(b$)",
			completion: "beta",
			expected:   LexicalFiltersMeta{InvalidArgument: true},
		},
		filterTC{
			buffer: `
beta = 4
alpha(b$)
`,
			completion: "beta",
		},
		filterTC{
			buffer:     "alpha = $",
			completion: "beta",
			expected:   LexicalFiltersMeta{InvalidAssignment: true},
		},
		filterTC{
			buffer: `
beta = 4
alpha = b$
`,
			completion: "beta",
		},
		filterTC{
			buffer:     "a$",
			completion: "alpha.beta",
			expected:   LexicalFiltersMeta{InvalidAttribute: true},
		},
		filterTC{
			buffer: `
import alpha
a$
`,
			completion: "alpha.beta",
		},
		filterTC{
			buffer:     "class Al$",
			completion: "Alpha",
		},
		filterTC{
			buffer: `
class Alpha:
	pass

class Al$
`,
			completion: "Alpha",
			expected:   LexicalFiltersMeta{InvalidClassDef: true},
		},
		filterTC{
			buffer:     "import $",
			completion: "alpha",
		},
		filterTC{
			buffer: `
import alpha
import a$
`,
			completion: "alpha",
			expected:   LexicalFiltersMeta{InvalidImport: true},
		},
		filterTC{
			buffer:     "def alpha(epsilon, e$)",
			completion: "eta",
		},
		filterTC{
			buffer:     "def alpha(epsilon, e$)",
			completion: "epsilon",
			expected:   LexicalFiltersMeta{InvalidFunctionDef: true},
		},
		filterTC{
			buffer:     "from alpha import e$",
			completion: "epsilon",
		},
		filterTC{
			buffer: `
from alpha import epsilon
from alpha import e$
`,
			completion: "eta",
		},
		filterTC{
			buffer: `
from alpha import epsilon
from alpha import e$
`,
			completion: "epsilon",
			expected:   LexicalFiltersMeta{InvalidImport: true},
		},
		filterTC{
			buffer: `
from alpha import epsilon, e$
`,
			completion: "eta",
		},
		filterTC{
			buffer: `
from alpha import epsilon, e$
`,
			completion: "epsilon",
			expected:   LexicalFiltersMeta{InvalidImport: true},
		},
	}
	for _, tc := range tcs {
		buffer := process(t, tc.buffer)
		completion := data.BuildSnippet(tc.completion)
		ctx := kitectx.Background()
		analyzer, err := newSemanticAnalyzer(ctx, buffer)
		require.NoError(t, err)
		meta := analyzer.filter(ctx, buffer, completion)
		require.Equal(t, tc.expected, meta)
	}
}

type representTC struct {
	completion string
	expected   string
}

func TestRepresent(t *testing.T) {
	tcs := []representTC{
		representTC{
			completion: "alpha(beta)",
			expected:   "alpha(beta)",
		},
		representTC{
			completion: "alpha(\002[beta]\003)",
			expected:   "alpha(kite_placeholder_representation)",
		},
		representTC{
			completion: "alpha(\002[beta]\003, \002[gamma]\003)",
			expected:   "alpha(kite_placeholder_representation, kite_placeholder_representation)",
		},
		representTC{
			completion: "alpha(\002\003)",
			expected:   "alpha()",
		},
		representTC{
			completion: "alpha(beta\002\003)",
			expected:   "alpha(beta)",
		},
		representTC{
			completion: "alpha(beta, gamma\002\003)",
			expected:   "alpha(beta, gamma)",
		},
	}
	for _, tc := range tcs {
		snippet := data.BuildSnippet(tc.completion)
		require.Equal(t, tc.expected, represent(snippet))
	}
}

type completeTC struct {
	given      string
	completion string
	expected   string
}

func TestComplete(t *testing.T) {
	tcs := []completeTC{
		completeTC{
			given:      "$",
			completion: "alpha",
			expected:   "alpha$",
		},
		completeTC{
			given:      "alpha(be$",
			completion: "beta, gamma",
			expected:   "alpha(beta, gamma$",
		},
		completeTC{
			given:      "alpha(be$)",
			completion: "beta, gamma",
			expected:   "alpha(beta, gamma$)",
		},
		completeTC{
			given: `
def main():
    alpha(b$)
    gamma()
`,
			completion: "beta",
			expected: `
def main():
    alpha(beta$)
    gamma()
`,
		},
		completeTC{
			given:      "al$pha albe$rta",
			completion: "alberta",
			expected:   "alberta$rta",
		},
	}
	for _, tc := range tcs {
		given := process(t, tc.given)
		expected := process(t, tc.expected)
		require.Equal(t, expected, complete(given, tc.completion))
	}
}
