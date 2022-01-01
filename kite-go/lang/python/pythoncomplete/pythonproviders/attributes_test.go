package pythonproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func TestAttr_JsonDot(t *testing.T) {
	src := `import json

json.$`
	res, err := runProvider(t, Attributes{}, src)
	require.NoError(t, err)
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("encoder"),
		Replace: data.Selection{},
	}))
}

func TestAttr_JsonDotD(t *testing.T) {
	src := `import json

json.d$`
	res, err := runProvider(t, Attributes{}, src)
	require.NoError(t, err)
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("dump"),
		Replace: data.Selection{Begin: -1},
	}))
}

func TestAttr_JsonDotSuffix(t *testing.T) {
	src := `import json

json.d$p`
	res, err := runProvider(t, Attributes{}, src)
	require.NoError(t, err)
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("dump"),
		Replace: data.Selection{Begin: -1, End: 1},
	}))

	comp := `import json

json.dump`

	var found bool
	for _, val := range res.out {
		for _, v := range val {
			buf := res.in.SelectedBuffer
			buf = buf.Select(v.Completion.Replace).ReplaceWithCursor(v.Completion.Snippet.Text)
			if buf.Buffer.Text() == comp {
				found = true
				break
			}
		}
	}
	require.True(t, found)
}

func TestAttr_JsonDotPartial(t *testing.T) {
	src := `import json

json.d$p`
	res, err := runProvider(t, Attributes{}, src)
	require.NoError(t, err)
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("dumps"),
		Replace: data.Selection{Begin: -1},
	}))
}
