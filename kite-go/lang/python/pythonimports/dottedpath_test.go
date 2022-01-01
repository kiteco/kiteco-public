package pythonimports

import (
	"testing"

	spooky "github.com/dgryski/go-spooky"
	"github.com/stretchr/testify/assert"
)

func TestDottedPath(t *testing.T) {
	p := NewDottedPath("abc.def.ghi")

	assert.Len(t, p.Parts, 3)
	assert.Equal(t, "abc", p.Parts[0])
	assert.Equal(t, "def", p.Parts[1])
	assert.Equal(t, "ghi", p.Parts[2])

	assert.EqualValues(t, spooky.Hash64([]byte("abc.def.ghi")), p.Hash)

	assert.Equal(t, "abc.def.ghi", p.String())

	assert.Equal(t, "abc", p.Head())
	assert.Equal(t, "ghi", p.Last())

	assert.True(t, p.Equals("abc.def.ghi"))
	assert.False(t, p.Equals(""))
	assert.False(t, p.Equals("."))
	assert.False(t, p.Equals("abc.def"))
	assert.False(t, p.Equals("abc.def.ghi."))
	assert.False(t, p.Equals("abc.def.ghi.jkl"))

	assert.False(t, p.Empty())
}

func TestDottedPath_Empty(t *testing.T) {
	p := NewDottedPath("")
	assert.Len(t, p.Parts, 0)
	assert.EqualValues(t, 0, p.Hash)
	assert.Equal(t, p.String(), "")

	assert.Equal(t, p.Head(), "")
	assert.Equal(t, p.Last(), "")

	assert.True(t, p.Equals(""))
	assert.False(t, p.Equals("a"))
	assert.False(t, p.Equals("."))

	assert.True(t, p.Empty())
}
