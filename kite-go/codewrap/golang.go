package codewrap

import (
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
)

func insertSpaceBeforeAndAfter(t token.Token) bool {
	if t.IsKeyword() {
		return true
	}

	switch t {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
		return false
	case token.AND, token.OR, token.SHL, token.SHR, token.AND_NOT:
		return false
	case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN:
		return true
	case token.AND_ASSIGN, token.OR_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN, token.AND_NOT_ASSIGN:
		return true
	case token.LAND, token.LOR, token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		return true
	case token.ARROW, token.ASSIGN, token.DEFINE:
		return true
	case token.INC, token.DEC, token.NOT:
		return false
	case token.COMMENT:
		return true
	}

	return false
}

func insertSpaceBefore(t token.Token) bool {
	return insertSpaceBeforeAndAfter(t)
}

func insertSpaceAfter(t, next token.Token) bool {
	switch {
	case insertSpaceBeforeAndAfter(t):
		return true
	case t == token.COMMA || t == token.SEMICOLON:
		return true
	case t == token.IDENT:
		return next == token.IDENT || next.IsKeyword() || next.IsLiteral()
	case t == token.RPAREN:
		return next == token.IDENT || next == token.MUL || isLeftBraceLike(next)
	default:
		return false
	}
}

func insertSpaceBetween(t1, t2 token.Token) bool {
	return insertSpaceBefore(t2) || insertSpaceAfter(t1, t2)
}

func isControlStructure(t token.Token) bool {
	switch t {
	case token.IF, token.FOR, token.SWITCH, token.SELECT, token.FUNC:
		return true
	default:
		return false
	}
}

// Return true if t is a brace, bracket, or parentheses
func isBraceLike(t token.Token) bool {
	return isLeftBraceLike(t) || isRightBraceLike(t)
}

// Return true if t is a left brace, bracket, or parentheses
func isLeftBraceLike(t token.Token) bool {
	return t == token.LBRACE || t == token.LBRACK || t == token.LPAREN
}

// Return true if t is a right brace, bracket, or parentheses
func isRightBraceLike(t token.Token) bool {
	return t == token.RBRACE || t == token.RBRACK || t == token.RPAREN
}

// Get the cost to split after the specified token, or -1 if it's illegal to split after this token
func splitCost(t token.Token, next token.Token) float64 {
	switch t {
	case token.SEMICOLON:
		return 2.
	case token.COMMA:
		return 3.
	case token.LBRACE, token.LBRACK, token.LPAREN:
		if isBraceLike(next) {
			return -1.
		}
		return 5.
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
		return 10.
	case token.AND, token.OR, token.SHL, token.SHR, token.AND_NOT:
		return 4.
	case token.ARROW:
		return 8.
	case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN:
		return 4.
	case token.AND_ASSIGN, token.OR_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN, token.AND_NOT_ASSIGN:
		return 4.
	case token.EQL, token.LSS, token.GTR, token.ASSIGN:
		return 3.
	case token.LAND, token.LOR:
		return 2.
	}
	return -1.
}

// Token represents a lexical token together with its position and the original
// characters that generated the token.
type golangToken struct {
	pos token.Pos
	id  token.Token
	lit string
}

// SplitCost gets the cost to split after the specified token, or -1 if it's illegal to split
// after this token
func (t *golangToken) SplitCost(next Token) (float64, string) {
	return splitCost(t.id, next.(*golangToken).id), ""
}

// InsertSpace decides whether a space should be inserted between this token and the next
func (t *golangToken) InsertSpace(next Token) bool {
	return insertSpaceBetween(t.id, next.(*golangToken).id)
}

// IsComment determines whether this token is a comment
func (t *golangToken) IsComment() bool {
	return t.id == token.COMMENT
}

// String gets the contents of this token
func (t *golangToken) String() string {
	if t.lit == "" {
		return t.id.String()
	}
	return t.lit
}

// Repr produces a debug string containing the token's ID and characters.
func (t *golangToken) Repr() string {
	return fmt.Sprintf("<id=%s lit=%s>", escaped(t.id.String()), escaped(t.lit))
}

// TokenizeGolang tokenizes a buffer containing golang code.
func TokenizeGolang(buf []byte) (*TokenizedBuffer, error) {
	var lexerrors []string
	handleError := func(pos token.Position, msg string) {
		lexerrors = append(lexerrors, fmt.Sprintf("%s: %s\n", pos, msg))
	}

	// Setup the lexer
	fset := token.NewFileSet()
	file := fset.AddFile("src.go", fset.Base(), len(buf))
	var lexer scanner.Scanner
	lexer.Init(file, buf, handleError, scanner.ScanComments)

	// Count lines
	lines := strings.Split(string(buf), "\n")

	out := TokenizedBuffer{
		Lines:  lines,
		Tokens: make([][]Token, len(lines)),
	}

	// Read as an array of lines, where each line is an array of tokens
	for {
		pos, id, lit := lexer.Scan()
		if id == token.EOF {
			break
		}

		// Trim auto-inserted semicolons
		if lit == "\n" && id == token.SEMICOLON {
			continue
		}

		k := file.Position(pos).Line - 1 // Position.Line is 1-based
		if k >= len(lines) {
			return nil, fmt.Errorf("line %d out of range", k)
		}
		out.Tokens[k] = append(out.Tokens[k], &golangToken{pos, id, lit})
	}

	// If there were lexer errors then construct an error
	var err error
	if len(lexerrors) > 0 {
		err = errors.New(strings.Join(lexerrors, "\n"))
	}

	return &out, err
}

// WrapGolang wraps golang code.
func WrapGolang(code string, opts Options) string {
	tokens, err := TokenizeGolang([]byte(code))
	if err != nil {
		return code
	}
	flow := Layout(tokens, opts)
	return strings.Join(flow.Lines, "\n")
}
