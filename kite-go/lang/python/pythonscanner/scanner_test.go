package pythonscanner

import (
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	opts = Options{
		OneBasedPositions: true,
		ScanComments:      true,
	}

	optsWithNewlines = Options{
		ScanComments:      true,
		ScanNewLines:      true,
		OneBasedPositions: true,
	}
)

func AssertScansOnce(t *testing.T, src string, expected Token) {
	scanner := NewScanner([]byte(src), opts)

	t.Logf("Scanning '%s'", src)

	begin, end, tok, _ := scanner.Scan()
	assert.Equal(t, expected.String(), tok.String())
	assert.Equal(t, token.Pos(1), begin)
	assert.Equal(t, token.Pos(len(src)+1), end)

	_, _, tok, _ = scanner.Scan()
	assert.Equal(t, EOF.String(), tok.String())
}

func AssertNextToken(t *testing.T, scanner *Scanner, expectedPos int, expectedTok Token) {
	begin, _, tok, _ := scanner.Scan()
	assert.Equal(t, token.Pos(expectedPos), begin)
	assert.Equal(t, expectedTok.String(), tok.String())
}

func AssertNextTokenLit(t *testing.T, scanner *Scanner, expectedPos int, expectedTok Token, expectedLit string) {
	begin, _, tok, lit := scanner.Scan()
	assert.Equal(t, token.Pos(expectedPos), begin)
	assert.Equal(t, expectedTok.String(), tok.String())
	assert.Equal(t, expectedLit, lit)
}

func AssertEOF(t *testing.T, scanner *Scanner) {
	_, _, tok, _ := scanner.Scan()
	assert.Equal(t, EOF, tok)
}

func AssertScan(t *testing.T, src string, expected string) {
	t.Log(src)

	scanner := NewScanner([]byte(src), opts)
	var actual []string
	for {
		_, _, tok, _ := scanner.Scan()
		if tok == EOF {
			break
		}
		actual = append(actual, tok.String())
	}
	assert.NoError(t, scanner.Errs)
	assert.Equal(t, expected, strings.Join(actual, " "))
}

func TestScanSimple(t *testing.T) {
	src := "abc + xyz"
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "abc")
	AssertNextToken(t, scanner, 5, Add)
	AssertNextTokenLit(t, scanner, 7, Ident, "xyz")
	AssertEOF(t, scanner)
}

func TestScanFunctionDef(t *testing.T) {
	src := "def foo():\n  pass"
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextToken(t, scanner, 1, Def)
	AssertNextTokenLit(t, scanner, 5, Ident, "foo")
	AssertNextToken(t, scanner, 8, Lparen)
	AssertNextToken(t, scanner, 9, Rparen)
	AssertNextToken(t, scanner, 10, Colon)
	AssertNextToken(t, scanner, 14, Pass)
	AssertEOF(t, scanner)
}

func TestScanEachOperator(t *testing.T) {
	for tok := operatorbegin + 1; tok < operatorend; tok++ {
		AssertScansOnce(t, tokenMap[tok], tok)
	}
}

func TestScanEachKeyword(t *testing.T) {
	for tok := keywordbegin + 1; tok < keywordend; tok++ {
		t.Logf("%v", tok)
		AssertScansOnce(t, tokenMap[tok], tok)
	}
}

func TestScanComment(t *testing.T) {
	src := "x # hello\ny # world"
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextTokenLit(t, scanner, 3, Comment, "# hello")
	AssertNextTokenLit(t, scanner, 11, Ident, "y")
	AssertNextTokenLit(t, scanner, 13, Comment, "# world")
	AssertEOF(t, scanner)
}

func TestScanInteger(t *testing.T) {
	strs := []string{"1", "0123", "0b1011", "0x3F"}
	for _, s := range strs {
		AssertScan(t, s, "Int")
	}
}

func TestScanLong(t *testing.T) {
	strs := []string{"123L", "0b110L", "0xA8L"}
	for _, s := range strs {
		AssertScan(t, s, "Long")
	}
}

func TestScanFloat(t *testing.T) {
	strs := []string{".123", "1.23", "123.", "123e-1", "12.3e+4"}
	for _, s := range strs {
		AssertScan(t, s, "Float")
	}
}

func TestScanImag(t *testing.T) {
	strs := []string{".123j", "1.23j", "123.j", "123e-1j", "12.3e+4j"}
	for _, s := range strs {
		AssertScan(t, s, "Imag")
	}
}

func TestScanDotted(t *testing.T) {
	src := `a = foo.bar.car`
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "a")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, Ident, "foo")
	AssertNextToken(t, scanner, 8, Period)
	AssertNextTokenLit(t, scanner, 9, Ident, "bar")
	AssertNextToken(t, scanner, 12, Period)
	AssertNextTokenLit(t, scanner, 13, Ident, "car")
}

func TestScan(t *testing.T) {
	words, err := Scan([]byte("x # hello\ny # world"))
	assert.NoError(t, err)
	assert.Len(t, words, 6)
}

func TestScanArrow(t *testing.T) {
	src := "a -> b"
	scanner := NewScanner([]byte(src), optsWithNewlines)
	AssertNextTokenLit(t, scanner, 1, Ident, "a")
	AssertNextToken(t, scanner, 3, Arrow)
	AssertNextTokenLit(t, scanner, 6, Ident, "b")
}

func TestScanLineContinuation(t *testing.T) {
	src := "foo = \\\n   bar"
	scanner := NewScanner([]byte(src), optsWithNewlines)
	AssertNextTokenLit(t, scanner, 1, Ident, "foo")
	AssertNextToken(t, scanner, 5, Assign)
	AssertNextToken(t, scanner, 7, LineContinuation)
	AssertNextTokenLit(t, scanner, 12, Ident, "bar")
}

func TestScanLineContinuation_CarriageReturn(t *testing.T) {
	src := "foo = \\\r\n   bar"
	scanner := NewScanner([]byte(src), optsWithNewlines)
	AssertNextTokenLit(t, scanner, 1, Ident, "foo")
	AssertNextToken(t, scanner, 5, Assign)
	AssertNextToken(t, scanner, 7, LineContinuation)
	AssertNextTokenLit(t, scanner, 13, Ident, "bar")
}

func TestScanLineContinuation_CarriageReturn2(t *testing.T) {
	src := "foo = \\\n\r   bar"
	scanner := NewScanner([]byte(src), optsWithNewlines)
	AssertNextTokenLit(t, scanner, 1, Ident, "foo")
	AssertNextToken(t, scanner, 5, Assign)
	AssertNextToken(t, scanner, 7, LineContinuation)
	AssertNextTokenLit(t, scanner, 13, Ident, "bar")
}

func TestScanNewLine(t *testing.T) {
	src := "foo = \n  bar"
	scanner := NewScanner([]byte(src), optsWithNewlines)
	AssertNextTokenLit(t, scanner, 1, Ident, "foo")
	AssertNextToken(t, scanner, 5, Assign)
	AssertNextTokenLit(t, scanner, 7, NewLine, "  ")
	AssertNextTokenLit(t, scanner, 10, Ident, "bar")
}

func TestScanCarriageReturnLinefeed(t *testing.T) {
	src := "def f():\r\n\tpass"

	t.Logf("Bytes:\n%v\n", []byte(src))

	scanner := NewScanner([]byte(src), optsWithNewlines)
	AssertNextToken(t, scanner, 1, Def)
	AssertNextTokenLit(t, scanner, 5, Ident, "f")
	AssertNextToken(t, scanner, 6, Lparen)
	AssertNextToken(t, scanner, 7, Rparen)
	AssertNextToken(t, scanner, 8, Colon)

	// CR, LF, \t
	// 9   10  11
	// SO src[begin:end] = [CR, LF, \t]
	// SO end = 12, thus beginning of Pass is 12
	AssertNextTokenLit(t, scanner, 9, NewLine, "\t")
	AssertNextToken(t, scanner, 12, Pass)

}

func TestScanLinefeedCarriageReturn(t *testing.T) {
	src := "def f():\n\r\tpass"

	t.Logf("Bytes:\n%v\n", []byte(src))

	scanner := NewScanner([]byte(src), optsWithNewlines)
	AssertNextToken(t, scanner, 1, Def)
	AssertNextTokenLit(t, scanner, 5, Ident, "f")
	AssertNextToken(t, scanner, 6, Lparen)
	AssertNextToken(t, scanner, 7, Rparen)
	AssertNextToken(t, scanner, 8, Colon)

	// LF, CR, \t
	// 9   10  11
	// SO src[begin:end] = [LF, CR, \t]
	// SO end = 12, thus beginning of Pass is 12
	AssertNextTokenLit(t, scanner, 9, NewLine, "\t")
	AssertNextToken(t, scanner, 12, Pass)

}

func TestScanMultiLineString(t *testing.T) {
	src := `x = """foo
bar"""`
	expected := `"""foo
bar"""`

	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, expected)
	AssertEOF(t, scanner)
}

func TestScanMultiLineString2(t *testing.T) {
	src := `x = '''foo
bar'''`
	expected := `'''foo
bar'''`

	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, expected)
	AssertEOF(t, scanner)
}

func TestScanEmptyMultiLineString(t *testing.T) {
	src := `x = """"""`

	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, `""""""`)
	AssertEOF(t, scanner)
}

func TestScanLineContinuationInString(t *testing.T) {
	src := `'abc\
def'`

	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, src)
	AssertEOF(t, scanner)
}

func TestScanDoubleQuoteString(t *testing.T) {
	src := `x = "foo"`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, `"foo"`)
	AssertEOF(t, scanner)
}

func TestScanSingleQuoteString(t *testing.T) {
	src := `x = 'foo'`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, `'foo'`)
	AssertEOF(t, scanner)
}

func TestScanStringWithEscapes(t *testing.T) {
	src := `x = '" \\ \n \' \"'`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, `'" \\ \n \' \"'`)
	AssertEOF(t, scanner)
}

func TestScanStringWithEscapes2(t *testing.T) {
	src := `x = "' \\ \n \' \""`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "x")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, `"' \\ \n \' \""`)
	AssertEOF(t, scanner)
}

func TestScanEmptyStringThenNewline(t *testing.T) {
	src := `
""
abc`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 2, String, `""`)
	AssertNextTokenLit(t, scanner, 5, Ident, "abc")
}

func TestScanStringWithPrefix(t *testing.T) {
	// All the valid python string prefixes:
	stringPrefixes := []string{
		"r", "u", "ur", "R", "U", "UR", "Ur", "uR",
		"b", "B", "br", "Br", "bR", "BR",
		"f", "F", "fr", "Fr", "fR", "FR", "rf", "Rf", "rF", "RF",
	}
	for _, prefix := range stringPrefixes {
		src := prefix + `"foo"`
		t.Log(src)
		scanner := NewScanner([]byte(src), opts)
		AssertNextTokenLit(t, scanner, 1, String, src)
		AssertEOF(t, scanner)
	}
}

func TestScanRawString(t *testing.T) {
	src := `r"\[\norules\\"`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, `r"\[\norules\\"`)
	AssertEOF(t, scanner)
}

func TestScanFormattedRawString(t *testing.T) {
	src := `fr"\[\norules{expr * 2}\\"`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, `fr"\[\norules{expr * 2}\\"`)
	AssertEOF(t, scanner)
}

func TestScanUnicodeString(t *testing.T) {
	src := `u"abc\u0123def"`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, `u"abc\u0123def"`)
	AssertEOF(t, scanner)
}

func TestScanRawUnicodeString(t *testing.T) {
	src := `uR"abc\u0123def"`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, `uR"abc\u0123def"`)
	AssertEOF(t, scanner)
}

func TestScanByteString(t *testing.T) {
	src := `B"xyz"`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, `B"xyz"`)
	AssertEOF(t, scanner)
}

func TestScanEmptyCommentBeforeSplitStmt(t *testing.T) {
	src := `
#
x = (y \
		+ z)`
	AssertScan(t, src, "Comment Ident = ( Ident + Ident )")
}

func TestScanUnicodeStringLiteral(t *testing.T) {
	// strings on RHS are unicode characters
	src := `M = '−' + '−' + "⌘"`
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, "M")
	AssertNextToken(t, scanner, 3, Assign)
	AssertNextTokenLit(t, scanner, 5, String, `'−'`)
	AssertNextToken(t, scanner, 11, Add)
	AssertNextTokenLit(t, scanner, 13, String, `'−'`)
	AssertNextToken(t, scanner, 19, Add)
	AssertNextTokenLit(t, scanner, 21, String, `"⌘"`)
	AssertEOF(t, scanner)
}

func TestScanNoBreakSpace(t *testing.T) {
	src := "\u00a0"

	scanner := NewScanner([]byte(src), opts)

	// 3 bytes for no break whitespace, then just EOF
	AssertNextToken(t, scanner, 3, EOF)
}

func TestScanStringLiteral(t *testing.T) {
	strs := []string{
		// single-quoted:
		`'abc'`,
		`''`,
		// double-quoted:
		`"abc"`,
		`""`,
		// single-quoted multi-line
		`'''abc'''`,
		`'''abc
		def'''`,
		`''''''`,
		// double-quoted multi-line
		`"""abc"""`,
		`"""abc
		def"""`,
		`""""""`,
		// quotes:
		`"'"`,
		`'"'`,
		`"""'"""`,
		`'''"'''`,
		// escaped quotes:
		`"\""`,
		`"\'"`,
		`'\''`,
		`'\"'`,
		`"X\""`,
		`"\'X"`,
		`'\'X'`,
		`'X\"'`,
		// escaped backslashes:
		`"\\"`,
		`"\\\""`,
		`'\\'`,
		`'\\\''`,
		`"\"\\"`,
		`'\'\\'`,
		// multi-line strings with escaped quotes and backslashes:
		`"""\""""`,
		`"""\'"""`,
		`'''\''''`,
		`'''\"'''`,
		`"""\"X"""`,
		`"""X\'"""`,
		`'''X\''''`,
		`'''\"X'''`,
		`"""\\\""""`,
		`'''\\\''''`,
		// special single-char sequences
		`"\a\b\f\n\r\t\v"`,
		`'\a\b\f\n\r\t\v'`,
		`"""\a\b\f\n\r\t\v"""`,
		`'''\a\b\f\n\r\t\v'''`,
		// the legacy \newline syntax:
		`"\newline"`,
		`'\newline'`,
		// unicode sequences:
		`"\u0f33"`,
		`'\u0f33'`,
		`"\U3d114daa"`,
		`'\U3d114daa'`,
		`"\u0f33"`,
		`'\u0f33'`,
		`"\U3d114daa"`,
		`'\U3d114daa'`,
		// named unicode sequences:
		`"\N{umlaut}"`,
		`'\N{umlaut}'`,
		`"""\N{umlaut}"""`,
		`'''\N{umlaut}'''`,
		// embedded hex
		`"\x3B"`,
		`"\xA1\xB2\x4F"`,
		`'\x3B'`,
		`'\xA1\xB2\x4F'`,
		// embedded octal
		`"\471"`,
		`"\143\101\633"`,
		`'\471'`,
		`'\143\101\633'`,
		// string control chars
		`r""`,
		`u""`,
		`ur""`,
		`R""`,
		`U""`,
		`UR""`,
		`Ur""`,
		`uR""`,
		`b""`,
		`B""`,
		`br""`,
		`Br""`,
		`bR""`,
		`BR""`,
		`f""`,
		`F""`,
		`fr""`,
		`fR""`,
		`Fr""`,
		`FR""`,
		`r''`,
		`u''`,
		`ur''`,
		`R''`,
		`U''`,
		`UR''`,
		`Ur''`,
		`uR''`,
		`b''`,
		`B''`,
		`br''`,
		`Br''`,
		`bR''`,
		`BR''`,
		`f''`,
		`F''`,
		`fr''`,
		`fR''`,
		`Fr''`,
		`FR''`,
	}
	for _, s := range strs {
		AssertScan(t, s, "String")
	}
}

func TestScanPct(t *testing.T) {
	src := `4 % 2`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Int, `4`)
	AssertNextToken(t, scanner, 3, Pct)
	AssertNextTokenLit(t, scanner, 5, Int, `2`)
	AssertEOF(t, scanner)
}

func TestScanPctAssign(t *testing.T) {
	src := `x %= 2`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Ident, `x`)
	AssertNextToken(t, scanner, 3, PctAssign)
	AssertNextTokenLit(t, scanner, 6, Int, `2`)
	AssertEOF(t, scanner)
}

func TestScanPctLineContinuation(t *testing.T) {
	src := `4 \
% 2`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Int, `4`)
	AssertNextToken(t, scanner, 5, Pct)
	AssertNextTokenLit(t, scanner, 7, Int, `2`)
	AssertEOF(t, scanner)
}

func TestScanMagicString(t *testing.T) {
	src := `%magic`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Magic, `%magic`)
	AssertEOF(t, scanner)
}

func TestScanMagicStringBeforeExpr(t *testing.T) {
	src := `%magic
"x"`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, Magic, `%magic`)
	AssertNextTokenLit(t, scanner, 8, String, `"x"`)
	AssertEOF(t, scanner)
}

func TestScanMagicStringAfterExpr(t *testing.T) {
	src := `"x"
%magic`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, `"x"`)
	AssertNextTokenLit(t, scanner, 5, Magic, `%magic`)
	AssertEOF(t, scanner)
}

func TestScanCellMagicString(t *testing.T) {
	src := `"x"
%%sh`
	t.Log(src)
	scanner := NewScanner([]byte(src), opts)
	AssertNextTokenLit(t, scanner, 1, String, `"x"`)
	AssertNextTokenLit(t, scanner, 5, Magic, `%%sh`)
	AssertEOF(t, scanner)
}
