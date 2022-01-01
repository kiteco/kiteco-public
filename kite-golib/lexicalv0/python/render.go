package python

import (
	"io"
	"strings"
	"time"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/python"
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
	never     ifMerge = 0
	reluctant ifMerge = 1
	willing   ifMerge = 2
	must      ifMerge = 3
)

func merge(lastRight, currLeft ifMerge) bool {
	// First priority: check if lastRight is either must or never
	if lastRight == must {
		return true
	}
	if lastRight == never {
		return false
	}

	// Second priority: check if currLeft is either must or never
	if currLeft == must {
		return true
	}
	if currLeft == never {
		return false
	}

	// Third priority: check if both currLeft and lastRight are willing
	return lastRight == willing && currLeft == willing
}

type mergeSig struct {
	left  ifMerge
	right ifMerge
}

type mergeLookUp map[int]mergeSig

// TypeLiterals are constants
var TypeLiterals = map[int]string{
	symString:  "str",
	symInteger: "int",
	symFloat:   "float",
}

var closings = map[int]bool{
	anonSymRbrace: true,
	anonSymRbrack: true,
	anonSymRparen: true,
}

var pairs = map[int]int{
	anonSymLparen: anonSymRparen,
	anonSymLbrace: anonSymRbrace,
	anonSymLbrack: anonSymRbrack,
}

func getMergeLookUp() mergeLookUp {
	lookup := make(map[int]mergeSig)
	for _, tok := range allTokens {
		lookup[tok] = mergeSig{left: never, right: never}
	}

	for tok := range TypeLiterals {
		lookup[tok] = mergeSig{left: willing, right: willing}
	}

	// Add special cases
	lookup[symIdentifier] = mergeSig{left: willing, right: willing}
	lookup[symTrue] = mergeSig{left: willing, right: willing}
	lookup[symFalse] = mergeSig{left: willing, right: willing}
	lookup[symNone] = mergeSig{left: willing, right: willing}

	lookup[anonSymDot] = mergeSig{left: must, right: must}

	lookup[anonSymRbrace] = mergeSig{left: must, right: willing}
	lookup[anonSymRbrack] = mergeSig{left: must, right: willing}
	lookup[anonSymRparen] = mergeSig{left: must, right: willing}

	lookup[anonSymComma] = mergeSig{left: must, right: never}
	lookup[anonSymColon] = mergeSig{left: must, right: never}

	lookup[anonSymLbrace] = mergeSig{left: willing, right: must}
	lookup[anonSymLbrack] = mergeSig{left: willing, right: must}
	lookup[anonSymLparen] = mergeSig{left: willing, right: must}

	lookup[anonSymElse] = mergeSig{left: never, right: reluctant}
	lookup[anonSymTry] = mergeSig{left: never, right: reluctant}
	lookup[anonSymFinally] = mergeSig{left: never, right: reluctant}

	return lookup
}

func ph(tok lexer.Token) string {
	if s, ok := TypeLiterals[int(tok.Token)]; ok {
		return data.HoleWithPlaceholderMarks(s)
	}
	switch tok.Token {
	case symTrue:
		return "True"
	case symFalse:
		return "False"
	case symNone:
		return "None"
	default:
		return tok.Lit
	}
}

func indent(num int, indentSymbol string) string {
	return strings.Repeat(indentSymbol, num)
}

// Render renders a completion
func Render(line, pred []lexer.Token, hasPrefix bool, precededBySpace bool, lineIndent int, indentSymbol string) (_ data.Snippet, ok bool) {
	lookup := getMergeLookUp()

	var lastRight = must
	var inCompletion bool
	completionStack := NewStack()

	// TODO(naman) more efficient way to do this?
	toks := append(append([]lexer.Token{}, line...), pred...)
	currentIndent := lineIndent

	var merged []string
	for i := 0; i < len(toks); i++ {

		curr := toks[i]
		if curr.Token == lexer.BPEEncodedTok {
			curr.Token = int(symIdentifier)
			toks[i].Token = int(symIdentifier)
		}

		currToken := curr.Token

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

		// Handle new lines
		if currToken == endOfStatement {
			merged = append(merged, "\n")
			merged = append(merged, indent(currentIndent, indentSymbol))
			lastRight = must
			continue
		}
		if currToken == startOfBlock {
			// only increment the indent count if we are in the completion
			// since the initial value of currentIndent == lineIndent
			// which already includes the appropriate indent for the current line.
			if inCompletion {
				currentIndent++
			}
			merged = append(merged, "\n")
			merged = append(merged, indent(currentIndent, indentSymbol))
			lastRight = must
			continue
		}
		if currToken == endOfBlock {
			if currentIndent == 0 {
				return data.Snippet{}, false
			}
			// see increment above
			if inCompletion {
				currentIndent--
			}
			merged = append(merged, "\n")
			merged = append(merged, indent(currentIndent, indentSymbol))
			lastRight = must
			continue
		}

		// For completions like `foo(bar,` at f$, we declare it as invalid
		// a.k.a. completions that has open paren, end in a syntax token that is not a close paren
		// It's only allowed to end in close paren, or identifiers or strings or constant placeholders
		if i == len(toks)-1 && completionStack.Len() > 0 {
			currentOpen := completionStack.Peek().(int)
			_, isOpen := pairs[currentOpen]
			_, ok := TypeLiterals[currToken]
			if isOpen && !ok && currToken != symIdentifier && currToken != pairs[currentOpen] {
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
				if pairs[completionStack.Peek().(int)] != currToken {
					return data.Snippet{}, false
				}
				completionStack.Pop()
			}
		}

		// Hack for lambda function, always merge left
		// so that even if we end up having ERROR nodes, rendering still looks correct
		if i > 1 && currToken == anonSymLambda {
			merged = append(merged, ph(curr))
			continue
		}

		// General cases, merge if both sides agree
		if merge(lastRight, lookup[currToken].left) {
			merged = append(merged, ph(curr))
		} else {
			merged = append(merged, " ", ph(curr))
		}

		lastRight = lookup[currToken].right
	}

	for completionStack.Len() > 0 {
		delim := completionStack.Pop().(int)
		if lastRight == never {
			merged = append(merged, " ")
		}
		ph := data.Hole("")
		switch delim {
		case anonSymLbrace:
			merged = append(merged, ph, "}")
		case anonSymLparen:
			merged = append(merged, ph, ")")
		case anonSymLbrack:
			merged = append(merged, ph, "]")
		}
	}

	return data.BuildSnippet(strings.Join(merged, "")), true
}

// DefaultPrettifyConfig ...
var DefaultPrettifyConfig = Config{
	Indent:                      strings.Repeat(" ", 4),
	SpaceAfterColonInPair:       true,
	SpaceAfterColonInTypedParam: true,
	SpaceAfterColonInLambda:     true,
	SpaceAfterComma:             true,
	SpaceInfixOps:               true,
	SpaceAroundArrow:            true,
	BlankLinesBeforeClassDef:    2,
	BlankLinesBeforeTopFuncDef:  2,
	BlankLinesBetweenMethods:    1,
	ListItemsNewLine:            -1,
	DictionaryItemsNewLine:      -1,
	FuncParamsNewLine:           -1,
}

// FormatCompletion formats the completion using Prettify
func FormatCompletion(input string, c data.Completion, config Config, match render.MatchOption) data.Snippet {
	defer status.FormatCompletionDuration.DeferRecord(time.Now())
	status.FormatBytes.Record(int64(len(input)))
	return render.FormatCompletion(input, c, python.GetLanguage(), match, func(w io.Writer, src []byte, n *sitter.Node) ([]render.OffsetMapping, error) {
		return Prettify(w, config, src, c.Replace.Begin, c.Replace.Begin+len(c.Snippet.Text), n)
	})
}
