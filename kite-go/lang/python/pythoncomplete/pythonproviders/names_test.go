package pythonproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

func TestNames_Import(t *testing.T) {
	src := `import os.$\n`
	_, err := runProvider(t, Names{}, src)
	require.Equal(t, data.ProviderNotApplicableError{}, err)
}

func TestNames_InScope(t *testing.T) {
	src := `import json

j$`
	res, err := runProvider(t, Names{}, src)
	require.NoError(t, err)
	require.True(t, res.containsRoot(data.Completion{
		Snippet: data.NewSnippet("json"),
		Replace: data.Selection{Begin: -1},
	}))
}

func TestNames_SuffixMatch(t *testing.T) {
	src := `
def foo_bar_rest(x):

foo_b$rest
`
	res, err := runProvider(t, Names{}, src)
	require.NoError(t, err)
	_, ok := res.getFromRoot(data.Completion{
		Snippet: data.NewSnippet("foo_bar_rest"),
		Replace: data.Selection{Begin: -5, End: 4},
	})
	require.True(t, ok, "completion should exist")
}

func TestNames_PartialMatch(t *testing.T) {
	src := `
def func(arg):
	return arg

def foobar(x):
	pass

f$bar
`
	res, err := runProvider(t, Names{}, src)
	require.NoError(t, err)
	_, ok := res.getFromRoot(data.Completion{
		Snippet: data.NewSnippet("foobar"),
		Replace: data.Selection{Begin: -1, End: 3},
	})
	require.True(t, ok, "completion should exist")
	_, ok = res.getFromRoot(data.Completion{
		Snippet: data.NewSnippet("func"),
		Replace: data.Selection{Begin: -1},
	})
	require.True(t, ok, "completion should exist")
}

func TestNames_ValidCall(t *testing.T) {
	src := `foo(bar$`
	_, err := runProvider(t, Names{}, src)
	require.NoError(t, err)
}

func TestNames_ValidKwargValue(t *testing.T) {
	src := `foo(bar, k=v$`
	_, err := runProvider(t, Names{}, src)
	require.NoError(t, err)
}

func TestNames_InValidKwarg(t *testing.T) {
	src := `foo(bar, k=v, baz$`
	_, err := runProvider(t, Names{}, src)
	require.Equal(t, data.ProviderNotApplicableError{}, err)
}

func TestNames_Ordering(t *testing.T) {
	src := `import json

def foo(x):
    def bar(y):
        $...$
`
	res, err := runProvider(t, Names{}, src)
	require.NoError(t, err)

	y, ok := res.getFromRoot(data.Completion{
		Snippet: data.NewSnippet("y"),
	})
	require.True(t, ok, "no completion `y`")

	x, ok := res.getFromRoot(data.Completion{
		Snippet: data.NewSnippet("x"),
	})
	require.True(t, ok, "no completion `x`")

	json, ok := res.getFromRoot(data.Completion{
		Snippet: data.NewSnippet("json"),
	})
	require.True(t, ok, "no completion `json`")

	none, ok := res.getFromRoot(data.Completion{
		Snippet: data.NewSnippet("None"),
	})
	require.True(t, ok, "no completion `None`")

	require.True(t, none.Score < json.Score && json.Score < x.Score && x.Score < y.Score, "wrong ordering")
}

func TestNames_FunctionDef(t *testing.T) {
	src := `
def foo$
`
	_, err := runProvider(t, Names{}, src)
	require.Error(t, err, "names provider should not emit for function definition")
}

func TestNames_BadIf(t *testing.T) {
	src := `
def search(phs: foobar) -> tf.tensor:
    if$
    pass
`
	_, err := runProvider(t, Names{}, src)
	require.Error(t, err, "names provider should not emit for bad if statement")
}
