package stringindex

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverter_Trivial(t *testing.T) {
	c := NewConverter("abc")
	for i := 0; i <= 3; i++ {
		assert.Equal(t, i, c.RunesFromBytes(i))
		assert.Equal(t, i, c.BytesFromRunes(i))
	}
}

func TestConverter_Nontrivial(t *testing.T) {
	c := NewConverter("a世界c")
	assert.Equal(t, 0, c.RunesFromBytes(0))
	assert.Equal(t, 1, c.RunesFromBytes(1))
	assert.Equal(t, 2, c.RunesFromBytes(2))
	assert.Equal(t, 2, c.RunesFromBytes(3))
	assert.Equal(t, 2, c.RunesFromBytes(4))
	assert.Equal(t, 3, c.RunesFromBytes(5))
	assert.Equal(t, 3, c.RunesFromBytes(6))
	assert.Equal(t, 3, c.RunesFromBytes(7))
	assert.Equal(t, 4, c.RunesFromBytes(8))

	assert.Equal(t, 0, c.BytesFromRunes(0))
	assert.Equal(t, 1, c.BytesFromRunes(1))
	assert.Equal(t, 4, c.BytesFromRunes(2))
	assert.Equal(t, 7, c.BytesFromRunes(3))
	assert.Equal(t, 8, c.BytesFromRunes(4))
}

func TestConverter_UTF32(t *testing.T) {
	// 1, 2, 3, and 4 utf-8 bytes, respectively
	c := NewConverter("$£ई𠜎")
	assertErrors(t, c, -1, UTF32)
	assertConverts(t, c, 0, UTF32, 0)
	assertConverts(t, c, 1, UTF32, 1)
	assertConverts(t, c, 2, UTF32, 3)
	assertConverts(t, c, 3, UTF32, 6)
	assertConverts(t, c, 4, UTF32, 10)
	assertErrors(t, c, 5, UTF32)

	// make sure empty string is handled as UTF32
	var enc OffsetEncoding
	require.NoError(t, json.Unmarshal([]byte(`""`), &enc))
	assertConverts(t, c, 4, enc, 10)
}

func TestConverter_UTF16(t *testing.T) {
	// 1, 1, 1, and 2 utf-16 code units, respectively
	// 1, 2, 3, and 4 utf-8 bytes, respectively
	c := NewConverter("$£ई𠜎")
	assertErrors(t, c, -1, UTF16)
	assertConverts(t, c, 0, UTF16, 0)
	assertConverts(t, c, 1, UTF16, 1)
	assertConverts(t, c, 2, UTF16, 3)
	assertConverts(t, c, 3, UTF16, 6)
	assertErrors(t, c, 4, UTF16)
	assertConverts(t, c, 5, UTF16, 10)
	assertErrors(t, c, 6, UTF16)
}

func TestConverter_UTF8(t *testing.T) {
	// 1, 2, 3, and 4 utf-8 bytes, respectively
	c := NewConverter("$£ई𠜎")
	assertErrors(t, c, -1, UTF8)
	assertConverts(t, c, 0, UTF8, 0)
	assertConverts(t, c, 1, UTF8, 1)
	assertErrors(t, c, 2, UTF8)
	assertConverts(t, c, 3, UTF8, 3)
	assertErrors(t, c, 4, UTF8)
	assertErrors(t, c, 5, UTF8)
	assertConverts(t, c, 6, UTF8, 6)
	assertErrors(t, c, 7, UTF8)
	assertErrors(t, c, 8, UTF8)
	assertErrors(t, c, 9, UTF8)
	assertConverts(t, c, 10, UTF8, 10)
	assertErrors(t, c, 11, UTF8)
}

func assertErrors(t *testing.T, c Converter, inp int, e OffsetEncoding) {
	_, err := c.OffsetToUTF8(inp, e)
	assert.Error(t, err)
}

func assertConverts(t *testing.T, c Converter, encoded int, e OffsetEncoding, decoded int) {
	decodedActual, err := c.EncodeOffset(encoded, e, UTF8)
	assert.NoError(t, err)
	assert.Equal(t, decoded, decodedActual)
	encodedActual, err := c.EncodeOffset(decoded, UTF8, e)
	assert.NoError(t, err)
	assert.Equal(t, encoded, encodedActual)
}
