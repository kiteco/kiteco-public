package inline

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/ast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/internal/testast"
	"github.com/stretchr/testify/require"
)

func assertParse(t *testing.T, expected string, src string) {
	p, err := Parse("", []byte(src))
	require.NoError(t, err)
	require.NotNil(t, p)
	testast.AssertNode(t, expected, p.(*ast.Paragraph))
}

func TestEmpty(t *testing.T) {
	src := ""
	expected := `
Paragraph
`
	assertParse(t, expected, src)
}

func TestSimpleText(t *testing.T) {
	src := "a"
	expected := `
Paragraph
	Text[a]
`
	assertParse(t, expected, src)
}

func TestInlineItalics(t *testing.T) {
	src := "*a*"
	expected := `
Paragraph
	Inline[i "a"]
`
	assertParse(t, expected, src)
}

func TestInlineBold(t *testing.T) {
	src := "**a**"
	expected := `
Paragraph
	Inline[b "a"]
`
	assertParse(t, expected, src)
}

func TestInlineMonospace(t *testing.T) {
	src := "``a``"
	expected := `
Paragraph
	Inline[m "a"]
`
	assertParse(t, expected, src)
}

func TestInlineCode(t *testing.T) {
	src := "`a`"
	expected := `
Paragraph
	Inline[c "a"]
`
	assertParse(t, expected, src)
}

func TestInlineItalicsInside(t *testing.T) {
	src := "a *b* c"
	expected := `
Paragraph
	Text[a ]
	Inline[i "b"]
	Text[ c]
`
	assertParse(t, expected, src)
}

func TestInlineBoldInside(t *testing.T) {
	src := "a **b** c"
	expected := `
Paragraph
	Text[a ]
	Inline[b "b"]
	Text[ c]
`
	assertParse(t, expected, src)
}

func TestInlineMonospaceInside(t *testing.T) {
	src := "a ``b`` c"
	expected := `
Paragraph
	Text[a ]
	Inline[m "b"]
	Text[ c]
`
	assertParse(t, expected, src)
}

func TestInlineCodeInside(t *testing.T) {
	src := "a `b` c"
	expected := `
Paragraph
	Text[a ]
	Inline[c "b"]
	Text[ c]
`
	assertParse(t, expected, src)
}

func TestAtStart(t *testing.T) {
	src := "*a b* c"
	expected := `
Paragraph
	Inline[i "a b"]
	Text[ c]
`
	assertParse(t, expected, src)
}

func TestAtEnd(t *testing.T) {
	src := "ab `c d`"
	expected := `
Paragraph
	Text[ab ]
	Inline[c "c d"]
`
	assertParse(t, expected, src)
}

func TestNested(t *testing.T) {
	src := "`a b *c* d`"
	expected := `
Paragraph
	Inline[c "a b *c* d"]
`
	assertParse(t, expected, src)
}

func TestMismatchedMarkers(t *testing.T) {
	src := "`a*"
	expected := "Paragraph\n\tText[`a*]\n"
	assertParse(t, expected, src)
}

func TestMissingStartSeparator(t *testing.T) {
	src := "a*b*"
	expected := `
Paragraph
	Text[a*b*]
`
	assertParse(t, expected, src)
}

func TestMissingEndSeparator(t *testing.T) {
	src := "a *b*c"
	expected := `
Paragraph
	Text[a *b*c]
`
	assertParse(t, expected, src)
}

func TestNestedSameMarker(t *testing.T) {
	src := "a *b*c* d"
	expected := `
Paragraph
	Text[a ]
	Inline[i "b*c"]
	Text[ d]
`
	assertParse(t, expected, src)
}

func TestConsecutiveInline(t *testing.T) {
	src := "*a* `b` **c**"
	expected := `
Paragraph
	Inline[i "a"]
	Text[ ]
	Inline[c "b"]
	Text[ ]
	Inline[b "c"]
`
	assertParse(t, expected, src)
}

func TestMismatchMarkerCount(t *testing.T) {
	src := "**a*"
	expected := `
Paragraph
	Text[**a*]
`
	assertParse(t, expected, src)
}

func TestMismatchMarkerCount2(t *testing.T) {
	src := "*a**"
	expected := `
Paragraph
	Inline[i "a*"]
`
	assertParse(t, expected, src)
}

func TestEscapedStartSingle(t *testing.T) {
	src := "a \\*b* c"
	expected := `
Paragraph
	Text[a \*b* c]
`
	assertParse(t, expected, src)
}

func TestEscapedStartDouble(t *testing.T) {
	src := "a \\**b** c"
	expected := `
Paragraph
	Text[a \**b** c]
`
	assertParse(t, expected, src)
}

func TestEscapedStartDoubleB(t *testing.T) {
	src := "a *\\*b** c"
	expected := `
Paragraph
	Text[a ]
	Inline[i "\\*b*"]
	Text[ c]
`
	assertParse(t, expected, src)
}

func TestEscapedEndSingle(t *testing.T) {
	src := "a *b\\* c"
	expected := `
Paragraph
	Text[a *b\* c]
`
	assertParse(t, expected, src)
}

func TestEscapedEndDouble(t *testing.T) {
	src := "a **b\\** c"
	expected := `
Paragraph
	Text[a **b\** c]
`
	assertParse(t, expected, src)
}

func TestEscapedEndDoubleB(t *testing.T) {
	src := "a **b*\\* c"
	expected := `
Paragraph
	Text[a **b*\* c]
`
	assertParse(t, expected, src)
}

func TestPunctuationSeparator(t *testing.T) {
	src := "'*a*]"
	expected := `
Paragraph
	Text[']
	Inline[i "a"]
	Text[]]
`
	assertParse(t, expected, src)
}

func TestPunctuationWrapRule(t *testing.T) {
	src := "'*'*"
	expected := `
Paragraph
	Text['*'*]
`
	assertParse(t, expected, src)
}

func TestPunctuationWrapRule2(t *testing.T) {
	src := "<*>abc*"
	expected := `
Paragraph
	Text[<*>abc*]
`
	assertParse(t, expected, src)
}

func TestPunctuationWrapRule3(t *testing.T) {
	src := "(*)abc*"
	expected := `
Paragraph
	Text[(*)abc*]
`
	assertParse(t, expected, src)
}

func TestPunctuationWrapRule4(t *testing.T) {
	src := "[*]abc*"
	expected := `
Paragraph
	Text[[*]abc*]
`
	assertParse(t, expected, src)
}

func TestPunctuationWrapRule5(t *testing.T) {
	src := "{*}abc*"
	expected := `
Paragraph
	Text[{*}abc*]
`
	assertParse(t, expected, src)
}

func TestPunctuationWrapRule6(t *testing.T) {
	src := "\"*\"abc*"
	expected := `
Paragraph
	Text["*"abc*]
`
	assertParse(t, expected, src)
}

func TestPunctuationWrapRuleUnmatched(t *testing.T) {
	src := "<*)abc*"
	expected := `
Paragraph
	Text[<]
	Inline[i ")abc"]
`
	assertParse(t, expected, src)
}

func TestPunctuationSeparated(t *testing.T) {
	src := "*a*-(**b**)"
	expected := `
Paragraph
	Inline[i "a"]
	Text[-(]
	Inline[b "b"]
	Text[)]
`
	assertParse(t, expected, src)
}

func TestWhitespaceAfterStart(t *testing.T) {
	src := "a * b*"
	expected := `
Paragraph
	Text[a * b*]
`
	assertParse(t, expected, src)
}

func TestEscapedWhitespaceAfterStart(t *testing.T) {
	t.Skip("skipping until proper string escaping is implemented in the grammar")
	// TODO: I think the backslash is supposed to be removed here, to
	// leave only the whitespace.
	src := "a *\\ b*"
	expected := `
Paragraph
	Text[a ]
	Inline[i " b"]
`
	assertParse(t, expected, src)
}

func TestEmptyMarker(t *testing.T) {
	src := "**"
	expected := `
Paragraph
	Text[**]
`
	assertParse(t, expected, src)
}

func TestEmptyDoubleMarker(t *testing.T) {
	src := "****"
	expected := `
Paragraph
	Text[****]
`
	assertParse(t, expected, src)
}
