package lexicalproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func Test_TextHTML_EmptyFile(t *testing.T) {
	src := `$`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.html")
	require.Empty(t, res.out)
}

func Test_TextHTML_Basic(t *testing.T) {
	src := `<h$
	
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.html")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("head"),
		Replace: data.Selection{Begin: -1, End: 0},
	}))
}

func Test_TextHTML_Basic1(t *testing.T) {
	src := `<html>
<$
	
	`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.html")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("head"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}

func Test_TextHTML_Basic2(t *testing.T) {
	src := `<html>
		<head>
		</head>
		<$
`
	initModels(t, lexicalmodels.DefaultModelOptions)
	res := requireRes(t, Text{}, src, "./src.html")
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.BuildSnippet("body"),
		Replace: data.Selection{Begin: 0, End: 0},
	}))
}
