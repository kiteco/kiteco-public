package css

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	cases := []struct {
		in   string
		lits []string
		toks []int
	}{
		{"", []string{""}, []int{symStylesheet}},
		{"* {\n}", []string{"*", "{", "}"}, []int{anonSymStar, anonSymLbrace, anonSymRbrace}},
		{`@import url("navigation.css");`, []string{"@import", "url", "(", `"navigation.css"`, ")", ";"}, []int{anonSymAtImport, symFunctionName, anonSymLparen, symStringValue, anonSymRparen, anonSymSemi}},
		{`
body {
  background-color: #001122;
  border-size: 1em;
}`, []string{"body", "{", "background-color", ":", "#", "001122", ";", "border-size", ":", "1", "em", ";", "}"}, []int{symTagName, anonSymLbrace, symPropertyName, anonSymColon, anonSymPound, symColorValue, anonSymSemi, symPropertyName, anonSymColon, symIntegerValue, symUnit, anonSymSemi, anonSymRbrace}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			lits, toks := requireLex(t, c.in)
			require.Equal(t, c.lits, lits)
			require.Equal(t, c.toks, toks)
		})
	}
}

func requireLex(t *testing.T, src string) ([]string, []int) {
	lexer, err := NewLexer()
	require.NoError(t, err)

	toks, err := lexer.Lex([]byte(src))
	require.NoError(t, err)

	var ids []int
	var lits []string
	for _, tok := range toks {
		lits = append(lits, tok.Lit)
		ids = append(ids, tok.Token)
	}

	return lits, ids
}
