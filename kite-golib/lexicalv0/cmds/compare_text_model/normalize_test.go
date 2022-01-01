package main

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type normalizeTC struct {
	NativeLang lang.Language
	Desc       string
	Input      string
	Expected   string
}

func assertNormalized(t *testing.T, tcs []normalizeTC) {
	textLexer, err := lexicalv0.NewLexerWithOpts(lang.Text, true)
	require.NoError(t, err)

	for i, tc := range tcs {
		nativeLexer, err := lexicalv0.NewLexerWithOpts(tc.NativeLang, true)
		require.NoError(t, err)

		textTokens, err := textLexer.Lex([]byte(tc.Input))
		require.NoError(t, err)

		actual, err := Normalize(textTokens, nativeLexer)
		require.NoError(t, err)

		assert.Equal(t, tc.Expected, actual, "test case %d: %s", i, tc.Desc)
	}
}

func Test_NormalizeGolang(t *testing.T) {
	tcs := []normalizeTC{
		{
			NativeLang: lang.Golang,
			Desc:       "basic for loop",
			Input:      "for i,x := range xs",
			Expected:   "for IDENT,IDENT := range IDENT",
		},
		{
			NativeLang: lang.Golang,
			Desc:       "number literals",
			Input:      "print(1, 2)",
			Expected:   "IDENT(LIT, LIT)",
		},
		{
			NativeLang: lang.Golang,
			Desc:       "string literal",
			Input:      `print("foo", "bar")`,
			Expected:   "IDENT(LIT, LIT)",
		},
	}

	assertNormalized(t, tcs)

}
