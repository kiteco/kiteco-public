package python

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireTestCase(t *testing.T, tc string) (string, int) {
	parts := strings.Split(tc, "$")
	require.Len(t, parts, 2, "test case must have exactly 2 parts, got %d", len(parts))

	return strings.Join(parts, ""), len(parts[0])
}

func assertIndentAndDepth(t *testing.T, tc string, expectedIndent string, expectedDepth int) {
	src, cursor := requireTestCase(t, tc)

	indent, depth, err := IndentInspect([]byte(src), cursor)
	require.NoError(t, err)

	assert.Equal(t, expectedIndent, indent)
	assert.Equal(t, expectedDepth, depth)
}

func TestIndentInspect_Basic(t *testing.T) {
	src := `
class ComplexNumber:
    def __init__(self,r: int = 0,i: int = 0, *vargs):
        self.real = r
        $self.img
`

	assertIndentAndDepth(t, src, "    ", 2)
}

func TestIndentInspect_Basic1(t *testing.T) {
	src := `
class ComplexNumber:
    def __init__(self,r: int = 0,i: int = 0, *vargs):
        self.real = r
        self.$img
`

	assertIndentAndDepth(t, src, "    ", 2)
}

func TestIndentInspect_Tab(t *testing.T) {
	src := "class ComplexNumber:\n" +
		"\tdef __init__(self,r: int = 0,i: int = 0, *vargs):\n" +
		"\t\tself.real = r\n" +
		"\t\t$self.img"

	assertIndentAndDepth(t, src, "\t", 2)
}

func TestIndentInspect_Tab1(t *testing.T) {
	src := "class ComplexNumber:\n" +
		"\tdef __init__(self,r: int = 0,i: int = 0, *vargs):\n" +
		"\t\tself.real = r\n" +
		"\t\tself.img$"

	assertIndentAndDepth(t, src, "\t", 2)
}

func TestIndentInspect_ErrorNode(t *testing.T) {
	src := `
import numpy as np

class ComplexNumber:
    def __init__(self,r: int = 0,i: int = 0, *vargs):
        self.real = r
        $self.
`

	assertIndentAndDepth(t, src, "    ", 2)
}

func TestIndentInspect_ErrorNode1(t *testing.T) {
	src := `
import numpy as np

class ComplexNumber:
    def __init__(self,r: int = 0,i: int = 0, *vargs):
        self.real = r
        self. $
`

	assertIndentAndDepth(t, src, "    ", 2)
}

func TestIndentInspect_Inconsistent(t *testing.T) {
	src := `
class ComplexNumber:
    def __init__(self,r: int = 0,i: int = 0, *vargs):
      self.real = r
      $self
`

	src, cursor := requireTestCase(t, src)

	_, _, err := IndentInspect([]byte(src), cursor)
	require.Error(t, err)
}

func TestIndentInspect_ZeroDepth(t *testing.T) {
	src := `
class ComplexNumber:
    def __init__(self,r: int = 0,i: int = 0, *vargs):
        self.real = r

$class
`

	assertIndentAndDepth(t, src, "    ", 0)
}

func TestIndentInspect_Windows(t *testing.T) {
	src := "class ComplexNumber:\r\n" +
		"    def __init__(self,r: int = 0,i: int = 0, *vargs):\r\n" +
		"        self.real = r\r\n" +
		"        $self.img"

	assertIndentAndDepth(t, src, "    ", 2)
}

func TestIndentInspect_Windows_2(t *testing.T) {
	src := "import numpy as np\r\n" +
		"\r\n" +
		"class ComplexNumber:\r\n" +
		"    def __init__(self,r: int = 0,i: int = 0, *vargs):\r\n" +
		"        self.real = r\r\n" +
		"        $self.img"

	assertIndentAndDepth(t, src, "    ", 2)
}

func TestIndentInspect_Window_3(t *testing.T) {
	src := "class ComplexNumber:\r\n" +
		"\tdef __init__(self,r: int = 0,i: int = 0, *vargs):\r\n" +
		"\t\tself.real = r\r\n" +
		"\r\n" +
		"$class"

	assertIndentAndDepth(t, src, "\t", 0)
}

func TestIndentInspect_Windows4(t *testing.T) {
	src := "class ComplexNumber:\r\n" +
		"    def __init__(self,r: int = 0,i: int = 0, *vargs):\r\n" +
		"        self.real = r\r\n" +
		"        self.img  $"

	assertIndentAndDepth(t, src, "    ", 2)
}
