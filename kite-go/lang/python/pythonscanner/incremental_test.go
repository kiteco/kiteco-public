package pythonscanner

import (
	"go/token"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -- Utils

func increment(begin, end int, replacement string) *Increment {
	return &Increment{
		Begin:       token.Pos(begin),
		End:         token.Pos(end),
		Replacement: []byte(replacement),
	}
}

// TODO(juan): figure out how to use this with TestIncrementalRandom
func assertUpdate(t *testing.T, src, newSrc string, update *Increment) {
	incr := NewIncrementalFromBuffer([]byte(src), opts)

	// assumes no relex for update
	if incr.Update(update) != nil {
		t.Error("\nError during update")
		return
	}

	// make sure underlying buffer is correct
	if len(newSrc) != len(incr.buf) {
		t.Errorf("\nExpected\n%v\nActual\n%v\n", string(newSrc), string(incr.buf))
		return
	}

	for i := range newSrc {
		if newSrc[i] != incr.buf[i] {
			t.Errorf("\nExpected\n%v\nActual\n%v\n", string(newSrc), string(incr.buf))
			return
		}
	}

	// make sure words are correct
	expectedWords, _ := Lex([]byte(newSrc), Options{
		ScanComments:      true,
		ScanNewLines:      true,
		OneBasedPositions: false,
	})
	if len(expectedWords) != len(incr.words) {
		t.Errorf("\nExpected\n%v\nActual\n%v\n", expectedWords, incr.words)
		return
	}

	for i := range expectedWords {
		if expectedWords[i] != incr.words[i] {
			t.Errorf("\nExpected %v,%v,%v,%v. Actual %v,%v,%v,%v.",
				expectedWords[i].Begin, expectedWords[i].End, expectedWords[i].Literal, expectedWords[i].Token,
				incr.words[i].Begin, incr.words[i].End, incr.words[i].Literal, incr.words[i].Token)
		}
	}

	// make sure lines are correct
	expectedLines := lines([]byte(newSrc))
	if len(incr.lines) != len(expectedLines) {
		t.Errorf("\nExpected\n%v\nActual\n%v\n", expectedLines, incr.lines)
		return
	}

	for i := range expectedLines {
		if expectedLines[i] != incr.lines[i] {
			t.Errorf("\nExpected\n%v\nActual\n%v\n", expectedLines, incr.lines)
			return
		}
	}
}

func assertRelex(t *testing.T, src string, update *Increment) {
	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, errRelex, incr.Update(update))
}

// -- Corner Cases Update Tests

func TestUpdate_EmptyInitialBuffer(t *testing.T) {
	src := ""

	update := increment(0, 0, "print x + 1")

	// empty initial buffer, trigger re lex
	assertRelex(t, src, update)
}

func TestUpdate_NoNewLines(t *testing.T) {
	src := "x + 1"

	update := increment(0, 0, "print ")

	// no new lines in text, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_InsertAtNewLine(t *testing.T) {
	src := "\nx + 1\n"

	update := increment(0, 0, "print ")

	// insert at a newline character, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_InsertAcossLines(t *testing.T) {
	src := "\nx = 1\nx + 1"

	update := increment(5, 7, "2; print ")

	// insert across lines, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_InsertNewLine(t *testing.T) {
	src := "\nx = 1\ny = 3"

	update := increment(6, 6, "\nprint x")

	// replacement contains newline, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_InsertMissingNewLineBefore(t *testing.T) {
	src := "x + 1\ny = 2\n"

	update := increment(0, 0, "print ")

	// This is to simplify corner cases,
	// no newline before update, trigger relex.
	assertRelex(t, src, update)
}

func TestUpdate_InsertMissingNewLineAfter(t *testing.T) {
	src := "\nprint hello\nprint"

	update := increment(17, 17, " x + 1")

	// This is to simplify corner cases,
	// no newline after update
	assertRelex(t, src, update)
}

func TestUpdate_InsertOnEmptyLine(t *testing.T) {
	src := "\nprint x + 1\n \n \n \nprint y + 1\n"

	update := increment(15, 15, "y = 1")

	// inserting on empty line, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_DeleteLine(t *testing.T) {
	src := "\nprint hello\nprint bar()\nfoo(car))\n"

	update := increment(13, 24, "")

	// deleting an entire line, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_DeleteFile(t *testing.T) {
	src := "\nprint hello\nprint bar()\nfoo(car))\n"

	update := increment(0, len(src), "")

	// deleting an entire file, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_Indent(t *testing.T) {
	src := "\nprint hello\nprint var()\nfoo"

	update := increment(13, 13, " ")

	// inserting indent triggers relex
	assertRelex(t, src, update)
}

func TestUpdate_InsertWhiteSpaceBeginLine(t *testing.T) {
	src := "print hello\n  print y + 1\nfoo"

	update := increment(13, 13, " ")

	// leading white space treated as indent by lexer
	// when account for new line, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_DeleteWhiteSpaceBeginLine(t *testing.T) {
	src := "print hello\n  print y + 1\nfoo"

	update := increment(12, 14, "")

	// leading white space treated as indent by lexer
	// when account for new line, even though white space
	// is deleted there will still be  a dedent on the
	// next line, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_EditWhiteSpaceBeginLine(t *testing.T) {
	src := "print hello\n print y + 1\nfoo"

	update := increment(12, 13, " ")

	// leading white space treated as indent by lexer
	// when account for new line, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_DeleteParen(t *testing.T) {
	src := "print x + 1\nprint(y + 2)\nlen(foo)"

	update := increment(17, 24, "")

	// line to modify contains paren, trigger relex
	assertRelex(t, src, update)
}

func TestUpdate_EditString(t *testing.T) {
	src := `
x = "this is some text"
`
	update := increment(6, 6, "hello, ")

	assertRelex(t, src, update)
}

// -- Insert Update Tests

func TestUpdate_InsertBeginLine(t *testing.T) {
	src := "\nx + 1\n"

	newSrc := "\nprint x + 1\n"

	update := increment(1, 1, "print ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertBeginLine1(t *testing.T) {
	src := "\nx + 1\ny = 2"

	newSrc := "\nprint x + 1\ny = 2"

	update := increment(1, 1, "print ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertBeginLine2(t *testing.T) {
	src := "print hello\nx + 1\ny = 2\n"

	newSrc := "print hello\nprint x + 1\ny = 2\n"

	update := increment(12, 12, "print ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertBeginLine3(t *testing.T) {
	src := "print hello(\nx + 1\ny = 2\n"

	newSrc := "print hello(\nprint x + 1\ny = 2\n"

	update := increment(13, 13, "print ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertBeginLine4(t *testing.T) {
	src := "print hello\nx + 1\n(y = 2\n"

	newSrc := "print hello\nprint x + 1\n(y = 2\n"

	update := increment(12, 12, "print ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertBeginLine5(t *testing.T) {
	src := "print hello(\nx + 1\n)y = 2\n"

	newSrc := "print hello(\nprint x + 1\n)y = 2\n"

	update := increment(13, 13, "print ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertEndLine(t *testing.T) {
	src := "\nprint\n"

	newSrc := "\nprint x + 1\n"

	update := increment(6, 6, " x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertEndLine1(t *testing.T) {
	src := "print hello\nprint\n"

	newSrc := "print hello\nprint x + 1\n"

	update := increment(17, 17, " x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertEndLine2(t *testing.T) {
	src := "print hello\nprint\nfoo(bar)"

	newSrc := "print hello\nprint x + 1\nfoo(bar)"

	update := increment(17, 17, " x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertEndLine3(t *testing.T) {
	src := "print hello(\nprint\nfoo(bar)"

	newSrc := "print hello(\nprint x + 1\nfoo(bar)"

	update := increment(18, 18, " x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertEndLine4(t *testing.T) {
	src := "print hello\nprint\n(foo(bar)"

	newSrc := "print hello\nprint x + 1\n(foo(bar)"

	update := increment(17, 17, " x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertEndLine5(t *testing.T) {
	src := "print hello(\nprint\nfoo(bar))"

	newSrc := "print hello(\nprint x + 1\nfoo(bar))"

	update := increment(18, 18, " x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertMiddleLine(t *testing.T) {
	src := "print hello\nprint y + 1\nfoo(bar)"

	newSrc := "print hello\nprint x + 1, y + 1\nfoo(bar)"

	update := increment(17, 17, " x + 1,")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertMiddleLine1(t *testing.T) {
	src := "print hello(\nprint y + 1\nfoo(bar)"

	newSrc := "print hello(\nprint x + 1, y + 1\nfoo(bar)"

	update := increment(18, 18, " x + 1,")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertMiddleLine2(t *testing.T) {
	src := "print hello(\nprint y + 1\nfoo(bar))"

	newSrc := "print hello(\nprint x + 1, y + 1\nfoo(bar))"

	update := increment(18, 18, " x + 1,")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertMiddleLine3(t *testing.T) {
	src := "print hello\nprint y + 1\n(foo(bar)"

	newSrc := "print hello\nprint x + 1, y + 1\n(foo(bar)"

	update := increment(17, 17, " x + 1,")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertWhiteSpaceEndLine(t *testing.T) {
	src := "print hello\nprint y + 1  \nfoo"

	newSrc := "print hello\nprint y + 1   \nfoo"

	update := increment(24, 24, " ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_InsertWhiteSpaceMiddleLine(t *testing.T) {
	src := "print hello\nprint  y + 1\nfoo"

	newSrc := "print hello\nprint   y + 1\nfoo"

	update := increment(18, 18, " ")

	assertUpdate(t, src, newSrc, update)
}

// -- Edit Update Tests

func TestUpdate_EditBeginLine(t *testing.T) {
	src := "print hello\nprnty + 1\nfoo(bar)"

	newSrc := "print hello\nprint y + 1\nfoo(bar)"

	update := increment(12, 16, "print ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditBeginLine1(t *testing.T) {
	src := "\nprint hello\nprnt y + 1\nfoo(bar)\n"

	newSrc := "\nprint hello\nprint x + 1\nfoo(bar)\n"

	update := increment(13, 19, "print x")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditBeginLine2(t *testing.T) {
	src := "\nprint (hello\nprnt y + 1\nfoo(bar)\n"

	newSrc := "\nprint (hello\nprint x + 1\nfoo(bar)\n"

	update := increment(14, 20, "print x")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditBeginLine3(t *testing.T) {
	src := "\nprint (hello\nprnt y + 1\n)foo(bar)\n"

	newSrc := "\nprint (hello\nprint x + 1\n)foo(bar)\n"

	update := increment(14, 20, "print x")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditBeginLine4(t *testing.T) {
	src := "\nprint (hello\nprnt y + 1\n(foo(bar)\n"

	newSrc := "\nprint (hello\nprint x + 1\n(foo(bar)\n"

	update := increment(14, 20, "print x")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditEndLine(t *testing.T) {
	src := "\nprint hello\nprint y + 1\nfoo(bar)\n"

	newSrc := "\nprint hello\nprint x + 1\nfoo(bar)\n"

	update := increment(19, 24, "x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditEndLine1(t *testing.T) {
	src := "\nprint hello\nprnt y + 1\nfoo(bar)\n"

	newSrc := "\nprint hello\nprint x + 1\nfoo(bar)\n"

	update := increment(15, 23, "int x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditEndLine2(t *testing.T) {
	src := "\nprint (hello\nprnt y + 1\n(foo(bar)\n"

	newSrc := "\nprint (hello\nprint x + 1\n(foo(bar)\n"

	update := increment(16, 24, "int x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditEndLine3(t *testing.T) {
	src := "\nprint (hello\nprnt y + 1\nfoo(bar)\n"

	newSrc := "\nprint (hello\nprint x + 1\nfoo(bar)\n"

	update := increment(16, 24, "int x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditEndLine4(t *testing.T) {
	src := "\nprint (hello\nprnt y + 1\nfoo(bar))\n"

	newSrc := "\nprint (hello\nprint x + 1\nfoo(bar))\n"

	update := increment(16, 24, "int x + 1")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditMiddleLine(t *testing.T) {
	src := "\nprint hello\npr y + 1\nfoo(bar)\n"

	newSrc := "\nprint hello\nprint x + 1\nfoo(bar)\n"

	update := increment(15, 17, "int x")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditMiddleLine1(t *testing.T) {
	src := "\nprint hello\nprnt y + 1\nfoo(bar)\n"

	newSrc := "\nprint hello\nprint x + 1\nfoo(bar)\n"

	update := increment(15, 21, "int x +")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditMiddleLine2(t *testing.T) {
	src := "\nprint hello(\nprnt y + 1\nfoo(bar)\n"

	newSrc := "\nprint hello(\nprint x + 1\nfoo(bar)\n"

	update := increment(16, 22, "int x +")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditMiddleLine3(t *testing.T) {
	src := "\nprint hello(\nprnt y + 1\nfoo(bar))\n"

	newSrc := "\nprint hello(\nprint x + 1\nfoo(bar))\n"

	update := increment(16, 22, "int x +")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditWhiteSpaceEndLine(t *testing.T) {
	src := "print hello\nprint y + 1  \nfoo"

	newSrc := "print hello\nprint y + 1 \nfoo"

	update := increment(23, 25, " ")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_EditWhiteSpaceMiddleLine(t *testing.T) {
	src := "print hello\nprint  y + 1\nfoo"

	newSrc := "print hello\nprint y + 1\nfoo"

	update := increment(17, 19, " ")

	assertUpdate(t, src, newSrc, update)
}

// -- Delete Update Tests

func TestUpdate_DeleteBeginLine(t *testing.T) {
	src := "\nprint hello\nprint y + 1\nfoo(bar())"

	newSrc := "\nprint hello\ny + 1\nfoo(bar())"

	update := increment(13, 19, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteBeginLine1(t *testing.T) {
	src := "\nprint hello\nprint y + 1\nfoo(bar())"

	newSrc := "\nprint hello\n1\nfoo(bar())"

	update := increment(13, 23, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteBeginLine2(t *testing.T) {
	src := "\nprint hello(\nprint y + 1\nfoo(bar())"

	newSrc := "\nprint hello(\n1\nfoo(bar())"

	update := increment(14, 24, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteBeginLine3(t *testing.T) {
	src := "\nprint hello(\nprint y + 1\n)foo(bar())"

	newSrc := "\nprint hello(\n1\n)foo(bar())"

	update := increment(14, 24, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteEndLine(t *testing.T) {
	src := "\nprint hello\nprint y + 1\nfoo(bar())"

	newSrc := "\nprint hello\nprint y\nfoo(bar())"

	update := increment(20, 24, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteEndLine1(t *testing.T) {
	src := "\nprint hello\nprint y + 1\nfoo(bar())"

	newSrc := "\nprint hello\nprint\nfoo(bar())"

	update := increment(18, 24, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteEndLine2(t *testing.T) {
	src := "\nprint (hello\nprint y + 1\nfoo(bar())"

	newSrc := "\nprint (hello\nprint\nfoo(bar())"

	update := increment(19, 25, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteEndLine3(t *testing.T) {
	src := "\nprint (hello\nprint y + 1\nfoo(bar()))"

	newSrc := "\nprint (hello\nprint\nfoo(bar()))"

	update := increment(19, 25, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteMiddleLine(t *testing.T) {
	src := "\nprint hello\nprint y + 1\nfoo(bar())"

	newSrc := "\nprint hello\nprint 1\nfoo(bar())"

	update := increment(19, 23, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteMiddleLine1(t *testing.T) {
	src := "\nprint hello\nprint y + 1\nfoo(bar())\n"

	newSrc := "\nprint hello\nprint 1\nfoo(bar())\n"

	update := increment(18, 22, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteMiddleLine2(t *testing.T) {
	src := "\n(print hello\nprint y + 1\nfoo(bar())\n"

	newSrc := "\n(print hello\nprint 1\nfoo(bar())\n"

	update := increment(19, 23, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteMiddleLine3(t *testing.T) {
	src := "\n(print hello\nprint y + 1\nfoo(bar()))\n"

	newSrc := "\n(print hello\nprint 1\nfoo(bar()))\n"

	update := increment(19, 23, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteWhiteSpaceEndLine(t *testing.T) {
	src := "print hello\nprint y + 1  \nfoo"

	newSrc := "print hello\nprint y + 1\nfoo"

	update := increment(23, 25, "")

	assertUpdate(t, src, newSrc, update)
}

func TestUpdate_DeleteWhiteSpaceMiddleLine(t *testing.T) {
	src := "print hello\nprint   y + 1\nfoo"

	newSrc := "print hello\nprint y + 1\nfoo"

	update := increment(18, 20, "")

	assertUpdate(t, src, newSrc, update)
}

// -- Random Tests

func TestRandomModifications(t *testing.T) {
	var files []string
	err := filepath.Walk("testdata/corpus/",
		func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && path.Base(p) != "inf_lex.py" {
				files = append(files, p)
			}
			return nil
		})
	require.Nil(t, err)
	rgen := rand.New(rand.NewSource(0))
	for _, file := range files {
		src, err := ioutil.ReadFile(file)
		require.NoError(t, err)

		failed, _, _, _ := IncrementalRandom(src, 500, rgen, false)

		if failed != 0 {
			t.Errorf("\nFile %s had %d fails", file, failed)
		}
	}
}

// -- Test lines

func TestLines_NoNewLines(t *testing.T) {
	src := "print x + 1"

	var expected []int

	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, expected, incr.lines)
}

func TestLines_LeadingNewLine(t *testing.T) {
	src := "\nprint x + 1"

	expected := []int{0}

	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, expected, incr.lines)
}

func TestLines_TrailingNewLine(t *testing.T) {
	src := "print x + 1\n"

	expected := []int{11}

	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, expected, incr.lines)
}

func TestLines_LeadingTrailingNewLine(t *testing.T) {
	src := "\nprint x + 1\n"

	expected := []int{0, 12}

	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, expected, incr.lines)
}

func TestLines_Empty(t *testing.T) {
	src := ""

	var expected []int

	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, expected, incr.lines)
}

func TestLines_Middle(t *testing.T) {
	src := "print x + 1\nprint x + 2"

	expected := []int{11}

	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, expected, incr.lines)
}

func TestLines_MiddleEmptyLine(t *testing.T) {
	src := "print x + 1\n\nprint x + 2"

	expected := []int{11, 12}

	incr := NewIncrementalFromBuffer([]byte(src), opts)

	assert.Equal(t, expected, incr.lines)
}
