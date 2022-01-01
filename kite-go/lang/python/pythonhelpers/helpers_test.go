package pythonhelpers

import (
	"go/token"
	"log"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockDiff(offset int32, diffType event.DiffType, text string) *event.Diff {
	return &event.Diff{
		Type:   &diffType,
		Offset: &offset,
		Text:   &text,
	}
}

func parseAnnotation(text string) ([]*event.Diff, string) {
	var diffs []*event.Diff

	var content string
	var inAnnotation, seenDot bool
	var prevText, curText string
	var offset int32

	for _, c := range text {
		switch c {
		case '|':
			inAnnotation = !inAnnotation
			if inAnnotation {
				offset = int32(len(content))
			}
			if !inAnnotation {
				if len(prevText) > 0 {
					diffs = append(diffs, mockDiff(offset, event.DiffType_DELETE, prevText))
				}
				if len(curText) > 0 {
					diffs = append(diffs, mockDiff(offset, event.DiffType_INSERT, curText))
				}
				content += curText
				prevText = ""
				curText = ""
				seenDot = false
			}
		case '.':
			// consume ...
			if inAnnotation {
				if !seenDot {
					prevText = curText
					curText = ""
					seenDot = true
				}
				continue
			}
			content += string(c)
		default:
			if inAnnotation {
				curText += string(c)
			} else {
				content += string(c)
			}
		}
	}
	return diffs, content
}

// parseTestSnippet parses the begin cursor position, end cursor position,
// and runnable python source code from the provided test snippet and returns
// these values.
func parseTestSnippet(snippet string) (int, int, string) {
	parts := strings.Split(snippet, "$")
	switch len(parts) {
	case 1:
		// no cursor found so assign cursor to be at end of the snippet
		return len(snippet), len(snippet), snippet
	case 2:
		// single cursor position found
		return len(parts[0]), len(parts[0]), strings.Join(parts, "")
	case 3:
		// start and end cursor position found
		return len(parts[0]), len(parts[0]) + len(parts[1]), strings.Join(parts, "")
	default:
		log.Fatalln("invalid test code snippet:", snippet)
		return -1, -1, ""
	}
}

func parseWithCursor(t testing.TB, snippet string) (*pythonast.Module, token.Pos) {
	cursor, _, src := parseTestSnippet(snippet)
	mod, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err)
	return mod, token.Pos(cursor)
}

// - hover and autosearch use DeepestContainingSelection to find name & attribute expressions
//   so we test that use-case here

func doDeepestNameTest(t testing.TB, snip string, expected string) {
	start, end, src := parseTestSnippet(snip)
	mod, err := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{})
	require.NoError(t, err)

	node := DeepestContainingSelection(kitectx.Background(), mod, int64(start), int64(end))
	switch node := node.(type) {
	case *pythonast.NameExpr:
		require.Equal(t, expected, node.Ident.Literal)
	case *pythonast.AttributeExpr:
		require.Equal(t, expected, node.Attribute.Literal)
	default:
		if expected != "" {
			require.Fail(t, "deepest node not name or attribute expression")
		}
	}
}

func TestDeepestName(t *testing.T) {
	// basic
	doDeepestNameTest(t, `$x = foo.bar`, "x")
	doDeepestNameTest(t, `x$ = foo.bar`, "x")
	doDeepestNameTest(t, `x $= foo.bar`, "")
	doDeepestNameTest(t, `x =$ foo.bar`, "")
	doDeepestNameTest(t, `x = $foo.bar`, "foo")
	doDeepestNameTest(t, `x = f$oo.bar`, "foo")
	doDeepestNameTest(t, `x = fo$o.bar`, "foo")
	doDeepestNameTest(t, `x = foo$.bar`, "foo")
	doDeepestNameTest(t, `x = foo.$bar`, "bar")
	doDeepestNameTest(t, `x = foo.b$ar`, "bar")
	doDeepestNameTest(t, `x = foo.ba$r`, "bar")
	doDeepestNameTest(t, `x = foo.bar$`, "bar")

	// with selection instead of cursor
	doDeepestNameTest(t, `$x$ = foo.bar`, "x")
	doDeepestNameTest(t, `$x $= foo.bar`, "")
	doDeepestNameTest(t, `x = $fo$o.bar`, "foo")
	doDeepestNameTest(t, `x = f$oo$.bar`, "foo")
	doDeepestNameTest(t, `x = f$oo.b$ar`, "bar") // note the behavior here!

	// try a call expression for fun
	doDeepestNameTest(t, `x = foo$(bar(0))`, "foo")
	doDeepestNameTest(t, `x = foo($bar(0))`, "bar")
	doDeepestNameTest(t, `x = foo(bar($0))`, "")
}

// -

func TestUnderCursor(t *testing.T) {
	snippet := `x = $foo(bar(0))`
	mod, cursor := parseWithCursor(t, snippet)

	foo := mod.Body[0].(*pythonast.AssignStmt).Value
	require.True(t, UnderCursor(foo, int64(cursor)))
	bar := foo.(*pythonast.CallExpr).Args[0].Value
	require.False(t, UnderCursor(bar, int64(cursor)))

	snippet = `x = foo(bar(0)$)`
	mod, cursor = parseWithCursor(t, snippet)
	foo = mod.Body[0].(*pythonast.AssignStmt).Value
	require.True(t, UnderCursor(foo, int64(cursor)))
	bar = foo.(*pythonast.CallExpr).Args[0].Value
	require.True(t, UnderCursor(bar, int64(cursor)))
}

func TestBetweenCallParens(t *testing.T) {
	snippet := `x = foo.bar(123)$`
	mod, cursor := parseWithCursor(t, snippet)
	foo := mod.Body[0].(*pythonast.AssignStmt).Value.(*pythonast.CallExpr)
	require.False(t, CursorBetweenCallParens(foo, cursor))

	snippet = `x = foo.bar$(123)`
	mod, cursor = parseWithCursor(t, snippet)
	foo = mod.Body[0].(*pythonast.AssignStmt).Value.(*pythonast.CallExpr)
	require.False(t, CursorBetweenCallParens(foo, cursor))

	snippet = `x = foo.bar(1$23)`
	mod, cursor = parseWithCursor(t, snippet)
	foo = mod.Body[0].(*pythonast.AssignStmt).Value.(*pythonast.CallExpr)
	require.True(t, CursorBetweenCallParens(foo, cursor))

	snippet = `x = foo.bar(123$)`
	mod, cursor = parseWithCursor(t, snippet)
	foo = mod.Body[0].(*pythonast.AssignStmt).Value.(*pythonast.CallExpr)
	require.True(t, CursorBetweenCallParens(foo, cursor))
}

func TestParseAnnotation1(t *testing.T) {
	annotate := `
import os

os.path.joina()

|...(|
`
	exp := `
import os

os.path.joina()

(
`
	_, content := parseAnnotation(annotate)
	assert.Equal(t, exp, content)
}

func TestParseAnnotation2(t *testing.T) {
	annotate := `
import os

os.path.joina()

|(test...(|
`
	exp := `
import os

os.path.joina()

(
`
	diffs, content := parseAnnotation(annotate)
	assert.Equal(t, exp, content)
	require.Len(t, diffs, 2)
	assert.Equal(t, "DELETE", diffs[0].GetType().String())
	assert.EqualValues(t, len(exp)-2, diffs[0].GetOffset())
	assert.Equal(t, "(test", diffs[0].GetText())
}

func TestNearestNonWhitespace_Normal(t *testing.T) {
	buf := []byte("  a.b  \n  c.d  ")
	assert.EqualValues(t, 0, NearestNonWhitespace(buf, 0, IsHSpace))
	assert.EqualValues(t, 0, NearestNonWhitespace(buf, 1, IsHSpace))
	assert.EqualValues(t, 2, NearestNonWhitespace(buf, 2, IsHSpace))
	assert.EqualValues(t, 3, NearestNonWhitespace(buf, 3, IsHSpace))
	assert.EqualValues(t, 4, NearestNonWhitespace(buf, 4, IsHSpace))
	assert.EqualValues(t, 5, NearestNonWhitespace(buf, 5, IsHSpace))
	assert.EqualValues(t, 5, NearestNonWhitespace(buf, 6, IsHSpace))
	assert.EqualValues(t, 5, NearestNonWhitespace(buf, 7, IsHSpace))
	assert.EqualValues(t, 8, NearestNonWhitespace(buf, 8, IsHSpace))
	assert.EqualValues(t, 8, NearestNonWhitespace(buf, 9, IsHSpace))
	assert.EqualValues(t, 10, NearestNonWhitespace(buf, 10, IsHSpace))
	assert.EqualValues(t, 11, NearestNonWhitespace(buf, 11, IsHSpace))
	assert.EqualValues(t, 12, NearestNonWhitespace(buf, 12, IsHSpace))
	assert.EqualValues(t, 13, NearestNonWhitespace(buf, 13, IsHSpace))
	assert.EqualValues(t, 13, NearestNonWhitespace(buf, 14, IsHSpace))
	assert.EqualValues(t, 13, NearestNonWhitespace(buf, 15, IsHSpace))

	// unfortunately go checks if a slice limit is out of bounds based
	// on the capacity of the slice not on the length,
	// and go allocates string slices to the nearest power of 2.
	assert.Equal(t, int64(cap(buf)+1), NearestNonWhitespace(buf, int64(cap(buf)+1), IsHSpace))
}

func TestNearestNonWhitespace_Empty(t *testing.T) {
	assert.EqualValues(t, 0, NearestNonWhitespace(nil, 0, IsHSpace))

	buf := []byte(" ")
	assert.EqualValues(t, 0, NearestNonWhitespace(buf, 0, IsHSpace))
	assert.EqualValues(t, 0, NearestNonWhitespace(buf, 1, IsHSpace))
}

func TestNearestNonWhitespace_Unicode(t *testing.T) {
	// the character below is 3 bytes long:
	buf := []byte("  世  ")
	assert.EqualValues(t, 0, NearestNonWhitespace(buf, 0, IsHSpace))
	assert.EqualValues(t, 0, NearestNonWhitespace(buf, 1, IsHSpace))
	assert.EqualValues(t, 2, NearestNonWhitespace(buf, 2, IsHSpace))
	// cursor=3 and cursor=4 are invalid
	assert.EqualValues(t, 5, NearestNonWhitespace(buf, 5, IsHSpace))
	assert.EqualValues(t, 5, NearestNonWhitespace(buf, 6, IsHSpace))
	assert.EqualValues(t, 5, NearestNonWhitespace(buf, 7, IsHSpace))
}
