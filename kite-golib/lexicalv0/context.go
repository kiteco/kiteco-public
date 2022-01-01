package lexicalv0

import (
	"go/token"
	"strings"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/javascript"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/vue"
)

// possibleGolangPrefix are syntax tokens that might be prefix of other syntax tokens, like `+` for `++` and `+=`
// TODO: where should this live?
var possibleGolangPrefix = map[token.Token]bool{
	token.ADD:     true,
	token.SUB:     true,
	token.MUL:     true,
	token.QUO:     true,
	token.REM:     true,
	token.AND:     true,
	token.OR:      true,
	token.XOR:     true,
	token.SHL:     true,
	token.SHR:     true,
	token.AND_NOT: true,
	token.ASSIGN:  true,
	token.LSS:     true,
	token.GTR:     true,
}

// PrecededBySpace ...
func PrecededBySpace(b data.SelectedBuffer, tokens []lexer.Token, l lang.Language) bool {
	if b.Selection.Begin == 0 {
		return true
	}
	for _, t := range tokens {
		// If it's in the middle of a token, and it's not a text lang -> return false
		if b.Selection.Begin > t.Start && b.Selection.Begin <= t.End && t.Token != javascript.SymJsxText && l != lang.Text {
			return false
		}
	}
	if unicode.IsSpace(b.RuneBefore(b.Selection.Begin)) {
		return true
	}
	return false
}

// CursorContext ...
type CursorContext struct {
	PrecededBySpace bool
	// Prefix is the part of token under the cursor that is left of the cursor
	Prefix string
	// LineContext contains tokens on the current line, up to but excluding the partial ident under the cursor
	LineContext []lexer.Token
	// LastTokenIdx is the index of the token immediately before the cursor
	LastTokenIdx int
}

// FindContext ...
// TODO: this should probably live in the predict package but that also needs a refactor to move all the model stuff
// into a separate package...
func FindContext(b data.SelectedBuffer, tokens []lexer.Token, langLexer lexer.Lexer) (CursorContext, error) {
	switch langLexer.Lang() {
	case lang.Golang:
		return findGolangContext(b, tokens, langLexer), nil
	case lang.JavaScript:
		return findJavaScriptContext(b, tokens, langLexer), nil
	case lang.Python:
		return findPythonContext(b, tokens, langLexer), nil
	case lang.Text:
		return findTextContext(b, tokens, langLexer), nil
	default:
		return CursorContext{}, errors.Errorf("unsupported language: %s", langLexer.Lang().Name())
	}
}

// LexSelectedBuffer ...
func LexSelectedBuffer(b data.SelectedBuffer, originalLang lang.Language, langLexer lexer.Lexer) ([]lexer.Token, error) {
	var tokens []lexer.Token
	if originalLang == lang.Vue && langLexer.Lang() != lang.Text {
		vueLexer, err := vue.NewLexer()
		if err != nil {
			return nil, err
		}
		raw, err := vueLexer.LexWithOptions([]byte(b.Text()), b.Begin, false)
		tokens = raw
	} else if langLexer.Lang() == lang.Text {
		// For text lexer, we only lex the before-cursor part to avoid weird merging issues
		// like https://github.com/kiteco/kiteco/pull/11749#issuecomment-694593407
		// or https://github.com/kiteco/kiteco/issues/11347#issuecomment-704574043
		raw, err := langLexer.Lex([]byte(b.Text()[:b.End]))
		if err != nil {
			return nil, err
		}
		if len(raw) > 0 && langLexer.IsType(lexer.EOF, raw[len(raw)-1]) {
			raw = raw[:len(raw)-1]
		}
		tokens = raw
	} else {
		raw, err := langLexer.Lex([]byte(b.Text()))
		if err != nil {
			return nil, err
		}
		tokens = raw
	}
	return tokens, nil
}

func getPrefix(full string, lastTokenIdx int, tokens []lexer.Token, sel data.Selection) string {
	if lastTokenIdx+1 >= len(tokens) {
		return full
	}
	excess := tokens[lastTokenIdx+1].End - sel.End
	if excess > len(full) || excess < 0 {
		return full
	}
	return full[:len(full)-excess]
}

func findGolangContext(b data.SelectedBuffer, tokens []lexer.Token, langLexer lexer.Lexer) CursorContext {
	// index of token before cursor
	var lastTokenIdx int
	for lastTokenIdx < len(tokens) && int(tokens[lastTokenIdx].Start+1) <= b.Selection.End {
		lastTokenIdx++
	}
	lastTokenIdx--

	precededBySpace := PrecededBySpace(b, tokens, langLexer.Lang())

	// check for a partially typed token under the cursor, updating lastTokenIdx as necessary
	var prefix string
	if !precededBySpace && lastTokenIdx >= 0 && tokens[lastTokenIdx].End >= b.Selection.End {
		tok := token.Token(tokens[lastTokenIdx].Token)
		var full string
		if tok == token.IDENT || langLexer.IsType(lexer.KEYWORD, tokens[lastTokenIdx]) {
			full = tokens[lastTokenIdx].Lit
			lastTokenIdx--
		} else if possibleGolangPrefix[tok] {
			full = langLexer.TokenName(tokens[lastTokenIdx].Token)
			lastTokenIdx--
		}
		prefix = getPrefix(full, lastTokenIdx, tokens, b.Selection)
	}

	return CursorContext{
		PrecededBySpace: precededBySpace,
		Prefix:          prefix,
		LineContext:     findLineContext(b, tokens, lastTokenIdx),
		LastTokenIdx:    lastTokenIdx,
	}
}

func findJavaScriptContext(b data.SelectedBuffer, tokens []lexer.Token, langLexer lexer.Lexer) CursorContext {
	// index of token before cursor
	var lastTokenIdx int
	for lastTokenIdx < len(tokens) && int(tokens[lastTokenIdx].Start+1) <= b.Selection.End {
		lastTokenIdx++
	}
	lastTokenIdx--

	// If the cursor is in the middle of an empty JSX test, don't include it in context
	if lastTokenIdx >= 0 {
		tok := tokens[lastTokenIdx]
		if tok.Token == javascript.SymJsxText && strings.TrimSpace(tok.Lit) == "" {
			lastTokenIdx--
		}
	}

	// TODO: hacky, for partially typed states like
	// var m $
	// var foo = 1
	// the last token will be on an automatic semicolon so we check for this
	if lastTokenIdx > 1 {
		tok := tokens[lastTokenIdx]
		if tok.Token == javascript.SymAutomaticSemicolon {
			// move the cursor back to the first non whitespace character (without crossing newline boundaries)
			begin := b.Selection.Begin
			b.Buffer.RangeReverse(b.Selection.Begin, func(i int, r rune) bool {
				if r == '\n' || r == '\r' || !unicode.IsSpace(r) {
					return false
				}
				begin--
				return true
			})
			if int(tok.Start) == begin {
				lastTokenIdx--
			}
		}
	}

	precededBySpace := PrecededBySpace(b, tokens, langLexer.Lang())

	// check for a partially typed token under the cursor, updating lastTokenIdx as necessary
	var prefix string
	if !precededBySpace && lastTokenIdx >= 0 && tokens[lastTokenIdx].End >= b.Selection.End {
		tok := tokens[lastTokenIdx]
		var full string
		_, bpe := langLexer.ShouldBPEEncode(tok)
		if bpe || langLexer.IsType(lexer.KEYWORD, tok) {
			full = tok.Lit
			lastTokenIdx--
		}
		prefix = getPrefix(full, lastTokenIdx, tokens, b.Selection)
	}

	return CursorContext{
		PrecededBySpace: precededBySpace,
		Prefix:          prefix,
		LineContext:     findLineContext(b, tokens, lastTokenIdx),
		LastTokenIdx:    lastTokenIdx,
	}
}

func findPythonContext(b data.SelectedBuffer, tokens []lexer.Token, langLexer lexer.Lexer) CursorContext {
	// index of token before cursor
	var lastTokenIdx int
	for lastTokenIdx < len(tokens) && int(tokens[lastTokenIdx].Start+1) <= b.Selection.End {
		lastTokenIdx++
	}
	lastTokenIdx--

	precededBySpace := PrecededBySpace(b, tokens, langLexer.Lang())

	// check for a partially typed token under the cursor, updating lastTokenIdx as necessary
	var prefix string
	if !precededBySpace && lastTokenIdx >= 0 && tokens[lastTokenIdx].End >= b.Selection.End {
		tok := tokens[lastTokenIdx]
		var full string
		_, bpe := langLexer.ShouldBPEEncode(tok)
		if bpe || langLexer.IsType(lexer.KEYWORD, tok) {
			full = tok.Lit
			lastTokenIdx--
		}
		prefix = getPrefix(full, lastTokenIdx, tokens, b.Selection)
	}

	return CursorContext{
		PrecededBySpace: precededBySpace,
		Prefix:          prefix,
		LineContext:     findLineContext(b, tokens, lastTokenIdx),
		LastTokenIdx:    lastTokenIdx,
	}
}

func findTextContext(b data.SelectedBuffer, tokens []lexer.Token, langLexer lexer.Lexer) CursorContext {
	// We only lex the before-cursor part of the buffer,
	// Therefore the lastTokenIdx is just id of the last token in `tokens`
	lastTokenIdx := len(tokens) - 1

	precededBySpace := PrecededBySpace(b, tokens, langLexer.Lang())
	var prefix string
	if !precededBySpace && lastTokenIdx >= 0 {
		tok := tokens[lastTokenIdx]
		var full string
		// If the cursor is directly after a literal token, treat it as a prefix instead
		// and update the lastTokenIdx accordingly
		if text.MaybeIdent(tok.Lit) {
			if _, ok := langLexer.ShouldBPEEncode(tok); ok {
				full = tok.Lit
				lastTokenIdx--
			}
			prefix = getPrefix(full, lastTokenIdx, tokens, b.Selection)
		}
	}

	return CursorContext{
		PrecededBySpace: precededBySpace,
		Prefix:          prefix,
		LineContext:     findLineContext(b, tokens, lastTokenIdx),
		LastTokenIdx:    lastTokenIdx,
	}
}

func findLineContext(b data.SelectedBuffer, tokens []lexer.Token, lastTokenIdx int) []lexer.Token {
	// context on the current line, up to but excluding the partial ident under the cursor
	var lineContext []lexer.Token
	for pos := b.Selection.End - 1; pos >= -1; pos-- {
		// find offset of last newline before cursor, or go to -1
		if pos >= 0 {
			if b.RuneAt(pos) != '\n' {
				continue
			}
		}
		// Position of the last token is before the \n
		if lastTokenIdx > -1 && int(tokens[lastTokenIdx].Start) <= pos {
			break
		}
		// index of first token after newline
		idx := lastTokenIdx
		for idx > 0 && int(tokens[idx-1].Start) > pos {
			idx--
		}
		// because lastTokenIdx might be -1
		if idx < 0 {
			idx = 0
		}

		lineContext = tokens[idx : lastTokenIdx+1]
		break
	}
	return lineContext
}
