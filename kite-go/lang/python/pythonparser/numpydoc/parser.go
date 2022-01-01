package numpydoc

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/parsing"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/ast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/internal/inline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/internal/pigeon"
)

const (
	defaultMaxLines = 1000
	defaultMaxExpr  = 1e5
)

// The list of section headers where content must be parsed as
// definition lists, lowercase as the lookup is case-insensitive.
var definitionListSections = map[string]bool{
	"parameters":       true,
	"returns":          true,
	"yields":           true,
	"other parameters": true,
	"raises":           true,
	"warns":            true,
	"see also":         true,
	"attributes":       true,
	"methods":          true,
}

// Option for parsing.
type Option func(*parser) Option

// MaxExpressions that the parser will try to
// parse (including backtracking) before failing.
//
// Default value is 1e5. A value of 0 means an unlimited number of
// expressions will be parsed.
func MaxExpressions(cnt uint64) Option {
	return func(p *parser) Option {
		oldCnt := p.maxExpressions
		p.maxExpressions = cnt
		return MaxExpressions(oldCnt)
	}
}

// MaxLines that the parser will try to
// parse before returning a partial node and stopping with an error.
//
// Default value is 1000. A value of 0 means an unlimited number of
// lines will be parsed.
func MaxLines(cnt uint64) Option {
	return func(p *parser) Option {
		oldCnt := p.maxLines
		p.maxLines = cnt
		return MaxLines(oldCnt)
	}
}

// Parse parses src using the argspec parser. It returns the
// resulting node and any error returned by the parser.
// It may return a partial result and an error if a limit was reached
// during parsing (e.g. a maximum number of lines).
func Parse(src []byte, opts ...Option) (*ast.Doc, error) {
	p := newParser(src, opts...)
	return p.parse(pigeon.MaxExpressions(p.maxExpressions),
		pigeon.GlobalStore(pigeon.DefinitionListSectionsKey, definitionListSections))
}

type parser struct {
	src            []byte
	maxExpressions uint64
	maxLines       uint64
	tooManyLines   bool
}

func newParser(src []byte, opts ...Option) *parser {
	p := &parser{
		src:            src,
		maxLines:       defaultMaxLines,
		maxExpressions: defaultMaxExpr,
	}
	for _, opt := range opts {
		opt(p)
	}

	p.src, p.tooManyLines = parsing.TrimMaxLines(src, p.maxLines)

	// NOTE: always add a blank line at the end of the input because the
	// PEG rules are line-based, so each line must end with a newline
	p.src = append(p.src, '\n')

	return p
}

func (p *parser) parse(opts ...pigeon.Option) (*ast.Doc, error) {
	n, err := pigeon.Parse("", p.src, opts...)
	doc, _ := n.(*ast.Doc)

	if err == nil && p.tooManyLines {
		err = errors.TooManyLines
	}

	// parse the inline markup
	if doc != nil {
		v := &inlineVisitor{maxExpressions: p.maxExpressions}
		ast.Walk(v, doc)
		if err == nil {
			err = v.err
		}
	}
	return doc, err
}

type inlineVisitor struct {
	maxExpressions uint64
	err            error
}

func (v *inlineVisitor) Visit(n ast.Node) ast.Visitor {
	if v.err != nil {
		return nil
	}

	// only process paragraphs with a single ast.Text node
	p, ok := n.(*ast.Paragraph)
	if !ok {
		return v
	}

	if len(p.Content) != 1 {
		return nil
	}
	text, ok := p.Content[0].(ast.Text)
	if !ok {
		return nil
	}

	val, err := inline.Parse("", []byte(text), inline.MaxExpressions(v.maxExpressions))
	if v.err == nil && err != nil {
		v.err = err
	}
	newp, ok := val.(*ast.Paragraph)
	if ok {
		p.Content = newp.Content
	}
	return nil
}
