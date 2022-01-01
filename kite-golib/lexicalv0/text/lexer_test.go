package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type maybeIdentTC struct {
	Desc     string
	Text     string
	Expected bool
}

func assertMaybeIdent(t *testing.T, tcs []maybeIdentTC) {
	for i, tc := range tcs {
		actual := MaybeIdent(tc.Text)
		assert.Equal(t, tc.Expected, actual, "TC %d: %s", i, tc.Desc)
	}
}

func Test_MaybeIdent(t *testing.T) {
	tcs := []maybeIdentTC{
		{
			Desc:     "dollar sign is ident",
			Text:     "$",
			Expected: true,
		},
		{
			Desc:     "ident with spaces is ident",
			Text:     " foo ",
			Expected: true,
		},
		{
			Desc:     "all punctuation is not ident",
			Text:     "..!",
			Expected: false,
		},
		{
			Desc:     "only underscore is ident",
			Text:     "_",
			Expected: true,
		},
		{
			Desc:     "only dash is not ident",
			Text:     "-",
			Expected: false,
		},
		{
			Desc:     "alpha numerics are ident",
			Text:     "hello1",
			Expected: true,
		},
		{
			Desc:     "dash in ident is ok",
			Text:     "color-red",
			Expected: true,
		},
	}

	assertMaybeIdent(t, tcs)
}

type lexerLitTC struct {
	Desc string
	Src  string

	Expected []string
}

func assertLexerLit(t *testing.T, tcs []lexerLitTC) {
	for i, tc := range tcs {
		toks, err := NewLexer().Lex([]byte(tc.Src))
		require.NoError(t, err)

		var actual []string
		for _, t := range toks {
			if t.Lit == "" {
				continue
			}
			actual = append(actual, t.Lit)
		}
		assert.Equal(t, tc.Expected, actual, "test case %d: %s", i, tc.Desc)
	}
}

func Test_LexerLit_Golang(t *testing.T) {
	tcs := []lexerLitTC{
		{
			Desc:     "merge keywords function header",
			Src:      "func (m *Manager) HandleLanguages(w )",
			Expected: []string{"func ", "(", "m", " ", "*", "Manager", ") ", "HandleLanguages", "(", "w", " ", ")"},
		},
		{
			Desc:     "slice variable declaration",
			Src:      "var resp []string",
			Expected: []string{"var ", "resp", " ", "[]", "string"},
		},
		{
			Desc:     "partial slice variable declaration",
			Src:      "var resp []str",
			Expected: []string{"var ", "resp", " ", "[]", "str"},
		},
	}

	assertLexerLit(t, tcs)
}

func Test_LexerLit_Javascript(t *testing.T) {
	tcs := []lexerLitTC{
		{
			Desc:     "jsx element",
			Src:      "<button onClick={ >",
			Expected: []string{"<", "button", " ", "onClick", "={", " ", ">"},
		},
	}
	assertLexerLit(t, tcs)
}
