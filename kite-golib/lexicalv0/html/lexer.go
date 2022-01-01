package html

import (
	"strings"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/html"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
)

const (
	terminalChar = "$"
)

// Lexer is an html lexer.
type Lexer struct {
	*lexer.TreeSitterLexer
	sitterLang *sitter.Language
}

// NewLexer returns a new html lexer.
func NewLexer() (*Lexer, error) {
	l := &Lexer{
		sitterLang: html.GetLanguage(),
	}
	ts, err := lexer.NewTreeSitterLexer(lang.HTML, int(l.sitterLang.SymbolCount()), l.extractTreeTokens)
	if err != nil {
		return nil, err
	}
	l.TreeSitterLexer = ts
	return l, nil
}

// Lang implements Lexer.
func (Lexer) Lang() lang.Language {
	return lang.HTML
}

// ShouldBPEEncode implements Lexer.
func (l Lexer) ShouldBPEEncode(tok lexer.Token) ([]string, bool) {
	// TODO: implement
	return nil, false
}

// MergeBPEEncoded implements Lexer.
func (Lexer) MergeBPEEncoded(in []string) []string {
	// TODO: implement
	return nil
}

// TrimTerminal implements lexer
func (Lexer) TrimTerminal(word string) string {
	return strings.TrimSuffix(word, terminalChar)
}

// IsIncompleteToken implements Lexer
func (Lexer) IsIncompleteToken(word string) bool {
	return !strings.HasSuffix(word, terminalChar)
}

// ContainsIdentOrKeyword returns true if a list of tokens contains idents or keywords
func (Lexer) ContainsIdentOrKeyword(toks []lexer.Token) bool {
	// TODO: implement...
	return false
}

// HasInvalidToken ...
func (Lexer) HasInvalidToken(tokens []lexer.Token) bool {
	for _, tok := range tokens {
		// Internal illegal token
		if tok.Lit == "KITE_ILLEGAL" {
			return true
		}
	}
	return false
}

// IsType returns whether a token is an Ident
func (Lexer) IsType(t lexer.TokenType, tok lexer.Token) bool {
	// TODO: implement...
	return false
}

// TokensInRanges returns the html tokens found in the specified ranges
// of the source input buf. It reuses the provided parser and sets its language
// and ranges to process only the html parts. If ranges is empty,
// it returns nil, nil. It does not close the parser when done - a caller
// should take care of this when it is no longer needed.
func (l *Lexer) TokensInRanges(parser *sitter.Parser, buf []byte, ranges []sitter.Range) (tokens []treesitter.Token, err error) {
	if len(ranges) == 0 {
		return nil, nil
	}
	parser.SetLanguage(l.sitterLang)
	parser.SetIncludedRanges(ranges)
	tree := parser.Parse(buf)
	defer tree.Close()
	return l.extractTreeTokens(buf, parser, tree)
}

func (l *Lexer) extractTreeTokens(buf []byte, parser *sitter.Parser, tree *sitter.Tree) (tokens []treesitter.Token, err error) {
	root := tree.RootNode()

	// extract all tokens from the parsed tree
	t := &tokenizer{
		buf:  buf,
		lang: l.sitterLang,
	}
	treesitter.Walk(t, root)
	return t.tokens, nil
}

type tokenizer struct {
	buf    []byte
	lang   *sitter.Language
	tokens []treesitter.Token
}

func (t *tokenizer) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		return nil
	}

	switch {
	case n.ChildCount() == 0:
		// a terminal token
		t.append(n)
	}
	return t
}

func (t *tokenizer) append(n *sitter.Node) treesitter.Token {
	sym := int(n.Symbol())
	return t.appendSym(sym, n.StartByte(), n.EndByte())
}

func (t *tokenizer) appendSym(sym int, start, end uint32) treesitter.Token {
	tok := treesitter.Token{
		Symbol:     sym,
		SymbolName: t.lang.SymbolName(sitter.Symbol(sym)),
		Start:      int(start),
		End:        int(end),
		Lit:        string(t.buf[start:end]),
	}
	t.tokens = append(t.tokens, tok)
	return tok
}
