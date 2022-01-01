package lexicalv0

import (
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/css"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/golang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/html"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/javascript"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/python"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/vue"
)

// ErrUnsupportedLexerLang ...
var ErrUnsupportedLexerLang = errors.New("language does not have a lexer")

// NewLexerForMetrics is specifically used for constructing a new lexer for completion metrics
// TODO: not sure where this should go
func NewLexerForMetrics(l lang.Language) (lexer.Lexer, error) {
	switch {
	case l == lang.Python:
		return python.Lexer{}, nil
	case AllLangsGroup.Contains(l):
		return text.Lexer{}, nil
	default:
		return nil, ErrUnsupportedLexerLang
	}
}

// NewLexer returns a lexer for the provided language
func NewLexer(l lang.Language) (lexer.Lexer, error) {
	return NewLexerWithOpts(l, false)
}

// NewLexerWithOpts ...
func NewLexerWithOpts(l lang.Language, useBytes bool) (lexer.Lexer, error) {
	switch l {
	case lang.Golang:
		return golang.Lexer{}, nil
	case lang.JavaScript:
		return javascript.NewLexer()
	case lang.Python:
		return python.Lexer{}, nil
	case lang.Vue:
		return vue.NewLexer()
	case lang.CSS:
		return css.NewLexer()
	case lang.HTML:
		return html.NewLexer()
	case lang.Text:
		return text.NewLexer(), nil
	}
	return nil, ErrUnsupportedLexerLang
}

// Lex is a helper function that simply provides lexical tokens for a buffer + language
func Lex(buf []byte, l lang.Language) ([]lexer.Token, error) {
	lexer, err := NewLexer(l)
	if err != nil {
		return nil, err
	}
	return lexer.Lex(buf)
}
