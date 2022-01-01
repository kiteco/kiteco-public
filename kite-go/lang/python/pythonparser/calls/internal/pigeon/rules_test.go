package pigeon

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

// testInput represents a parser test case, in is the input data to
// test, and desc is an optional description of that test case. If
// desc is not empyt, it is printed along with the input data in
// error messages.
type testInput struct {
	in   string
	desc string
}

func (ti testInput) String() string {
	if ti.desc == "" {
		return fmt.Sprintf("%q", ti.in)
	}
	return fmt.Sprintf("%q - %s", ti.in, ti.desc)
}

var validIDs = []testInput{
	{in: "a"},
	{in: "_"},
	{in: "_a"},
	{in: "_a_"},
	{in: "__a__"},
	{in: "a1"},
	{in: "℮"},
	{in: "℮·"},
	{in: "℮·1"},
	{in: "℮·1b_"},
}

var invalidIDs = []testInput{
	{in: ""},
	{in: "1"},
	{in: "·"},
	{in: "·a"},
	{in: "1a"},
}

func TestValidIDs(t *testing.T) {
	for _, id := range validIDs {
		got, err := Parse("", []byte(id.in), Entrypoint("TestID"))
		if err != nil {
			t.Errorf("%s: failed: %v", id, err)
			continue
		}
		ne := got.(*pythonast.NameExpr)
		if ne.Ident.Literal != id.in {
			t.Errorf("%s: got %q", id, ne.Ident.Literal)
		}
	}
}

func TestInvalidIDs(t *testing.T) {
	for _, id := range invalidIDs {
		_, err := Parse("", []byte(id.in), Entrypoint("TestID"))
		if err == nil {
			t.Errorf("%s: want error, got none", id)
		}
	}
}

var validAttr = []struct {
	testInput
	out []string
}{
	{testInput{in: "a"}, []string{"a"}},
	{testInput{in: "a.b"}, []string{"a", "b"}},
	{testInput{in: "a.b.c"}, []string{"a", "b", "c"}},
	{testInput{in: "a.b.c.de.fgh"}, []string{"a", "b", "c", "de", "fgh"}},

	{testInput{in: "foo.\nbar"}, []string{"foo", "bar"}},
	{testInput{in: "foo\n.\nbar"}, []string{"foo", "bar"}},
	{testInput{in: "foo . bar"}, []string{"foo", "bar"}},
	{testInput{in: "foo\n.\nbar\n\r . \rbaz"}, []string{"foo", "bar", "baz"}},

	{testInput{in: "foo."}, []string{"foo", ""}},
	{testInput{in: "foo. bar\n."}, []string{"foo", "bar", ""}},
	{testInput{in: "foo. bar\n. \tbaz."}, []string{"foo", "bar", "baz", ""}},
}

func TestValidAttr(t *testing.T) {
	for _, c := range validAttr {
		got, err := Parse("", []byte(c.in), Entrypoint("TestAttr"))
		if err != nil {
			t.Errorf("%s: failed: %v", c.testInput, err)
			continue
		}

		expr := got.(pythonast.Expr)
		parts := attributeParts(t, expr, nil)
		if !reflect.DeepEqual(c.out, parts) {
			t.Errorf("%s: want %v, got %v", c.testInput, c.out, parts)
		}
	}
}

func attributeParts(t *testing.T, expr pythonast.Expr, parts []string) []string {
	switch v := expr.(type) {
	case *pythonast.NameExpr:
		parts = append(parts, v.Ident.Literal)
	case *pythonast.AttributeExpr:
		parts = attributeParts(t, v.Value, parts)
		parts = append(parts, v.Attribute.Literal)
	default:
		t.Fatalf("unexpected expression type: %T", expr)
	}
	return parts
}

var validInts = []testInput{
	{in: "0"},
	{in: "1"},
	{in: "2"},
	{in: "3"},
	{in: "4"},
	{in: "5"},
	{in: "6"},
	{in: "7"},
	{in: "8"},
	{in: "9"},
	{in: "10"},
	{in: "11"},
	{in: "120"},
	{in: "1200"},
	{in: "12123"},
	{in: "121234"},
	{in: "00"},
	{in: "0_000"},
	{in: "1_234_567"},

	{in: "0b0"},
	{in: "0b1"},
	{in: "0B00"},
	{in: "0b01"},
	{in: "0B10"},
	{in: "0b11"},
	{in: "0B_1_1"},

	{in: "0o0"},
	{in: "0o1"},
	{in: "0O2"},
	{in: "0o3"},
	{in: "0O4"},
	{in: "0o5"},
	{in: "0O6"},
	{in: "0o7"},
	{in: "0o_123"},
	{in: "0O1_23_456_7"},

	{in: "0x0"},
	{in: "0X1"},
	{in: "0x2"},
	{in: "0X3"},
	{in: "0x4"},
	{in: "0x5"},
	{in: "0X6"},
	{in: "0x7"},
	{in: "0x8"},
	{in: "0x9"},
	{in: "0XA"},
	{in: "0xb"},
	{in: "0xc"},
	{in: "0XD"},
	{in: "0xe"},
	{in: "0XF"},
	{in: "0x_0"},
	{in: "0x1_ab_23"},
}

var validLongs = []testInput{
	{in: "123L"},
	{in: "1_2_3l"},
	{in: "0b0101L"},
	{in: "0B01_01l"},
	{in: "0o13L"},
	{in: "0O1_3L"},
	{in: "0x0L"},
	{in: "0xABCDEFL"},
}

var validFloats = []testInput{
	{in: "1."},
	{in: "3.14"},
	{in: ".1"},
	{in: ".001"},
	{in: "1e100"},
	{in: "3.14E-10"},
	{in: "0e0"},
	{in: "2_4e+10"},
	{in: "3.14_15_93"},
}

var validImag = []testInput{
	{in: "1.j"},
	{in: "3.14j"},
	{in: ".001j"},
	{in: "1e100j"},
	{in: "2_4e+10j"},
	{in: "3.14_15_93j"},
	{in: "123j"},
	{in: "000j"},
	{in: "0123j"},
}

var invalidInts = []testInput{
	{in: ""},
	{in: "_"},
	{in: "a"},
	{in: "01", desc: "only 0 can be written as decimal integer with leading 0s"},
	{in: "0a"},
	{in: "0b2"},
	{in: "0o8"},
	{in: "0xg"},
	{in: "0_b0"},
	{in: "0_o0"},
	{in: "0_x0"},
	{in: "0_"},
	{in: "0b0_"},
	{in: "0o0_"},
	{in: "0x0_"},
}

var invalidLongs = []testInput{
	{in: "l"},
	{in: "L"},
	{in: "02L"},
	{in: "0b0_l"},
	{in: "0o0_L"},
	{in: "0x0_L"},
}

var invalidFloats = []testInput{
	{in: "."},
	{in: "_1."},
	{in: "1._"},
	{in: "0.0_"},
	{in: "0.1e"},
	{in: "0.1e_"},
	{in: "0.1e1_"},
	{in: "0.1e+"},
	{in: "0.1e-"},
	{in: "0.1e--1"},
	{in: "0.1e+-1"},
}

var invalidImag = []testInput{
	{in: ".j"},
	{in: "_j"},
	{in: "1e+j"},
	{in: "1e-j"},
	{in: "0b1j"},
	{in: "0o1j"},
	{in: "0x1j"},
}

func TestValidNums(t *testing.T) {
	cases := make(map[testInput]pythonscanner.Token)
	for _, n := range validInts {
		cases[n] = pythonscanner.Int
	}
	for _, n := range validLongs {
		cases[n] = pythonscanner.Long
	}
	for _, n := range validFloats {
		cases[n] = pythonscanner.Float
	}
	for _, n := range validImag {
		cases[n] = pythonscanner.Imag
	}

	for n, tok := range cases {
		got, err := Parse("", []byte(n.in), Entrypoint("TestNumber"))
		if err != nil {
			t.Errorf("%s: failed: %v", n, err)
			continue
		}
		ne := got.(*pythonast.NumberExpr)
		if ne.Number.Literal != n.in {
			t.Errorf("%s: got %q", n, ne.Number.Literal)
		}
		if ne.Number.Token != tok {
			t.Errorf("%s: want token %v, got %v", n, tok, ne.Number.Token)
		}
	}
}

func TestInvalidNums(t *testing.T) {
	cases := append(invalidInts, append(invalidLongs, append(invalidFloats, invalidImag...)...)...)
	for _, n := range cases {
		_, err := Parse("", []byte(n.in), Entrypoint("TestNumber"))
		if err == nil {
			t.Errorf("%s: want error, got none", n)
		}
	}
}

var validStrings = []struct {
	testInput
	out []string
}{
	{testInput{in: `""`}, []string{`""`}},
	{testInput{in: `''`}, []string{`''`}},
	{testInput{in: `""''`}, []string{`""`, `''`}},
	{testInput{in: `"" ''`}, []string{`""`, `''`}},
	{testInput{in: "\"\"\n''"}, []string{`""`, `''`}},

	{testInput{in: `""""""`, desc: "empty triple-quoted string, not 3 empty strings"}, []string{`""""""`}},
	{testInput{in: `''''''`, desc: "empty triple-quoted string, not 3 empty strings"}, []string{`''''''`}},
	{testInput{in: `"""""\""""`}, []string{`"""""\""""`}},
	{testInput{in: `'''''\''''`}, []string{`'''''\''''`}},
	{testInput{in: `""""""""''`}, []string{`""""""`, `""`, `''`}},
	{testInput{in: `''''''''""`}, []string{`''''''`, `''`, `""`}},

	{testInput{in: `b""`}, []string{`b""`}},
	{testInput{in: `br""`}, []string{`br""`}},
	{testInput{in: `rb""`}, []string{`rb""`}},
	{testInput{in: `RB""`}, []string{`RB""`}},
	{testInput{in: `u""`}, []string{`u""`}},
	{testInput{in: `rf""`}, []string{`rf""`}},
	{testInput{in: `fr""`}, []string{`fr""`}},
	{testInput{in: `FR""`}, []string{`FR""`}},
	{testInput{in: `FR""b''`}, []string{`FR""`, `b''`}},
	{testInput{in: `FR""b''""""""u''`}, []string{`FR""`, `b''`, `""""""`, `u''`}},

	{testInput{in: `b'abc\x80\xff'`}, []string{`b'abc\x80\xff'`}},
	{testInput{in: `"hello"`}, []string{`"hello"`}},
	{testInput{in: `"hello" 'world'`}, []string{`"hello"`, `'world'`}},
	{testInput{in: "''''\n\"\\''''"}, []string{"''''\n\"\\''''"}},

	{testInput{in: `"Ŝόмẽ ŝẳოρļё ẦŞÇÌỊ-ŧεхţ"`}, []string{`"Ŝόмẽ ŝẳოρļё ẦŞÇÌỊ-ŧεхţ"`}},
	{testInput{in: `'--̪͝ḭ̮͢n̹̦̺͍͜v͓̻̘͇͈̞o̧k̞͔͎̼̲̦e̛͈̬̫̜ ̡̲t̹̥͕͚̘͟h̖̪̲̥͞e̷̯͙̲̬̘ ̵̥̲hi̯͙̙̝v̟͇e҉-̀m̪̩i̭͉̰̘̼̣͜ń̜͍d̠̬̞͔̰͇͞--'`}, []string{`'--̪͝ḭ̮͢n̹̦̺͍͜v͓̻̘͇͈̞o̧k̞͔͎̼̲̦e̛͈̬̫̜ ̡̲t̹̥͕͚̘͟h̖̪̲̥͞e̷̯͙̲̬̘ ̵̥̲hi̯͙̙̝v̟͇e҉-̀m̪̩i̭͉̰̘̼̣͜ń̜͍d̠̬̞͔̰͇͞--'`}},

	{testInput{in: "\"\"\"hello\n\n\tworld\n\n\r\"\"\""}, []string{"\"\"\"hello\n\n\tworld\n\n\r\"\"\""}},

	// explicit line continuation
	{testInput{in: "'hello\\\nworld!'"}, []string{"'hello\\\nworld!'"}},
	{testInput{in: "\"hello\\\nworld!\""}, []string{"\"hello\\\nworld!\""}},
	{testInput{in: "b'hello\\\nworld!'"}, []string{"b'hello\\\nworld!'"}},
	{testInput{in: "b\"hello\\\nworld!\""}, []string{"b\"hello\\\nworld!\""}},

	// newlines within a string literal do not count towards the max lines limit
	{testInput{in: "'''a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\nm\nn\no\np'''"}, []string{"'''a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\nm\nn\no\np'''"}},
	{testInput{in: "'a\\\nb\\\nc\\\nd\\\ne\\\nf\\\ng\\\nh\\\ni\\\nj\\\nk\\\nl\\\nm\\\no\\\np'"}, []string{"'a\\\nb\\\nc\\\nd\\\ne\\\nf\\\ng\\\nh\\\ni\\\nj\\\nk\\\nl\\\nm\\\no\\\np'"}},

	// unclosed string literals are supported
	{testInput{in: `"`}, []string{`"`}},
	{testInput{in: `'`}, []string{`'`}},
	{testInput{in: `"""`, desc: "unclosed triple-quoted string"}, []string{`"""`}},
	{testInput{in: `'''`, desc: "unclosed triple-quoted string"}, []string{`'''`}},
	{testInput{in: `"""""`, desc: "unclosed triple-quoted string that contains 2 `\"`"}, []string{`"""""`}},
	{testInput{in: `'''''`, desc: "unclosed triple-quoted string that contains 2 `'`"}, []string{`'''''`}},
	{testInput{in: `"""""""`, desc: "empty triple-quoted string followed by unclosed single-quoted"}, []string{`""""""`, `"`}},
	{testInput{in: `'''''''`, desc: "empty triple-quoted string followed by unclosed single-quoted"}, []string{`''''''`, `'`}},
	{testInput{in: `"\"`}, []string{`"\"`}},
	{testInput{in: `'\'`}, []string{`'\'`}},
	{testInput{in: `"""\"""`}, []string{`"""\"""`}},
	{testInput{in: `'''\'''`}, []string{`'''\'''`}},
	{testInput{in: "\"\n\"", desc: "2 unclosed literals"}, []string{`"`, `"`}},
	{testInput{in: "'\n'", desc: "2 unclosed literals"}, []string{`'`, `'`}},
	{testInput{in: "'hello\\'"}, []string{`'hello\'`}},

	// unclosed bytes literals are supported
	{testInput{in: `b"`}, []string{`b"`}},
	{testInput{in: `b'`}, []string{`b'`}},
	{testInput{in: `b"""`, desc: "unclosed triple-quoted string"}, []string{`b"""`}},
	{testInput{in: `b'''`, desc: "unclosed triple-quoted string"}, []string{`b'''`}},
	{testInput{in: `b"""""`, desc: "unclosed triple-quoted string that contains 2 `\"`"}, []string{`b"""""`}},
	{testInput{in: `b'''''`, desc: "unclosed triple-quoted string that contains 2 `'`"}, []string{`b'''''`}},
	{testInput{in: `b"""""""`, desc: "empty triple-quoted string followed by unclosed single-quoted"}, []string{`b""""""`, `"`}},
	{testInput{in: `b'''''''`, desc: "empty triple-quoted string followed by unclosed single-quoted"}, []string{`b''''''`, `'`}},
	{testInput{in: `b"\"`}, []string{`b"\"`}},
	{testInput{in: `b'\'`}, []string{`b'\'`}},
	{testInput{in: `b"""\"""`}, []string{`b"""\"""`}},
	{testInput{in: `b'''\'''`}, []string{`b'''\'''`}},
	{testInput{in: "b\"\nb\"", desc: "2 unclosed literals"}, []string{`b"`, `b"`}},
	{testInput{in: "b'\nb'", desc: "2 unclosed literals"}, []string{`b'`, `b'`}},
	{testInput{in: "b'hello\\'"}, []string{`b'hello\'`}},

	{testInput{in: "r'this'\rb\"is\"\r\n'''a somewhat\ncomplex''' b'string"}, []string{`r'this'`, `b"is"`, "'''a somewhat\ncomplex'''", "b'string"}},
}

var invalidStrings = []testInput{
	{in: `"\`},
	{in: `'\`},
	{in: `b"\`},
	{in: `b'\`},
	{in: `b"Ŝόмẽ ŝẳოρļё ẦŞÇÌỊ-ŧεхţ"`},
	{in: `b"é"`},
}

func TestValidStrings(t *testing.T) {
	for _, c := range validStrings {
		got, err := Parse("", []byte(c.in), Entrypoint("TestStrings"))
		if err != nil {
			t.Errorf("%s: failed: %v", c.testInput, err)
			continue
		}

		se := got.(*pythonast.StringExpr)
		var gotStrings []string
		for _, w := range se.Strings {
			gotStrings = append(gotStrings, w.Literal)
		}

		if !reflect.DeepEqual(c.out, gotStrings) {
			t.Errorf("%s: want %v, got %v", c.testInput, c.out, gotStrings)
		}
	}
}

func TestInvalidStrings(t *testing.T) {
	for _, s := range invalidStrings {
		_, err := Parse("", []byte(s.in), Entrypoint("TestStrings"))
		if err == nil {
			t.Errorf("%s: want error, got none", s)
		}
	}
}
