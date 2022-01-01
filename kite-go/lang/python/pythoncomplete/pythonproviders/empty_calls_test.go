package pythonproviders

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func Test_EmptyCalls_LambdaBad(t *testing.T) {
	src := `lambda: foo$`
	res, err := runProvider(t, EmptyCalls{}, src)
	require.NoError(t, err)
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("()"),
		Replace: data.Selection{},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf("(%s)", data.Hole(""))),
		Replace: data.Selection{},
	}))
}

func Test_EmptyCalls_LambdaGood(t *testing.T) {
	src := `import json
lambda: json.dumps$`
	res, err := runProvider(t, EmptyCalls{}, src)
	require.NoError(t, err)
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf("(%s)", data.Hole(""))),
		Replace: data.Selection{},
	}))
}

func Test_EmptyCalls_LambdaGoodBad(t *testing.T) {
	src := `import json
lambda: json.dumps()$`
	res, err := runProvider(t, EmptyCalls{}, src)
	require.NoError(t, err)
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("()"),
		Replace: data.Selection{},
	}))
	require.False(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf("(%s)", data.Hole(""))),
		Replace: data.Selection{},
	}))
}
