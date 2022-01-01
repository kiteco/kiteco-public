package lexer

import (
	"C"
)
import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
)

const (
	// SubtokenChar marks the end of subtoken
	SubtokenChar = "+"
	// TerminalChar marks the end of work
	TerminalChar = "$"
)

const (
	// BPEEncodedTok represents a token that was BPE encoded. The meaning of this is language specific
	BPEEncodedTok = -1
	// SepTok is a separator token. We put this here to make it consistent with BPEEncodedToken
	SepTok = -2
	// SepTokStr ...
	SepTokStr = "kite-septoken184-SEP"
)

// TokenType represents a single or group of tokens that represent a category,
// e.g strings, literals, or a semicolon. This isn't a super crisp abstraction,
// but allows for language-agnostic logic that depends on these categories/types.
type TokenType int

// A few TokenTypes...
const (
	IDENT TokenType = iota
	STRING
	COMMENT
	LITERAL
	SEMICOLON
	EOF
	KEYWORD
	IMPORT
)

// Token ...
type Token struct {
	Token int
	// Lit is the literal contents of the token in the original buffer (may be empty for keywords or tokens like auto semicolon)
	Lit string
	// Start position of the token in the original buffer (0 based index)
	Start int
	// End position of the token in the original buffer (0 based index)
	End int
}

// Lexer ...
type Lexer interface {
	Lang() lang.Language
	Lex(buf []byte) ([]Token, error)
	NumTokens() int
	Tokens() []Token
	TokenName(int) string
	ShouldBPEEncode(Token) ([]string, bool)
	MergeBPEEncoded([]string) []string
	IsIncompleteToken(string) bool
	ContainsIdentOrKeyword([]Token) bool
	HasInvalidToken([]Token) bool
	TrimTerminal(string) string
	IsType(TokenType, Token) bool
}

// MergeBPEEncoded ...
func MergeBPEEncoded(parts []string, terminalChar string) []string {
	var joined []string
	var pending []string
	for i, s := range parts {
		pending = append(pending, s)
		if strings.HasSuffix(s, terminalChar) || i == len(parts)-1 {
			joined = append(joined, strings.TrimSuffix(strings.Join(pending, ""), terminalChar))
			pending = nil
		}
	}
	return joined
}

// IsIncompleteToken ...
func IsIncompleteToken(word string, terminalChar string) bool {
	return !strings.HasSuffix(word, terminalChar)
}

// TrimTerminal ...
func TrimTerminal(word string, terminalChar string) string {
	return strings.TrimSuffix(word, terminalChar)
}
