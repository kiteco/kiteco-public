package html

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
		{"", []string{""}, []int{symFragment}},
		{"<div/>", []string{"<", "div", "/>"}, []int{anonSymLt, symTagName, anonSymSlashGt}},
		{"<body>hello</body>", []string{"<", "body", ">", "hello", "</", "body", ">"}, []int{anonSymLt, symTagName, anonSymGt, symText, anonSymLtSlash, symTagName4, anonSymGt}},
		{`<a href="x" disabled>`, []string{"<", "a", "href", "=", `"`, "x", `"`, "disabled", ">"}, []int{anonSymLt, symTagName, symAttributeName, anonSymEq, anonSymDquote, symAttributeValue3, anonSymDquote, symAttributeName, anonSymGt}},
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
