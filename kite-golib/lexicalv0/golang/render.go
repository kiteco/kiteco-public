package golang

import (
	"go/token"
	"io"
	"strings"
	"time"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/golang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
)

type (
	// Stack ...
	Stack struct {
		top    *node
		length int
	}
	node struct {
		value interface{}
		prev  *node
	}
)

// NewStack ...
func NewStack() *Stack {
	return &Stack{nil, 0}
}

// Len ...
func (s *Stack) Len() int {
	return s.length
}

// Peek ...
func (s *Stack) Peek() interface{} {
	if s.length == 0 {
		return nil
	}
	return s.top.value
}

// Pop ...
func (s *Stack) Pop() interface{} {
	if s.length == 0 {
		return nil
	}

	n := s.top
	s.top = n.prev
	s.length--
	return n.value
}

// Push ...
func (s *Stack) Push(value interface{}) {
	n := &node{value, s.top}
	s.top = n
	s.length++
}

type ifMerge int

const (
	never ifMerge = 0
	can   ifMerge = 1
	must  ifMerge = 2
)

type mergeSig struct {
	left  ifMerge
	right ifMerge
}

type mergeLookUp map[token.Token]mergeSig

var toPush = map[token.Token]bool{
	token.FOR:    true,
	token.IF:     true,
	token.ELSE:   true,
	token.FUNC:   true,
	token.SWITCH: true,
	token.SELECT: true,
	token.LBRACE: true,
	token.LBRACK: true,
	token.LPAREN: true,
}

var possibleInlineSemicolon = map[token.Token]bool{
	token.FOR:    true,
	token.IF:     true,
	token.SWITCH: true,
}

var closings = map[token.Token]bool{
	token.RBRACE: true,
	token.RBRACK: true,
	token.RPAREN: true,
}

// TypeLiterals enumerates token types we consider literals.
var TypeLiterals = map[token.Token]bool{
	token.INT:    true,
	token.FLOAT:  true,
	token.IMAG:   true,
	token.STRING: true,
	token.CHAR:   true,
}

var pairs = map[token.Token]token.Token{
	token.LBRACE: token.RBRACE,
	token.LBRACK: token.RBRACK,
	token.LPAREN: token.RPAREN,
}

func getMergeLookUp() mergeLookUp {
	lookup := make(map[token.Token]mergeSig)

	// Default is never merge both ways
	for _, tok := range AllTokens {
		lookup[tok] = mergeSig{left: never, right: never}
	}

	for tok := range TypeLiterals {
		lookup[tok] = mergeSig{left: can, right: can}
	}

	// Add special cases
	lookup[token.IDENT] = mergeSig{left: can, right: can}

	lookup[token.PERIOD] = mergeSig{left: must, right: must}
	lookup[token.LPAREN] = mergeSig{left: must, right: must}

	lookup[token.ELLIPSIS] = mergeSig{left: must, right: can}
	lookup[token.INC] = mergeSig{left: must, right: can}
	lookup[token.DEC] = mergeSig{left: must, right: can}
	lookup[token.RBRACE] = mergeSig{left: must, right: can}
	lookup[token.RBRACK] = mergeSig{left: must, right: can}
	lookup[token.RPAREN] = mergeSig{left: must, right: can}

	lookup[token.COMMA] = mergeSig{left: must, right: never}
	lookup[token.COLON] = mergeSig{left: must, right: never}
	lookup[token.SEMICOLON] = mergeSig{left: must, right: never}

	lookup[token.LBRACE] = mergeSig{left: can, right: must}
	lookup[token.LBRACK] = mergeSig{left: can, right: must}

	lookup[token.MAP] = mergeSig{left: never, right: must}
	lookup[token.MUL] = mergeSig{left: never, right: must}
	lookup[token.AND] = mergeSig{left: never, right: must}

	lookup[token.STRUCT] = mergeSig{left: never, right: can}
	lookup[token.DEFAULT] = mergeSig{left: never, right: can}

	return lookup
}

func ifIdentifierLike(tok token.Token) bool {
	return tok == token.IDENT || TypeLiterals[tok]
}

func ph(tok lexer.Token) string {
	if TypeLiterals[token.Token(tok.Token)] {
		return data.Hole(render.GoTempPlaceholder)
	}
	return tok.Lit
}

// Render renders a completion
func Render(line, pred []lexer.Token, hasPrefix bool, precededBySpace bool) (_ data.Snippet, ok bool) {
	lookup := getMergeLookUp()

	var lastRight = must
	var inCase bool
	var newLine bool
	var inCompletion bool
	braceStack := NewStack()
	completionStack := NewStack()

	// TODO(naman) more efficient way to do this?
	toks := append(append([]lexer.Token{}, line...), pred...)

	var merged []string
	for i := 0; i < len(toks); i++ {
		curr := toks[i]
		if curr.Token == lexer.BPEEncodedTok {
			curr.Token = int(token.IDENT)
			toks[i].Token = int(token.IDENT)
		}

		currToken := token.Token(curr.Token)

		// If there's proceeded space but the current completion wants to merge, invalidate the completion
		if i == len(line) && precededBySpace && (lastRight == must || lookup[currToken].left == must) {
			return data.Snippet{}, false
		}

		// Actually start to render
		if i == len(line) {
			merged = []string{}
			inCompletion = true
			if hasPrefix || precededBySpace {
				lastRight = must
			}
		}

		if newLine {
			merged = append(merged, "\n")
			newLine = false
		}

		// For completions like `foo(bar,` at f$, we declare it as invalid
		// a.k.a. completions that has open paren, end in a syntax token that is not a close paren
		// It's only allowed to end in close paren, or identifiers or constant placeholders
		if i == len(toks)-1 && completionStack.Len() > 0 {
			currentOpen := completionStack.Peek().(token.Token)
			_, isOpen := pairs[currentOpen]
			_, ok := TypeLiterals[currToken]
			if isOpen && !ok && currToken != token.IDENT && currToken != pairs[currentOpen] {
				return data.Snippet{}, false
			}
		}

		// Handle parens included in the completion
		if inCompletion {
			if _, ok := pairs[currToken]; ok {
				completionStack.Push(currToken)
			}
			if closings[currToken] {
				// Found a single closing paren
				if completionStack.Len() == 0 {
					return data.Snippet{}, false
				}

				// Found a un-matching closing paren
				if pairs[completionStack.Peek().(token.Token)] != currToken {
					return data.Snippet{}, false
				}

				completionStack.Pop()
			}
		}

		// The following logic decides if we are in the middle of func/for/if/else statement
		// If so, turn on newLine
		if toPush[currToken] {
			braceStack.Push(currToken)
			if braceStack.Len() == 2 && currToken == token.LBRACE {
				braceStack.Pop()
				braceStack.Pop()
				if i == len(line) && !precededBySpace && !hasPrefix {
					merged = append(merged, " ")
				}
				merged = append(merged, "{")
				newLine = true
				lastRight = must
				continue
			}
		}

		if closings[currToken] && braceStack.Len() > 0 {
			braceStack.Pop()
		}

		// Add new line after `case foo:`
		if currToken == token.CASE || currToken == token.DEFAULT {
			inCase = true
		}

		if inCase && currToken == token.COLON {
			merged = append(merged, ph(curr))
			newLine = true
			lastRight = must
			inCase = false
			continue
		}

		// Handle semicolon
		if currToken == token.SEMICOLON {
			if braceStack.Len() > 0 && possibleInlineSemicolon[braceStack.Peek().(token.Token)] {
				merged = append(merged, ph(curr))
				lastRight = never
			} else {
				newLine = true
				lastRight = must
			}
			continue
		}

		// IDENT or TypeLiterals can't follow each other, otherwise they are flexible
		if i >= 1 && ifIdentifierLike(currToken) {
			last := token.Token(toks[i-1].Token)
			if ifIdentifierLike(last) {
				if i == len(line) && (hasPrefix || precededBySpace) {
					merged = append(merged, ph(curr))
				} else {
					merged = append(merged, " ", ph(curr))
				}
				lastRight = can
				continue
			}
		}

		// General cases, merge if both sides agree
		switch lastRight {
		case must:
			merged = append(merged, ph(curr))
		case never:
			merged = append(merged, " ", ph(curr))
		case can:
			switch lookup[currToken].left {
			case must, can:
				merged = append(merged, ph(curr))
			case never:
				merged = append(merged, " ", ph(curr))
			}
		}

		lastRight = lookup[currToken].right
	}

	for completionStack.Len() > 0 {
		delim := completionStack.Pop().(token.Token)
		if lastRight == never {
			merged = append(merged, " ")
		}
		ph := data.Hole("")
		switch delim {
		case token.LBRACE:
			merged = append(merged, ph, token.RBRACE.String())
		case token.LBRACK:
			merged = append(merged, ph, token.RBRACK.String())
		case token.LPAREN:
			merged = append(merged, ph, token.RPAREN.String())
		}
	}

	return data.BuildSnippet(strings.Join(merged, "")), true
}

// DefaultPrettifyConfig ...
var DefaultPrettifyConfig = Config{
	Indent:          "\t",
	SpaceAfterComma: true,
}

// FormatCompletion formats the completion using Prettify
func FormatCompletion(input string, c data.Completion, config Config, match render.MatchOption) data.Snippet {
	defer status.FormatCompletionDuration.DeferRecord(time.Now())
	status.FormatBytes.Record(int64(len(input)))
	return render.FormatCompletion(input, c, golang.GetLanguage(), match, func(w io.Writer, src []byte, n *sitter.Node) ([]render.OffsetMapping, error) {
		return Prettify(w, config, src, c.Replace.Begin, c.Replace.Begin+len(c.Snippet.Text), n)
	})
}
