package python

import (
	"strings"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/go-tree-sitter/python"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer/treesitter"
)

const (
	terminalChar = "$"
)

var (
	pyIdentLike = map[int]bool{
		symIdentifier: true,
		anonSymPrint:  true,
		anonSymExec:   true,
	}
	pyStringLike = map[int]bool{
		symString:             true,
		symConcatenatedString: true,
		// Right now we don't lex `symString` further, so we won't see these as single tokens
		symStringContent:  true,
		symEscapeSequence: true,
	}
	pyStatements = map[int]bool{
		symStatement:             true,
		symSimpleStatements:      true,
		symImportStatement:       true,
		symImportFromStatement:   true,
		symFutureImportStatement: true,
		symAssertStatement:       true,
		symPrintStatement:        true,
		symExpressionStatement:   true,
		symReturnStatement:       true,
		symDeleteStatement:       true,
		symRaiseStatement:        true,
		symPassStatement:         true,
		symBreakStatement:        true,
		symContinueStatement:     true,
		symIfStatement:           true,
		symWhileStatement:        true,
		symForStatement:          true,
		symTryStatement:          true,
		symWithStatement:         true,
		symGlobalStatement:       true,
		symNonlocalStatement:     true,
		symExecStatement:         true,
		symClassDefinition:       true,
		symFunctionDefinition:    true,
		symDecoratedDefinition:   true,
	}
	pyClauses = map[int]bool{
		symIfClause:      true,
		symElifClause:    true,
		symElseClause:    true,
		symExceptClause:  true,
		symFinallyClause: true,
	}
	pyKeywords = map[int]bool{
		anonSymAnd:      true,
		anonSymAs:       true,
		anonSymAssert:   true,
		anonSymAsync:    true,
		anonSymAwait:    true,
		anonSymBreak:    true,
		anonSymClass:    true,
		anonSymContinue: true,
		anonSymDef:      true,
		anonSymDel:      true,
		anonSymElif:     true,
		anonSymElse:     true,
		anonSymExcept:   true,
		anonSymExec:     true,
		anonSymFinally:  true,
		anonSymFor:      true,
		anonSymFrom:     true,
		anonSymGlobal:   true,
		anonSymIf:       true,
		anonSymImport:   true,
		anonSymIn:       true,
		anonSymIs:       true,
		anonSymLambda:   true,
		anonSymNonlocal: true,
		anonSymNot:      true,
		anonSymOr:       true,
		anonSymPrint:    true,
		anonSymRaise:    true,
		anonSymReturn:   true,
		anonSymTry:      true,
		anonSymWhile:    true,
		anonSymWith:     true,
		anonSymYield:    true,
	}

	pyParens = map[int]bool{
		anonSymLparen: true,
		anonSymRparen: true,
		anonSymLbrace: true,
		anonSymRbrace: true,
		anonSymLbrack: true,
		anonSymRbrack: true,
	}

	extraTokens = map[int]extraToken{
		// endOfStatement indicates a new line
		endOfStatement: {id: endOfStatement, name: "end_of_statement"},
		// startOfBlock indicates a new line and indent
		startOfBlock: {id: startOfBlock, name: "start_of_block"},
		// endOfBlock always follows endOfStatement, and it indicates an extra dedent
		endOfBlock: {id: endOfBlock, name: "end_of_block"},
	}
	remappings = map[int]remappedToken{
		65535: {offset: 1, name: "KITE_ILLEGAL"},
	}
)

type extraToken struct {
	id   int
	name string
}

type remappedToken struct {
	offset int
	name   string
}

// NewLexer returns a new python lexer.
func NewLexer() (lexer.Lexer, error) {
	return Lexer{}, nil
}

// Lexer is a python lexer.
type Lexer struct{}

// Lang implements Lexer.
func (Lexer) Lang() lang.Language {
	return lang.Python
}

// Lex implements lexer
func (l Lexer) Lex(buf []byte) ([]lexer.Token, error) {
	tokens, err := treesitter.Lex(buf, python.GetLanguage(), l.extractTreeTokens)
	if err != nil {
		return nil, err
	}

	var ret []lexer.Token
	for _, tok := range tokens {
		if remap, ok := remappings[tok.Symbol]; ok {
			tok.Symbol = len(allTokens) + remap.offset
		}
		ret = append(ret, lexer.Token{
			Token: tok.Symbol,
			Lit:   tok.Lit,
			Start: int(tok.Start),
			End:   int(tok.End),
		})
	}
	return ret, nil
}

// ShouldBPEEncode implements Lexer.
func (l Lexer) ShouldBPEEncode(tok lexer.Token) ([]string, bool) {
	// Hack to filter comments - we say we want to BPE encode, but then return nothing
	if l.IsType(lexer.COMMENT, tok) {
		return nil, true
	}

	// Only need to encode idents, and we don't use subtokens, so just use the terminalChar
	if l.IsType(lexer.IDENT, tok) {
		// NOTE(tarak): There are some crazy long idents, some of which cause BPE encoding
		// to hang. This was a quick hack to get around that issue.
		if len(tok.Lit) <= 80 {
			return []string{tok.Lit + terminalChar}, true
		}
	}

	return nil, false
}

// MergeBPEEncoded implements Lexer.
func (Lexer) MergeBPEEncoded(in []string) []string {
	var idents []string
	var pending []string
	for i, s := range in {
		pending = append(pending, s)
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
		if tok.Token == lexer.BPEEncodedTok || pyIdentLike[tok.Token] || pyKeywords[tok.Token] {
			return true
		}
		if tok.Token == symTrue || tok.Token == symFalse || tok.Token == symNone {
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

// NumTokens implements Lexer
func (Lexer) NumTokens() int {
	// All tokens and the error token
	return int(python.GetLanguage().SymbolCount()) + len(extraTokens) + len(remappings)
}

// Tokens implements Lexer
func (l Lexer) Tokens() []lexer.Token {
	var toks []lexer.Token
	for i := 0; i < int(python.GetLanguage().SymbolCount()); i++ {
		toks = append(toks, lexer.Token{
			Token: i,
			Lit:   python.GetLanguage().SymbolName(sitter.Symbol(i)),
		})
	}
	// Add extra tokens
	for _, e := range extraTokens {
		toks = append(toks, lexer.Token{
			Token: e.id,
			Lit:   e.name,
		})
	}
	for _, remap := range remappings {
		toks = append(toks, lexer.Token{
			Token: l.NumTokens() + remap.offset,
			Lit:   remap.name,
		})

	}
	return toks
}

// TokenName implements Lexer
func (l Lexer) TokenName(tok int) string {
	if tok >= len(allTokens) {
		for _, remap := range remappings {
			if tok == len(allTokens)+remap.offset {
				return remap.name
			}
		}
	}
	if tok >= int(python.GetLanguage().SymbolCount()) {
		for _, e := range extraTokens {
			if tok == e.id {
				return e.name
			}
		}
	}
	return python.GetLanguage().SymbolName(sitter.Symbol(tok))
}

// IsType returns whether a token is an Ident
func (Lexer) IsType(t lexer.TokenType, tok lexer.Token) bool {
	switch t {
	case lexer.IDENT:
		_, ok := pyIdentLike[tok.Token]
		return ok
	case lexer.STRING:
		_, ok := pyStringLike[tok.Token]
		return ok
	case lexer.COMMENT:
		return tok.Token == symComment
	case lexer.LITERAL:
		_, ok := pyStringLike[tok.Token]
		return ok ||
			tok.Token == symInteger ||
			tok.Token == symTrue ||
			tok.Token == symFalse ||
			tok.Token == symNone ||
			tok.Token == symFloat
	case lexer.SEMICOLON:
		// TODO: would indent/dedent tokens be considered semicolons for python?
		// Note that currently the lexer doesn't generate those tokens (in fact
		// treesitter doesn't create nodes for those)
		return tok.Token == symSemicolon
	case lexer.EOF:
		return tok.Token == 0
	case lexer.KEYWORD:
		_, ok := pyKeywords[tok.Token]
		return ok
	case lexer.IMPORT:
		return tok.Token == anonSymImport
	}
	return false
}

func (l *Lexer) extractTreeTokens(buf []byte, parser *sitter.Parser, tree *sitter.Tree) (tokens []treesitter.Token, err error) {
	root := tree.RootNode()

	// extract all tokens from the parsed tree
	t := &tokenizer{
		buf:  buf,
		lang: python.GetLanguage(),
	}
	treesitter.Walk(t, root)
	return t.tokens, nil
}

type tokenizer struct {
	buf    []byte
	lang   *sitter.Language
	tokens []treesitter.Token
}

func (t *tokenizer) Visit(n *sitter.Node) treesitter.Visitor {
	if n == nil {
		return nil
	}
	sym := int(n.Symbol())

	switch {
	case sym == symString:
		// TODO: when we enable BPE-encode string content, include the following part.
		//// NOTE: the python treesitter parser maps all "string start" and "string end"
		//// nodes to the '"' double quote token, so regardless of the actual string
		//// delimiters used in the code (e.g. 'abc', "abc", """abc""", '''abc'''),
		//// all get mapped to the token types '"' 'abc' and '"' (but the associated
		//// literals have the right values).
		//var nextPos uint32
		//count := int(n.ChildCount())
		//for i := 0; i < count; i++ {
		//	child := n.Child(i)
		//	start := child.StartByte()
		//
		//	if start > nextPos && nextPos != 0 {
		//		t.appendSym(symStringContent, nextPos, start)
		//	}
		//	treesitter.Walk(t, child)
		//	nextPos = child.EndByte()
		//}
		t.append(n)
		return nil
	case pyStatements[sym]:
		count := int(n.ChildCount())
		var hasBlock bool
		for i := 0; i < count; i++ {
			child := n.Child(i)
			if int(child.Symbol()) == symBlock {
				hasBlock = true
			}
			treesitter.Walk(t, child)
		}
		if !hasBlock {
			t.appendExtraToken(endOfStatement, n.EndByte())
		}
		return nil
	case sym == symBlock:
		t.appendExtraToken(startOfBlock, n.StartByte())
		count := int(n.ChildCount())
		for i := 0; i < count; i++ {
			treesitter.Walk(t, n.Child(i))
		}
		t.appendExtraToken(endOfBlock, n.EndByte())
		return nil
	case n.ChildCount() == 0:
		// a terminal token
		t.append(n)
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

func (t *tokenizer) appendExtraToken(sym int, pos uint32) treesitter.Token {
	tok := treesitter.Token{
		Symbol:     sym,
		SymbolName: extraTokens[sym].name,
		Start:      int(pos),
		End:        int(pos),
		Lit:        extraTokens[sym].name,
	}
	t.tokens = append(t.tokens, tok)
	return tok
}
