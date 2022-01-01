package lexer

import (
	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/css"
	"github.com/kiteco/go-tree-sitter/golang"
	"github.com/kiteco/go-tree-sitter/html"
	"github.com/kiteco/go-tree-sitter/javascript"
	"github.com/kiteco/go-tree-sitter/python"
	"github.com/kiteco/go-tree-sitter/vue"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
)

type symbolRemap struct {
	originalSymbol int
	symbolName     string
	symbolOffset   int
}

var (
	// symbolRemapping holds any global treesitter symbol remappings we need to make.
	// this remapping strictly remaps to a *new* symbol - so this increases the total
	// number of lexical symbols.
	//
	// THE ORDER HERE MATTERS - e.g each of these tokens is added to the end of the
	// SymbolCount tokens that already exist for a language, in the order listed here.
	//
	// See the init method in this file to see how symbol ids are assigned for remappings
	symbolRemapping = []*symbolRemap{
		&symbolRemap{65535, "KITE_ILLEGAL", -1}, // treesitter returns max uint16-1 for errors, we just map to internal illegal token
	}
	remappings = map[int]*symbolRemap{}
)

func init() {
	for idx, remap := range symbolRemapping {
		remap.symbolOffset = idx
		remappings[remap.originalSymbol] = remap
	}
}

// TreeSitterLexer is a lexer backed by tree-sitter
type TreeSitterLexer struct {
	lang *sitter.Language
	// number of symbols in the language (can be different than lang.SymbolCount() for
	// multi-language containers, e.g. vue).
	symCount       int
	tokenExtractor func([]byte, *sitter.Parser, *sitter.Tree) ([]treesitter.Token, error)
}

// NewTreeSitterLexer returns a lexer for the provided language
func NewTreeSitterLexer(l lang.Language, symCount int, tokenExtractor func([]byte, *sitter.Parser, *sitter.Tree) ([]treesitter.Token, error)) (*TreeSitterLexer, error) {
	tsl, err := tsLang(l)
	if err != nil {
		return nil, err
	}
	return &TreeSitterLexer{
		lang:           tsl,
		symCount:       symCount,
		tokenExtractor: tokenExtractor,
	}, nil
}

// Lex implements lexer
func (t *TreeSitterLexer) Lex(buf []byte) ([]Token, error) {
	tokens, err := treesitter.Lex(buf, t.lang, t.tokenExtractor)
	if err != nil {
		return nil, err
	}

	var ret []Token
	for _, tok := range tokens {
		if remap, ok := remappings[tok.Symbol]; ok {
			tok.Symbol = t.symCount + remap.symbolOffset
		}
		ret = append(ret, Token{
			Token: tok.Symbol,
			Lit:   tok.Lit,
			Start: int(tok.Start),
			End:   int(tok.End),
		})
	}

	return ret, nil
}

// NumTokens implements Lexer
func (t *TreeSitterLexer) NumTokens() int {
	return t.symCount + len(symbolRemapping)
}

// Tokens implements Lexer
func (t *TreeSitterLexer) Tokens() []Token {
	var toks []Token
	for i := 0; i < t.symCount; i++ {
		toks = append(toks, Token{
			Token: i,
			Lit:   t.lang.SymbolName(sitter.Symbol(i)),
		})
	}
	for _, remap := range symbolRemapping {
		toks = append(toks, Token{
			Token: t.symCount + remap.symbolOffset,
			Lit:   remap.symbolName,
		})

	}
	return toks
}

// TokenName implements Lexer
func (t *TreeSitterLexer) TokenName(tok int) string {
	if tok >= t.symCount {
		for _, remap := range symbolRemapping {
			if tok == t.symCount+remap.symbolOffset {
				return remap.symbolName
			}
		}
	}
	return t.lang.SymbolName(sitter.Symbol(tok))
}

// --

func tsLang(l lang.Language) (*sitter.Language, error) {
	switch l {
	case lang.JavaScript:
		return javascript.GetLanguage(), nil
	case lang.Python:
		return python.GetLanguage(), nil
	case lang.Vue:
		return vue.GetLanguage(), nil
	case lang.CSS:
		return css.GetLanguage(), nil
	case lang.HTML:
		return html.GetLanguage(), nil
	case lang.Golang:
		return golang.GetLanguage(), nil
	}
	return nil, errors.Errorf("unsupported treesitter language: %s", l.Name())
}
