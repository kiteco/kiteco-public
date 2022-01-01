package pythonscanner

import (
	"io/ioutil"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var Name = Ident

func AssertWords(t *testing.T, src string, expected []Word, actual []Word) {
	t.Logf("\nSource:\n%s\n", src)
	t.Logf("\nBytes:\n%v\n", []byte(src))
	for i := 0; i < len(expected) && i < len(actual); i++ {
		e := expected[i]
		a := actual[i]
		if e != a {
			t.Errorf("Expected {%d, %d, Literal: %s, Token: %s} != {%d, %d, Literal: %s, Token: %s} Actual\n",
				e.Begin, e.End, e.Literal, e.Token,
				a.Begin, a.End, a.Literal, a.Token)
		}
	}
	if len(expected) < len(actual) {
		t.Errorf("Got extra tokens:\n")
		for i := len(expected); i < len(actual); i++ {
			a := actual[i]
			t.Errorf("%d %d %s (lit) %s (token)\n", a.Begin, a.End, a.Literal, a.Token)
		}
	}

	if len(actual) < len(expected) {
		t.Errorf("Missing tokens:\n")
		for i := len(actual); i < len(expected); i++ {
			e := expected[i]
			t.Errorf("%d %d %s (lit) %s (token)\n", e.Begin, e.End, e.Literal, e.Token)
		}
	}

	for _, w := range actual {
		if !w.Valid() {
			t.Errorf("invalid word: begin: %d end: %d tok: %s lit: '%s'", w.Begin, w.End, w.Token.String(), w.Literal)
		}
	}
}

func TestLexer_SingleLine(t *testing.T) {
	src := `foo(bar)`
	expected := []Token{Name, Lparen, Name, Rparen, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_Indents(t *testing.T) {
	src := `
if foo:
   bar
   baz 456
`
	expected := []Token{If, Name, Colon, NewLine, Indent, Name, NewLine, Name, Int, NewLine, Dedent, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_Indents_NoFinalNewline(t *testing.T) {
	src := `
if foo:
   bar
   baz 456`
	expected := []Token{If, Name, Colon, NewLine, Indent, Name, NewLine, Name, Int, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_Dedents(t *testing.T) {
	src := `
if ham:
  123
456
`
	expected := []Token{If, Name, Colon, NewLine, Indent, Int, NewLine, Dedent, Int, NewLine, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_EmptyLine(t *testing.T) {
	src := `
if foo:
  123

  456
`
	expected := []Token{If, Name, Colon, NewLine, Indent, Int, NewLine, Int, NewLine, Dedent, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_EmptyLineWithComment(t *testing.T) {
	src := `
if foo:
  123
# does not count
  456
`
	expected := []Token{If, Name, Colon, NewLine, Indent, Int, Comment, NewLine, Int, NewLine, Dedent, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_Parens(t *testing.T) {
	src := `
a = [
  1,
  2,
]
`
	expected := []Token{Name, Assign, Lbrack, Int, Comma, Int, Comma, Rbrack, NewLine, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_LineContinuations(t *testing.T) {
	src := `
a = 1 + \
    2
`
	expected := []Token{Name, Assign, Int, Add, Int, NewLine, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))
	require.NoError(t, err)

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_MissingParen(t *testing.T) {
	src := `
foo(:
	pass
bar()
`
	expected := []Token{Name, Lparen, Colon, Pass, NewLine, Name, Lparen, Rparen, NewLine, EOF}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))

	require.Error(t, err)
	require.Equal(t, 1, err.(errors.Errors).Len())

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestLexer_BadIndentationLevel(t *testing.T) {
	// this should generate an "invalid indentation level" error but we
	// should still get a complete token stream with indents matching dedents
	src := `
class x:
	class y:
		class z:
  bad
`
	expected := []Token{
		Class, Ident, Colon, NewLine,
		Indent, Class, Ident, Colon, NewLine,
		Indent, Class, Ident, Colon, NewLine,
		Dedent, Ident, NewLine,
		Dedent,
		EOF,
	}

	t.Log(src)
	words, err := Lex([]byte(src), Options{})
	require.Len(t, words, len(expected))

	require.Error(t, err)
	require.Equal(t, 1, err.(errors.Errors).Len())

	for i, word := range words {
		assert.Equal(t, expected[i].String(), word.Token.String())
	}
}

func TestListLexer_Empty(t *testing.T) {
	lexer := NewListLexer(nil)
	expected := Word{
		Begin:   0,
		End:     0,
		Token:   EOF,
		Literal: EOF.String(),
	}
	actual := *lexer.Next()
	assert.Equal(t, expected, actual)
}

func TestListLexer_End(t *testing.T) {
	words := []Word{
		Word{
			Begin:   0,
			End:     5,
			Token:   Name,
			Literal: "hello",
		},
		Word{
			Begin:   5,
			End:     11,
			Token:   Name,
			Literal: "world",
		},
		Word{
			Begin:   11,
			End:     11,
			Token:   EOF,
			Literal: EOF.String(),
		},
	}
	listLexer := NewListLexer(words)
	var actual []Word
	for i := 0; i < 4; i++ {
		actual = append(actual, *listLexer.Next())
	}

	expected := []Word{
		Word{
			Begin:   0,
			End:     5,
			Token:   Name,
			Literal: "hello",
		},
		Word{
			Begin:   5,
			End:     11,
			Token:   Name,
			Literal: "world",
		},
		Word{
			Begin:   11,
			End:     11,
			Token:   EOF,
			Literal: EOF.String(),
		},
		Word{
			Begin:   11,
			End:     11,
			Token:   EOF,
			Literal: EOF.String(),
		},
	}
	assert.Equal(t, expected, actual)
}

func AssertNextWord(t *testing.T, lexer Lexer, expected Word) {
	actual := lexer.Next()
	if assert.NotNil(t, actual, "got nil word, expected %v", expected) {
		return
	}

	assert.Equal(t, expected, *actual, "expected %v got %v", expected, *actual)
}

func TestLexDotted(t *testing.T) {
	src := `a = foo.bar.car`
	lexer := NewStreamLexer([]byte(src), opts)

	AssertNextWord(t, lexer, Word{
		Begin:   0,
		End:     1,
		Token:   Ident,
		Literal: "a",
	})

	AssertNextWord(t, lexer, Word{
		Begin:   2,
		End:     3,
		Token:   Assign,
		Literal: "=",
	})

	AssertNextWord(t, lexer, Word{
		Begin:   4,
		End:     7,
		Token:   Ident,
		Literal: "foo",
	})

	AssertNextWord(t, lexer, Word{
		Begin:   7,
		End:     8,
		Token:   Period,
		Literal: ".",
	})

	AssertNextWord(t, lexer, Word{
		Begin:   8,
		End:     11,
		Token:   Ident,
		Literal: "bar",
	})

	AssertNextWord(t, lexer, Word{
		Begin:   11,
		End:     12,
		Token:   Period,
		Literal: ".",
	})

	AssertNextWord(t, lexer, Word{
		Begin:   12,
		End:     15,
		Token:   Ident,
		Literal: "car",
	})
}

func TestLexNoBreakWhiteSpace(t *testing.T) {
	src := "def foo():\n\u00a0\u00a0\u00a0\u00a0pass"

	expected := []Word{
		Word{
			Begin: 1,
			End:   4,
			Token: Def,
		},
		Word{
			Begin:   5,
			End:     8,
			Token:   Ident,
			Literal: "foo",
		},
		Word{
			Begin: 8,
			End:   9,
			Token: Lparen,
		},
		Word{
			Begin: 9,
			End:   10,
			Token: Rparen,
		},
		Word{
			Begin: 10,
			End:   11,
			Token: Colon,
		},
		Word{
			Begin: 11,
			End:   20,
			Token: NewLine,
		},
		Word{
			Begin: 20,
			End:   24,
			Token: Indent,
		},
		Word{
			Begin: 20,
			End:   24,
			Token: Pass,
		},
		Word{
			Begin: 24,
			End:   24,
			Token: EOF,
		},
	}

	words, err := Lex([]byte(src), opts)
	require.NoError(t, err)

	assert.Equal(t, expected, words)
}

func TestCarriageReturnLinefeed(t *testing.T) {
	src := "def f():\r\n\tpass"

	expected := []Word{
		Word{
			Begin: 1,
			End:   4,
			Token: Def,
		},

		Word{
			Begin:   5,
			End:     6,
			Token:   Ident,
			Literal: "f",
		},
		Word{
			Begin: 6,
			End:   7,
			Token: Lparen,
		},
		Word{
			Begin: 7,
			End:   8,
			Token: Rparen,
		},
		Word{
			Begin: 8,
			End:   9,
			Token: Colon,
		},
		Word{
			// CR, LF, \t
			// 9   10  11
			// SO src[begin:end] = [CR, LF, \t]
			// SO end = 12
			Begin: 9,
			End:   12,
			Token: NewLine,
		},
		Word{
			Begin: 12,
			End:   16,
			Token: Indent,
		},
		Word{
			Begin: 12,
			End:   16,
			Token: Pass,
		},
		Word{
			Begin: 16,
			End:   16,
			Token: EOF,
		},
	}

	words, err := Lex([]byte(src), opts)
	require.NoError(t, err)

	AssertWords(t, src, expected, words)
}

func TestLinefeedCarriageReturn(t *testing.T) {
	src := "def f():\n\r\tpass"

	expected := []Word{
		Word{
			Begin: 1,
			End:   4,
			Token: Def,
		},

		Word{
			Begin:   5,
			End:     6,
			Token:   Ident,
			Literal: "f",
		},
		Word{
			Begin: 6,
			End:   7,
			Token: Lparen,
		},
		Word{
			Begin: 7,
			End:   8,
			Token: Rparen,
		},
		Word{
			Begin: 8,
			End:   9,
			Token: Colon,
		},
		Word{
			// LF, CR, \t
			// 9   10  11
			// SO src[begin:end] = [LF, CR, \t]
			// SO end = 12
			Begin: 9,
			End:   12,
			Token: NewLine,
		},
		Word{
			Begin: 12,
			End:   16,
			Token: Indent,
		},
		Word{
			Begin: 12,
			End:   16,
			Token: Pass,
		},
		Word{
			Begin: 16,
			End:   16,
			Token: EOF,
		},
	}

	words, err := Lex([]byte(src), opts)
	require.NoError(t, err)

	AssertWords(t, src, expected, words)
}

// Cannot figure out why this file causes an infinite lex
// if we do not check for illegal Tokens when lexing
func TestIllegalCharacters(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/inf_lex.py")
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		Lex(src, Options{})
		done <- struct{}{}
	}()

	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout exceeded")
	case <-done:
	}
}

func TestIndentStack(t *testing.T) {
	s := newIndentStack(32)
	require.Equal(t, cap(s.levels), 32)
	beforePtr := ((*reflect.SliceHeader)((unsafe.Pointer)(&s.levels))).Data

	for i := 0; i < 32; i++ {
		s.push(i)
		require.Equal(t, i, s.peek())
	}
	require.Equal(t, 31, s.pop())
	require.Equal(t, 30, s.pop())
	s.push(32)
	require.Equal(t, 32, s.peek())
	require.Equal(t, 32, s.pop())
	require.Equal(t, 29, s.pop())

	afterPtr := ((*reflect.SliceHeader)((unsafe.Pointer)(&s.levels))).Data
	require.Equal(t, beforePtr, afterPtr)
}

func TestWordQueue(t *testing.T) {
	q := newWordQueue(16)
	require.Equal(t, cap(q.ring), 16)
	beforePtr := ((*reflect.SliceHeader)((unsafe.Pointer)(&q.ring))).Data

	for i := 0; i < 16; i++ {
		q.add(Word{Token: Token(i)})
	}
	for i := 0; i < 8; i++ {
		require.Equal(t, Word{Token: Token(i)}, q.remove())
	}
	for i := 16; i < 24; i++ {
		q.add(Word{Token: Token(i)})
	}
	for i := 8; i < 16; i++ {
		require.Equal(t, Word{Token: Token(i)}, q.remove())
	}

	afterPtr := ((*reflect.SliceHeader)((unsafe.Pointer)(&q.ring))).Data
	require.Equal(t, beforePtr, afterPtr)
}
