package diff

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	// DefaultGapSize is the default initial gap size.
	DefaultGapSize = 32
)

var (
	// ErrDeleteMismatch is an error corresponding to a mismatch between the
	// deleted bytes and the provided data.
	ErrDeleteMismatch = errors.New("deleted bytes do not match provided data")
)

// GapBuffer is a buffer that allows for efficient insertion/deletion
// around an offset. This works well for a stream of edits coming from
// an editor.
type GapBuffer struct {
	buf      []byte
	gapStart int
	gapEnd   int
}

// NewGapBuffer returns a new gap buffer. It copies the data from buf,
func NewGapBuffer(buf []byte) *GapBuffer {
	bufSize := len(buf) + DefaultGapSize
	gapBuf := make([]byte, bufSize)

	copy(gapBuf[DefaultGapSize:], buf)

	return &GapBuffer{
		buf:      gapBuf,
		gapStart: 0,
		gapEnd:   DefaultGapSize,
	}
}

// Bytes returns a copy of the underlying byte slice for the gap buffer.
func (g *GapBuffer) Bytes() []byte {
	ret := make([]byte, len(g.buf)-g.gapSize())
	n := copy(ret, g.buf[:g.gapStart])
	copy(ret[n:], g.buf[g.gapEnd:])
	return ret
}

// Insert takes data and inserts it at the given offset.
func (g *GapBuffer) Insert(offset int, data []byte) {
	g.ensureGapCapacity(len(data))
	g.setGapStart(offset)
	copy(g.buf[g.gapStart:], data)
	g.gapStart += len(data)
}

// Delete takes data and removes it, starting at the given offset. Optional
// verify boolean checks if the bytes removed matches the bytes provided in the
// data buffer.
func (g *GapBuffer) Delete(offset int, data []byte, verify bool) error {
	g.setGapStart(offset)

	var toDelete []byte
	if g.gapEnd+len(data) > len(g.buf) {
		toDelete = g.buf[g.gapEnd:]
	} else {
		toDelete = g.buf[g.gapEnd : g.gapEnd+len(data)]
	}

	if verify && !bytes.Equal(data, toDelete) {
		return ErrDeleteMismatch
	}

	g.gapEnd += len(toDelete)
	for idx := range toDelete {
		toDelete[idx] = 0
	}

	return nil
}

// --

func (g *GapBuffer) gapSize() int {
	return g.gapEnd - g.gapStart
}

// setGapStart is responsible for setting the gapStart to the provided offset.
func (g *GapBuffer) setGapStart(offset int) {
	if offset == g.gapStart {
		return
	}

	if g.gapStart == g.gapEnd {
		g.gapStart = offset
		g.gapEnd = offset
		return
	}

	newStart := offset
	newEnd := newStart + g.gapSize()
	if newEnd > len(g.buf) {
		newBuf := make([]byte, newEnd)
		copy(newBuf, g.buf)
		g.buf = newBuf
	}
	if newStart < g.gapStart {
		copy(g.buf[newEnd:g.gapEnd], g.buf[newStart:g.gapStart])
	} else {
		copy(g.buf[g.gapStart:newStart], g.buf[g.gapEnd:newEnd])
	}

	g.gapStart = newStart
	g.gapEnd = newEnd

	gap := g.buf[g.gapStart:g.gapEnd]
	for idx := range gap {
		gap[idx] = 0
	}
}

// ensureCapacity is responsible for managing the gap (i.e resizing if needed).
// It checks to make sure there is enough gap space to insert n bytes. If not, adjusts
// buffers accordingly.
func (g *GapBuffer) ensureGapCapacity(n int) {
	if n >= g.gapSize() {
		extraGap := g.gapSize()
		newStart := g.gapStart
		newEnd := newStart + n + extraGap

		buf := make([]byte, len(g.buf)+n)
		copy(buf, g.buf[:g.gapStart])
		copy(buf[newEnd:], g.buf[g.gapEnd:])

		g.buf = buf
		g.gapStart = newStart
		g.gapEnd = newEnd
	}
}

func (g *GapBuffer) debug(s string) {
	fmt.Printf("%s: len: %d, gs: %d, ge: %d, gsize: %d\n", s, len(g.buf), g.gapStart, g.gapEnd, g.gapSize())
}
