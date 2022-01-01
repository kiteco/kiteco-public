package pythonscanner

import (
	"bytes"
	"errors"
	"go/token"
	"unicode"
)

// Increment represents a contiguous update to a buffer
type Increment struct {
	// Begin position in previous buffer
	Begin token.Pos
	// End position in previous buffer
	End token.Pos
	// Replacement of oldbuffer[Begin:End]
	Replacement []byte
}

// Incremental builds and updates a token stream in response to updates
type Incremental struct {
	opts Options
	buf  []byte
	// lines are the positions of the newline (return carriage) runes in the buffer,
	// these are not the same as the line offset used by token.File.
	lines []int
	words []Word
}

// NewIncrementalFromBuffer returns a pointer to an Incremental Lexer using the provided buffer.
// NOTE: the lexer expects `buf` to be UTF8 encoded
func NewIncrementalFromBuffer(buf []byte, opts Options) *Incremental {
	opts.ScanNewLines = true
	opts.ScanComments = true
	opts.OneBasedPositions = false

	words, _ := Lex(buf, opts)
	return NewIncremental(buf, words, opts)
}

// NewIncremental returns an Incremental Lexer using the provided buffer and words
// the provided slice of words should not be modified outside of a single go routine.
// NOTE: the lexer expects `buf` to be UTF8 encoded
func NewIncremental(buf []byte, words []Word, opts Options) *Incremental {
	opts.ScanNewLines = true
	opts.ScanComments = true
	opts.OneBasedPositions = false

	return &Incremental{
		opts:  opts,
		buf:   buf,
		lines: lines(buf),
		words: words,
	}
}

// Buffer returns a reference to the underlying buffer,
// this buffer should be considered read only and is not safe to access from multiple go routines.
func (l *Incremental) Buffer() []byte {
	return l.buf
}

// Words in the buffer. This returns a reference
// to the underlying slice of words and should be
// considered READ ONLY. It is safe to read from
// this slice in multiple go routines, however it
// is not safe to call Words() and Update() from separate go routines
// without locking.
func (l *Incremental) Words() []Word {
	return l.words
}

var errRelex = errors.New("relex")

// Update modifies the token stream to match the given edit to the underlying buffer.
// It is not safe to call Update() and Words() from separate go routines without locking.
func (l *Incremental) Update(update *Increment) error {
	if len(l.buf) == 0 || len(l.lines) == 0 {
		// buffer empty or original text contained no newlines,
		// trigger relex
		return errRelex
	}

	// Find line that we need to update
	startIdx, endIdx := -1, -1
	for i := 1; i < len(l.lines); i++ {
		if int(update.Begin) > l.lines[i-1] {
			if int(update.End) <= l.lines[i] {
				startIdx = i - 1
				endIdx = i
				break
			}
		}
	}

	if startIdx == -1 || endIdx == -1 {
		// This is to simplify corner cases, occurs if:
		// 1) begining of update is at a new line,
		// 2) no newline before the update,
		// 3) no newline after the update.
		// With these simplifications, each update now
		// has the form "...\n`update`\n...".
		return errRelex
	}

	// +1 since we store the position of the new line character
	// thus start is the postion of the first character on the line to be replaced, and
	// end is the position of the first character immediately after the line to be replaced.
	start, end := l.lines[startIdx]+1, l.lines[endIdx]
	startPos, endPos := token.Pos(start), token.Pos(end)

	if len(bytes.TrimSpace(l.buf[start:end])) == 0 {
		// inserting on empty line, trigger relex
		return errRelex
	}

	if hasSpecialWords(wordsInRegion(l.words, startPos, endPos-1)...) {
		// old words contain relex word, trigger relex
		return errRelex
	}

	// build chunk to scan
	scan := bytes.Join([][]byte{
		l.buf[start:update.Begin],
		update.Replacement,
		l.buf[update.End:end],
	}, nil)

	if len(scan) > 0 && unicode.IsSpace(rune(scan[0])) || unicode.IsSpace(rune(l.buf[start])) {
		// first character in line is whitespace,
		// trigger relex
		return errRelex
	}

	// scan
	newWords, err := Scan(scan)
	if err != nil {
		return err
	}

	if hasSpecialWords(newWords...) {
		// new words contain relex word, tirgger relex
		return errRelex
	}

	// newWords always contains EOF token,
	// if this is the only token then the user
	// must have deleted an entire line.
	if len(newWords) == 1 {
		return errRelex
	}

	// drop EOF token
	newWords = newWords[:len(newWords)-1]

	// build new buffer
	buf := bytes.Join([][]byte{
		l.buf[:start],
		scan,
		l.buf[end:],
	}, nil)

	offset := token.Pos(len(buf) - len(l.buf))

	l.buf = buf

	// build new token stream
	var idx int
	words := make([]Word, 0, len(l.words)+len(newWords))
	for ; idx < len(l.words); idx++ {
		word := l.words[idx]
		// always add newlines since lexer changes their begin and end
		// and already checked that they were not touched by the update.
		if word.End <= startPos || word.Token == NewLine {
			words = append(words, word)
		} else {
			break
		}
	}

	// Here we use the beginning of the prelude to account for the text that occured in the
	// buffer before the insertion.
	// We can't just use the offset here because we also need
	// to account for the words that appeared before the words we are going to insert.
	// e.g `print x + 1` -> `print x + 22`
	// the offset is 1 but the new words start at position 10.
	// We also cant just use the end of the last word from the old set of words because this
	// may miss some text that is not part of any token.
	// e.g `print x + 1` -> `print x + 22` the end of the last word (+)
	// is 9, but the begining of the new word (22) is 10.
	for _, word := range newWords {
		word.Begin += startPos
		word.End += startPos
		words = append(words, word)
	}

	// Here we need to use the offset because we could have deleted text
	// e.g `print x + 1` -> ` x + 1`
	// the begining of the prelude is 0, but
	// x originally has begin = 6 and we need it to be 1 when we are done.
	for ; idx < len(l.words); idx++ {
		word := l.words[idx]
		if word.Begin > endPos || word.Token == EOF || word.Token == NewLine {
			word.Begin += offset
			word.End += offset
			words = append(words, word)
		}
	}

	l.words = words

	// update lines
	for i := endIdx; i < len(l.lines); i++ {
		l.lines[i] += int(offset)
	}

	return nil
}

func hasSpecialWords(words ...Word) bool {
	for _, word := range words {
		switch word.Token {
		case Indent, Dedent, Lparen, Lbrace, Lbrack,
			Rparen, Rbrace, Rbrack, NewLine, Illegal, String:
			return true
		}
	}
	return false
}

func wordsInRegion(words []Word, from, to token.Pos) []Word {
	start, end := -1, -1
	for i := range words {
		if words[i].Begin >= to {
			end = i
			break
		}
		if start == -1 && words[i].Begin >= from {
			start = i
		}
	}
	if start == -1 {
		return nil
	}
	if end == -1 {
		return words[start:len(words)]
	}
	return words[start:end]
}

func lines(buf []byte) []int {
	var lines []int
	for i, c := range buf {
		if c == byte('\n') || c == byte('\r') {
			lines = append(lines, i)
		}
	}
	return lines
}
