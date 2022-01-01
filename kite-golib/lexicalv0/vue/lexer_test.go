package vue

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
		{"", nil, nil},
		{"<template></template>", []string{""}, []int{1024}},                          // HTML fragment
		{"<script>x</script>", []string{"x", ""}, []int{2001, 2117}},                  // ident, automatic semi
		{"<style>div {}</style>", []string{"div", "{", "}"}, []int{3111, 3009, 3010}}, // tag name, lbrace, rbrace
		{"<style>p {\n  font-size: 2em;\n}\n</style>", []string{"p", "{", "font-size", ":", "2", "em", ";", "}"}, []int{3111, 3009, 3110, 3017, 3088, 3044, 3003, 3010}},
		{"<script>function() {}</script><style>div {}</style>", []string{"function", "(", ")", "{", "}", "", "div", "{", "}"}, []int{2051, 2018, 2019, 2006, 2008, 2117, 3111, 3009, 3010}},
		{"<script>|/#</script>", []string{"|", "/", "#"}, []int{2075, 2102, 3112}}, // 3112 = ERROR remapped to css.SymbolCount()

		{`
<template>
  <p>{{ greeting }} World</p>
</template>

<script>
module.exports = {
  data: function() {
    return {'greeting': 'Hello'}
  }
}
</script>

<style>
p {
  font-size: 2em;
}
</style>
`,
			[]string{
				"\n  ", "<", "p", ">", "{{ greeting }} World", "</", "p", ">", "\n",
				"module", ".", "exports", "=", "{", "data", ":", "function", "(", ")", "{", "return", "{", "'", "greeting", "'", ":", "'", "Hello", "'", "}", "", "}", "}", "",
				"p", "{", "font-size", ":", "2", "em", ";", "}",
			},
			[]int{
				1015, 1005, 1016, 1003, 1015, 1007, 1019, 1003, 1015,
				2001, 2047, 2227, 2039, 2006, 2227, 2034, 2051, 2018, 2019, 2006, 2031, 2006, 2096, 2097, 2096, 2034, 2096, 2097, 2096, 2008, 2117, 2008, 2008, 2117,
				3111, 3009, 3110, 3017, 3088, 3044, 3003, 3010,
			},
		},
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
	lexer, err := NewCompleteLexer()
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
