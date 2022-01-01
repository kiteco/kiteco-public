package main

import (
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
)

// Normalize text tokens by collapsing identifiers to IDENT and literals to LIT
func Normalize(textToks []lexer.Token, nativeLexer lexer.Lexer) (string, error) {
	rendered, ok := text.Render(nativeLexer.Lang(), []lexer.Token{}, textToks)
	if !ok {
		return "", errors.New("rendering unvalid")
	}
	s := rendered.ForFormat()
	nativeToks, err := nativeLexer.Lex([]byte(s))
	if err != nil {
		return "", errors.Wrapf(err, "unable to lex")
	}

	var parts []string
	var last int
	for _, tok := range nativeToks {
		if last < tok.Start {
			parts = append(parts, s[last:tok.Start])
		}
		switch {
		case nativeLexer.IsType(lexer.IDENT, tok):
			parts = append(parts, "IDENT")
		case nativeLexer.IsType(lexer.COMMENT, tok):
			parts = append(parts, "COMMENT")
		case nativeLexer.IsType(lexer.LITERAL, tok):
			parts = append(parts, "LIT")
		case nativeLexer.IsType(lexer.SEMICOLON, tok) && tok.Lit != ";":
			// skip automatically inserted semicolons, still need to
			// move last forward below since for golang an auto semicolon
			// has a literal of "\n" which actually appears in the file
		default:
			parts = append(parts, tok.Lit)
		}
		last = tok.End
	}

	return strings.Join(parts, ""), nil
}
