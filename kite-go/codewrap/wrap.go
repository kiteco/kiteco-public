package codewrap

// TODO:
//  - split string literals when necessary
//  - include comments in output

import (
	"fmt"
	"strings"
)

const (
	// ExtraColumnCost is the per-column penalty for failing to fit a code block
	// into the specified width.
	ExtraColumnCost = 15
)

// Token represents a lexical element
type Token interface {
	// SplitCost returns the cost to insert a newline after the given token, or -1 if a newline
	// must not be inserted here. It also returns the appropriate line continuation token, or
	// empty string if no line continuation token is necessary at this position.
	SplitCost(next Token) (float64, string)
	// InsertSpace returns whether a space should be inserted between this token and the next
	InsertSpace(next Token) bool
	// IsComment determines whether this token is a comment
	IsComment() bool
	// String returns the textual representation for this token that should appear in the output
	// stream
	String() string
	// Repr returns a string repreentation for this token appropriate for debugging
	Repr() string
}

// Options represents parameters for wrapping
type Options struct {
	Columns  int // maximum length of any line
	TabWidth int // number of space characters per indentation level
}

// TokenizedBuffer represents a code block that has been divided into lines, and
// then each line has been processed into a sequence of zero or more tokens.
type TokenizedBuffer struct {
	Lines  []string
	Tokens [][]Token
}

// Flow represents a buffer that has been reformatted to fit within a specified
// column limit.
type Flow struct {
	Lines   []string
	Tokens  *TokenizedBuffer
	LineMap []int // LineMap has length equal to Lines and has values that are indices into the input lines
}

// Layout lays out all tokens using no more than the specified number of columns, breaking lines at
// syntactically valid positions chosen to minimize a cost function (see splitCost).
// The tabWidth parameter determines by how many columns each indent level differs from the
// previous one. The returned object contains a list of lines that can be printed to output,
// as well as a mapping back to the input line numbers.
func Layout(buf *TokenizedBuffer, opts Options) *Flow {
	out := Flow{
		Tokens: buf,
	}
	for i, tokens := range buf.Tokens {
		layout := layoutLine(tokens, buf.Lines[i], opts)
		out.Lines = append(out.Lines, layout...)
		for j := 0; j < len(layout); j++ {
			out.LineMap = append(out.LineMap, i)
		}
	}
	return &out
}

// PrintTokens prints all tokens, with annotation info (for debugging).
func PrintTokens(t *TokenizedBuffer) {
	// Print the tokens line-by-line
	for i, tokens := range t.Tokens {
		fmt.Println(t.Lines[i])
		fmt.Print(leadingIndentStr(t.Lines[i]))
		for _, t := range tokens {
			fmt.Print(t.Repr(), " ")
		}
		fmt.Print("\n\n")
	}
}

// PrintSideBySide prints a re-formatted code block side-by-side with the the
// original code.
func PrintSideBySide(flow *Flow, opts Options) {
	// Print final output
	fmt.Println(strings.Repeat("=", 80))
	spacesForTab := strings.Repeat(" ", opts.TabWidth)
	for i, line := range flow.Lines {
		line = strings.Replace(line, "\t", spacesForTab, -1)
		lineWidth := width(line, opts.TabWidth)
		if lineWidth > opts.Columns {
			n := 60 - lineWidth
			var pad string
			if n >= 0 {
				pad = strings.Repeat(" ", n)
			}
			fmt.Printf("%s%s# overshoot by %d #", line, pad, lineWidth-opts.Columns)
		} else {
			fmt.Print(line + strings.Repeat(" ", opts.Columns-lineWidth) + "#")
		}
		if i == 0 || flow.LineMap[i] != flow.LineMap[i-1] {
			srcLine := flow.Tokens.Lines[flow.LineMap[i]]
			fmt.Print(strings.Replace(srcLine, "\t", spacesForTab, -1))
		}
		fmt.Print("\n")
	}
}
