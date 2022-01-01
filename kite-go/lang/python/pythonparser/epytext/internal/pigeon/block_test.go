package pigeon

import (
	"reflect"
	"testing"
)

func assertBlocks(t *testing.T, in string, want []block) {
	got, err := Parse("", []byte(in), Entrypoint("TestInternalBlocks"))
	if err != nil {
		t.Fatal(err)
	}
	blocks := toIfaceBlocks(got.([]interface{}))

	if !reflect.DeepEqual(blocks, want) {
		t.Errorf("want %#v, got %#v", want, blocks)
	}
}

func TestList(t *testing.T) {
	cases := []struct {
		in   string
		want []block
	}{
		{"1. \n", []block{makeList(0, "1.", "", false, false)}},
		{" \t\n1. \n", []block{makeList(0, "1.", "", false, false)}},
		{" \t\n  1. \n", []block{makeList(2, "1.", "", false, false)}},
		{"1.2.3. \n", []block{makeList(0, "1.2.3.", "", false, false)}},
		{"- \n", []block{makeList(0, "-", "", false, false)}},
		{" \t\n- \n", []block{makeList(0, "-", "", false, false)}},
		{" \t\n\t- \n", []block{makeList(8, "-", "", false, false)}},
		{"- \n  2. \n    3.   \n4. a\n", []block{
			makeList(0, "-", "", false, false),
			makeList(2, "2.", "", false, false),
			makeList(4, "3.", "", false, false),
			makeList(0, "4.", "a", true, false),
		}},

		{"- abc\n", []block{makeList(0, "-", "abc", true, false)}},
		{"- abc \n", []block{makeList(0, "-", "abc ", true, false)}},
		{"- abc \ndef\n", []block{
			makeList(0, "-", "abc ", true, false),
			makeParagraph(0, "def", false),
		}},
		{"\t1. abc \n\tdef\n", []block{
			makeList(8, "1.", "abc ", true, false),
			makeParagraph(8, "def", false),
		}},
		{"\t1. abc \n\n\tdef\n", []block{
			makeList(8, "1.", "abc ", false, false),
			makeParagraph(8, "def", false),
		}},
		{"- ::\n", []block{makeList(0, "-", ":", true, true)}},

		{"1 \n", []block{makeParagraph(0, "1 ", false)}},
		{"1.\n", []block{makeList(0, "1.", "", false, false)}},
		{"-\n", []block{makeList(0, "-", "", false, false)}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assertBlocks(t, c.in, c.want)
		})
	}
}

func TestDoctest(t *testing.T) {
	cases := []struct {
		in   string
		want []block
	}{
		{"\n>>> test\n", []block{makeDoctest(0, ">>> test")}},
		{"\n>>> test\n\n", []block{makeDoctest(0, ">>> test")}},
		{"\n  >>> test\n  other line\n\n", []block{makeDoctest(2, ">>> test\nother line")}},
		{"\n  >>> test\n  other line\n\t\tnot this one\n", []block{
			makeParagraph(2, ">>> test\nother line", false),
			makeParagraph(16, "not this one", false),
		}},
		{"\n  >>> test\n  other line\n\n  >>> new test\n  hello\n\n", []block{
			makeDoctest(2, ">>> test\nother line"),
			makeDoctest(2, ">>> new test\nhello"),
		}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assertBlocks(t, c.in, c.want)
		})
	}
}

func TestParagraph(t *testing.T) {
	cases := []struct {
		in   string
		want []block
	}{
		{"a\n", []block{makeParagraph(0, "a", false)}},
		{"abc\ndef\n", []block{makeParagraph(0, "abc\ndef", false)}},
		{"abc\n def\n", []block{makeParagraph(0, "abc", false), makeParagraph(1, "def", false)}},
		{"abc\ndef\n  ghi\n  jkl\n    mno\npqr\n", []block{
			makeParagraph(0, "abc\ndef", false),
			makeParagraph(2, "ghi\njkl", false),
			makeParagraph(4, "mno", false),
			makeParagraph(0, "pqr", false),
		}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assertBlocks(t, c.in, c.want)
		})
	}
}

func TestSection(t *testing.T) {
	cases := []struct {
		in   string
		want []block
	}{
		{"", []block{}},
		{"\n\t \n \t", []block{}},

		{"a\n=\n", []block{makeSection(0, "a", '=')}},
		{" a\n =\n", []block{makeSection(1, "a", '=')}},
		{"a\n=\n \n  b\n  ~\n", []block{
			makeSection(0, "a", '='),
			makeSection(2, "b", '~'),
		}},
		{"a\n=\n \n  b\n  ~\n c\n -\n\nd\n=\n", []block{
			makeSection(0, "a", '='),
			makeSection(2, "b", '~'),
			makeSection(1, "c", '-'),
			makeSection(0, "d", '='),
		}},
		{"\n\n\tabc\n  =====\n\n", []block{
			makeParagraph(8, "abc", false),
			makeParagraph(2, "=====", false),
		}},
		{"\n\n\tabc\n\t=====\n\n", []block{
			makeParagraph(8, "abc\n=====", false),
		}},
		{"\n\n\tabcde\n\t=====\n\n", []block{
			makeSection(8, "abcde", '='),
		}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assertBlocks(t, c.in, c.want)
		})
	}
}

func TestLiteral(t *testing.T) {
	cases := []struct {
		in   string
		want []block
	}{
		{"p::\n  literal\n", []block{
			makeParagraph(0, "p:", true),
			makeLiteral(0, "  literal"),
		}},
		{"::\n  a\n\n\n  b\n", []block{
			makeParagraph(0, ":", true),
			makeLiteral(0, "  a\n\n\n  b"),
		}},
		{"  p1\n  p1cont\n  ::\n   a\n", []block{
			makeParagraph(2, "p1\np1cont\n:", true),
			makeLiteral(2, " a"),
		}},
		{"- ::\n  a\n", []block{
			makeList(0, "-", ":", true, true),
			makeLiteral(0, "  a"),
		}},
		{"1. meet lit::\n  a\n", []block{
			makeList(0, "1.", "meet lit:", true, true),
			makeLiteral(0, "  a"),
		}},
		{"p::\ncont'd\n  a\n", []block{
			makeParagraph(0, "p::\ncont'd", false),
			makeParagraph(2, "a", false),
		}},
		{"  p::\notherp\n", []block{
			makeParagraph(2, "p:", true),
			makeParagraph(0, "otherp", false),
		}},
		{"p1.a\np1.b::\n\n  lit1.a\n  lit1.b\np2\n\n  1. l1::\n    lit2\n  - \n    p3::\n      lit3\n    p4\n", []block{
			makeParagraph(0, "p1.a\np1.b:", true),
			makeLiteral(0, "\n  lit1.a\n  lit1.b"),
			makeParagraph(0, "p2", false),
			makeList(2, "1.", "l1:", true, true),
			makeLiteral(2, "  lit2"),
			makeList(2, "-", "", false, false),
			makeParagraph(4, "p3:", true),
			makeLiteral(4, "  lit3"),
			makeParagraph(4, "p4", false),
		}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assertBlocks(t, c.in, c.want)
		})
	}
}

func TestField(t *testing.T) {
	cases := []struct {
		in   string
		want []block
	}{
		{"@f:\n", []block{makeField(0, "f", "", "", false, false)}},
		{"@f arg:\n", []block{makeField(0, "f", "arg", "", false, false)}},
		{"@f arg :\n", []block{makeField(0, "f", "arg", "", false, false)}},
		{"@f :\n", []block{makeField(0, "f", "", "", false, false)}},
		{"  @return: inline\n", []block{makeField(2, "return", "", "inline", true, false)}},
		{"  @return: inline\n\n", []block{makeField(2, "return", "", "inline", false, false)}},
		{"  @f a: p::\n    lit\n", []block{
			makeField(2, "f", "a", "p:", true, true),
			makeLiteral(2, "  lit"),
		}},
		{"@(c): copy\n\n", []block{makeField(0, "(c)", "", "copy", false, false)}},
		{"@a b c: \n\n", []block{makeParagraph(0, "@a b c: ", false)}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assertBlocks(t, c.in, c.want)
		})
	}
}
