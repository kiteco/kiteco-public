package pythonproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func TestKWArgs_TypedPrefix(t *testing.T) {
	// skipkeys is important here:
	// it's the second argument as per the argspec;
	// we must not consider it "filled" by the partial argument `sk`
	// which is an easy mistake to make.
	src := `import json
json.dumps(1, sk$)
`
	res, err := runProvider(t, KWArgs{}, src)
	require.NoError(t, err)
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("skipkeys=" + data.Hole("...")),
		Replace: data.Selection{Begin: -2},
	}))
}
