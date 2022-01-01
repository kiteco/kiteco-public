package codewrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapPython_NoWrap(t *testing.T) {
	src := "a = b+c"
	out := WrapPython(src, Options{
		Columns:  40,
		TabWidth: 4,
	})

	assert.Equal(t, src, out)
}

func TestWrapPython_OneLine(t *testing.T) {
	src := "myLongVariableName = someVeryLongNumber * someOtherNumber"
	out := WrapPython(src, Options{
		Columns:  40,
		TabWidth: 4,
	})

	expected := "myLongVariableName = \\\n    someVeryLongNumber*someOtherNumber"

	assert.Equal(t, expected, out)
}

func TestWrapPython_OneLineNoContinuation(t *testing.T) {
	src := "myLongVariableName = (foo, someOtherVeryLongNumber)"
	out := WrapPython(src, Options{
		Columns:  40,
		TabWidth: 4,
	})

	expected := "myLongVariableName = (foo,\n    someOtherVeryLongNumber)"

	assert.Equal(t, expected, out)
}

func TestWrapPython_MultiLineine(t *testing.T) {
	src := `
if foo:
    bar
`
	out := WrapPython(src, Options{
		Columns:  40,
		TabWidth: 4,
	})

	expected := `
if foo:
    bar
`
	t.Log(expected)
	t.Log(out)
	assert.Equal(t, expected, out)
}
