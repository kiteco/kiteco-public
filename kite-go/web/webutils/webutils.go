package webutils

import (
	"fmt"
	"strconv"

	"github.com/kiteco/kiteco/kite-golib/stringindex"
	"github.com/kiteco/kiteco/kite-golib/syntaxcolors"
)

// ColorizeCode renders syntax highlighted code snippet.
func ColorizeCode(code []byte) string {
	return string(syntaxcolors.Colorize(code, -1))
}

// ParseByteOrRuneOffset parses an offset that be provided as a query
// parameter either in terms of bytes or runes (e.g. selection_begin_bytes and
// selection_begin_runes). If both are non-empty then an error is returned. If
// the runes variant is provided then it is converted to a byte offset.
func ParseByteOrRuneOffset(content []byte, byteOffs, runeOffs string) (int, error) {
	switch {
	case byteOffs != "" && runeOffs != "":
		return 0, fmt.Errorf("byte and rune offsets cannot both be provided")
	case byteOffs != "":
		return strconv.Atoi(byteOffs)
	case runeOffs != "":
		idx, err := strconv.Atoi(runeOffs)
		if err != nil {
			return 0, err
		}
		byteOffset := stringindex.NewConverter(string(content)).BytesFromRunes(idx)
		if byteOffset < idx {
			// BytesFromRunes() returns the length of the string in bytes if idx is out of bounds
			// this must be true for valid offsets: byte offset >= rune offset
			return 0, fmt.Errorf("runnes index out of bounds")
		}
		return byteOffset, nil
	default:
		return 0, fmt.Errorf("neither bytes nor runes were provided")
	}
}

// OffsetToBytes converts either a byte offset or rune offset to its byte
// offset equivalent. It has the same semantics as ParseByteOrRuneOffset
// above.
func OffsetToBytes(content []byte, byteOffs, runeOffs int) (int, error) {
	switch {
	case byteOffs != 0 && runeOffs != 0:
		return 0, fmt.Errorf("byte and rune offsets cannot both be provided")
	case byteOffs != 0:
		if byteOffs < 0 {
			return 0, fmt.Errorf("byte offset cannot be negative")
		}
		return byteOffs, nil
	case runeOffs != 0:
		if runeOffs < 0 {
			return 0, fmt.Errorf("rune offset cannot be negative")
		}
		return stringindex.NewConverter(string(content)).BytesFromRunes(runeOffs), nil
	default:
		return 0, nil
	}
}

// ParseOffsetToUTF8 converts a string unicode offset to its UTF8 byte offset equivalent.
func ParseOffsetToUTF8(content []byte, offsetStr string, encodingStr string) (int, error) {
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, err
	}
	return stringindex.NewConverter(string(content)).OffsetToUTF8(offset, stringindex.GetOffsetEncoding(encodingStr))
}

// OffsetToUTF8 converts an unicode offset to its UTF8 byte offset equivalent.
func OffsetToUTF8(content []byte, offset int, e stringindex.OffsetEncoding) (int, error) {
	if offset < 0 {
		return 0, fmt.Errorf("unicode offset cannot be negative")
	}
	return stringindex.NewConverter(string(content)).OffsetToUTF8(offset, e)
}
