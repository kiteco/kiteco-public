package vue

import (
	"strings"

	"github.com/go-errors/errors"
	sitter "github.com/kiteco/go-tree-sitter"
	sittercss "github.com/kiteco/go-tree-sitter/css"
	sitterjs "github.com/kiteco/go-tree-sitter/javascript"
	"github.com/kiteco/go-tree-sitter/vue"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/css"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/html"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/javascript"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
)

const (
	terminalChar = "$"
	symERROR     = 65535
)

// Vue is a "container" type of language - it returns tokens for other
// languages such as HTML, Javascript and CSS. By default, the token IDs of
// those various languages would overlap - they all start at 1, so a token
// ID 19 would have no meaning in a Vue source, it would be impossible to
// tell if this is a javascript 19 or an HTML 19, etc.
//
// To address this, the IDs are offset in a predefined order so it is
// possible to tell them apart. Note that the 0-1000 range is kept for
// vue-specific tokens, should we need those eventually (vue itself has
// a tree-sitter parser and tokens, although there are no *terminal* tokens
// of interest at the moment).
const (
	HTMLTokenIDBase       = 1000
	JavascriptTokenIDBase = 2000
	CSSTokenIDBase        = 3000
)

// Lexer is a vue lexer.
type Lexer struct {
	*lexer.TreeSitterLexer
	sitterLang     *sitter.Language
	cursor         int
	currentTagOnly bool
}

// NewLexer returns the default vue lexer that uses JS lexer on certain tag content
func NewLexer() (*Lexer, error) {
	l := &Lexer{
		sitterLang: vue.GetLanguage(),
	}
	symCount := int(sitterjs.GetLanguage().SymbolCount())
	// Use extractTreeTokens unless you want only JS tokens
	ts, err := lexer.NewTreeSitterLexer(lang.Vue, symCount, l.extractJsTokens)
	if err != nil {
		return nil, err
	}
	l.TreeSitterLexer = ts
	return l, nil
}

// NewCompleteLexer returns a complete vue lexer that uses separate JS/html/css lexers
func NewCompleteLexer() (*Lexer, error) {
	l := &Lexer{
		sitterLang: vue.GetLanguage(),
	}
	symCount := int(sittercss.GetLanguage().SymbolCount()) + CSSTokenIDBase
	ts, err := lexer.NewTreeSitterLexer(lang.Vue, symCount, l.extractTreeTokens)
	if err != nil {
		return nil, err
	}
	l.TreeSitterLexer = ts
	return l, nil
}

// Lang implements Lexer.
func (Lexer) Lang() lang.Language {
	return lang.Vue
}

// ShouldBPEEncode implements Lexer.
func (l Lexer) ShouldBPEEncode(tok lexer.Token) ([]string, bool) {
	// TODO: implement
	return nil, false
}

// MergeBPEEncoded implements Lexer.
func (Lexer) MergeBPEEncoded(in []string) []string {
	// TODO: implement
	return nil
}

// TrimTerminal implements lexer
func (Lexer) TrimTerminal(word string) string {
	return strings.TrimSuffix(word, terminalChar)
}

// IsIncompleteToken implements Lexer
func (Lexer) IsIncompleteToken(word string) bool {
	return !strings.HasSuffix(word, terminalChar)
}

// ContainsIdentOrKeyword returns true if a list of tokens contains idents or keywords
func (Lexer) ContainsIdentOrKeyword(toks []lexer.Token) bool {
	// TODO: implement...
	return false
}

// HasInvalidToken ...
func (Lexer) HasInvalidToken(tokens []lexer.Token) bool {
	for _, tok := range tokens {
		// Internal illegal token
		if tok.Lit == "KITE_ILLEGAL" {
			return true
		}
	}
	return false
}

// IsType returns whether a token is an Ident
func (Lexer) IsType(t lexer.TokenType, tok lexer.Token) bool {
	// TODO: implement...
	return false
}

// LexWithOptions sets the current cursor, and an option to only extract the current tag, then lex the source code
func (l *Lexer) LexWithOptions(buf []byte, cursor int, currentTagOnly bool) ([]lexer.Token, error) {
	l.cursor = cursor
	l.currentTagOnly = currentTagOnly
	return l.Lex(buf)
}

type jsTokenizer struct {
	buf              []byte
	lang             *sitter.Language
	jsLexer          *javascript.Lexer
	parser           *sitter.Parser
	cursor           int
	jsRanges         []sitter.Range
	cursorInValidTag bool
	currentTagOnly   bool

	tokens []treesitter.Token
	err    error
}

func (l *Lexer) extractJsTokens(buf []byte, parser *sitter.Parser, tree *sitter.Tree) ([]treesitter.Token, error) {
	root := tree.RootNode()
	jsLexer, err := javascript.NewLexer()
	if err != nil {
		return nil, err
	}

	j := &jsTokenizer{
		buf:            buf,
		lang:           l.sitterLang,
		cursor:         l.cursor,
		jsLexer:        jsLexer,
		parser:         parser,
		currentTagOnly: l.currentTagOnly,
	}
	treesitter.Walk(j, root)

	if j.err != nil {
		return nil, j.err
	}
	if j.cursor != -1 && !j.cursorInValidTag {
		return nil, errors.New("cursor not in a valid tag")
	}
	return j.tokens, nil
}

func (j *jsTokenizer) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		return nil
	}
	// fmt.Printf("%s | sym: %d (type: %s) | %d-%d | nchild: %d | %q\n", n, n.Symbol(), n.Type(), n.StartByte(), n.EndByte(), n.ChildCount(), n.Content(j.buf))
	switch int(n.Symbol()) {
	case symERROR:
		if n.ChildCount() > 0 {
			first := n.Child(0).Content(j.buf)
			var rng sitter.Range
			if first == "<script>" && n.ChildCount() > 1 {
				rng = sitter.Range{
					StartByte:  n.Child(0).EndByte(),
					EndByte:    n.EndByte(),
					StartPoint: n.Child(0).EndPoint(),
					EndPoint:   n.EndPoint(),
				}
			}
			if first == "<template>" {
				rng = sitter.Range{
					StartByte:  n.StartByte(),
					EndByte:    n.EndByte(),
					StartPoint: n.StartPoint(),
					EndPoint:   n.EndPoint(),
				}
			}
			if rng.EndByte > rng.StartByte {
				err := j.maybeAddTokensInRange(rng)
				if err != nil {
					return nil
				}
			}
		}
	case symTemplateElement:
		// If it's HTML tag, treat it as JSX and parse it with existing JS parser
		tag, _ := getStartTagNameAndRange(n)
		if l := getTagLangAttr(j.buf, tag); l == "" || l == "html" {
			rng := sitter.Range{
				StartByte:  n.StartByte(),
				EndByte:    n.EndByte(),
				StartPoint: n.StartPoint(),
				EndPoint:   n.EndPoint(),
			}
			err := j.maybeAddTokensInRange(rng)
			if err != nil {
				return nil
			}
		}
	case symScriptElement:
		// Extract JS script
		tag, rng := getStartTagNameAndRange(n)
		if l := getTagLangAttr(j.buf, tag); l == "" || l == "js" || l == "javascript" {
			err := j.maybeAddTokensInRange(rng)
			if err != nil {
				return nil
			}
		}
	}
	return j
}

func (j *jsTokenizer) maybeAddTokensInRange(rng sitter.Range) error {
	tokens, err := j.jsLexer.TokensInRanges(j.parser, j.buf, []sitter.Range{rng})
	if err != nil {
		j.err = errors.Errorf("error getting jsx tokens from template tag: %s", err)
		return err
	}
	inCurrentTag := j.cursor >= int(rng.StartByte) && j.cursor <= int(rng.EndByte)
	if inCurrentTag {
		j.cursorInValidTag = true
	}
	if inCurrentTag || !j.currentTagOnly {
		j.tokens = append(j.tokens, tokens...)
	}
	return nil
}

func (l *Lexer) extractTreeTokens(buf []byte, parser *sitter.Parser, tree *sitter.Tree) (tokens []treesitter.Token, err error) {
	root := tree.RootNode()

	// first, get the ranges from the vue file that contain code in
	// other languages
	t := &tokenizer{
		buf:  buf,
		lang: l.sitterLang,
	}
	treesitter.Walk(t, root)

	// next, for each supported language, extract the tokens of those ranges
	type rangeTokenizer interface {
		TokensInRanges(*sitter.Parser, []byte, []sitter.Range) ([]treesitter.Token, error)
	}
	extractRangeTokens := func(rt rangeTokenizer, ranges []sitter.Range, offset int) error {
		toks, err := rt.TokensInRanges(parser, buf, ranges)
		if err != nil {
			return err
		}
		for _, tok := range toks {
			// for the ERROR node, let's keep it without offset so that it gets remapped
			// to KITE_ILLEGAL in ../lexer/treesitter.go, regardless of whether this
			// is an HTML, CSS or Javascript error.
			if tok.Symbol != symERROR {
				tok.Symbol += offset
			}
			tokens = append(tokens, tok)
		}
		return nil
	}

	if len(t.htmlRanges) > 0 {
		htmlLex, err := html.NewLexer()
		if err != nil {
			return nil, err
		}
		if err := extractRangeTokens(htmlLex, t.htmlRanges, HTMLTokenIDBase); err != nil {
			return nil, err
		}
	}
	if len(t.jsRanges) > 0 {
		jsLex, err := javascript.NewLexer()
		if err != nil {
			return nil, err
		}
		if err := extractRangeTokens(jsLex, t.jsRanges, JavascriptTokenIDBase); err != nil {
			return nil, err
		}
	}
	if len(t.cssRanges) > 0 {
		cssLex, err := css.NewLexer()
		if err != nil {
			return nil, err
		}
		if err := extractRangeTokens(cssLex, t.cssRanges, CSSTokenIDBase); err != nil {
			return nil, err
		}
	}

	return tokens, nil
}

type tokenizer struct {
	buf  []byte
	lang *sitter.Language

	htmlRanges []sitter.Range
	jsRanges   []sitter.Range
	cssRanges  []sitter.Range
}

func (t *tokenizer) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		return nil
	}

	// When processing Vue files, there are 3 sections:
	// * Template (typically html)
	// * Script (typically js)
	// * Style (typically css)
	//
	// The Vue templates are valid HTML, so they can be parsed with the HTML parser.
	// However, each of those sections can also indicate a different language, using
	// preprocessors/transpilers. For example, jade/pug may be used instead of html
	// for the template, typescript instead of js for script, and stylus instead of css.
	//
	// We have to grab the language attribute of each section (if any) and make sure
	// we can parse it before saving the range for language-specific parsing.

	switch int(n.Symbol()) {
	case symTemplateElement:
		// TODO: we could parse the html with the vue parser itself, it has the advantage of
		// understanding the interpolation notation "{{ ... }}", but is likely less complete
		// and powerful than the actual html parser (it is indeed very minimal in the vue grammar).
		// With the interpolation nodes, we could add those to the js ranges and parse that
		// yet again with the javascript parser (!), but that might be a tad too much for now at least
		// (also, not sure how treesitter would handle such short unrelated snippets in a single
		// batch of ranges).
		tag, rng := getStartTagNameAndRange(n)
		if lang := getTagLangAttr(t.buf, tag); lang == "" || lang == "html" {
			t.htmlRanges = append(t.htmlRanges, rng)
		}

	case symScriptElement:
		// TODO: we could probably support typescript here too
		tag, rng := getStartTagNameAndRange(n)
		if lang := getTagLangAttr(t.buf, tag); lang == "" || lang == "js" || lang == "javascript" {
			t.jsRanges = append(t.jsRanges, rng)
		}

	case symStyleElement:
		tag, rng := getStartTagNameAndRange(n)
		if lang := getTagLangAttr(t.buf, tag); lang == "" || lang == "css" {
			t.cssRanges = append(t.cssRanges, rng)
		}
	}
	return t
}

func getStartTagNameAndRange(elem *sitter.Node) (*sitter.Node, sitter.Range) {
	// from an element, the start tag is the first child and the end tag is the last
	start, end := elem.Child(0), elem.Child(int(elem.ChildCount())-1)
	if start == nil || end == nil {
		return nil, sitter.Range{}
	}

	// the tag name is the 2nd child of the start tag, after the opening "<"
	tag := start.Child(1)
	// the range is from the end of the start to the start of the end
	rng := sitter.Range{
		StartPoint: start.EndPoint(),
		EndPoint:   end.StartPoint(),
		StartByte:  start.EndByte(),
		EndByte:    end.StartByte(),
	}
	return tag, rng
}

func getTagLangAttr(src []byte, n *sitter.Node) string {
	if n == nil {
		return ""
	}

	// attributes are siblings of the tag
	for sib := n.NextSibling(); sib != nil && sib.Type() == "attribute"; sib = sib.NextSibling() {
		nm := sib.Child(0)
		if nm == nil || nm.Content(src) != "lang" {
			continue
		}

		// we have the lang attribute, get its value
		val := sib.Child(2)
		if val == nil {
			return ""
		}
		return strings.Trim(val.Content(src), `"`)
	}
	return ""
}
