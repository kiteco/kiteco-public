package epytext

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/internal/testast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/testparser"
	"github.com/stretchr/testify/require"
)

// for tests to be isolated from changes to the defaultMaxLines.
const testMaxLines = 10

func assertParse(t *testing.T, expected string, src string, opts ...Option) {
	assertParseWithError(t, expected, src, errors.Unknown, opts...)
}

func assertParseWithError(t *testing.T, expected string, src string, expectedReason errors.Reason, opts ...Option) {
	opts = append([]Option{MaxLines(testMaxLines)}, opts...)
	doc, err := Parse([]byte(src), opts...)

	if expectedReason == errors.Unknown {
		require.NoError(t, err)
	} else {
		require.Error(t, err)
		require.Equal(t, errors.ErrorReason(err), expectedReason)
	}
	require.NotNil(t, doc)

	testast.Assert(t, expected, doc)
}

func TestGeneratedParserUpToDate(t *testing.T) {
	testparser.ParserUpToDate(t, "internal/pigeon/parser.peg")
}

func TestSingleParagraph(t *testing.T) {
	src := `p`
	expected := `
Doc
	Paragraph
		Text[p]
`
	assertParse(t, expected, src)
}

func TestSingleSection(t *testing.T) {
	src := `Head
====`
	expected := `
Doc
	Section[=]
		Text[Head]
`
	assertParse(t, expected, src)
}

func TestSingleList(t *testing.T) {
	src := `- list`
	expected := `
Doc
	List[- (0)]
		Paragraph
			Text[list]
`
	assertParse(t, expected, src)
}

func TestSingleField(t *testing.T) {
	src := `@param p: desc`
	expected := `
Doc
	Field[param (p)]
		Paragraph
			Text[desc]
`
	assertParse(t, expected, src)
}

func TestSingleLiteral(t *testing.T) {
	src := `
p::
  lit`
	expected := `
Doc
	Paragraph
		Text[p:]
		Literal[  lit]
`
	assertParse(t, expected, src)
}

func TestSingleDoctest(t *testing.T) {
	src := `
>>> doctest
(result)

`
	expected := `
Doc
	Doctest[>>> doctest
(result)]
`
	assertParse(t, expected, src)
}

func TestMergeParagraph(t *testing.T) {
	src := `
  - p1
  p2

  p3
p4`
	expected := `
Doc
	List[- (0)]
		Paragraph
			Text[p1
p2]
		Paragraph
			Text[p3]
	Paragraph
		Text[p4]
`
	assertParse(t, expected, src)
}

func TestGenerateParagraph(t *testing.T) {
	src := `
	@field: p1

	p2
p3
`
	expected := `
Doc
	Field[field ()]
		Paragraph
			Text[p1]
		Paragraph
			Text[p2]
	Paragraph
		Text[p3]
`
	assertParse(t, expected, src)
}

func TestTooManyLines(t *testing.T) {
	src := `p1

	p2

	p3


p4`
	expected := `
Doc
	Paragraph
		Text[p1]
		Paragraph
			Text[p2]
`
	assertParseWithError(t, expected, src, errors.TooManyLines, MaxLines(3))
}

func TestOptimizedResult(t *testing.T) {
	src := `a B{I{}} b`
	expected := `
Doc
	Paragraph
		Text[a  b]
`
	assertParse(t, expected, src)
}
