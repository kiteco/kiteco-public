package python

import (
	"bytes"
	"go/token"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireParse(t *testing.T, src string, cursor int64) pythonast.Node {
	cursorPos := token.Pos(cursor)
	mod, _ := pythonparser.Parse(kitectx.Background(), []byte(src), pythonparser.Options{
		Approximate: true,
		Cursor:      &cursorPos,
	})
	require.NotNil(t, mod)
	return mod
}

func requireCursorAndSource(t *testing.T, src string) (int64, string) {
	parts := strings.Split(src, "$")
	require.Len(t, parts, 2)

	return int64(len(parts[0])), strings.Join(parts, "")
}

func assertFindCall(t *testing.T, src string, outsideParens bool, start, end int) {
	cursor, src := requireCursorAndSource(t, src)
	ast := requireParse(t, src, cursor)

	t.Logf("Cursor: %v\n", cursor)

	var buf bytes.Buffer
	t.Logf("\nAST:\n")
	pythonast.PrintPositions(ast, &buf, "\t")
	t.Logf("\n%s\n", buf.String())

	call, aop, _ := FindCallExpr(kitectx.Background(), ast, []byte(src), cursor)
	require.NotNil(t, call)

	buf.Reset()
	t.Logf("\nFound call:\n")
	pythonast.PrintPositions(call, &buf, "\t")
	t.Logf("\n%s\n", buf.String())
	t.Log(call.LeftParen, call.LeftParen.Begin, call.LeftParen.End)

	assert.Equal(t, outsideParens, aop, "expected %v for outsideParens, got %v", outsideParens, aop)

	assert.Equal(t, token.Pos(start), call.Begin(), "expeceted %v for start but got %v", start, call.Begin())
	assert.Equal(t, token.Pos(end), call.End(), "expeceted %v for end but got %v", end, call.End())
}

func TestFindCallExpr_Simple(t *testing.T) {
	src := `foo($)`
	assertFindCall(t, src, false, 0, len(src)-1)

	src = "fo$o()"
	assertFindCall(t, src, true, 0, len(src)-1)
}

func TestFindCallExpr_SimpleIncomplete(t *testing.T) {
	src := `foo($`
	assertFindCall(t, src, false, 0, len(src)-1)

	src = `fo$o(`
	assertFindCall(t, src, true, 0, len(src)-1)

	src = "foo( \n$"
	assertFindCall(t, src, false, 0, len(src)-3)
}

func TestFindCallExpr_Nested(t *testing.T) {
	src := `foo(bar($))`
	assertFindCall(t, src, false, 4, 9)
}

func TestFindCallExpr_NestedIncomplete(t *testing.T) {
	src := `foo(bar($`
	assertFindCall(t, src, false, 4, len(src)-1)

	src = `foo(bar($)`
	assertFindCall(t, src, false, 4, len(src)-1)

	src = `foo(ba$r(`
	assertFindCall(t, src, false, 0, len(src)-1)

	src = `foo(bar()$`
	assertFindCall(t, src, false, 0, len(src)-1)

	src = `foo(bar(),$`
	assertFindCall(t, src, false, 0, len(src)-1)
}

func TestFindCallExpr_NestedArg(t *testing.T) {
	src := `foo(bar()$)`
	assertFindCall(t, src, false, 0, len(src)-1)

	src = `foo(ba$r())`
	assertFindCall(t, src, false, 0, len(src)-1)
}

func TestFindCallExpr_Chained(t *testing.T) {
	src := `foo(b$b).bar(aa)`
	assertFindCall(t, src, false, 0, 7)

	src = `foo(bb).bar(a$a)`
	assertFindCall(t, src, false, 0, len(src)-1)
}

func TestFindCallExpr_ChainedIncomplete(t *testing.T) {
	src := `foo(bb).bar($`
	assertFindCall(t, src, false, 0, len(src)-1)
}
