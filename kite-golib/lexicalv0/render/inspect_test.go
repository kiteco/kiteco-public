package render

import (
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireTestCase(t *testing.T, tc string) (src string, expected int, cursor int) {
	parts := strings.Split(tc, "$")
	require.Len(t, parts, 3, "test case should have exactly 3 parts, got %d", len(parts))

	src = strings.Join(parts, "")

	log.Println(len(parts[0]), len(parts[1]))
	cursor = len(parts[0]) + len(parts[1])

	expected = len(parts[0])

	return src, expected, cursor
}

func assertStartOfLine(t *testing.T, tc string) {
	src, expected, cursor := requireTestCase(t, tc)

	actual := findStartOfLine([]byte(src), cursor)

	log.Printf("\n%s\n", src)
	log.Println("cursor", cursor)
	log.Println("actual", actual)

	assert.Equal(t, expected, actual)
}

func Test_StartOfLine_EmptyIndentedEOF(t *testing.T) {
	src := "class foo():\n  $$"

	assertStartOfLine(t, src)
}

func Test_StartOfLine_Empty(t *testing.T) {
	src := "class foo():\n$$"

	assertStartOfLine(t, src)

}

func Test_StartOfLine_NotEmpty(t *testing.T) {
	src := "class foo():\n\t$bar= $"
	assertStartOfLine(t, src)
}

func Test_StartOfFirstLine(t *testing.T) {
	src := "$class foo$"
	assertStartOfLine(t, src)
}
