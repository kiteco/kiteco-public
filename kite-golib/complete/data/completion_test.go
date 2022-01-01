package data

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	_, ok := Completion{
		Snippet: BuildSnippet(fmt.Sprintf("(%s)", Hole(""))),
		Replace: Selection{3, 5},
	}.Validate(NewBuffer("foo()").Select(Cursor(4)))
	require.True(t, ok)
}

func TestCompletion_AfterEmptyPlaceholder(t *testing.T) {
	first := Completion{Snippet: Snippet{Text: "()", placeholders: []Selection{{1, 1}}}}
	second := Completion{
		Snippet: Snippet{Text: "a=[...]", placeholders: []Selection{{2, 7}}},
		Replace: Selection{1, 1},
	}

	result, err := second.After(first)
	require.NoError(t, err)
	require.Equal(t, 1, len(result.Snippet.placeholders))
	require.Equal(t, "(a=[...])", result.Snippet.Text)
}

func Test_HasMultipleIdents(t *testing.T) {
	tests := []struct {
		compl   Completion
		expects bool
	}{
		{
			compl: Completion{
				Snippet: Snippet{Text: "foo"},
			},
			expects: false,
		},
		{
			compl: Completion{
				Snippet: Snippet{Text: "foo."},
			},
			expects: false,
		},
		{
			compl: Completion{
				Snippet: Snippet{Text: "foo()"},
			},
			expects: false,
		},
		{
			compl: Completion{
				Snippet: Snippet{Text: "foo.bar"},
			},
			expects: true,
		},
		{
			compl: Completion{
				Snippet: Snippet{Text: "foo(bar)"},
			},
			expects: true,
		},
	}
	for _, test := range tests {
		result, err := test.compl.HasMultiIdents()
		require.NoError(t, err)
		assert.EqualValues(t, test.expects, result, "test: "+test.compl.Snippet.Text)
	}
}

func TestValidSuffix(t *testing.T) {
	// Consider sample program with cursor at $:
	//
	// abc012xyz = 1
	// foo(a$012xyz)
	//
	// The following completion should be valid and snippet should include the suffix.
	c, ok := Completion{
		Snippet: Snippet{Text: "abc012xyz"},
		Replace: Selection{18, 25},
	}.Validate(NewBuffer(`abc012xyz = 1
foo(a012xyz)`).Select(Cursor(19)))
	require.True(t, ok)
	require.Equal(t, "abc012xyz", c.Snippet.Text)
}

func TestDisplay_EmptyPlaceholder(t *testing.T) {
	comp := Completion{Snippet: Snippet{Text: "))", placeholders: []Selection{
		{0, 0}, {1, 1}}}, Replace: Selection{
		Begin: 6,
		End:   8,
	}}
	b := SelectedBuffer{
		Buffer:    "then(())",
		Selection: Selection{Begin: 6, End: 6},
	}
	opts := DisplayOptions{TrimBeforeEmptyPH: true}
	display := comp.DisplayText(b, opts)
	require.Equal(t, "", display)
}
