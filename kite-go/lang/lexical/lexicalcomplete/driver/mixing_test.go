package driver

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/assert"
)

type similarSetTC struct {
	completions []string
	expected    []string
}

func TestLexicallySimilarSet(t *testing.T) {
	similarSetTCs := []similarSetTC{
		{},
		{
			completions: []string{"alpha", "alpha(", "alpha()"},
			expected:    []string{"alpha"},
		},
		{
			completions: []string{"alpha(beta)", "alpha(beta"},
			expected:    []string{"alpha(beta)"},
		},
		{
			completions: []string{"alpha(beta", "alpha("},
			expected:    []string{"alpha(beta", "alpha("},
		},
		{
			completions: []string{"beta++", "beta", "alpha", "alpha--"},
			expected:    []string{"alpha", "beta++"},
		},
		{
			completions: []string{
				"alpha(", "alpha()",
				"alpha[", "alpha[]",
				"alpha{}", "alpha{",
				"alpha++", "alpha--", "alpha",
			},
			expected: []string{"alpha("},
		},
		{
			completions: []string{"alpha()beta", "alpha(beta"},
			expected:    []string{"alpha()beta", "alpha(beta"},
		},
		{
			completions: []string{"()", "("},
			expected:    []string{"()"},
		},
	}

	for _, tc := range similarSetTCs {
		keep := make(map[string]bool)
		for _, expected := range tc.expected {
			keep[expected] = true
		}
		similarSet := NewLexicallySimilarSet(data.NewBuffer("").Select(data.Selection{}))
		for _, text := range tc.completions {
			completion := data.Completion{
				Snippet: data.BuildSnippet(text),
			}
			exclude := similarSet.CheckExcludeAndUpdate(completion)
			assert.Equal(t, !exclude, keep[text])
		}
	}
}
