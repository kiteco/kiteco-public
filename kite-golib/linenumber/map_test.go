package linenumber

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	for _, s := range []string{"abc", "abc\ndef", "", "\n", "a\na\na\n"} {
		numlines := strings.Count(s, "\n") + 1
		t.Logf("%q", s)
		m := NewMap([]byte(s))
		assert.Equal(t, len(s), m.ByteCount)
		assert.Equal(t, numlines, m.LineCount())
		var curline, curcol int
		for i := 0; i <= len(s); i++ {
			assert.Equal(t, curline, m.Line(i))
			assert.Equal(t, curcol, m.Column(i))
			assert.Equal(t, i, m.Offset(curline, curcol))
			if i < len(s) && s[i] == '\n' {
				curline++
				curcol = 0
			} else {
				curcol++
			}
		}
	}
}

func TestMap_LineBounds(t *testing.T) {
	m := NewMap([]byte("...\n...\n..."))
	a, b := m.LineBounds(0)
	assert.Equal(t, 0, a)
	assert.Equal(t, 3, b)
	c, d := m.LineBounds(1)
	assert.Equal(t, 4, c)
	assert.Equal(t, 7, d)
	e, f := m.LineBounds(2)
	assert.Equal(t, 8, e)
	assert.Equal(t, 11, f)
}

func TestMap_LineBounds_LastLine(t *testing.T) {
	m := NewMap([]byte("...\n...\n...\n"))
	a, b := m.LineBounds(0)
	assert.Equal(t, 0, a)
	assert.Equal(t, 3, b)
	c, d := m.LineBounds(1)
	assert.Equal(t, 4, c)
	assert.Equal(t, 7, d)
	e, f := m.LineBounds(2)
	assert.Equal(t, 8, e)
	assert.Equal(t, 11, f)
	g, h := m.LineBounds(3)
	assert.Equal(t, m.ByteCount, g)
	assert.Equal(t, m.ByteCount, h)
}
