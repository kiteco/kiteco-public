package codewrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapGolang_NoWrap(t *testing.T) {
	src := "a := b+c"
	out := WrapGolang(src, Options{
		Columns:  40,
		TabWidth: 4,
	})

	assert.Equal(t, src, out)
}

func TestWrapGolang_OneLine(t *testing.T) {
	src := "myLongVariableName := someVeryLongNumber * someOtherNumber"
	out := WrapGolang(src, Options{
		Columns:  40,
		TabWidth: 4,
	})

	expected := "myLongVariableName := someVeryLongNumber*\n    someOtherNumber"

	assert.Equal(t, expected, out)
}
