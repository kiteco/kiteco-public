package golang

import (
	"go/scanner"
	"go/token"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

const (
	terminalChar = "$"
)

var (
	parens = map[token.Token]bool{
		token.LPAREN: true,
		token.LBRACK: true,
		token.LBRACE: true,

		token.RPAREN: true,
		token.RBRACK: true,
		token.RBRACE: true,
	}
)

// Lexer is a golang lexer
type Lexer struct {
}

// Lang implements Lexer
func (Lexer) Lang() lang.Language {
	return lang.Golang
}

// Lex implements Lexer
func (Lexer) Lex(buf []byte) ([]lexer.Token, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(buf))

	var tokens []lexer.Token
	addToken := func(t token.Token, lit string, pos token.Pos) {
		// go uses 1 based indexing, we use 0 based
		start := int(pos - 1)
		end := start + len(lit)

		if t == token.SEMICOLON && lit == "\n" {
			// this is an automatically inserted semi colon,
			// so it does not appear in the buffer, so start == end
			end = start
		}

		tokens = append(tokens, lexer.Token{
			Token: int(t),
			Lit:   lit,
			Start: start,
			End:   end,
		})
	}

	var s scanner.Scanner
	s.Init(file, buf, nil, scanner.ScanComments)
	for {
		pos, t, lit := s.Scan()
		addToken(t, lit, pos)
		if t == token.EOF {
			break
		}
	}

	// Do some normalization

	return tokens, nil
}

// NumTokens implements Lexer
func (Lexer) NumTokens() int {
	return len(AllTokens)
}

// Tokens implements Lexer
func (Lexer) Tokens() []lexer.Token {
	var toks []lexer.Token
	for _, tok := range AllTokens {
		toks = append(toks, lexer.Token{
			Token: int(tok),
			Lit:   tok.String(),
		})
	}
	return toks
}

// IsIncompleteToken implements Lexer
func (Lexer) IsIncompleteToken(word string) bool {
	return !strings.HasSuffix(word, terminalChar)
}

// TrimTerminal implements lexer
func (Lexer) TrimTerminal(word string) string {
	return strings.TrimSuffix(word, terminalChar)
}

// TokenName implements Lexer
func (Lexer) TokenName(t int) string {
	return token.Token(t).String()
}

// ShouldBPEEncode implements Lexer
func (Lexer) ShouldBPEEncode(tok lexer.Token) ([]string, bool) {
	// Hack to filter comments - we say we want to BPE encode, but then return nothing
	if tok.Token == int(token.COMMENT) {
		return nil, true
	}

	// Only need to encode idents, and we don't use subtokens, so just use the terminalChar
	if tok.Token == int(token.IDENT) {
		// NOTE(tarak): There are some crazy long idents, some of which cause BPE encoding
		// to hang. This was a quick hack to get around that issue.
		if len(tok.Lit) <= 80 {
			return []string{tok.Lit + terminalChar}, true
		}
		return nil, true
	}

	return nil, false
}

// MergeBPEEncoded implements Lexer
func (Lexer) MergeBPEEncoded(in []string) []string {
	var idents []string
	var pending []string
	for i, s := range in {
		pending = append(pending, s)
		if strings.HasSuffix(s, terminalChar) || i == len(in)-1 {
			idents = append(idents, strings.TrimSuffix(strings.Join(pending, ""), terminalChar))
			pending = nil
		}
	}
	return idents
}

// ContainsIdentOrKeyword ...
func (Lexer) ContainsIdentOrKeyword(tokens []lexer.Token) bool {
	for _, tok := range tokens {
		if tok.Token == lexer.BPEEncodedTok || token.Token(tok.Token).IsKeyword() {
			return true
		}
	}
	return false
}

// HasInvalidToken ...
func (Lexer) HasInvalidToken(tokens []lexer.Token) bool {
	for _, tok := range tokens {
		if token.Token(tok.Token) == token.ILLEGAL {
			return true
		}
	}
	return false
}

// IsType returns whether a token is an of the given type
func (Lexer) IsType(t lexer.TokenType, tok lexer.Token) bool {
	switch t {
	case lexer.IDENT:
		return tok.Token == int(token.IDENT)
	case lexer.STRING:
		return tok.Token == int(token.STRING)
	case lexer.COMMENT:
		return tok.Token == int(token.COMMENT)
	case lexer.LITERAL:
		return token.Token(tok.Token).IsLiteral()
	case lexer.SEMICOLON:
		return tok.Token == int(token.SEMICOLON)
	case lexer.EOF:
		return tok.Token == int(token.EOF)
	case lexer.KEYWORD:
		return token.Token(tok.Token).IsKeyword()
	case lexer.IMPORT:
		return tok.Token == int(token.IMPORT)
	}
	return tok.Token == int(token.IDENT)
}
