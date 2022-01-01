package linenumber

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUTF16Map(t *testing.T) {
	for _, s := range []string{"abc", "abc\ndef", "", "\n", "a\na\na\n"} {
		numlines := strings.Count(s, "\n") + 1
		t.Logf("%q", s)
		m := NewUTF16Map(s)
		assert.Equal(t, len(s), m.CodeUnitCount)
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

func TestUTF16Map_LineBounds(t *testing.T) {
	m := NewUTF16Map("...\n...\n...")
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

func TestUTF16Map_LineBounds_LastLine(t *testing.T) {
	m := NewUTF16Map("...\n...\n...\n")
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
	assert.Equal(t, m.CodeUnitCount, g)
	assert.Equal(t, m.CodeUnitCount, h)
}

func TestUTF16Map_Offset(t *testing.T) {
	// . = 1 utf16 (1 utf8) code-unit
	// 𠈔 = 2 utf16 (4 utf8) code-units https://www.fileformat.info/info/unicode/char/20214/index.htm
	// ᘄ = 1 utf16 (3 utf8) code-unit https://www.fileformat.info/info/unicode/char/1604/index.htm
	m := NewUTF16Map("...\n.ᘄ.\n...\n.𠈔.\n...")
	a := m.Offset(0, 1)
	assert.Equal(t, 1, a)
	b := m.Offset(2, 1)
	assert.Equal(t, 9, b)
	c := m.Offset(4, 0)
	assert.Equal(t, 17, c)
}
