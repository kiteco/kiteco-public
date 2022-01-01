package lexicalv0

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireLexer(t *testing.T, l lang.Language) lexer.Lexer {
	ll, err := NewLexer(l)
	require.NoError(t, err)
	return ll
}

type testData struct {
	Cursor string
	Lexer  lexer.Lexer
	Tokens []lexer.Token
	SB     data.SelectedBuffer
}

func requireTestData(t *testing.T, src, cursor string, originalLang, lexerLang lang.Language) testData {
	var cursorPos int
	parts := strings.Split(src, cursor)
	switch len(parts) {
	case 1:
		cursorPos = len(src)
	case 2:
		src = parts[0] + parts[1]
		cursorPos = len(parts[0])
	default:
		require.Fail(t, "bad test case format, got %d parts should be 1 or 2", len(parts))
	}

	sb := data.NewBuffer(src).Select(data.Selection{
		Begin: cursorPos,
		End:   cursorPos,
	})

	ll := requireLexer(t, lexerLang)
	toks, err := LexSelectedBuffer(sb, originalLang, ll)
	require.NoError(t, err)

	// Make sure we exclude EOF token for text lexer
	// since when we only lex before-cursor part
	if lexerLang == lang.Text && strings.TrimSpace(src[cursorPos:]) != "" && len(toks) > 0 {
		assert.False(t, ll.IsType(lexer.EOF, toks[len(toks)-1]),
			"cursor before end of file but token EOF before cursor for text lexer")
	}

	return testData{
		Cursor: cursor,
		Lexer:  ll,
		Tokens: toks,
		SB:     sb,
	}
}

type findContextTC struct {
	Desc         string
	Src          string
	Cursor       string
	OriginalLang lang.Language
	LexerLang    lang.Language

	Prefix       string
	LastTokenIdx int
}

func assertFindContext(t *testing.T, tcs []findContextTC) {
	for i, tc := range tcs {
		require.NotEmpty(t, tc.Cursor)
		td := requireTestData(t, tc.Src, tc.Cursor, tc.OriginalLang, tc.LexerLang)

		cc, err := FindContext(td.SB, td.Tokens, td.Lexer)
		require.NoError(t, err)
		assert.Equal(t, tc.Prefix, cc.Prefix, "test case %d: %s for lang %s", i, tc.Desc, tc.LexerLang.Name())

		expectedTok := "NO TOKEN"
		if tc.LastTokenIdx > -1 {
			expectedTok = td.Tokens[tc.LastTokenIdx].Lit
		}
		actualTok := "NO TOKEN"
		if cc.LastTokenIdx > -1 {
			actualTok = td.Tokens[cc.LastTokenIdx].Lit
		}

		assert.Equal(t, tc.LastTokenIdx, cc.LastTokenIdx,
			"test case %d: %s for lang %s, expected token '%s' but got '%s'",
			i, tc.Desc, tc.LexerLang.Name(), expectedTok, actualTok,
		)
		if tc.LastTokenIdx >= 0 {
			assert.True(t, td.Tokens[tc.LastTokenIdx].End <= td.SB.End,
				"test case %d: %s for lang %s, last token %s before cursor pos %d",
				i, tc.Desc, tc.LexerLang.Name(), expectedTok, td.SB.End)
		}
	}
}

func Test_FindContext_Golang(t *testing.T) {
	tcs := []findContextTC{
		{
			Desc:         "cursor after space, no prefix",
			Src:          "package $",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Golang,
			Prefix:       "",
			LastTokenIdx: 0,
		},
		{
			Desc:         "cursor end of ident, prefix",
			Src:          "package$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Golang,
			Prefix:       "package",
			LastTokenIdx: -1,
		},
		{
			Desc:         "cursor end of number",
			Src:          "for i := 0$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Golang,
			Prefix:       "",
			LastTokenIdx: 3,
		},
		{
			Desc:         "cursor after dot",
			Src:          "defer f.$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Golang,
			Prefix:       "",
			LastTokenIdx: 2,
		},
	}

	assertFindContext(t, tcs)
}

func Test_FindContext_TextGolang(t *testing.T) {
	// TODO: maybe only allow keywords to merge with spaces that occur _after_ the keyword,
	// it would avoid the unintuitive behavior below. Basically the "merging" logic
	// we use in text/split.go requires looking back at the tokens after we process them
	// to merge things
	tcs := []findContextTC{
		{
			Desc:         "keyword space cursor -> no prefix",
			Src:          "package $",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "",
			LastTokenIdx: 0,
		},
		{
			Desc:         "keyword space cursor space -> no prefix",
			Src:          "package $ ",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "",
			LastTokenIdx: 0,
		},
		{
			Desc:         "test before space followed by keyword",
			Src:          "func (m *Manager) HandleLanguages(w$ )",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "w",
			LastTokenIdx: 8,
		},
		{
			Desc:         "test after space followed by keyword",
			Src:          "func (m *Manager) HandleLanguages(w $)",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "",
			LastTokenIdx: 10,
		},
		{
			Desc:         "test after space not followed by keyword",
			Src:          "func (m *Manager) HandleLanguages(w $",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "",
			LastTokenIdx: 10,
		},
		{
			Desc:         "test cursor after keyword",
			Src:          "for$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "for",
			LastTokenIdx: -1,
		},
		{
			Desc:         "test cursor after keyword and space",
			Src:          "for $ ",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			LastTokenIdx: 0,
		},
		{
			Desc:         "test cursor after partial keyword",
			Src:          "var resp []str$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "str",
			LastTokenIdx: 3,
		},
		{
			Desc:         "cursor end of keyword, prefix",
			Src:          "package$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "package",
			LastTokenIdx: -1,
		},
		{
			Desc:         "cursor end of ident, prefix",
			Src:          "m$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "m",
			LastTokenIdx: -1,
		},
		{
			Desc:         "cursor end of number",
			Src:          "for i := 0$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "0",
			LastTokenIdx: 3,
		},
		{
			Desc:         "cursor after dot",
			Src:          "defer f.$",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "",
			LastTokenIdx: 2,
		},
		{
			Desc:         "cursor inside keyword",
			Src:          "fo$r",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "fo",
			LastTokenIdx: -1,
		},
		{
			Desc:         "cursor inside brackets",
			Src:          "var m = map[$]bool{}",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "",
			LastTokenIdx: 5,
		},
		{
			Desc:         "cursor inside brackets",
			Src:          "data := make(map[$])",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Prefix:       "",
			LastTokenIdx: 6,
		},
	}

	assertFindContext(t, tcs)
}

func Test_FindContext_TextGolang_Spacing(t *testing.T) {
	src := `package permissions

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
)

func (m *Manager) HandleLanguages(w $)
`
	tcs := []findContextTC{
		{
			Desc:         "cursor after space, no prefix",
			Src:          src,
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			LastTokenIdx: 44,
		},
	}

	assertFindContext(t, tcs)
}

func Test_FindContext_TextGolang_BetweenBrackets(t *testing.T) {
	src := `package main
func main() {
	data := make(map[$])
}
`
	tcs := []findContextTC{
		{
			Desc:         "cursor in between brackets, no prefix",
			Src:          src,
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			LastTokenIdx: 16,
			Prefix:       "",
		},
	}
	assertFindContext(t, tcs)
}

func Test_FindContext_TextGolang_Return(t *testing.T) {
	src := `func ChainClientHooks(hooks ...*ClientHooks) *ClientHooks {
	if len(hooks) == 0 {
		return nil
	}
	if len(hooks) == 1 {
		return hooks[0]
	}

	return $
}`
	tcs := []findContextTC{
		{
			Desc:         "cursor after space, no prefix",
			Src:          src,
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			LastTokenIdx: 53,
		},
	}

	assertFindContext(t, tcs)
}

type precededBySpaceTC struct {
	Desc         string
	Src          string
	Cursor       string
	OriginalLang lang.Language
	LexerLang    lang.Language

	Expected bool
}

func assertPrecededBySpace(t *testing.T, tcs []precededBySpaceTC) {
	for i, tc := range tcs {
		require.NotEmpty(t, tc.Cursor)
		td := requireTestData(t, tc.Src, tc.Cursor, tc.OriginalLang, tc.LexerLang)

		actual := PrecededBySpace(td.SB, td.Tokens, tc.LexerLang)
		assert.Equal(t, tc.Expected, actual, "test case %d: %s for lang %s", i, tc.Desc, tc.LexerLang.Name())
	}
}

func Test_PrecededBySpace_Golang(t *testing.T) {
	tcs := []precededBySpaceTC{
		{
			Desc:         "cursor after space of keyword",
			Src:          "package $",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Golang,
			Expected:     true,
		},
		{
			Desc:         "cursor after keyword",
			Src:          "package$ ",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Golang,
		},
	}

	assertPrecededBySpace(t, tcs)
}

func Test_PrecededBySpace_TextGolang(t *testing.T) {
	tcs := []precededBySpaceTC{
		{
			Desc:         "cursor after space of keyword",
			Src:          "package $",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
			Expected:     true,
		},
		{
			Desc:         "cursor after keyword",
			Src:          "package$ ",
			Cursor:       "$",
			OriginalLang: lang.Golang,
			LexerLang:    lang.Text,
		},
	}

	assertPrecededBySpace(t, tcs)
}

func Test_PrecededBySpace_Javascript(t *testing.T) {
	tcs := []precededBySpaceTC{
		{
			Desc:         "TODO: inside string after space, we do not count the space in the string",
			Src:          `"foo ^"`,
			Cursor:       "^",
			OriginalLang: lang.JavaScript,
			LexerLang:    lang.JavaScript,
			Expected:     false,
		},
		{
			Desc:         "inside string",
			Src:          `"foo^"`,
			Cursor:       "^",
			OriginalLang: lang.JavaScript,
			LexerLang:    lang.JavaScript,
			Expected:     false,
		},
	}

	assertPrecededBySpace(t, tcs)
}

func Test_PrecededBySpace_TextJavascript(t *testing.T) {
	tcs := []precededBySpaceTC{
		{
			Desc:         "inside string after space",
			Src:          `"foo ^"`,
			Cursor:       "^",
			OriginalLang: lang.JavaScript,
			LexerLang:    lang.Text,
			Expected:     true,
		},
		{
			Desc:         "inside string",
			Src:          `"foo^"`,
			Cursor:       "^",
			OriginalLang: lang.JavaScript,
			LexerLang:    lang.Text,
			Expected:     false,
		},
	}

	assertPrecededBySpace(t, tcs)
}
