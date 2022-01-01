package python

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
		{"", []string{""}, []int{symModule}},
		{"x", []string{"x", "end_of_statement"}, []int{symIdentifier, endOfStatement}},
		// KITE_ILLEGAL was mapped to 222
		{"x ?", []string{"x", "?"}, []int{symIdentifier, 222}},
		{`
def fn():
  pass
  `,
			[]string{"def", "fn", "(", ")", ":", "start_of_block", "pass", "end_of_statement", "end_of_block"},
			[]int{anonSymDef, symIdentifier, anonSymLparen, anonSymRparen, anonSymColon, startOfBlock, anonSymPass, endOfStatement, endOfBlock},
		},
		{`"abc"`,
			[]string{"\"abc\"", "end_of_statement"},
			[]int{symString, endOfStatement},
		},
		{`"\r\n\t"`,
			[]string{"\"\\r\\n\\t\"", "end_of_statement"},
			[]int{symString, endOfStatement},
		},
		{`[ x for x in range(20) if x % 2 == 0]`,
			[]string{"[", "x", "for", "x", "in", "range", "(", "20", ")", "if", "x", "%", "2", "==", "0", "]", "end_of_statement"},
			[]int{anonSymLbrack, symIdentifier, anonSymFor, symIdentifier, anonSymIn, symIdentifier, anonSymLparen, symInteger, anonSymRparen, anonSymIf, symIdentifier, anonSymPercent, symInteger, anonSymEqEq, symInteger, anonSymRbrack, endOfStatement},
		},
		{`3.1415l # a float`,
			[]string{"3.1415l", "end_of_statement", "# a float"},
			[]int{symFloat, endOfStatement, symComment},
		},
		{`print x`,
			[]string{"print", "x", "end_of_statement"},
			[]int{anonSymPrint, symIdentifier, endOfStatement},
		},
		{`
if x:
  y = 1
  z = 2
  `,
			[]string{"if", "x", ":", "start_of_block", "y", "=", "1", "end_of_statement", "z", "=", "2", "end_of_statement", "end_of_block"},
			[]int{anonSymIf, symIdentifier, anonSymColon, startOfBlock, symIdentifier, anonSymEq, symInteger, endOfStatement, symIdentifier, anonSymEq, symInteger, endOfStatement, endOfBlock},
		},
		{`
class Foo():
  def bar(): pass
`,
			[]string{"class", "Foo", "(", ")", ":", "start_of_block", "def", "bar", "(", ")", ":", "start_of_block", "pass", "end_of_statement", "end_of_block", "end_of_block"},
			[]int{anonSymClass, symIdentifier, anonSymLparen, anonSymRparen, anonSymColon, startOfBlock, anonSymDef, symIdentifier, anonSymLparen, anonSymRparen, anonSymColon, startOfBlock, anonSymPass, endOfStatement, endOfBlock, endOfBlock},
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
	lexer := Lexer{}
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
