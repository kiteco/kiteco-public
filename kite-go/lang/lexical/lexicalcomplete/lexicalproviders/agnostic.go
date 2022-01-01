package lexicalproviders

import (
	"math"
	"strings"
	"unicode"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

func requireSingleToken(in Inputs) bool {
	var singleToken bool
	in.Range(in.Selection.End, func(i int, r rune) bool {
		if r == '\n' {
			// don't replace past a newline TODO(naman) loosen this constraint?
			return false
		}
		if unicode.IsSpace(r) {
			return true
		}
		switch r {
		case ')', ']', '}', '>', '"', '`', '\'':
			return true
		}

		// found a non-close-brace, non-whitespace character on the current line
		singleToken = true
		return false
	})
	return singleToken
}

// OverlapSize returns the maximal length of a prefix that matches the buffer.
func OverlapSize(given data.SelectedBuffer, completion string) int {
	begin := given.Selection.Begin - len(completion)
	if begin < 0 {
		begin = 0
	}
	lookBack := data.Selection{
		Begin: begin,
		End:   given.Selection.Begin,
	}
	lowerLeft := strings.ToLower(given.Buffer.TextAt(lookBack))
	lowerCompletion := strings.ToLower(completion)
	for i := len(lowerCompletion); i > 0; i-- {
		if strings.HasSuffix(lowerLeft, lowerCompletion[:i]) {
			return i
		}
	}
	return 0
}

func containsType(pred []lexer.Token, tokenType lexer.TokenType, lex lexer.Lexer) bool {
	for _, token := range pred {
		if lex.IsType(tokenType, token) {
			return true
		}
	}
	return false
}

func computeValue(c data.Completion, tokens []lexer.Token) float64 {
	numNewLines := strings.Count(c.Snippet.Text, "\n")
	numPlaceholders := len(c.Snippet.Placeholders())

	value := len(tokens) - 2*numNewLines - numPlaceholders
	if value < 0 {
		return 0
	}
	return math.Sqrt(float64(value))
}
