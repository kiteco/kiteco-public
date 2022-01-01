package linenumber

import "sort"

// Map converts byte offsets to and from (line number, column) pairs.
// Byte offsets, line numbers, and columns are all zero-based. This struct
// is a replacement for token.File with the following differences:
//   - it uses zero-based offsets, line numbers, and column numbers
//   - it is immutable
//   - it is serializable via gob/json/etc
//   - it is not (and does not need to be) synchronized
type Map struct {
	ByteCount   int   // ByteCount is the number of bytes in the buffer
	LineOffsets []int // LineOffsets contains the byte offset of the first char of each line
}

// NewMap creates a map for the given buffer.
func NewMap(buf []byte) *Map {
	m := Map{
		ByteCount:   len(buf),
		LineOffsets: []int{0},
	}
	for i, c := range buf {
		if c == '\n' {
			m.LineOffsets = append(m.LineOffsets, i+1)
		}
	}
	return &m
}

// Offset converts a line number and column offset (both zero-based) to a byte offset.
func (m *Map) Offset(line, column int) int {
	return m.LineOffsets[line] + column
}

// LineCol converts a byte offset to a line number and column offset (both zero based).
// If offset is the position of a newline character then its line number will be number of
// LineOffsets that come before it and its column will be the ByteCount of the line that it ends.
func (m *Map) LineCol(offset int) (line, column int) {
	line = sort.Search(len(m.LineOffsets)-1, func(i int) bool { return offset < m.LineOffsets[i+1] })
	return line, offset - m.LineOffsets[line]
}

// Column gets the zero-based column for a byte offset
func (m *Map) Column(offset int) int {
	_, col := m.LineCol(offset)
	return col
}

// Line gets the zero-based line number for a byte offset
func (m *Map) Line(offset int) int {
	line, _ := m.LineCol(offset)
	return line
}

// LineBounds gets the begin and end of the given line number, such that buf[begin:end] will
// contain the complete contents of the line without any newline characters.
func (m *Map) LineBounds(line int) (begin, end int) {
	begin = m.LineOffsets[line]
	end = m.ByteCount
	if line+1 < len(m.LineOffsets) {
		end = m.LineOffsets[line+1] - 1
	}
	return
}

// LineCount gets the number of lines (equal to the number of newline characters plus one)
func (m *Map) LineCount() int {
	return len(m.LineOffsets)
}
