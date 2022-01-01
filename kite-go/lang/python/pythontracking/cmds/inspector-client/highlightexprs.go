package main

import (
	"bytes"
	"fmt"
	"html"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/internal/inspectorapi"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

const (
	resolvedClass    = "resolved"
	unresolvedClass  = "unresolved"
	selectedClass    = "selected"
	cursorClass      = "cursor"
	lineNumbersClass = "line-numbers"
)

// tokenToHighlight describes a token that should be highlighted, its position in the buffer, and the style that should
// be applied.
// Constraints:
// - tokens cannot have overlapping positions.
// - tokens cannot span newlines.
type tokenToHighlight struct {
	begin    int64
	end      int64
	cssClass string
	toolTip  string
}

func highlightExprs(messageID analyze.MessageID, resp *inspectorapi.EventDetail) string {
	var tokens []tokenToHighlight
	for _, expr := range resp.Exprs {
		toolTip := "unresolved"
		class := unresolvedClass
		if expr.ResolvesTo != "" {
			class = resolvedClass
			toolTip = fmt.Sprintf("resolves to: %s", expr.ResolvesTo)
		}
		if expr.Begin == resp.ExprDetail.Begin {
			class += " " + selectedClass
		}
		tokens = append(tokens, tokenToHighlight{
			begin:    expr.Begin,
			end:      expr.End,
			cssClass: class,
			toolTip:  toolTip,
		})
	}
	baseURL := eventDetailURL(messageID.URI, messageID.ID)
	return highlightCode(resp.Buffer, tokens, baseURL, resp.Cursor, true)
}

func highlightPlainCode(buffer string) string {
	return highlightCode(buffer, nil, "", -1, false)
}

func highlightCode(buffer string, tokens []tokenToHighlight, tokenBaseURL string, userCursor int64, lineLinks bool) string {
	// Make sure buffer always has a trailing newline
	if len(buffer) > 0 && buffer[len(buffer)-1] != '\n' {
		buffer += "\n"
	}

	var buf bytes.Buffer

	// First, render the line number
	buf.Write([]byte(fmt.Sprintf("<span class=\"%s\">", lineNumbersClass)))
	for i := 0; i < numLines(buffer); i++ {
		if lineLinks {
			buf.Write([]byte(fmt.Sprintf("<a href=\"#%s\">%d</a>\n", lineID(i+1), i+1)))
		} else {
			buf.Write([]byte(fmt.Sprintf("<span>%d</span>\n", i+1)))
		}
	}
	buf.Write([]byte("</span>"))

	// Then, render the code
	buf.Write([]byte("<code>"))

	var curToken int
	line := 1

	// Windows-style newlines (\r\n) need to be handled a little differently
	var inCRLF bool

	buf.Write(beforeLine(line, lineLinks))

	for pos, ch := range buffer {
		if inCRLF {
			inCRLF = false
			// If we're encountering an \n in a \r\n sequence, we know we don't have to do anything
			if ch == '\n' {
				continue
			}
		}

		cursor := int64(pos)
		if curToken < len(tokens) {
			token := tokens[curToken]
			if cursor == token.begin {
				buf.Write(beforeToken(token, tokenBaseURL))
			} else if cursor == token.end {
				buf.Write(afterToken())
				curToken++
			}
		}

		if ch == '\r' || ch == '\n' {
			if ch == '\r' {
				inCRLF = true
			}
			if cursor == userCursor {
				// If the cursor is at the end of the line, draw a blank space so we have something to highlight
				buf.Write(userCursorToHTML(' '))
			}
			buf.Write(afterLine())
			buf.Write([]byte("\n"))
			line++
			if pos != len(buffer)-1 {
				buf.Write(beforeLine(line, lineLinks))
			}
		} else {
			if cursor == userCursor {
				buf.Write(userCursorToHTML(ch))
			} else {
				buf.Write([]byte(charToHTML(ch)))
			}
		}
	}

	buf.Write([]byte("</code>"))

	return buf.String()
}

func beforeToken(token tokenToHighlight, baseURL string) []byte {
	url := fmt.Sprintf("%s&cursor=%d", baseURL, token.begin)
	toolTip := ""
	if token.toolTip != "" {
		toolTip = fmt.Sprintf(" data-toggle=\"tooltip\" title=\"%s\"", html.EscapeString(token.toolTip))
	}
	return []byte(fmt.Sprintf(
		"<a href=\"%s\" class=\"%s\"%s>", url, token.cssClass, toolTip))
}

func afterToken() []byte {
	return []byte("</a>")
}

func beforeLine(line int, lineLinks bool) []byte {
	if lineLinks {
		return []byte(fmt.Sprintf("<span id=\"%s\">", lineID(line)))
	}
	return []byte("<span>")
}

func afterLine() []byte {
	return []byte("</span>")
}

func userCursorToHTML(ch int32) []byte {
	return []byte(fmt.Sprintf("<span class=\"%s\">%s</span>", cursorClass, charToHTML(ch)))
}

func charToHTML(ch int32) string {
	return html.EscapeString(string([]rune{ch}))
}

func numLines(buffer string) int {
	var count int
	for _, ch := range buffer {
		if ch == '\n' {
			count++
		}
	}
	return count
}

func lineID(line int) string {
	return fmt.Sprintf("line-%d", line)
}
