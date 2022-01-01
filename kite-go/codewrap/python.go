package codewrap

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

// The line continuation token for python source
var pythonContinuation = " \\"

func insertSpaceBeforeAndAfterPy(t pythonscanner.Token) bool {
	if t.IsKeyword() {
		return true
	}

	switch t {
	case pythonscanner.Add,
		pythonscanner.Sub,
		pythonscanner.Mul,
		pythonscanner.Div,
		pythonscanner.Pct,
		pythonscanner.BitNot:
		return false
	case pythonscanner.BitAnd,
		pythonscanner.BitOr,
		pythonscanner.BitLshift,
		pythonscanner.BitRshift,
		pythonscanner.BitXor,
		pythonscanner.Lg:
		return false
	case pythonscanner.AddAssign,
		pythonscanner.SubAssign,
		pythonscanner.MulAssign,
		pythonscanner.DivAssign,
		pythonscanner.PctAssign:
		return true
	case pythonscanner.BitAndAssign,
		pythonscanner.BitOrAssign,
		pythonscanner.BitLshiftAssign,
		pythonscanner.BitRshiftAssign,
		pythonscanner.BitXorAssign:
		return true
	case pythonscanner.And,
		pythonscanner.Or,
		pythonscanner.Eq,
		pythonscanner.Lt,
		pythonscanner.Gt,
		pythonscanner.Ne,
		pythonscanner.Le,
		pythonscanner.Ge:
		return true
	case pythonscanner.Assign:
		return true
	case pythonscanner.Not:
		return false
	case pythonscanner.Comment, pythonscanner.Magic:
		return true
	}

	return false
}

func insertSpaceBeforePy(t pythonscanner.Token) bool {
	return insertSpaceBeforeAndAfterPy(t)
}

func insertSpaceAfterPy(t, next pythonscanner.Token) bool {
	switch {
	case insertSpaceBeforeAndAfterPy(t):
		return true
	case t == pythonscanner.Comma || t == pythonscanner.Semicolon || t == pythonscanner.Colon:
		return true
	case t == pythonscanner.Ident:
		return next == pythonscanner.Ident || next.IsKeyword() || next.IsLiteral()
	case t == pythonscanner.Rparen:
		return next == pythonscanner.Ident || next == pythonscanner.Mul || isLeftBraceLikePy(next)
	default:
		return false
	}
}

func insertSpaceBetweenPy(t1, t2 pythonscanner.Token) bool {
	return t2 != pythonscanner.Colon && (insertSpaceBeforePy(t2) || insertSpaceAfterPy(t1, t2))
}

func isControlStructurePy(t pythonscanner.Token) bool {
	switch t {
	case pythonscanner.If, pythonscanner.For, pythonscanner.While, pythonscanner.With, pythonscanner.Def:
		return true
	default:
		return false
	}
}

// Return true if t is a brace, bracket, or parentheses
func isBraceLikePy(t pythonscanner.Token) bool {
	return isLeftBraceLikePy(t) || isRightBraceLikePy(t)
}

// Return true if t is a left brace, bracket, or parentheses
func isLeftBraceLikePy(t pythonscanner.Token) bool {
	return t == pythonscanner.Lbrace || t == pythonscanner.Lbrack || t == pythonscanner.Lparen
}

// Return true if t is a right brace, bracket, or parentheses
func isRightBraceLikePy(t pythonscanner.Token) bool {
	return t == pythonscanner.Rbrace || t == pythonscanner.Rbrack || t == pythonscanner.Rparen
}

// Get the cost to split after the specified pythonscanner, or -1 if it's illegal to split after this pythonscanner
func splitCostPy(t pythonscanner.Token, next pythonscanner.Token) float64 {
	switch t {
	case pythonscanner.Semicolon:
		return 2.
	case pythonscanner.Comma:
		return 3.
	case pythonscanner.Lbrace, pythonscanner.Lbrack, pythonscanner.Lparen:
		if isBraceLikePy(next) {
			return -1.
		}
		return 5.
	case pythonscanner.Add, pythonscanner.Sub, pythonscanner.Mul, pythonscanner.Div, pythonscanner.Pct:
		return 10.
	case pythonscanner.BitAnd, pythonscanner.BitOr, pythonscanner.BitLshift, pythonscanner.BitRshift, pythonscanner.BitXor, pythonscanner.Lg:
		return 4.
	case pythonscanner.AddAssign, pythonscanner.SubAssign, pythonscanner.MulAssign, pythonscanner.DivAssign, pythonscanner.PctAssign:
		return 4.
	case pythonscanner.BitAndAssign, pythonscanner.BitOrAssign, pythonscanner.BitLshiftAssign, pythonscanner.BitRshiftAssign, pythonscanner.BitXorAssign:
		return 4.
	case pythonscanner.Eq, pythonscanner.Lt, pythonscanner.Gt, pythonscanner.Assign:
		return 3.
	case pythonscanner.And, pythonscanner.Or:
		return 2.
	}
	return -1.
}

// pythonToken represents a lexical token together with its position and the original
// characters that generated the pythonscanner.
type pythonToken struct {
	word         pythonscanner.Word
	insideParens bool // true if this token is inside any set of parens, brackets, or braces
}

// SplitCost gets the cost to split after the specified pythonscanner, or -1 if it's illegal to split
// after this pythonscanner
func (t *pythonToken) SplitCost(next Token) (float64, string) {
	cost := splitCostPy(t.word.Token, next.(*pythonToken).word.Token)
	if t.insideParens {
		return cost, ""
	}
	return cost, pythonContinuation
}

// InsertSpace decides whether a space should be inserted between this pythonscanner and the next
func (t *pythonToken) InsertSpace(next Token) bool {
	return insertSpaceBetweenPy(t.word.Token, next.(*pythonToken).word.Token)
}

// IsComment determines whether this pythonscanner is a comment
func (t *pythonToken) IsComment() bool {
	return t.word.Token == pythonscanner.Comment || t.word.Token == pythonscanner.Magic
}

// String gets the contents of this pythonscanner
func (t *pythonToken) String() string {
	if t.word.Literal == "" {
		return t.word.Token.String()
	}
	return t.word.Literal
}

// Repr produces a debug string containing the pythonscanner's Id and characters.
func (t *pythonToken) Repr() string {
	return fmt.Sprintf("<id=%s lit=%s>", escaped(t.word.Token.String()), escaped(t.word.Literal))
}

// TokenizePython tokenizes a buffer containing python code.
func TokenizePython(buf []byte) (*TokenizedBuffer, error) {
	f := pythonscanner.File(buf)

	// Setup the lexer
	opts := pythonscanner.Options{
		ScanComments: true,
	}
	lexer := pythonscanner.NewScanner(buf, opts)

	// Count lines
	lines := strings.Split(string(buf), "\n")

	out := TokenizedBuffer{
		Lines:  lines,
		Tokens: make([][]Token, len(lines)),
	}

	// Read as an array of lines, where each line is an array of tokens
	var numParens int // number of unclosed parens
	for {
		begin, end, id, lit := lexer.Scan()
		if id == pythonscanner.EOF {
			break
		}

		if isLeftBraceLikePy(id) {
			numParens++
		} else if isRightBraceLikePy(id) {
			numParens--
		}

		// We must add 1 to begin because the file expects 1-based byte offsets, but then
		// we must subtract 1 from the line number because it returns 1-based line numbers
		k := f.Position(begin+1).Line - 1
		if k >= len(lines) {
			return nil, fmt.Errorf("line %d out of range", k)
		}
		out.Tokens[k] = append(out.Tokens[k], &pythonToken{
			insideParens: numParens > 0,
			word: pythonscanner.Word{
				Begin:   begin,
				End:     end,
				Token:   id,
				Literal: lit,
			},
		})
	}

	return &out, lexer.Errs
}

// WrapPython wraps python code.
func WrapPython(code string, opts Options) string {
	if spaces := strings.Repeat(" ", detectTabWidth(code)); len(spaces) != 0 {
		code = strings.Replace(code, spaces, "\t", -1)
	}
	tokens, err := TokenizePython([]byte(code))
	if err != nil {
		return code
	}
	flow := Layout(tokens, opts)
	return strings.Join(flow.Lines, "\n")
}

// WrapPythonLines wraps python code that has been split into individual lines
// i.e. split by newlines
func WrapPythonLines(lines []string, opts Options) [][]string {
	spaces := strings.Repeat(" ", detectTabWidth(strings.Join(lines, "\n")))
	wrap := make([][]string, len(lines))
	for i, line := range lines {
		if len(spaces) != 0 {
			line = strings.Replace(line, spaces, "\t", -1)
		}
		tokens, err := TokenizePython([]byte(line))
		if err != nil {
			// tokenization error: qw cannot wrap this line so just return the in¯put
			wrap[i] = []string{line}
			continue
		}
		flow := Layout(tokens, opts)
		wrap[i] = flow.Lines
	}
	return wrap
}

// For files that use spaces for indents, this function detects the indent
// width by finding the smallest indentation (non-zero) in the file and
// reporting its length in spaces.
func detectTabWidth(code string) int {
	var width int
	var set bool
	for _, line := range strings.Split(code, "\n") {
		cur := 0
		for _, ch := range line {
			if ch == ' ' {
				cur++
			} else {
				break
			}
		}
		if cur == 0 {
			continue
		}
		if !set {
			width = cur
			set = true
		} else if cur < width {
			width = cur
		}
	}
	return width
}
