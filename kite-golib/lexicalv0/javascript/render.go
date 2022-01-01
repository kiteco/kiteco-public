package javascript

import (
	"io"
	"strings"
	"time"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/javascript"
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

type mergeLookUp map[int]mergeSig

var closings = map[int]bool{
	anonSymRbrace: true,
	anonSymRbrack: true,
	anonSymRparen: true,
}

var pairs = map[int]int{
	anonSymLparen:       anonSymRparen,
	anonSymLbrace:       anonSymRbrace,
	anonSymLbrack:       anonSymRbrack,
	anonSymDollarLbrace: anonSymRbrace,
}

// TypeLiterals ...
var TypeLiterals = map[int]bool{
	symNumber:          true,
	symJsxText:         true,
	auxSymStringToken1: true,
	auxSymStringToken2: true,
	symRegexPattern:    true,
	symRegexFlags:      true,
	symTemplateChars:   true,
}

func getMergeLookUp() mergeLookUp {
	lookup := make(map[int]mergeSig)
	for _, tok := range allTokens {
		lookup[tok] = mergeSig{left: never, right: never}
	}

	for tok := range TypeLiterals {
		lookup[tok] = mergeSig{left: can, right: can}
	}
	// Add special cases
	lookup[symIdentifier] = mergeSig{left: can, right: can}
	lookup[symThis] = mergeSig{left: can, right: can}
	lookup[anonSymTarget] = mergeSig{left: can, right: can}
	lookup[symSuper] = mergeSig{left: can, right: can}
	lookup[symTrue] = mergeSig{left: can, right: can}
	lookup[symFalse] = mergeSig{left: can, right: can}
	lookup[symUndefined] = mergeSig{left: can, right: can}
	lookup[symNull] = mergeSig{left: can, right: can}

	// Default is `can` so that JSX behaves correct
	lookup[anonSymGt] = mergeSig{left: can, right: can}
	lookup[anonSymLt] = mergeSig{left: can, right: can}

	lookup[anonSymDquote] = mergeSig{left: can, right: can}
	lookup[anonSymSquote] = mergeSig{left: can, right: can}
	lookup[anonSymBquote] = mergeSig{left: can, right: can}
	lookup[anonSymBang] = mergeSig{left: can, right: can}
	lookup[anonSymTilde] = mergeSig{left: can, right: can}

	lookup[anonSymDot] = mergeSig{left: must, right: must}

	lookup[anonSymPlusPlus] = mergeSig{left: must, right: can}
	lookup[anonSymDashDash] = mergeSig{left: must, right: can}
	lookup[anonSymRbrace] = mergeSig{left: must, right: can}
	lookup[anonSymRbrack] = mergeSig{left: must, right: can}
	lookup[anonSymRparen] = mergeSig{left: must, right: can}

	lookup[anonSymComma] = mergeSig{left: must, right: never}
	lookup[anonSymColon] = mergeSig{left: must, right: never}
	lookup[anonSymSemi] = mergeSig{left: must, right: never}

	lookup[anonSymLbrace] = mergeSig{left: can, right: must}
	lookup[anonSymLbrack] = mergeSig{left: can, right: must}
	lookup[anonSymLparen] = mergeSig{left: can, right: must}
	lookup[anonSymDotDotDot] = mergeSig{left: can, right: must}
	lookup[anonSymDollarLbrace] = mergeSig{left: can, right: must}

	lookup[anonSymAt] = mergeSig{left: never, right: must}

	return lookup
}

func ph(tok lexer.Token) string {
	if TypeLiterals[tok.Token] {
		// Here the placeholder is a string to make `FormatCompletion` easier
		// Later it will be replaced by `...`
		return data.Hole(render.JsTempPlaceholder)
	}
	return tok.Lit
}

func endsWithJsxClosingTag(toks []lexer.Token) bool {
	l := len(toks)
	if l < 4 {
		return false
	}
	if toks[l-1].Lit == ">" && toks[l-2].Token == lexer.BPEEncodedTok &&
		toks[l-3].Lit == "/" && toks[l-4].Lit == "<" {
		return true
	}
	return false
}

func endsJsxSelfClosingTag(toks []lexer.Token) bool {
	l := len(toks)
	if l < 2 {
		return false
	}
	if toks[l-1].Lit == ">" && toks[l-2].Lit == "/" {
		return true
	}
	return false
}

// Render renders a completion
func Render(line, pred []lexer.Token, hasPrefix bool, precededBySpace bool) (_ data.Snippet, ok bool) {
	// Invalid if has only one quote in completion
	var numSQ, numDQ, numBQ int
	for _, p := range pred {
		if p.Token == anonSymSquote {
			numSQ++
		}
		if p.Token == anonSymDquote {
			numDQ++
		}
		if p.Token == anonSymBquote {
			numBQ++
		}
	}
	if numSQ%2 != 0 || numDQ%2 != 0 || numBQ%2 != 0 {
		return data.Snippet{}, false
	}

	// TODO(naman) more efficient way to do this?
	toks := append(append([]lexer.Token{}, line...), pred...)

	// Add a filter for https://github.com/kiteco/kiteco/issues/10448
	var hasGt bool
	for _, tok := range pred {
		if tok.Lit == ">" {
			hasGt = true
			break
		}
	}
	if hasGt && !endsJsxSelfClosingTag(toks) && !endsWithJsxClosingTag(toks) {
		return data.Snippet{}, false
	}

	lookup := getMergeLookUp()
	var lastRight = must
	var inCase bool
	var newLine bool
	var inCompletion bool
	completionStack := NewStack()
	var merged []string

	for i := 0; i < len(toks); i++ {
		curr := toks[i]
		if curr.Token == lexer.BPEEncodedTok {
			curr.Token = int(symIdentifier)
			toks[i].Token = int(symIdentifier)
		}

		currToken := curr.Token

		// If it's not the first token in the current line and there's proceeded space in front
		// but the current completion wants to merge, invalidate the completion
		if i == len(line) && i > 0 && precededBySpace && (lastRight == must || lookup[currToken].left == must) {
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

		// Add new line after `case foo:`
		if currToken == anonSymCase {
			inCase = true
		}

		if inCase && currToken == anonSymColon {
			merged = append(merged, ph(curr))
			newLine = true
			lastRight = must
			inCase = false
			continue
		}

		// Handle semicolon
		if currToken == symAutomaticSemicolon {
			newLine = true
			lastRight = must
			continue
		}

		if currToken == anonSymSemi {
			merged = append(merged, ph(curr))
			newLine = true
			lastRight = must
			continue
		}

		// Handle < and >, see if they are actually used as GT/LT or part of JSX tags
		if currToken == anonSymGt || currToken == anonSymLt {
			// If it's not first or last token
			if i > 0 && i < len(toks)-1 {
				last := toks[i-1].Token
				next := toks[i+1].Token
				// Check if they are in between identifiers or numbers
				if (last == symIdentifier || last == symNumber) && (next == symIdentifier || next == symNumber) {
					if i != len(line) {
						merged = append(merged, " ")
					}
					merged = append(merged, ph(curr))
					lastRight = never
					continue
				}
			}
		}

		// Handle the `/` following `<` in potential JSX tags
		if currToken == anonSymSlash {
			if i > 0 && toks[i-1].Token == anonSymLt {
				merged = append(merged, ph(curr))
				lastRight = must
				continue
			}
		}

		// Handle IDENT: they can't follow other IDENT or RPAREN, otherwise they are flexible
		if i >= 1 && currToken == symIdentifier {
			last := toks[i-1].Token
			if last == symIdentifier || last == anonSymRparen {
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
		case anonSymDollarLbrace:
			merged = append(merged, ph, "}")
		}
	}

	return data.BuildSnippet(strings.Join(merged, "")), true
}

// DefaultPrettifyConfig ...
var DefaultPrettifyConfig = Config{
	ArrayBracketNewline:        -1,
	ArrayElementNewline:        -1,
	ArrowSpacingBefore:         true,
	ArrowSpacingAfter:          true,
	CommaSpacingAfter:          true,
	FuncParamArgumentNewline:   -1,
	FuncParenNewline:           -1,
	Indent:                     2,
	KeySpacingAfterColon:       true,
	KeywordSpacingBefore:       true,
	KeywordSpacingAfter:        true,
	ObjectCurlySpacing:         true,
	ObjectCurlyNewline:         -1,
	ObjectPropertyNewline:      -1,
	SpaceBeforeBlocks:          true,
	SpaceInfixOps:              true,
	SpaceUnaryOpsWords:         true,
	StatementNewline:           true,
	SwitchColonNewLine:         true,
	JsxElementChildrenNewline:  -1,
	JsxFragmentChildrenNewline: true,
	JsxAttributeNewline:        -1,
}

// FormatCompletion formats the completion using Prettify
func FormatCompletion(input string, c data.Completion, config Config, match render.MatchOption) data.Snippet {
	defer status.FormatCompletionDuration.DeferRecord(time.Now())
	status.FormatBytes.Record(int64(len(input)))
	return render.FormatCompletion(input, c, javascript.GetLanguage(), match, func(w io.Writer, src []byte, n *sitter.Node) ([]render.OffsetMapping, error) {
		return Prettify(w, config, src, c.Replace.Begin, c.Replace.Begin+len(c.Snippet.Text), n)
	})
}
