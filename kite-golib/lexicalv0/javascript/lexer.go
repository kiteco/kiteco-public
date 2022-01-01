package javascript

import (
	"bytes"
	"regexp"
	"strings"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/javascript"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
)

const (
	subtokenChar = "+"
	terminalChar = "$"

	maxStringLength = 1000

	// We do not add the end token to match the golang behavior and to avoid having to
	// account for this case when doing inference and the user's cursor is at the end
	// of the file.
	includeEndToken = false
)

var (
	// these token ids are coming from treesitter's internal enum representing
	// different token types. for us, this is currently vendored at:
	// vendor/github.com/kiteco/go-tree-sitter/javascript/parser.c
	// w/ the repo @ https://github.com/kiteco/go-tree-sitter
	jsIdentLike = map[int]bool{
		symIdentifier:                       true,
		symJsxIdentifier:                    true,
		symNestedIdentifier:                 true,
		aliasSymShorthandPropertyIdentifier: true,
		aliasSymPropertyIdentifier:          true,
		aliasSymStatementIdentifier:         true,
	}
	jsStringLike = map[int]bool{
		symJsxText:         true,
		auxSymStringToken1: true,
		auxSymStringToken2: true,
		symRegexPattern:    true,
		symRegexFlags:      true,
		symTemplateChars:   true,
	}
	jsStringBPE = map[int]bool{
		auxSymStringToken1: true,
		auxSymStringToken2: true,
		symTemplateChars:   true,
	}
	jsKeywords = map[int]bool{
		anonSymAs:         true,
		anonSymAsync:      true,
		anonSymAwait:      true,
		anonSymBreak:      true,
		anonSymCase:       true,
		anonSymClass:      true,
		anonSymContinue:   true,
		anonSymConst:      true,
		anonSymCatch:      true,
		anonSymDebugger:   true,
		anonSymDefault:    true,
		anonSymDelete:     true,
		anonSymDo:         true,
		anonSymElse:       true,
		anonSymExport:     true,
		anonSymExtends:    true,
		anonSymFinally:    true,
		anonSymFrom:       true,
		anonSymFunction:   true,
		anonSymFor:        true,
		anonSymGet:        true,
		anonSymIf:         true,
		anonSymIn:         true,
		anonSymInstanceof: true,
		anonSymImport:     true,
		anonSymLet:        true,
		anonSymNew:        true,
		anonSymOf:         true,
		anonSymReturn:     true,
		anonSymStatic:     true,
		anonSymSwitch:     true,
		anonSymSet:        true,
		anonSymTarget:     true,
		anonSymThrow:      true,
		anonSymTry:        true,
		anonSymTypeof:     true,
		anonSymVar:        true,
		anonSymVoid:       true,
		anonSymWhile:      true,
		anonSymWith:       true,
		anonSymYield:      true,
		symThis:           true,
		symSuper:          true,
		symTrue:           true,
		symFalse:          true,
		symNull:           true,
		symUndefined:      true,
	}

	jsQuotes = map[int]bool{
		anonSymBquote: true,
		anonSymSquote: true,
		anonSymDquote: true,
	}

	jsParens = map[int]bool{
		anonSymLparen:       true,
		anonSymRparen:       true,
		anonSymLbrace:       true,
		anonSymRbrace:       true,
		anonSymLbrack:       true,
		anonSymRbrack:       true,
		anonSymDollarLbrace: true,
	}
)

// Lexer is a javascript lexer
type Lexer struct {
	*lexer.TreeSitterLexer
	sitterLang *sitter.Language
}

// NewLexer returns a new javascript lexer
func NewLexer() (*Lexer, error) {
	j := &Lexer{
		sitterLang: javascript.GetLanguage(),
	}
	ts, err := lexer.NewTreeSitterLexer(lang.JavaScript, int(j.sitterLang.SymbolCount()), j.extractTreeTokens)
	if err != nil {
		return nil, err
	}
	j.TreeSitterLexer = ts
	return j, nil
}

// Lang implements Lexer
func (j *Lexer) Lang() lang.Language {
	return lang.JavaScript
}

// ShouldBPEEncode implements Lexer
func (j *Lexer) ShouldBPEEncode(tok lexer.Token) ([]string, bool) {
	// Hack to filter comments
	if j.IsType(lexer.COMMENT, tok) {
		return nil, true
	}

	var tokens []string
	if j.IsType(lexer.IDENT, tok) {
		if j.isValidJSIdent(tok.Lit) {
			tokens = append(tokens, j.subtokens(tok)...)
		}
	}

	if _, ok := jsStringBPE[tok.Token]; ok {
		if j.isValidJSString(tok.Lit) {
			tokens = append(tokens, j.subtokens(tok)...)
		}
	}

	return tokens, len(tokens) > 0
}

// MergeBPEEncoded implements Lexer
func (Lexer) MergeBPEEncoded(in []string) []string {
	var idents []string
	var pending []string
	for i, s := range in {
		pending = append(pending, strings.TrimSuffix(s, subtokenChar))
		if strings.HasSuffix(s, terminalChar) || i == len(in)-1 {
			idents = append(idents, strings.TrimSuffix(strings.Join(pending, ""), terminalChar))
			pending = nil
		}
	}
	return idents
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
	for _, tok := range toks {
		if tok.Token == lexer.BPEEncodedTok || jsIdentLike[tok.Token] || jsStringBPE[tok.Token] || jsKeywords[tok.Token] {
			return true
		}
	}
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
func (j *Lexer) IsType(t lexer.TokenType, tok lexer.Token) bool {
	switch t {
	case lexer.IDENT:
		_, ok := jsIdentLike[tok.Token]
		return ok
	case lexer.STRING:
		_, ok := jsStringLike[tok.Token]
		return ok
	case lexer.COMMENT:
		return tok.Token == symComment
	case lexer.LITERAL:
		_, ok := jsStringLike[tok.Token]
		return ok ||
			tok.Token == symNumber ||
			tok.Token == symTrue ||
			tok.Token == symFalse ||
			tok.Token == symNull ||
			tok.Token == symUndefined
	case lexer.SEMICOLON:
		return tok.Token == anonSymSemi || tok.Token == symAutomaticSemicolon
	case lexer.EOF:
		return tok.Token == 0
	case lexer.KEYWORD:
		_, ok := jsKeywords[tok.Token]
		return ok
	case lexer.IMPORT:
		return tok.Token == anonSymImport
	}
	return false
}

// --

// subtokens will simply append terminalChar($) to idents, and split strings by
// space and /, adding the subtokenChar(+) for any part within a string, and terminate
// with the terminalChar($)
func (j *Lexer) subtokens(tok lexer.Token) []string {
	if j.IsType(lexer.IDENT, tok) {
		return []string{tok.Lit + terminalChar}
	}

	// Only create subtokens for strings, and only then split on whitespace
	if _, ok := jsStringBPE[tok.Token]; !ok {
		return nil
	}

	parts := SplitString(tok.Lit)

	var subtokens []string
	for idx, p := range parts {
		if idx < len(parts)-1 {
			subtokens = append(subtokens, p+subtokenChar)
		} else {
			subtokens = append(subtokens, p+terminalChar)
		}
	}
	return subtokens
}

// SplitString splits strings by certain characters
func SplitString(str string) []string {
	var parts []string
	for {
		pos := strings.IndexAny(str, " /\n")
		if pos < 0 {
			if len(str) > 0 {
				parts = append(parts, str)
			}
			break
		}
		if pos == 0 {
			parts = append(parts, str[:pos+1])
			str = str[pos+1:]
			continue
		}
		parts = append(parts, str[:pos])
		str = str[pos:]
	}
	return parts
}

func (j *Lexer) isValidJSString(str string) bool {
	if len(str) > maxStringLength {
		return false
	}
	for _, c := range str {
		if !j.isAlphaNum(c) && !strings.ContainsRune("_-.#/ \n", c) {
			return false
		}
	}
	return true
}

func (j *Lexer) isValidJSIdent(str string) bool {
	for _, c := range str {
		if !j.isAlphaNum(c) && !strings.ContainsRune("_-$", c) {
			return false
		}
	}
	return true
}

func (j *Lexer) isAlphaNum(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '1' && c <= '9')
}

// TokensInRanges returns the javascript tokens found in the specified ranges
// of the source input buf. It reuses the provided parser and sets its language
// and ranges to process only the javascript parts. If ranges is empty,
// it returns nil, nil. It does not close the parser when done - a caller
// should take care of this when it is no longer needed.
func (j *Lexer) TokensInRanges(parser *sitter.Parser, buf []byte, ranges []sitter.Range) (tokens []treesitter.Token, err error) {
	if len(ranges) == 0 {
		return nil, nil
	}
	parser.SetLanguage(j.sitterLang)
	parser.SetIncludedRanges(ranges)
	tree := parser.Parse(buf)
	defer tree.Close()
	return j.extractTreeTokens(buf, parser, tree)
}

func (j *Lexer) extractTreeTokens(buf []byte, parser *sitter.Parser, tree *sitter.Tree) (tokens []treesitter.Token, err error) {
	root := tree.RootNode()

	// extract all tokens from the parsed tree
	t := &tokenizer{
		buf:  buf,
		lang: j.sitterLang,
	}
	treesitter.Walk(t, root)

	// might need a final ASI
	if t.fromFunOrClassDecl {
		if len(t.tokens) > 0 {
			if lastTok := t.tokens[len(t.tokens)-1]; lastTok.Lit == "}" {
				t.appendSym(symAutomaticSemicolon, uint32(lastTok.End), uint32(lastTok.End))
			}
		}
	}
	tokens = t.tokens

	if includeEndToken {
		endb := int(root.EndByte())
		tokens = append(tokens, treesitter.Token{
			SymbolName: treesitter.EndSymbolName,
			Symbol:     treesitter.EndSymbolIdx,
			Start:      endb,
			End:        endb,
		})
	}

	// make sure that any semi colon that has start == end is mapped to symAutomaticSemicolon
	// any other tokens with start == end are removed and no consecutive symAutomaticSemicolons are allowed
	// TODO: could do this as part of the walk.
	var cleaned []treesitter.Token
	for i, t := range tokens {
		// End-of-file is allowed
		if i == len(tokens)-1 && includeEndToken {
			cleaned = append(cleaned, t)
			continue
		}
		if t.Symbol == anonSymSemi && t.Start == t.End {
			t.Symbol = symAutomaticSemicolon
			t.SymbolName = j.sitterLang.SymbolName(sitter.Symbol(symAutomaticSemicolon))
		}
		// Avoid repeating auto-semicolons
		if t.Symbol == symAutomaticSemicolon && len(cleaned) > 0 && cleaned[len(cleaned)-1].Symbol == symAutomaticSemicolon {
			continue
		}
		if t.Start == t.End && t.Symbol != symAutomaticSemicolon {
			continue
		}

		cleaned = append(cleaned, t)
	}

	return cleaned, nil
}

type tokenizer struct {
	buf                []byte
	lang               *sitter.Language
	tokens             []treesitter.Token
	fromFunOrClassDecl bool
}

var jsNodeTypeRequireASI = map[string]bool{
	"empty_statement":      true,
	"variable_declaration": true,
	"lexical_declaration":  true,
	"return_statement":     true,
	"debugger_statement":   true,
	"expression_statement": true,
	"do_statement":         true,
	"break_statement":      true,
	"continue_statement":   true,
	"throw_statement":      true,
	"import_statement":     true,
	"export_statement":     true,
}

// regular expression to check if a string could be a valid JS identifier.
var reIdentLike = regexp.MustCompile(`^[\pL$_][\pL$_\d]*$`)

func (t *tokenizer) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		return nil
	}

	if t.fromFunOrClassDecl {
		// special ASI case: after a function declaration, if the trailing '}' is followed
		// by a newline before the next node (and that node doesn't start with '(' or '['),
		// insert a semicolon.
		t.fromFunOrClassDecl = false
		if len(t.tokens) > 0 {
			// if the last character was '}' and it is separated from this node
			// by at least one newline, auto-add a semicolon.
			if lastTok := t.tokens[len(t.tokens)-1]; lastTok.Lit == "}" {
				start := int(n.StartByte())
				if bytes.Contains(t.buf[lastTok.End:start], []byte{'\n'}) && t.buf[start] != '[' && t.buf[start] != '(' {
					t.appendSym(symAutomaticSemicolon, uint32(lastTok.End), uint32(lastTok.End))
				}
			}
		}
	}

	//fmt.Printf(">>> %s | %d | %s | err? %t | miss? %t | %s\n", n, n.Symbol(), n.Type(), n.HasError(), n.IsMissing(), n.Content(t.buf))
	typ := n.Type()
	switch {
	case int(n.Symbol()) == symERROR:
		errEnd := n.EndByte()
		remainStart := n.StartByte()
		count := int(n.ChildCount())

		// process each child of the ERROR node
		for i := 0; i < count; i++ {
			child := n.Child(i)
			treesitter.Walk(t, child)
			remainStart = child.EndByte()
		}
		// and then generate an ERROR token for the remaining source not
		// covered by any child node (if there is such source)
		if remainStart < errEnd {
			if reIdentLike.Match(t.buf[remainStart:errEnd]) {
				t.appendSym(symIdentifier, remainStart, errEnd)
			} else {
				t.appendSym(symERROR, remainStart, errEnd)
			}
		}
		return nil

	case n.ChildCount() == 0:
		// a terminal token
		t.append(n)

	case jsNodeTypeRequireASI[typ]:
		// this is a node type that requires automatic semicolon insertion.
		count := int(n.ChildCount())

		// visit all children, then add automatic semicolon if it is still required
		// (i.e. the last child might not be a terminal, so this needs to happen
		// recursively).
		for i := 0; i < count; i++ {
			treesitter.Walk(t, n.Child(i))
		}
		if len(t.tokens) > 0 {
			lastTok := t.tokens[len(t.tokens)-1]
			if lastTok.Symbol != anonSymSemi && lastTok.Symbol != symAutomaticSemicolon {
				t.appendSym(symAutomaticSemicolon, uint32(lastTok.End), uint32(lastTok.End))
			}
		}
		return nil

	case typ == "function_declaration" || typ == "class_declaration":
		// keep track when we exit from a func or class decl for automatic
		// semicolon insertion.
		count := int(n.ChildCount())
		for i := 0; i < count; i++ {
			treesitter.Walk(t, n.Child(i))
		}
		t.fromFunOrClassDecl = true
		return nil

	case typ == "string" || typ == "template_string":
		// the format of such nodes is e.g. (warning: ascii art)
		// "this\nis a string\n"
		// |    |            | |
		// ↓    |            | |
		// Terminal (")      | |
		//      |            | |
		//      ↓            | |
		//   Escape sequence (\n)
		//                   | |
		//                   | |
		//                   ↓ |
		//            Escape sequence (\n)
		//                     |
		//                     ↓
		//                  Terminal (")
		//
		// And the actual string literal is to be extracted from souce in-between
		// those nodes.
		var (
			openSym int
			nextPos uint32
		)
		if typ == "template_string" {
			openSym = symTemplateChars
		}
		count := int(n.ChildCount())
		for i := 0; i < count; i++ {
			child := n.Child(i)
			start := child.StartByte()

			if start > nextPos && nextPos != 0 {
				t.appendSym(openSym, nextPos, start)
			}

			treesitter.Walk(t, child)
			nextPos = child.EndByte()
			if openSym == 0 {
				// for strings, the symbol of the string literal between quotes is
				// whatever the open quote's symbol is + 1 (i.e. if the quote is
				// anonSymDquote, the string literal auxSymStringToken1 is
				// anonSymDquote+1, same for anonSymSquote), but for template_string,
				// always use _template_chars (see if typ == "template_string" above,
				// where symTemplateChars is used if that's the case).
				openSym = int(child.Symbol()) + 1
			}
		}
		return nil
	}

	return t
}

func (t *tokenizer) append(n *sitter.Node) treesitter.Token {
	sym := int(n.Symbol())
	return t.appendSym(sym, n.StartByte(), n.EndByte())
}

func (t *tokenizer) appendSym(sym int, start, end uint32) treesitter.Token {
	tok := treesitter.Token{
		Symbol:     sym,
		SymbolName: t.lang.SymbolName(sitter.Symbol(sym)),
		Start:      int(start),
		End:        int(end),
		Lit:        string(t.buf[start:end]),
	}
	t.tokens = append(t.tokens, tok)
	return tok
}
