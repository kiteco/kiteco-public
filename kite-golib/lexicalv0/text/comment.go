package text

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// SingleLineCommentSymbols ...
func SingleLineCommentSymbols(l lang.Language) []string {
	switch l {
	case lang.JavaScript, lang.JSX, lang.Vue, lang.Golang, lang.Less,
		lang.TSX, lang.TypeScript, lang.CSS, lang.Kotlin, lang.Java, lang.Scala,
		lang.C, lang.Cpp, lang.ObjectiveC, lang.CSharp:
		return []string{"//"}
	case lang.Python, lang.Ruby, lang.Bash:
		return []string{"#"}
	case lang.PHP:
		return []string{"//", "#"}
	default:
		panic(fmt.Sprintf("unsupported line comment for language %s", l.Name()))
	}
}

func multiLineCommentSymbols(l lang.Language) (string, string) {
	switch l {
	case lang.JavaScript, lang.JSX, lang.Vue, lang.Golang, lang.Less,
		lang.TSX, lang.TypeScript, lang.CSS, lang.Kotlin, lang.Java, lang.Scala,
		lang.C, lang.Cpp, lang.ObjectiveC, lang.CSharp, lang.PHP:
		return "/*", "*/"
	case lang.HTML:
		return "<!--", "-->"
	case lang.Python:
		// TODO: other comment symbols
		return `'''`, `'''`
	case lang.Ruby:
		return "=begin", "=end"
	default:
		panic(fmt.Sprintf("unsupported multi line comment for language %s", l.Name()))
	}
}

func cursorLine(sb data.SelectedBuffer) (string, int) {
	var start int
	sb.RangeReverse(sb.Selection.Begin, func(i int, r rune) bool {
		if r == '\n' {
			start = i
			return false
		}
		return true
	})

	end := -1
	sb.Range(sb.Selection.Begin, func(i int, r rune) bool {
		if r == '\n' {
			end = i
			return false
		}
		return true
	})
	if end < 0 {
		// can happen if we don't have an ending newline
		end = sb.Buffer.Len()
	}

	line := sb.TextAt(data.Selection{Begin: start, End: end})

	// cursor is always to the "right" of the index,
	// so e.g line[pos-1] is the character immediately before
	// the cursor
	pos := sb.Selection.Begin - start

	return line, pos
}

// CursorInComment ...
func CursorInComment(sb data.SelectedBuffer, l lang.Language) bool {
	if l != lang.HTML {
		line, cursorPos := cursorLine(sb)
		for _, lineComment := range SingleLineCommentSymbols(l) {
			if idx := strings.Index(line, lineComment); idx != -1 {
				// cursor is always to the "right" of the index,
				// so e.g line[pos-1] is the character immediately before
				// the cusor, idx is an index into the line so we need
				// to add one to convert it to a cursor position
				return cursorPos >= idx+1
			}
		}
	}

	if l == lang.Bash {
		return false
	}

	startComment, endComment := multiLineCommentSymbols(l)

	beforeCursorCode := sb.Text()[:sb.Selection.Begin]
	startCommentCount := strings.Count(beforeCursorCode, startComment)
	if startCommentCount == 0 {
		return false
	}
	if startComment == endComment {
		return startCommentCount%2 != 0
	}

	pieces := strings.Split(beforeCursorCode, startComment)
	lastPiece := pieces[len(pieces)-1]
	return !strings.Contains(lastPiece, endComment)
}
