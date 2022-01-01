package highlight

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers/p"
	"github.com/alecthomas/chroma/styles"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

// CursorAnchor is the id of the anchor tag near the cursor position
const CursorAnchor = "cursor"

// The CSS style of the highlighted cursor
const cursorStyle = "background-color: #ff7070;"

// Highlight returns some HTML representing the highlighted code, and also highlights the cursor at the given offset.
func Highlight(src string, cursor int64) (string, error) {
	lm := linenumber.NewMap([]byte(src))

	lineNo := lm.Line(int(cursor)) + 1
	columnNo := lm.Column(int(cursor)) + 1

	var lineRange [][2]int
	lineRange = append(lineRange, [2]int{lineNo, lineNo})

	code, err := highlightPythonCode(src, lineRange)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	for i, line := range strings.Split(code, "\n") {
		s := line
		if i == lineNo-1 {
			s = highlightLineWithCursor(line, columnNo)
		}
		buf.Write([]byte(s + "\n"))
	}
	return buf.String(), nil
}

// Returns some HTML representing the highlighted code.
func highlightPythonCode(src string, highlightLines [][2]int) (string, error) {
	var buf bytes.Buffer

	l := p.Python
	l = chroma.Coalesce(l)

	f := html.New(
		html.WithLineNumbers(),
		html.HighlightLines(highlightLines),
	)

	s := styles.Get("monokailight")
	if s == nil {
		s = styles.Fallback
	}

	it, err := l.Tokenise(nil, src)
	if err != nil {
		return "", err
	}
	err = f.Format(&buf, s, it)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Walk the HTML-formatted code line to find the column position indicated by columnNo and highlight it.
// To find the position:
// - Ignore HTML tags, treat escaped characters (e.g. &#xx;) as single characters.
// - Ignore the contents of the second HTML tag, which we assume is the line number.
func highlightLineWithCursor(line string, columnNo int) string {
	// If the column is the last character of the line, we add a space so we have something to highlight
	if columnNo == len(line)-1 {
		line += " "
	}

	const (
		normal int = iota
		inSpan
		inEscapedChar
	)
	state := normal
	var col int
	// Count the number of times we see opening or closing HTML tags. This is so that we don't count the line number.
	// We assume that the format is like this:
	// <span ...><span ...>{line number}</span>{code....}...
	// So after seeing three tags (two open and one close), we assume we're no longer in the line number part.
	var numTags int
	// true if we've written the opening <span> tag for the cursor position but not the closing one
	var inCursorSpan bool
	var buf bytes.Buffer

	for _, ch := range []rune(line) {
		switch state {
		case normal:
			if col == columnNo-1 && numTags >= 3 && !inCursorSpan && ch != '<' {
				buf.Write(beforeCursor())
				inCursorSpan = true
			} else if col == columnNo && inCursorSpan {
				buf.Write(afterCursor())
				inCursorSpan = false
			}
			if ch == '<' {
				state = inSpan
			} else if ch == '&' {
				state = inEscapedChar
			} else if numTags >= 3 {
				col++
			}
		case inSpan:
			if ch == '>' {
				state = normal
				numTags++
			}
		case inEscapedChar:
			if ch == ';' {
				state = normal
				col++
			}
		}
		buf.Write([]byte(string([]rune{ch})))
	}
	// If the cursor is at the very end of the line, write a space so we have something to highlight
	if col == columnNo-1 && !inCursorSpan {
		buf.Write(beforeCursor())
		buf.Write([]byte(" "))
		inCursorSpan = true
	}
	// Close the cursor tags if we haven't yet
	if inCursorSpan {
		buf.Write(afterCursor())
	}
	return buf.String()
}

func beforeCursor() []byte {
	return []byte(fmt.Sprintf(
		"<a id=\"%s\"><span style=\"%s\">", CursorAnchor, cursorStyle))
}

func afterCursor() []byte {
	return []byte("</span></a>")
}
