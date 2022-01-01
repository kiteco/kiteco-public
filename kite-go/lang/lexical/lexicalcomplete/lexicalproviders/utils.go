package lexicalproviders

import (
	"go/token"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/golang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/javascript"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/python"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

func suppressLang(nativeLang lang.Language, in Inputs) bool {
	if in.Begin == 0 {
		return false
	}

	if in.SelectedBuffer.Begin != in.SelectedBuffer.End {
		return true
	}

	// turn off completions in comments for any language
	if in.Lexer.Lang() == lang.Text && text.CursorInComment(in.SelectedBuffer, nativeLang) {
		return true
	}

	beforeCursor := in.TextAt(data.Selection{Begin: in.Begin - 1, End: in.Begin})
	if in.Lexer.Lang() == lang.Text {
		var ext string
		if e := filepath.Ext(in.PredictInputs.FilePath); len(e) > 0 {
			ext = e[1:]
		}
		if _, ok := textSuppressAfter[ext]; ok {
			return textSuppressAfter[ext][beforeCursor]
		}
	}

	switch nativeLang {
	case lang.Golang:
		if in.Lexer.Lang() != lang.Text {
			if in.LastTokenIdx >= 0 {
				lastToken := in.Tokens[in.LastTokenIdx]
				// make sure we turn off completions in comments and strings
				if in.Lexer.IsType(lexer.COMMENT, lastToken) || in.Lexer.IsType(lexer.STRING, lastToken) {
					return true
				}
			}
		}

		// Do not trigger completions if the cursor is immediately after ){};,
		// Or cursor is after ( unless there's an ident immediately before the parens
		if golangSuppressAfter[beforeCursor] {
			return true
		}
		if beforeCursor == "(" && in.LastTokenIdx > 1 &&
			!in.Lexer.IsType(lexer.IDENT, in.Tokens[in.LastTokenIdx-1]) {
			return true
		}
	case lang.Python:
		if in.Lexer.Lang() != lang.Text {
			// Turn off completions in string
			if in.LastTokenIdx >= 0 {
				inToken := in.Tokens[in.LastTokenIdx].Token
				if inToken == python.SymString || inToken == python.SymComment {
					return true
				}
			}

			// Turn off completion for incomplete strings
			// Look for quotes in the context (Completed strings will be lexed as one token `symString`
			// But incomplete ones will have single quotes
			for _, tok := range in.Tokens {
				if tok.Token == python.AnonSymDquoteStart || tok.Token == python.AnonSymDquoteEnd {
					return true
				}
			}
		}

		// Do not trigger completions if the cursor is immediately after "':)}]\
		// Or cursor is after ({[ unless there's an ident immediately before the parens
		if pythonSuppressAfter[beforeCursor] {
			return true
		}
		if pythonMaybeSuppressAfter[beforeCursor] && in.LastTokenIdx > 1 &&
			!in.Lexer.IsType(lexer.IDENT, in.Tokens[in.LastTokenIdx-1]) {
			return true
		}
	case lang.JavaScript:
		if in.Lexer.Lang() != lang.Text {
			// Turn off completions in JSX text or in comments
			if in.LastTokenIdx >= 0 {
				if last := in.Tokens[in.LastTokenIdx].Token; last == javascript.SymJsxText || last == javascript.SymComment {
					return true
				}
			}
		}

		// Do not trigger completions if the cursor is immediately after ){};,>
		// Or cursor is after ( unless there's an ident immediately before the parens
		if javascriptSuppressAfter[beforeCursor] {
			return true
		}
		if beforeCursor == "(" && in.LastTokenIdx > 1 &&
			!in.Lexer.IsType(lexer.IDENT, in.Tokens[in.LastTokenIdx-1]) {
			return true
		}
	}

	return false
}

func useCompletionPreRender(nativeLang lang.Language, p predict.Predicted, in Inputs) bool {
	// Abandon empty completions
	if len(p.Tokens) == 0 {
		return false
	}

	// Abandon completions that end with incomplete tokens
	if p.EndsWithIncompleteTok {
		return false
	}

	// Abandon completions with no identifiers or keywords
	if !in.Lexer.ContainsIdentOrKeyword(p.Tokens) || in.Lexer.HasInvalidToken(p.Tokens) {
		return false
	}

	if !HasValidSuffix(p.Tokens[len(p.Tokens)-1].Lit, in.PredictInputs.FilePath) {
		return false
	}

	// In some cases, abandon completions that are not single token
	if requireSingleToken(in) && len(p.Tokens) != 1 {
		return false
	}

	switch nativeLang {
	case lang.Golang:
		// Abandon completions with only a placeholder
		if len(p.Tokens) == 1 && golang.TypeLiterals[token.Token(p.Tokens[0].Token)] {
			return false
		}

		// Abandon completions with import, since users probably use goimports
		if containsType(p.Tokens, lexer.IMPORT, in.Lexer) {
			return false
		}
	case lang.Python:
		// Abandon completions with only a placeholder
		if len(p.Tokens) == 1 {
			if _, ok := python.TypeLiterals[p.Tokens[0].Token]; ok {
				return false
			}
		}

		// If there's a prefix, disallow completions that start with a placeholder
		if _, ok := python.TypeLiterals[p.Tokens[0].Token]; ok && p.Prefix != "" {
			return false
		}
	case lang.JavaScript:
		// Abandon completions with only a placeholder
		if len(p.Tokens) == 1 {
			if _, ok := javascript.TypeLiterals[p.Tokens[0].Token]; ok {
				return false
			}
		}

		// For completions ending with (potentially part of) JSX closing tag
		// abandon completion if that part already appears after the cursor
		ltIndex := -1
		for i, tok := range p.Tokens {
			if tok.Lit == "<" {
				ltIndex = i
				break
			}
		}
		if ltIndex != -1 {
			var closingTag string
			for _, tok := range p.Tokens[ltIndex:] {
				closingTag += tok.Lit
			}
			afterCursor := strings.TrimLeft(in.Text()[in.End:], " ")
			if strings.HasPrefix(afterCursor, closingTag) {
				return false
			}
		}
	}

	return true
}

func useCompletionPostRender(nativeLang lang.Language, c data.Completion, in Inputs, p predict.Predicted) bool {
	// Abandon if the prefix does not match
	if !strings.HasPrefix(strings.ToLower(c.Snippet.ForFormat()), strings.ToLower(p.Prefix)) {
		return false
	}

	// Safety checks for replace position
	if c.Replace.Begin < 0 || c.Replace.End > len(in.Text()) || c.Replace.Begin > c.Replace.End {
		rollbar.Error(errors.New("invalid completion replace bounds"), c)
		return false
	}

	// Safety checks for placeholder positions
	for _, p := range c.Snippet.Placeholders() {
		if p.Begin < 0 || p.End > len(c.Snippet.Text) || p.Begin > p.End {
			rollbar.Error(errors.New("invalid completion placeholder bounds"), c)
			return false
		}
	}

	overlap := OverlapSize(in.SelectedBuffer, c.Snippet.Text)
	extension := c.Snippet.Text[overlap:]
	// Abandon completions whose extension is empty or starts with \n
	if len(extension) == 0 {
		return false
	}
	if strings.HasPrefix(extension, "\n") {
		return false
	}

	//Abandon completions which have consecutive \n
	if strings.Contains(c.Snippet.Text, "\n\n") || strings.Count(c.Snippet.Text, "\r\n") > 1 {
		return false
	}

	return true
}
