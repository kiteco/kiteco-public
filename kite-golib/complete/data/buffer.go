package data

import (
	"bytes"
	"encoding/binary"
	"hash"
	"io"
	"unicode/utf8"

	"github.com/kiteco/kiteco/kite-golib/stringindex"
	"github.com/kiteco/kiteco/kite-golib/unsafe"
	"github.com/minio/highwayhash"
)

var hashKey [32]byte // use a zero key for all hashes for convenience

var cr = []byte("\r") // carriage return
func writeNoCR(w io.Writer, text string) error {
	bs := unsafe.StringToBytes(text)
	for idx := bytes.Index(bs, cr); len(bs) > 0; idx = bytes.Index(bs, cr) {
		if idx < 0 {
			idx = len(bs)
		}

		if _, err := w.Write(bs[:idx]); err != nil {
			return err
		}

		// chop off until the CR or end of buffer
		bs = bs[idx:]
		if len(bs) > 0 {
			// there must be a CR at the start
			bs = bs[len(cr):]
		}
	}
	return nil
}

// BufferHash can be used as a map key
type BufferHash struct {
	Low  uint64 `json:"low"`
	High uint64 `json:"high"`
}

func newBufferHash(h hash.Hash) BufferHash {
	var sum [16]byte
	h.Sum(sum[:0])
	return BufferHash{
		Low:  binary.LittleEndian.Uint64(sum[:8]),
		High: binary.LittleEndian.Uint64(sum[8:16]),
	}
}

// Hash64 returns a uint64
func (h BufferHash) Hash64() uint64 {
	// this is what spooky.Hash64 does
	return h.Low
}

// AddHashInfo combines the buffer hash with the hash of the string argument to produce a new Hash
// for example it is used for adding placeholder information to the completion hash to avoid collision
func (h BufferHash) AddHashInfo(s string) BufferHash {
	if s == "" {
		return h
	}
	hash, _ := highwayhash.New128(hashKey[:])
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(h.Low))
	hash.Write(b[:])
	binary.LittleEndian.PutUint64(b[:], uint64(h.High))
	hash.Write(b[:])
	writeNoCR(hash, s)

	return newBufferHash(hash)
}

// Buffer encapsulates represents the contents of a single Python file, as it is edited
type Buffer string

// NewBuffer allocates a new Buffer
func NewBuffer(text string) Buffer {
	return Buffer(text)
}

// Hash returns a BufferHash
func (b Buffer) Hash() BufferHash {
	hash, _ := highwayhash.New128(hashKey[:])
	writeNoCR(hash, string(b))
	return newBufferHash(hash)
}

// Len returns the length of the buffer in bytes
func (b Buffer) Len() int {
	return len(b)
}

// TextAt returns the text under a selection
func (b Buffer) TextAt(sel Selection) string {
	if sel.End < 0 {
		sel.End = len(b)
	}
	return string(b[sel.Begin:sel.End])
}

// Range iterates over the buffer string by runes starting at index `start`.
// It is analogous to a `for i, r := range s`. But when start > 0,
// i is an absolute byte offset, unlike doing `for i, r := range s[start:]`
func (b Buffer) Range(start int, cb func(int, rune) bool) bool {
	for i, r := range b[start:] {
		if !cb(start+i, r) {
			return false
		}
	}
	return true
}

// RangeReverse iterates backwards over the buffer string from the rune before index `start`,
// decrementing by the rune length until `cb` returns false.
// The callback should handle RuneError from RuneBefore if needed.
func (b Buffer) RangeReverse(start int, cb func(int, rune) bool) int {
	for start > 0 {
		r := b.RuneBefore(start)
		cont := cb(start, r)

		if r == utf8.RuneError {
			break
		}
		start -= utf8.RuneLen(r)

		if !cont {
			break
		}
	}
	return start
}

// RuneAt is similar to utf8.DecodeRune
func (b Buffer) RuneAt(off int) rune {
	r, _ := utf8.DecodeRuneInString(string(b[off:]))
	return r
}

// RuneBefore returns the last rune before offset.
func (b Buffer) RuneBefore(off int) rune {
	r, _ := utf8.DecodeLastRuneInString(string(b[:off]))
	return r
}

// Text returns the text string of the buffer
func (b Buffer) Text() string {
	return string(b)
}

// Replace replaces the given selection with the given text
func (b Buffer) Replace(sel Selection, text string) Buffer {
	return b[:sel.Begin] + Buffer(text) + b[sel.End:]
}

// ReplaceHash is a potentially optimized version of b.Replace(sel, text).Hash()
func (b Buffer) ReplaceHash(sel Selection, text string) BufferHash {
	return b.Replace(sel, text).Hash()
}

// Select returns a SelectedBuffer
func (b Buffer) Select(sel Selection) SelectedBuffer {
	return SelectedBuffer{b, sel}
}

// SelectedBuffer pairs a Buffer with a Selection
type SelectedBuffer struct {
	Buffer    `json:"text"`
	Selection `json:"selection"`
}

// SelectedBufferHash can be used as a map key
type SelectedBufferHash struct {
	BufferHash BufferHash
	Selection  Selection
}

// Hash returns a SelectedBufferHash
func (b SelectedBuffer) Hash() SelectedBufferHash {
	return SelectedBufferHash{b.Buffer.Hash(), b.Selection}
}

// Replace is equivalent to Buffer.Replace
func (b SelectedBuffer) Replace(text string) Buffer {
	return b.Buffer.Replace(b.Selection, text)
}

// ReplaceWithCursor is equivalent to Buffer.Replace, followed by moving the cursor to the end of the replacement text
func (b SelectedBuffer) ReplaceWithCursor(text string) SelectedBuffer {
	return b.Buffer.Replace(b.Selection, text).Select(Cursor(b.Selection.Begin + len(text)))
}

// Identity returns a "no-op"/"identity" completion at b
func (b SelectedBuffer) Identity() Completion {
	return Completion{
		Replace: b.Selection,
		Snippet: Snippet{Text: b.TextAt(b.Selection)},
	}
}

func (b SelectedBuffer) String() string {
	prefix := b.Buffer.TextAt(Selection{
		Begin: 0,
		End:   b.Begin,
	})
	sel := b.Buffer.TextAt(b.Selection)
	suffix := b.Buffer.TextAt(Selection{
		Begin: b.End,
		End:   b.Buffer.Len(),
	})
	return prefix + "⦉" + sel + "⦊" + suffix
}

// EncodeOffsets decodes the selection offsets according to the given encoding.
func (b *SelectedBuffer) EncodeOffsets(from, to stringindex.OffsetEncoding) error {
	return b.Selection.EncodeOffsets(b.Buffer.Text(), from, to)
}
