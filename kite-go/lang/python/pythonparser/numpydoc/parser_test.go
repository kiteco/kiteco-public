package numpydoc

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/testparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/internal/testast"
	"github.com/stretchr/testify/require"
)

// for tests to be isolated from changes to the defaultMaxLines.
const testMaxLines = 20

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
	testast.AssertNode(t, expected, doc)
}

func TestGeneratedParserUpToDate(t *testing.T) {
	testparser.ParserUpToDate(t, "internal/pigeon/parser.peg")
}

func TestDocstringFiles(t *testing.T) {
	// for larger test cases, read the source from the testdata/*.docstring
	// files, and the expected AST result in the corresponding testdata/*.ast.
	const dir = "testdata"
	files, err := ioutil.ReadDir(dir)
	require.NoError(t, err)
	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if ext != ".docstring" {
			continue
		}
		t.Run(f.Name(), func(t *testing.T) {
			srcName := filepath.Join(dir, f.Name())
			expName := filepath.Join(dir, strings.TrimSuffix(f.Name(), ext)+".ast")

			src, err := ioutil.ReadFile(srcName)
			require.NoError(t, err)
			exp, err := ioutil.ReadFile(expName)
			require.NoError(t, err)

			assertParse(t, string(exp), string(src), MaxLines(1000))
		})
	}
}

func TestEmptyInputs(t *testing.T) {
	cases := []struct {
		src, expected string
	}{
		{"", "Doc"},
		{"\t", "Doc"},
		{"   ", "Doc"},
		{"   \t \f", "Doc"},
		{"\n", "Doc"},
		{"\n \t ", "Doc"},
		{"\n \r\n \t \r", "Doc"},
	}
	for _, c := range cases {
		assertParse(t, c.expected, c.src)
	}
}

func TestSingleLineParagraph(t *testing.T) {
	src := `abc`
	expected := `
Doc
	Paragraph
		Text[abc]`
	assertParse(t, expected, src)
}

func TestMultiLineParagraph(t *testing.T) {
	src := `a
		b
  c`
	expected := `
Doc
	Paragraph
		Text[a b c]`
	assertParse(t, expected, src)
}

func TestMultipleParagraphs(t *testing.T) {
	src := `a
		b
  c

d
	e
		f`
	expected := `
Doc
	Paragraph
		Text[a b c]
	Paragraph
		Text[d e f]`
	assertParse(t, expected, src)
}

func TestUnderlineSectionIndentMismatch(t *testing.T) {
	src := `
 abc
    ---`
	expected := `
Doc
	Paragraph
		Text[abc ---]
`
	assertParse(t, expected, src)
}

func TestUnderlineSectionLenMismatch(t *testing.T) {
	src := `
 abc
 ----`
	expected := `
Doc
	Paragraph
		Text[abc ----]
`
	assertParse(t, expected, src)
}

func TestUnderlineSection(t *testing.T) {
	src := `
 abc
 ---`
	expected := `
Doc
	Section[abc]
`
	assertParse(t, expected, src)
}

func TestMultipleSections(t *testing.T) {
	src := `
 s1
 --
 s2
 --
 s3
 --`
	expected := `
Doc
	Section[s1]
	Section[s2]
	Section[s3]
`
	assertParse(t, expected, src)
}

func TestUnderlineWithParagraph(t *testing.T) {
	src := `
 abc
 ---
 def`

	expected := `
Doc
	Section[abc]
		Paragraph
			Text[def]
`
	assertParse(t, expected, src)
}

func TestUnderlineWithParagraphs(t *testing.T) {
	src := `
 abc
 ---
 def

 g
  h
		i`

	expected := `
Doc
	Section[abc]
		Paragraph
			Text[def]
		Paragraph
			Text[g h i]
`
	assertParse(t, expected, src)
}

func TestTopLevelParagraphsWithUnderlineSections(t *testing.T) {
	src := `
paragraph
one

paragraph
  two

 s1
 --
 s1 paragraph

s2
--

  s2 paragraph`

	expected := `
Doc
	Paragraph
		Text[paragraph one]
	Paragraph
		Text[paragraph two]
	Section[s1]
		Paragraph
			Text[s1 paragraph]
	Section[s2]
		Paragraph
			Text[s2 paragraph]
`
	assertParse(t, expected, src)
}

func TestTopLevelDirective(t *testing.T) {
	src := `
.. deprecated:: 0.1.1
  paragraph
`
	expected := `
Doc
	Directive[deprecated]
		Paragraph
			Text[0.1.1 paragraph]
`
	assertParse(t, expected, src)
}

func TestMultipleDirectives(t *testing.T) {
	src := `
.. d1:: p1
.. d2:: p2
.. [d3] p3
`
	expected := `
Doc
	Directive[d1]
		Paragraph
			Text[p1]
	Directive[d2]
		Paragraph
			Text[p2]
	Directive[[d3]]
		Paragraph
			Text[p3]
`
	assertParse(t, expected, src)
}

func TestTopLevelDirectiveContentIndent(t *testing.T) {
	src := `
  .. [1] a
    b
      c
  d
`
	expected := `
Doc
	Directive[[1]]
		Paragraph
			Text[a b c]
	Paragraph
		Text[d]
`
	assertParse(t, expected, src)
}

func TestSectionWithDirective(t *testing.T) {
	src := `
s1
--
p1

.. d1:: a
    b
      c
p2
`
	expected := `
Doc
	Section[s1]
		Paragraph
			Text[p1]
		Directive[d1]
			Paragraph
				Text[a b c]
		Paragraph
			Text[p2]
`
	assertParse(t, expected, src)
}

func TestDoctestOutsideSection(t *testing.T) {
	// Outside an underlined section, it parses as a standard paragraph.
	src := `
>>> a
b
`
	expected := `
Doc
	Paragraph
		Text[>>> a b]
`
	assertParse(t, expected, src)
}

func TestDoctestInsideSectionNoBlankLine(t *testing.T) {
	// Without a leading blank line, parses as a standard paragraph.
	src := `
s1
--
>>> a
b
`
	expected := `
Doc
	Section[s1]
		Paragraph
			Text[>>> a b]
`
	assertParse(t, expected, src)
}

func TestDoctest(t *testing.T) {
	src := `
s1
--

>>> a
b
`
	expected := `
Doc
	Section[s1]
		Doctest[">>> a\nb"]
`
	assertParse(t, expected, src)
}

func TestMultipleDoctest(t *testing.T) {
	src := `
s1
--

>>> a
b

  >>> c
  d
e
`
	expected := `
Doc
	Section[s1]
		Doctest[">>> a\nb"]
		Doctest[">>> c\nd"]
		Paragraph
			Text[e]
`
	assertParse(t, expected, src)
}

func TestDefinition(t *testing.T) {
	src := `
Parameters
----------
x : int
`
	expected := `
Doc
	Section[Parameters]
		Definition
			Text[x]
			Text[int]
`
	assertParse(t, expected, src)
}

func TestDefinitionWithContent(t *testing.T) {
	src := `
Parameters
----------
x : int
  p1
`
	expected := `
Doc
	Section[Parameters]
		Definition
			Text[x]
			Text[int]
			Paragraph
				Text[p1]
`
	assertParse(t, expected, src)
}

func TestDefinitionContentIndent(t *testing.T) {
	src := `
Parameters
----------

x : int

  a
    b

  c
d
s2
--
y : z
`
	expected := `
Doc
	Section[Parameters]
		Definition
			Text[x]
			Text[int]
			Paragraph
				Text[a b]
			Paragraph
				Text[c]
		Definition
			Text[d]
	Section[s2]
		Paragraph
			Text[y : z]
`
	assertParse(t, expected, src)
}

func TestDefinitionWrongSection(t *testing.T) {
	src := `
Not parameters
--------------
x : int
`
	expected := `
Doc
	Section[Not parameters]
		Paragraph
			Text[x : int]
`
	assertParse(t, expected, src)
}

func TestMultipleDefinitions(t *testing.T) {
	src := `
Parameters
----------
x : int
y : bool
a, b, c
d
`
	expected := `
Doc
	Section[Parameters]
		Definition
			Text[x]
			Text[int]
		Definition
			Text[y]
			Text[bool]
		Definition
			Text[a, b, c]
		Definition
			Text[d]
`
	assertParse(t, expected, src)
}

func TestWithMarkup(t *testing.T) {
	src := `
  Notes
  -----
a *b* **c** <*>d<*>
`
	expected := `
Doc
	Section[Notes]
		Paragraph
			Text[a ]
			Inline[i "b"]
			Text[ ]
			Inline[b "c"]
			Text[ <*>d<*>]
`
	assertParse(t, expected, src)
}
