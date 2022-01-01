package pythongraph

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

func printWords(words []pythonscanner.Word) string {
	var parts []string
	for _, word := range words {
		parts = append(parts, fmt.Sprintf("[%d:%d]%s", word.Begin, word.End, word.String()))
	}
	return strings.Join(parts, ", ")
}

func tokensString(toks []pythonscanner.Token) string {
	var parts []string
	for _, tok := range toks {
		parts = append(parts, tok.String())
	}
	return strings.Join(parts, ", ")
}

func assertWords(t *testing.T, expected []pythonscanner.Token, actual []pythonscanner.Word) {
	if len(expected) < len(actual) {
		t.Errorf("got extra words:\nExpected\n\t%s\nActual\n\t%s\n", tokensString(expected), printWords(actual))
		return
	}

	if len(expected) > len(actual) {
		t.Errorf("missing words:\nExpected\n\t%s\nActual\n\t%s\n", tokensString(expected), printWords(actual))
		return
	}

	for i := range expected {
		e := expected[i]
		a := actual[i].Token
		if e != a {
			t.Errorf("mismatch at word %d, expected: %s, actual: %s", i, e.String(), a.String())
		}
	}
}

func requireLexAndParse(t *testing.T, src string) ([]pythonscanner.Word, *pythonast.Module) {
	words, err := pythonscanner.Lex([]byte(src), scanOpts)
	require.NoError(t, err)

	mod, _ := pythonparser.ParseWords(kitectx.Background(), []byte(src), words, parseOpts)
	require.NotNil(t, mod)

	var pwords []pythonscanner.Word
	for i := range words {
		pwords = append(pwords, words[i])
	}

	return pwords, mod
}

func printAST(node pythonast.Node) string {
	var b bytes.Buffer
	pythonast.PrintPositions(node, &b, "\t")
	return b.String()
}

func assertWordsForStmt(t *testing.T, src string, expected ...pythonscanner.Token) {
	words, mod := requireLexAndParse(t, src)

	require.Len(t, mod.Body, 1)

	all := wordsForNodes(mod, words)

	stmtWords := all[mod.Body[0]]

	t.Logf("\nStmt:\n%s\n", printAST(mod.Body[0]))

	assertWords(t, expected, stmtWords)
}

func assertWordsForExpr(t *testing.T, src string, expected ...pythonscanner.Token) {
	words, mod := requireLexAndParse(t, src)

	require.Len(t, mod.Body, 1)

	all := wordsForNodes(mod, words)

	require.IsType(t, &pythonast.ExprStmt{}, mod.Body[0])

	es := mod.Body[0].(*pythonast.ExprStmt)

	t.Logf("\nExpr:\n%s\n", printAST(es.Value))

	assertWords(t, expected, all[es.Value])
}

func TestWordsForClass(t *testing.T) {
	src := `
class foo(mar(bar,star)):
	pass
`

	assertWordsForStmt(t, src,
		pythonscanner.Class,
		pythonscanner.Lparen,
		pythonscanner.Rparen,
		pythonscanner.Colon,
	)
}

func TestWordsForFunc(t *testing.T) {
	src := `
def foo(bar=star, car, **mar):
	pass
	`

	assertWordsForStmt(t, src,
		pythonscanner.Def,
		pythonscanner.Lparen,
		pythonscanner.Comma,
		pythonscanner.Comma,
		pythonscanner.Pow,
		pythonscanner.Rparen,
		pythonscanner.Colon,
	)
}

func TestWordsForCall(t *testing.T) {
	src := `
mar(bar,star)
	`

	assertWordsForExpr(t, src,
		pythonscanner.Lparen,
		pythonscanner.Comma,
		pythonscanner.Rparen,
	)
}

func TestWordsForAssign(t *testing.T) {
	src := `x = y`
	assertWordsForStmt(t, src,
		pythonscanner.Assign,
	)
}

func TestWordsForModule(t *testing.T) {
	src := `foo

`

	words, mod := requireLexAndParse(t, src)

	all := wordsForNodes(mod, words)

	for n, ws := range all {
		log.Printf("DEBUG: %s %s", pythonast.String(n), printWords(ws))
	}
	assertWords(t, []pythonscanner.Token{pythonscanner.NewLine, pythonscanner.EOF}, all[mod])

}
