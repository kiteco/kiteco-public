package linenumber

import (
	"sort"
	"unicode/utf16"
)

// UTF16Map converts utf16 code-unit offsets to and from (line number, column) pairs.
// Code-unit offsets, line numbers, and columns are all zero-based.
type UTF16Map struct {
	CodeUnitCount int   // CodeUnitCount is the number of utf16 code-units in the buffer
	LineOffsets   []int // LineOffsets contains the utf16 code-unit offset of the first char of each line
}

// NewUTF16Map creates a map for the given string.
func NewUTF16Map(s string) *UTF16Map {
	runes := []rune(s)
	codeUnits := utf16.Encode(runes)
	m := UTF16Map{
		CodeUnitCount: len(codeUnits),
		LineOffsets:   []int{0},
	}
	for i, c := range codeUnits {
		if c == '\n' {
			m.LineOffsets = append(m.LineOffsets, i+1)
		}
	}
	return &m
}

// Offset converts a line number and column offset (both zero-based) to a code-unit offset.
func (m *UTF16Map) Offset(line, column int) int {
	return m.LineOffsets[line] + column
}

// LineCol converts a code-unit offset to a line number and column offset (both zero based).
// If offset is the position of a newline character then its line number will be number of
// LineOffsets that come before it and its column will be the CodeUnitCount of the line that it ends.
func (m *UTF16Map) LineCol(offset int) (line, column int) {
	line = sort.Search(len(m.LineOffsets)-1, func(i int) bool { return offset < m.LineOffsets[i+1] })
	return line, offset - m.LineOffsets[line]
}

// Column gets the zero-based column for a code-unit offset
func (m *UTF16Map) Column(offset int) int {
	_, col := m.LineCol(offset)
	return col
}

// Line gets the zero-based line number for a code-unit offset
func (m *UTF16Map) Line(offset int) int {
	line, _ := m.LineCol(offset)
	return line
}

// LineBounds gets the begin and end of the given line number, such that codeUnits[begin:end] will
// contain the complete contents of the line without any newline characters.
func (m *UTF16Map) LineBounds(line int) (begin, end int) {
	begin = m.LineOffsets[line]
	end = m.CodeUnitCount
	if line+1 < len(m.LineOffsets) {
		end = m.LineOffsets[line+1] - 1
	}
	return
}

// LineCount gets the number of lines (equal to the number of newline characters plus one)
func (m *UTF16Map) LineCount() int {
	return len(m.LineOffsets)
}
