package text

import (
	"strings"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
)

type stack []rune

func (s *stack) Push(r rune) {
	*s = append(*s, r)
}

func (s *stack) Pop() rune {
	if len(*s) == 0 {
		return rune(0)
	}
	elem := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return elem
}

func (s *stack) Len() int {
	return len(*s)
}

var quotes = map[rune]bool{
	'\'': true,
	'"':  true,
	'`':  true,
}

var syntaxPairs = map[rune]rune{
	'{': '}',
	'[': ']',
	'(': ')',
}

var closeSyntax = map[rune]bool{
	'}': true,
	']': true,
	')': true,
}

var syntaxPairsExtended = map[rune]rune{
	'{': '}',
	'[': ']',
	'(': ')',
	'<': '>',
}

var closeSyntaxExtended = map[rune]bool{
	'}': true,
	']': true,
	')': true,
	'>': true,
}

var webTagLangs = map[lang.Language]bool{
	lang.CSS:  true,
	lang.HTML: true,
	lang.JSX:  true,
	lang.TSX:  true,
	lang.Vue:  true,
}

var genericLangs = map[lang.Language]bool{
	lang.Cpp:        true,
	lang.CSharp:     true,
	lang.Java:       true,
	lang.ObjectiveC: true,
	lang.Kotlin:     true,
	lang.Scala:      true,
}

// Simple heuristics for if enforcing paired <>
func enforcePairedGtLt(nativeLang lang.Language, line []lexer.Token) bool {
	if webTagLangs[nativeLang] {
		return true
	}
	if !genericLangs[nativeLang] {
		return false
	}
	lits := make([]string, 0, len(line))
	for _, t := range line {
		lits = append(lits, t.Lit)
	}
	before := strings.Join(lits, "")
	if strings.Contains(before, "class") || strings.Contains(before, "template") || strings.Contains(before, "private") || strings.Contains(before, "var") {
		return true
	}

	// Enforce pairing if the content before cursor has a opening `<`
	var pointy int
	for _, r := range before {
		if r == '<' {
			pointy++
		}
		if r == '>' {
			pointy--
		}
	}
	return pointy == 1
}

// Render ...
func Render(nativeLang lang.Language, line, toks []lexer.Token) (data.Snippet, bool) {
	stack := new(stack)
	var merged []string

	var addClosing bool
	var badClosing bool
	sp := syntaxPairs
	cs := closeSyntax
	if enforcePairedGtLt(nativeLang, line) {
		sp = syntaxPairsExtended
		cs = closeSyntaxExtended
	}
	quoteCounts := make(map[rune]int)
	for _, tok := range toks {
		for _, r := range tok.Lit {
			if quotes[r] {
				quoteCounts[r]++
			}
			merged = append(merged, string(r))
			if c, ok := sp[r]; ok {
				stack.Push(c)
				addClosing = true
			} else if cs[r] {
				if stack.Pop() != r {
					badClosing = true
				}
				addClosing = true
			} else {
				addClosing = unicode.IsSpace(r) || isIdentRune(r)
			}
		}
	}

	// Make sure we have even number of each quotes in the predicted text
	for _, c := range quoteCounts {
		if c%2 != 0 {
			return data.Snippet{}, false
		}
	}

	if badClosing {
		return data.Snippet{}, false
	}
	if !addClosing || stack.Len() == 0 {
		str := strings.TrimRightFunc(strings.Join(merged, ""), unicode.IsSpace)
		return data.BuildSnippet(str), true
	}
	for stack.Len() > 0 {
		merged = append(merged, data.Hole(""), string(stack.Pop()))
	}

	// Trim trailing spaces from rendered snippet
	str := strings.TrimRightFunc(strings.Join(merged, ""), unicode.IsSpace)
	return data.BuildSnippet(str), true
}
