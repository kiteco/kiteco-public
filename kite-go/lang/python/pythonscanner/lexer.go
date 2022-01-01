package pythonscanner

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Count counts the number of tokens, without allocating space for them.
// It counts EOF, so the output might be one greater than what you expect.
// NOTE: the lexer expects `buf` to be UTF8 encoded
func Count(buf []byte, opts Options) (int, error) {
	var count int

	lexer := newStreamLexer(buf, opts)
	var prev Word
	for prev.Token != EOF {
		prev = lexer.Next()
		count++
	}

	return count, lexer.errs
}

// Lex converts a byte array to an array of lexical elements
// NOTE: the lexer expects `buf` to be UTF8 encoded
func Lex(buf []byte, opts Options) ([]Word, error) {
	opts.ScanComments = true
	opts.ScanNewLines = true

	// preallocate token slice
	count, _ := Count(buf, opts)
	words := make([]Word, 0, count)

	lexer := newStreamLexer(buf, opts)
	for len(words) == 0 || words[len(words)-1].Token != EOF {
		words = append(words, lexer.Next())
	}
	return words, lexer.errs
}

// Lexer extracts words from python source
type Lexer interface {
	Next() *Word
}

// ListLexer reads words from a provided list of tokens
type ListLexer struct {
	Words []Word
	Curr  int
	eof   Word
}

// NewListLexer returns a pointer to a ListLexer that
// reads words from the provided slice of words.
// NOTE: the lexer expects `buf` to be UTF8 encoded
func NewListLexer(words []Word) *ListLexer {
	eof := Word{
		Token:   EOF,
		Literal: EOF.String(),
	}
	if len(words) > 0 {
		eof.Begin = words[len(words)-1].End
		eof.End = eof.Begin
	}
	return &ListLexer{
		Words: words,
		eof:   eof,
	}
}

// Next satisfies the Lexer interface
func (l *ListLexer) Next() *Word {
	if l.Curr < len(l.Words) {
		w := &l.Words[l.Curr]
		l.Curr++
		return w
	}
	return &l.eof
}

// StreamLexer extracts words from python source
type StreamLexer streamLexer

// NewStreamLexer constructs a lexer that will return tokens from the provided file
// NOTE: the lexer expects `buf` to be UTF8 encoded
func NewStreamLexer(src []byte, opts Options) *StreamLexer {
	return (*StreamLexer)(newStreamLexer(src, opts))
}

// Next gets the next lexical token
func (s *StreamLexer) Next() *Word {
	w := (*streamLexer)(s).Next()
	return &w
}

// -

type wordQueue struct {
	ring          []Word
	start, length int
}

func newWordQueue(sz int) *wordQueue {
	var ring []Word
	if sz > 0 {
		ring = make([]Word, sz)
	}
	return &wordQueue{
		ring: ring,
	}
}

func (q *wordQueue) resize() {
	newCapacity := q.length << 1
	if newCapacity == 0 {
		newCapacity = 8 // arbitrary
	}
	newBuf := make([]Word, newCapacity)

	if q.start+q.length <= len(q.ring) {
		copy(newBuf, q.ring[q.start:q.start+q.length])
	} else {
		n := copy(newBuf, q.ring[q.start:])
		copy(newBuf[n:], q.ring[:q.start+q.length-len(q.ring)])
	}

	q.start = 0
	q.ring = newBuf
}

func (q *wordQueue) add(w Word) {
	if q.length == len(q.ring) {
		q.resize()
	}

	idx := q.start + q.length
	if idx >= len(q.ring) {
		idx -= len(q.ring)
	}

	q.ring[idx] = w
	q.length++
}

func (q *wordQueue) remove() Word {
	if q.length == 0 {
		panic("wordQueue: remove called on empty queue")
	}
	w := q.ring[q.start]
	q.length--
	q.start++
	if q.start == len(q.ring) {
		q.start = 0
	}
	return w
}

type indentStack struct {
	levels []int
	length int
}

func newIndentStack(sz int) *indentStack {
	var levels []int
	if sz > 0 {
		levels = make([]int, sz)
	}
	return &indentStack{
		levels: levels,
	}
}

func (s *indentStack) peek() int {
	if s.length == 0 {
		return 0 // top-level indent level is 0 by definition
	}
	return s.levels[s.length-1]
}

func (s *indentStack) push(lvl int) {
	if s.length == len(s.levels) {
		s.levels = append(s.levels, lvl)
	} else { // s.length < len(s.levels)
		s.levels[s.length] = lvl
	}
	s.length++
}

func (s *indentStack) pop() int {
	// s.length should be greater than 0; panics otherwise
	lvl := s.levels[s.length-1]
	s.length--
	return lvl
}

type streamLexer struct {
	scanner      *Scanner
	parenDepth   int
	indents      *indentStack
	queue        *wordQueue
	curIndent    string
	needsNewline bool
	hasFirst     bool
	opts         Options

	errs errors.Errors
}

func newStreamLexer(src []byte, opts Options) *streamLexer {
	opts.ScanNewLines = true
	lexer := &streamLexer{
		opts:    opts,
		scanner: NewScanner(src, opts),
		indents: newIndentStack(16),
		queue:   newWordQueue(16),
	}
	return lexer
}

func (l *streamLexer) error(offs int, msg string) {
	l.errs = errors.Append(l.errs, PosError{token.Pos(offs), msg})
}

// Compute an indentation level from an indentation string
func (l *streamLexer) computeIndentLevel(s string) int {
	var level int
	for _, c := range s {
		// normal space or no break line space
		if c == ' ' || c == '\u00a0' {
			level++
		} else if c == '\t' {
			// increase indent to level multiple of eight
			level += 8 - (level % 8)
		} else {
			// this is an error but treat it as a single whitespace character soo
			// that we can keep processing
			level++
			l.error(l.scanner.offset, fmt.Sprintf("invalid character %q within indentation whitespace", c))
		}
	}
	return level
}

func (l *streamLexer) processIndent(indent string, begin, end token.Pos) {
	lastLevel := l.indents.peek()
	curLevel := l.computeIndentLevel(indent)

	switch {
	// Case 1: indentation is unchanged; emit a newline
	case curLevel == lastLevel:
		return

	// Case 2: indentation has increased; emit a newline and queue an indent
	case curLevel > lastLevel:
		l.indents.push(curLevel)
		l.queue.add(Word{
			Begin: begin,
			End:   end,
			Token: Indent,
		})

	// Case 3: indentation has decreased; emit a newline and queue dedents
	default:
		// pop indent levels until we reach a level <= current level
		var numDedents int
		for l.indents.peek() > curLevel {
			numDedents++
			l.indents.pop()
		}

		// now the top of the stack should equal curLevel, if not then we have a lexical error
		if l.indents.peek() != curLevel {
			l.error(l.scanner.offset, "invalid indentation level")
			// Insert a new indent level so that we can keep processing
			l.indents.push(curLevel)
			// This "uses up" one of the dedents since we must exactly match indents with dedents
			numDedents--
		}

		// queue up the dedents
		for i := 0; i < numDedents; i++ {
			l.queue.add(Word{
				Begin: begin,
				End:   end,
				Token: Dedent,
			})
		}
	}
}

func (l *streamLexer) Next() Word {
	if l.queue.length > 0 {
		return l.queue.remove()
	}

	newLineBegin, newLineEnd := token.Pos(-1), token.Pos(-1)

	for {
		begin, end, tok, lit := l.scanner.Scan()
		word := Word{
			Begin:   begin,
			End:     end,
			Token:   tok,
			Literal: lit,
		}

		// Process open and close parens. If we get a keyword that cannot appear within a
		// parenthesized region then drop out of the parentheses and set an error.
		switch tok {
		case Lparen, Lbrace, Lbrack:
			l.parenDepth++
		case Rparen, Rbrace, Rbrack:
			if l.parenDepth > 0 {
				l.parenDepth--
			}
		case Class, Def, Del, Pass, With, Raise, Import,
			Break, Continue, Assert, Except, Finally,
			Global, Try, While, Semicolon, NonLocal:
			if l.parenDepth != 0 {
				l.error(l.scanner.offset, fmt.Sprintf("invalid keyword in parenthesized region: %s", tok.String()))
				l.parenDepth = 0
			}
		}

		// Process newlines
		switch tok {
		case Comment, Magic:
			break

		case LineContinuation:
			// Do not surface this token at all
			continue

		case NewLine:
			for strings.HasPrefix(lit, "\n") || strings.HasPrefix(lit, "\r") {
				lit = lit[1:]
			}
			l.curIndent = lit
			if l.parenDepth == 0 {
				// we cannot emit a newline char token here because we may have multiple
				// consecutive newlines, which need to be treated as one newline
				l.needsNewline = true
			}
			if newLineBegin == -1 {
				newLineBegin = begin
			}
			newLineEnd = end
			continue

		default:
			// if we have a newline pending then we need to emit one of:
			//    NEWLINE <CURTOKEN>                       if the indentation level has not changed
			//    NEWLINE INDENT <CURTOKEN>                if the indentation level has increased
			//    NEWLINE DEDENT ... DEDENT <CURTOKEN>     if the indentation level has decreased
			if l.needsNewline && l.hasFirst {
				// if we have an EOF then ignore any indentation on the last line
				if tok == EOF {
					if !l.opts.KeepEOFIndent {
						l.curIndent = ""
					}
					newLineBegin, newLineEnd = begin, end
				}

				// queue up indents/dedents
				l.processIndent(l.curIndent, begin, end)
				// add the current word to the end of the queue
				l.queue.add(word)

				// finall emit a newline character
				word = Word{
					Begin: newLineBegin,
					End:   newLineEnd,
					Token: NewLine,
				}
			}

			newLineBegin, newLineEnd = token.Pos(-1), token.Pos(-1)
			l.needsNewline = false
			l.hasFirst = true
		}

		return word
	}
}
