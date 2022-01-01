package text

import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

const maxLitLen = 160

// Lexer ...
type Lexer struct{}

// NewLexer ...
func NewLexer() Lexer {
	return Lexer{}
}

// Lang ...
func (Lexer) Lang() lang.Language {
	return lang.Text
}

// Lex ...
func (Lexer) Lex(buf []byte) ([]lexer.Token, error) {
	words := SplitWithOpts(string(buf), true)
	toks := make([]lexer.Token, 0, len(words)+1) // +1 for eof

	var offset int
	for _, w := range words {
		end := offset + len(w)
		tok := lexer.Token{
			Token: int(text),
			Lit:   w,
			Start: offset,
			End:   end,
		}
		if len(w) >= maxLitLen {
			tok.Lit = ""
			tok.Token = int(unknownToken)
		}

		toks = append(toks, tok)
		offset = end
	}

	toks = append(toks, lexer.Token{
		Token: int(eof),
		Start: offset,
		End:   offset,
	})

	return toks, nil
}

// NumTokens ...
func (Lexer) NumTokens() int {
	return len(allTokens)
}

// Tokens ...
func (Lexer) Tokens() []lexer.Token {
	var tokens []lexer.Token
	for _, tok := range allTokens {
		tokens = append(tokens, lexer.Token{
			Token: int(tok),
			Lit:   tok.String(),
		})
	}
	return tokens
}

// TokenName ...
func (Lexer) TokenName(t int) string {
	return token(t).String()
}

// ShouldBPEEncode ...
func (Lexer) ShouldBPEEncode(t lexer.Token) ([]string, bool) {
	if shouldBPEEncode(t) {
		return []string{t.Lit + lexer.TerminalChar}, true
	}
	return nil, false
}

func shouldBPEEncode(t lexer.Token) bool {
	return token(t.Token) == text && len(t.Lit) < maxLitLen
}

// MergeBPEEncoded ...
func (Lexer) MergeBPEEncoded(parts []string) []string {
	return lexer.MergeBPEEncoded(parts, lexer.TerminalChar)
}

// IsIncompleteToken ...
func (Lexer) IsIncompleteToken(t string) bool {
	return lexer.IsIncompleteToken(t, lexer.TerminalChar)
}

// ContainsIdentOrKeyword ...
func (Lexer) ContainsIdentOrKeyword(ts []lexer.Token) bool {
	for _, t := range ts {
		tt := strings.TrimSpace(t.Lit)
		if keywords[tt] {
			return true
		}

		if MaybeIdent(t.Lit) {
			return true
		}
	}
	return false
}

// HasInvalidToken ...
func (Lexer) HasInvalidToken(ts []lexer.Token) bool {
	for _, t := range ts {
		// everything is BPE encoded
		if t.Token != lexer.BPEEncodedTok {
			return true
		}
	}
	return false
}

// TrimTerminal ...
func (Lexer) TrimTerminal(t string) string {
	return lexer.TrimTerminal(t, lexer.TerminalChar)
}

// IsType ...
func (Lexer) IsType(tt lexer.TokenType, t lexer.Token) bool {
	switch tt {
	case lexer.IMPORT:
		return strings.Contains(t.Lit, "import")
	case lexer.IDENT:
		return MaybeIdent(t.Lit)
	case lexer.EOF:
		return t.Token == int(eof)
	case lexer.SEMICOLON:
		return strings.Contains(t.Lit, ";")
	default:
		return true
	}
}

// MaybeIdent ...
func MaybeIdent(s string) bool {
	for _, r := range s {
		if isIdentRune(r) {
			return true
		}
	}
	return false
}

func isIdentRune(r rune) bool {
	cc := charCategory(r)
	if isLetterOrNumber(cc) {
		return true
	}

	if r == '$' || r == '_' {
		return true
	}
	return false
}
