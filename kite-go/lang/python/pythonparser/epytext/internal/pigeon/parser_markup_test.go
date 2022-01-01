package pigeon

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/internal/testast"
)

func assertText(t *testing.T, expected, src string) {
	nodes := parseText(src)
	// wrap in a single parent
	root := &ast.DocBlock{Nodes: nodes}
	testast.Assert(t, expected, root)
}

func TestMarkupNone(t *testing.T) {
	src := `test`
	expected := `
Doc
	Text[test]
`
	assertText(t, expected, src)
}

func TestClosingOnly(t *testing.T) {
	src := `te}st`
	expected := `
Doc
	Text[te}st]
`
	assertText(t, expected, src)
}

func TestUnclosed(t *testing.T) {
	src := `teB{st`
	expected := `
Doc
	Text[te]
	BasicMarkup[B]
		Text[st]
`
	assertText(t, expected, src)
}

func TestBasicMarkupStart(t *testing.T) {
	src := `B{te}st`
	expected := `
Doc
	BasicMarkup[B]
		Text[te]
	Text[st]
`
	assertText(t, expected, src)
}

func TestBasicMarkupMiddle(t *testing.T) {
	src := `tB{e}st`
	expected := `
Doc
	Text[t]
	BasicMarkup[B]
		Text[e]
	Text[st]
`
	assertText(t, expected, src)
}

func TestBasicMarkupEnd(t *testing.T) {
	src := `tesB{t}`
	expected := `
Doc
	Text[tes]
	BasicMarkup[B]
		Text[t]
`
	assertText(t, expected, src)
}

func TestBasicMarkupEmptyStart(t *testing.T) {
	src := `B{}test`
	expected := `
Doc
	BasicMarkup[B]
	Text[test]
`
	assertText(t, expected, src)
}

func TestBasicMarkupEmptyMiddle(t *testing.T) {
	src := `tesB{}t`
	expected := `
Doc
	Text[tes]
	BasicMarkup[B]
	Text[t]
`
	assertText(t, expected, src)
}

func TestBasicMarkupEmptyEnd(t *testing.T) {
	src := `testB{}`
	expected := `
Doc
	Text[test]
	BasicMarkup[B]
`
	assertText(t, expected, src)
}

func TestMultiBasicMarkup(t *testing.T) {
	src := `tB{e}sI{t}`
	expected := `
Doc
	Text[t]
	BasicMarkup[B]
		Text[e]
	Text[s]
	BasicMarkup[I]
		Text[t]
`
	assertText(t, expected, src)
}

func TestMultiBasicMarkupEnds(t *testing.T) {
	src := `B{te}I{st}`
	expected := `
Doc
	BasicMarkup[B]
		Text[te]
	BasicMarkup[I]
		Text[st]
`
	assertText(t, expected, src)
}

func TestNestedBasicMarkup(t *testing.T) {
	src := `B{tI{e}}st`
	expected := `
Doc
	BasicMarkup[B]
		Text[t]
		BasicMarkup[I]
			Text[e]
	Text[st]
`
	assertText(t, expected, src)
}

func TestEscapeMarkupStart(t *testing.T) {
	src := `E{t}est`
	expected := `
Doc
	Text[t]
	Text[est]
`
	assertText(t, expected, src)
}

func TestEscapeMarkupMiddle(t *testing.T) {
	src := `tE{e}st`
	expected := `
Doc
	Text[t]
	Text[e]
	Text[st]
`
	assertText(t, expected, src)
}

func TestEscapeMarkupEnd(t *testing.T) {
	src := `tesE{t}`
	expected := `
Doc
	Text[tes]
	Text[t]
`
	assertText(t, expected, src)
}

func TestEscapeMarkupReplace(t *testing.T) {
	src := `tE{lb}estE{rb}`
	expected := `
Doc
	Text[t]
	Text[{]
	Text[est]
	Text[}]
`
	assertText(t, expected, src)
}

func TestURLMarkupNoArg(t *testing.T) {
	src := `U{xyz}`
	expected := `
Doc
	URLMarkup[xyz]
		Text[xyz]
`
	assertText(t, expected, src)
}

func TestURLMarkupWithBasicNoArg(t *testing.T) {
	src := `U{this B{is} a test}`
	expected := `
Doc
	URLMarkup[this is a test]
		Text[this ]
		BasicMarkup[B]
			Text[is]
		Text[ a test]
`
	assertText(t, expected, src)
}

func TestURLMarkupWithArg(t *testing.T) {
	src := `U{x<y>}`
	expected := `
Doc
	URLMarkup[y]
		Text[x]
`
	assertText(t, expected, src)
}

func TestURLMarkupWithBasicWithArg(t *testing.T) {
	src := `U{this B{is} a I{test}<http://test>}`
	expected := `
Doc
	URLMarkup[http://test]
		Text[this ]
		BasicMarkup[B]
			Text[is]
		Text[ a ]
		BasicMarkup[I]
			Text[test]
		Text[]
`
	assertText(t, expected, src)
}

func TestCrossRefNoArg(t *testing.T) {
	src := `L{xyz}`
	expected := `
Doc
	CrossRefMarkup[xyz]
		Text[xyz]
`
	assertText(t, expected, src)
}

func TestCrossRefWithArg(t *testing.T) {
	src := `L{xyz<abc>}`
	expected := `
Doc
	CrossRefMarkup[abc]
		Text[xyz]
`
	assertText(t, expected, src)
}

func TestCrossRefMarkupWithBasicWithArg(t *testing.T) {
	src := `L{this B{is} a I{test}<test>}`
	expected := `
Doc
	CrossRefMarkup[test]
		Text[this ]
		BasicMarkup[B]
			Text[is]
		Text[ a ]
		BasicMarkup[I]
			Text[test]
		Text[]
`
	assertText(t, expected, src)
}

func TestCrossRefMarkupWithBasicNoArg(t *testing.T) {
	src := `L{this B{is} a test}`
	expected := `
Doc
	CrossRefMarkup[this is a test]
		Text[this ]
		BasicMarkup[B]
			Text[is]
		Text[ a test]
`
	assertText(t, expected, src)
}
