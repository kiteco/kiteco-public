package stringindex

import (
	"unicode/utf8"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/unsafe"
)

// Converter converts between byte and rune offsets.
type Converter struct {
	Bytes []byte
}

// NewConverter creates a converter
func NewConverter(s string) Converter {
	bytes := unsafe.StringToBytes(s)
	return Converter{bytes}
}

// BytesFromRunes is deprecated. Use EncodeOffset(n, UTF32, UTF8).
func (c Converter) BytesFromRunes(nrunes int) int {
	var nbytes int
	for i := 0; i < nrunes; i++ {
		_, sz := utf8.DecodeRune(c.Bytes[nbytes:])
		if sz == 0 { // if there was invalid utf8 then just more forward one byte
			sz = 1
		}
		nbytes += sz
		if nbytes >= len(c.Bytes) {
			return len(c.Bytes)
		}
	}
	return nbytes
}

// RunesFromBytes is deprecated. Use EncodeOffset(n, UTF8, UTF32)..
func (c Converter) RunesFromBytes(n int) int {
	var nbytes, nrunes int
	for nbytes < n && nbytes < len(c.Bytes) {
		_, sz := utf8.DecodeRune(c.Bytes[nbytes:])
		if sz == 0 { // if there was invalid utf8 then just more forward one byte
			sz = 1
		}
		nbytes += sz
		nrunes++
	}
	return nrunes
}

// OffsetToUTF8 is deprecated. Use EncodeOffset(n, from, UTF8)
func (c Converter) OffsetToUTF8(offset int, from OffsetEncoding) (int, error) {
	return c.EncodeOffset(offset, from, UTF8)
}

// EncodeOffset changes the offset encoding
func (c Converter) EncodeOffset(offset int, from, to OffsetEncoding) (int, error) {
	if offset < 0 {
		return 0, errors.Errorf("negative offset")
	}

	var encoded int
	bytes := c.Bytes
	for offset > 0 {
		if len(bytes) == 0 {
			return encoded, errors.Errorf("offset overflows buffer")
		}

		r, n := utf8.DecodeRune(bytes)
		if r == utf8.RuneError {
			return encoded, errors.Errorf("invalid string")
		}
		bytes = bytes[n:]

		offset -= runeLen(r, from)
		encoded += runeLen(r, to)
	}
	if offset < 0 {
		return encoded, errors.Errorf("offset not aligned to code point boundary")
	}
	return encoded, nil
}

func runeLen(r rune, enc OffsetEncoding) int {
	switch enc {
	case UTF8:
		return utf8.RuneLen(r)
	case UTF16:
		if r > 0xFFFF {
			return 2
		}
		return 1
	case UTF32:
		return 1
	default:
		panic("unhandled encoding")
	}
}
