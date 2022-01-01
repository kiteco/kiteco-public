package pythonscanner

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Options represents configuration for the scanner
type Options struct {
	ScanComments      bool
	ScanNewLines      bool
	OneBasedPositions bool
	KeepEOFIndent     bool
	Label             string // Label is the filename for error reporting
}

// DefaultOptions is an Options object with default values.
var DefaultOptions = Options{
	ScanComments: false,
	ScanNewLines: true,
}

// A Scanner holds the scanner's internal state while processing
// a given text.  It can be allocated as part of another data
// structure but must be initialized via Init before use.
//
type Scanner struct {
	// immutable state
	src  []byte  // source
	opts Options // scanner options

	// scanning state
	ch       rune  // current character
	offset   int   // character offset
	rdOffset int   // reading offset (position after current character)
	prevTok  Token // previous token, set to Illegal before first

	// public state - ok to modify
	Errs errors.Errors // errors encountered
}

// Word represents a token together with its position and literal content
type Word struct {
	Token   Token
	Begin   token.Pos
	End     token.Pos
	Literal string
}

// String gets a string representation of a lexical symbol
func (w Word) String() string {
	switch {
	case w.Token.IsLiteral():
		s := w.Token.String()
		if len(w.Literal) > 50 || strings.Contains(w.Literal, "\n") {
			return s + fmt.Sprintf("[%d chars]", len(w.Literal))
		}
		return s + "[" + w.Literal + "]"
	case w.Token.IsOperator(), w.Token.IsKeyword():
		return `"` + w.Token.String() + `"`
	case w.Token == Illegal:
		return w.Token.String() + "[" + w.Literal + "]"
	default:
		return w.Token.String()
	}
}

// Valid checks if the Word is valid; it is intended for use in testing
func (w Word) Valid() bool {
	if w.Begin > w.End {
		return false
	}
	if canHaveLiteral(w.Token) {
		return true
	}
	return w.Literal == ""
}

func canHaveLiteral(tok Token) bool {
	if tok.IsWhitespace() || tok.IsLiteral() || tok == Comment || tok == Illegal || tok == BadToken || tok == Magic {
		// TODO(naman) should BadToken have text?
		return true
	}
	return false
}

// ScanError represents an error encountered during scanning
type ScanError struct {
	Message  string
	Position token.Pos
}

// Error returns a string representation of the error
func (e ScanError) Error() string {
	return fmt.Sprintf("%d: %s", e.Position, e.Message)
}

// Scan extracts all tokens from the buffer. Even if there is an error, a token
// stream will also be returned, though it may (or may not) contain Illegal tokens.
// NOTE: the scanner expects `buf` to be UTF8 encoded
func Scan(buf []byte) ([]Word, error) {
	scanner := NewScanner(buf, Options{
		ScanComments: true,
		ScanNewLines: true,
	})
	var words []Word
	for {
		begin, end, tok, lit := scanner.Scan()
		words = append(words, Word{
			Begin:   begin,
			End:     end,
			Token:   tok,
			Literal: lit,
		})
		if tok == EOF {
			break
		}
	}
	return words, scanner.Errs
}

// NewScanner creates a scanner to tokenize the text src by setting the
// scanner at the beginning of src. The scanner uses the file set file
// for position information and it adds line information for each line.
// It is ok to re-use the same file when re-scanning the same file as
// line information which is already present is ignored. Init causes a
// panic if the file size does not match the src size.
//
// Calls to Scan will track encountered errors in the Errors field.
//
// Note that Init may call err if there is an error in the first character
// of the file.
//
func NewScanner(src []byte, opts Options) *Scanner {
	s := &Scanner{
		src:      src,
		opts:     opts,
		ch:       ' ',
		offset:   0,
		rdOffset: 0,
		prevTok:  Illegal,
	}

	s.next()
	if s.ch == bom {
		s.next() // ignore Bom at file beginning
	}

	return s
}

const bom = 0xFeff // byte order mark, only permitted as very first character

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
//
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character Nul")
		case r >= 0x80:
			// not Ascii
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal Utf-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		s.ch = -1 // eof
	}
}

func (s *Scanner) error(offs int, msg string) {
	s.Errs = errors.Append(s.Errs, PosError{token.Pos(offs + 1), msg})
}

func (s *Scanner) scanComment() string {
	// initial '#' already consumed
	offs := s.offset - 1 // position of initial '#'
	hasCr := false

	for s.ch != '\n' && s.ch >= 0 {
		if s.ch == '\r' {
			hasCr = true
		}
		s.next()
	}

	lit := s.src[offs:s.offset]
	if hasCr {
		lit = stripCr(lit)
	}

	return string(lit)
}

// IsLetter checks if the given rune is a valid "letter" according to the python spec
func IsLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

// IsDigit checks if the given rune is a valid "digit" according to the python spec
func IsDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch)
}

func parseStringPrefix(s string) (stringModifiers, bool) {
	// This function recognizes the valid string literal prefixes as defined by
	// https://docs.python.org/2/reference/lexical_analysis.html#string-literals
	var mod stringModifiers
	if len(s) != 1 && len(s) != 2 {
		return mod, false
	}
	for _, ch := range s {
		switch ch {
		case 'r', 'R':
			mod.raw = true
		case 'b', 'B':
			mod.bytes = true
		case 'u', 'U':
			mod.unicode = true
		case 'f', 'F':
			mod.formatted = true
		default:
			return mod, false
		}
	}
	return mod, true
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func stripCr(b []byte) []byte {
	c := make([]byte, len(b))
	i := 0
	for _, ch := range b {
		if ch != '\r' {
			c[i] = ch
			i++
		}
	}
	return c[:i]
}

func (s *Scanner) scanIdentifier() string {
	offs := s.offset
	for IsLetter(s.ch) || IsDigit(s.ch) {
		s.next()
	}
	identBuf := s.src[offs:s.offset]
	// unsafe code taken from Go stdlib strings.Builder.String()
	return *(*string)(unsafe.Pointer(&identBuf))
}

func (s *Scanner) scanMantissa(base int) {
	for digitVal(s.ch) < base {
		s.next()
	}
}

func (s *Scanner) scanNumber(seenDecimalPoint bool) (Token, string) {
	// digitVal(s.ch) < 10
	offs := s.offset
	tok := Int

	if seenDecimalPoint {
		offs--
		tok = Float
		s.scanMantissa(10)
		goto exponent
	}

	if s.ch == '0' {
		// int or float
		offs := s.offset
		s.next()
		if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.error(offs, "illegal hexadecimal number")
			}
			goto long
		} else if s.ch == 'o' || s.ch == 'O' {
			s.next()
			s.scanMantissa(8)
			goto long
		} else if s.ch == 'b' || s.ch == 'B' {
			s.next()
			s.scanMantissa(2)
			goto long
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(8)
			if s.ch == '8' || s.ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(10)
			}
			if s.ch == 'l' || s.ch == 'L' {
				goto long
			} else if s.ch == '.' {
				goto fraction
			} else if s.ch == 'e' || s.ch == 'E' {
				goto exponent
			} else if s.ch == 'j' || s.ch == 'J' {
				goto imaginary
			}

			// octal int
			if seenDecimalDigit {
				s.error(offs, "illegal octal number")
			}
		}
		goto exit
	}

	// decimal int or float
	s.scanMantissa(10)

fraction:
	if s.ch == '.' {
		tok = Float
		s.next()
		s.scanMantissa(10)
	}

exponent:
	if s.ch == 'e' || s.ch == 'E' {
		tok = Float
		s.next()
		if s.ch == '-' || s.ch == '+' {
			s.next()
		}
		s.scanMantissa(10)
	}

imaginary:
	if s.ch == 'j' || s.ch == 'J' {
		tok = Imag
		s.next()
	}

long:
	if s.ch == 'l' || s.ch == 'L' {
		// long integer
		tok = Long
		s.next()
	}

exit:
	return tok, string(s.src[offs:s.offset])
}

type stringModifiers struct {
	raw       bool // indicates a raw string literal like r"foo" or R"spam"
	bytes     bool // indicates a bytes literal like b"foo" or B"spam"
	unicode   bool // indicates a unicode literal like u"foo" or U"spam"
	formatted bool // indicates a formatted string literal like f"name: {name}" or F"count: {n}"
}

// scan a string
func (s *Scanner) scanString(quote rune, modifiers stringModifiers) string {
	// opening quote already consumed
	offs := s.offset - 1

	// determine whether we are at a triple-quote string
	if s.ch == quote {
		s.next()

		// Now we have two quotes: either it was an empty string or
		// the beginning of a triple-quoted string
		if s.ch == quote {
			s.next()
			return s.scanMultiLineString(quote, modifiers)
		}
		return string(s.src[offs:s.offset])
	}

	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == quote {
			break
		}
		if ch == '\\' {
			// We have a backslash so skip the next character and keep
			// scanning. Note that this is still valid even for multi-char
			// escape sequences since valid escape sequences cannot include
			// quotes or backslashes after the first two chars. Do not bother
			// interpreting string literals since we do not need to know
			// their interpreted value.
			s.next()
		}
	}

	return string(s.src[offs:s.offset])
}

// scan a multi-line string
func (s *Scanner) scanMultiLineString(quote rune, modifiers stringModifiers) string {
	// opening three chars already consumed
	offs := s.offset - 3

	var numQuotes int
	hasCr := false
	for {
		ch := s.ch
		if ch < 0 {
			s.error(offs, "multi-line string literal not terminated")
			break
		}
		s.next()
		if ch == quote {
			numQuotes++
			if numQuotes == 3 {
				break
			}
		} else {
			numQuotes = 0
		}
		if ch == '\r' {
			hasCr = true
		}
		if ch == '\\' {
			// We have a backslash so skip the next character and keep
			// scanning. Note that this is still valid even for multi-char
			// escape sequences since valid escape sequences cannot include
			// quotes or backslashes after the first two chars. Do not bother
			// interpreting string literals since we do not need to know
			// their interpreted value.
			s.next()
		}
	}

	lit := s.src[offs:s.offset]
	if hasCr {
		lit = stripCr(lit)
	}

	return string(lit)
}

func (s *Scanner) scanWhitespace() string {
	offs := s.offset
	// '\u00a0' -> no break whitespace
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\f' || s.ch == '\v' || s.ch == '\u00a0' {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

// Helper functions for scanning multi-byte tokens such as >> += >>= .
// Different routines recognize different length toki based on matches
// of chi. If a ends in '=', the result is tok1 or tok3
// respectively. Otherwise, the result is tok0 if there was no other
// matching character, or tok2 if the matching character was ch2.

func (s *Scanner) switch2(tok0, tok1 Token) Token {
	if s.ch == '=' {
		s.next()
		return tok1
	}
	return tok0
}

func (s *Scanner) switch3(tok0, tok1 Token, ch2 rune, tok2 Token) Token {
	if s.ch == '=' {
		s.next()
		return tok1
	}
	if s.ch == ch2 {
		s.next()
		return tok2
	}
	return tok0
}

func (s *Scanner) switch4(tok0, tok1 Token, ch2 rune, tok2, tok3 Token) Token {
	if s.ch == '=' {
		s.next()
		return tok1
	}
	if s.ch == ch2 {
		s.next()
		if s.ch == '=' {
			s.next()
			return tok3
		}
		return tok2
	}
	return tok0
}

// Scan scans the next and returns the position, the
// and its literal string if applicable. The source end is indicated by
// EOF.
//
// If the returned is a literal (Ident, Int, Float,
// Imag, Char, String) or Comment, the literal string
// has the corresponding value.
//
// If the returned is a keyword, the literal string is the keyword.
//
// If the returned is Semicolon, the corresponding
// literal string is ";" if the semicolon was present in the source,
// and "\n" if the semicolon was inserted because of a newline or
// at EOF.
//
// If the returned is Illegal, the literal string is the
// offending character.
//
// In all other cases, Scan returns an empty literal string.
//
// For more tolerant parsing, Scan will return a valid if
// possible even if a syntax error was encountered. Thus, even
// if the resulting sequence contains no illegal tokens,
// a client may not assume that no error occurred. Instead it
// must check that s.Errs == nil.
//
// Scan adds line information to the file added to the file
// set with Init. Token positions are relative to that file
// and thus relative to the file set.
//
func (s *Scanner) Scan() (begin, end token.Pos, tok Token, lit string) {
rescan:
	if tok != Illegal {
		// handle the cases when a token was found but not returned (via goto)
		s.prevTok = tok
	}

	s.scanWhitespace()

	// current start
	begin = token.Pos(s.offset + 1) // +1 because token.File is 1-based

	// determine value
	switch ch := s.ch; {
	case IsLetter(ch):
		lit = s.scanIdentifier()
		if mod, ok := parseStringPrefix(lit); ok && (s.ch == '"' || s.ch == '\'') {
			quote := s.ch
			s.next()
			lit += s.scanString(quote, mod)
			tok = String
		} else {
			tok = Lookup(lit)
			if !canHaveLiteral(tok) {
				lit = ""
			}
		}
	case '0' <= ch && ch <= '9':
		tok, lit = s.scanNumber(false)
	default:
		s.next() // always make progress
		switch ch {
		case -1:
			tok = EOF
		case '\\':
			if s.ch == '\r' || s.ch == '\n' {
				tok = LineContinuation
				for s.ch == '\n' || s.ch == '\r' {
					// keep advancing until we have removed all new lines
					s.next()
				}
				if !s.opts.ScanNewLines {
					goto rescan
				}
			} else {
				s.error(int(begin-1), fmt.Sprintf("backslash not followed by newline"))
			}
		case '\n':
			tok = NewLine
			if s.ch == '\r' {
				// if \n is followed by \r then ignore the \r (e.g interpret \n\r as an end-of-line sequence)
				// SEE: https://docs.python.org/3/reference/lexical_analysis.html in particular Physical lines section.
				// NOTE: we intentionally ignore spec above and interpret \r\n as a single "physical line", this is not
				// strictly neccesary since it is unclear what the python interpreter would do here.
				s.next()
			}
			lit = s.scanWhitespace()
			if !s.opts.ScanNewLines {
				goto rescan
			}
		case '\r':
			tok = NewLine
			if s.ch == '\n' {
				// if \r is followed by \n then ignore the \n (e.g interpret \r\n as an end-of-line sequence per the spec)
				// SEE: https://docs.python.org/3/reference/lexical_analysis.html in particular Physical lines section.
				s.next()
			}
			lit = s.scanWhitespace()
			if !s.opts.ScanNewLines {
				goto rescan
			}
		case '"', '\'':
			tok = String
			lit = s.scanString(ch, stringModifiers{})
		case '#':
			tok = Comment
			lit = s.scanComment()
			if !s.opts.ScanComments {
				goto rescan
			}
		case '.':
			if '0' <= s.ch && s.ch <= '9' {
				tok, lit = s.scanNumber(true)
			} else {
				tok = Period
			}
		case ',':
			tok = Comma
		case ';':
			tok = Semicolon
		case '(':
			tok = Lparen
		case ')':
			tok = Rparen
		case '[':
			tok = Lbrack
		case ']':
			tok = Rbrack
		case '{':
			tok = Lbrace
		case '}':
			tok = Rbrace
		case '@':
			tok = At
		case '`':
			tok = Backtick
		case ':':
			tok = Colon
		case '+':
			tok = s.switch2(Add, AddAssign)
		case '-':
			if s.ch == '>' {
				s.next()
				tok = Arrow
			} else {
				tok = s.switch2(Sub, SubAssign)
			}
		case '*':
			tok = s.switch4(Mul, MulAssign, '*', Pow, PowAssign)
		case '/':
			tok = s.switch4(Div, DivAssign, '/', Truediv, TruedivAssign)
		case '<':
			if s.ch == '>' {
				s.next()
				tok = Lg
			} else {
				tok = s.switch4(Lt, Le, '<', BitLshift, BitLshiftAssign)
			}
		case '>':
			tok = s.switch4(Gt, Ge, '>', BitRshift, BitRshiftAssign)
		case '%':
			tok = s.switch2(Pct, PctAssign)

			// special-case for IPython's "magic" lines: if the % is the first symbol
			// on a line and is followed by "%" or a letter, treat it as a comment.
			// Line continuations are reported as LineContinuation tokens, not as NewLine,
			// and so won't result in a magic line.
			if s.prevTok == Illegal || s.prevTok == NewLine {
				if s.ch == '%' || IsLetter(s.ch) {
					tok = Magic
					lit = s.scanComment()
					if !s.opts.ScanComments {
						goto rescan
					}
				}
			}

		case '=':
			tok = s.switch2(Assign, Eq)
		case '&':
			tok = s.switch2(BitAnd, BitAndAssign)
		case '|':
			tok = s.switch2(BitOr, BitOrAssign)
		case '^':
			tok = s.switch2(BitXor, BitXorAssign)
		case '~':
			tok = BitNot
		case '!':
			if s.ch == '=' {
				s.next()
				tok = Ne
			} else {
				s.error(int(begin), "'!' not allowed outside '!='")
			}
		default:
			// next reports unexpected Boms - don't repeat
			if ch != bom {
				s.error(int(begin), fmt.Sprintf("illegal character %#U", ch))
			}
			tok = Illegal
			lit = fmt.Sprintf("%q", ch)
		}
	}

	end = token.Pos(s.offset + 1) // +1 because token.File is 1-based
	if !s.opts.OneBasedPositions {
		begin--
		end--
	}
	s.prevTok = tok
	return
}
