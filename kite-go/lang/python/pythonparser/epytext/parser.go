package epytext

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/internal/pigeon"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/parsing"
)

const (
	defaultMaxLines = 2000
	defaultMaxExpr  = 1e6
	defaultOptimize = true
)

// Option for parsing.
type Option func(*parser) Option

// MaxExpressions that the parser will try to
// parse (including backtracking) before failing.
//
// Default value is 1e6. A value of 0 means an unlimited number of
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
// Default value is 2000. A value of 0 means an unlimited number of
// lines will be parsed.
func MaxLines(cnt uint64) Option {
	return func(p *parser) Option {
		oldCnt := p.maxLines
		p.maxLines = cnt
		return MaxLines(oldCnt)
	}
}

// Optimize indicates if the returned AST is optimized. An optimized AST
// has its empty nodes removed and merges consecutive Text nodes into one.
// Default value is true.
func Optimize(b bool) Option {
	return func(p *parser) Option {
		oldOpt := p.optimizeAST
		p.optimizeAST = b
		return Optimize(oldOpt)
	}
}

// Parse parses src using the epytext parser. It returns the
// resulting *ast.DocBlock node and any error returned by the parser.
// It may return a partial result and an error if a limit was reached
// during parsing (e.g. a maximum number of lines).
func Parse(src []byte, opts ...Option) (*ast.DocBlock, error) {
	p := newParser(src, opts...)
	return p.parse(pigeon.MaxExpressions(p.maxExpressions))
}

type parser struct {
	src            []byte
	maxExpressions uint64
	maxLines       uint64
	optimizeAST    bool
	tooManyLines   bool
}

func newParser(src []byte, opts ...Option) *parser {
	p := &parser{
		src:            src,
		maxLines:       defaultMaxLines,
		maxExpressions: defaultMaxExpr,
		optimizeAST:    defaultOptimize,
	}
	for _, opt := range opts {
		opt(p)
	}

	p.src, p.tooManyLines = parsing.TrimMaxLines(src, p.maxLines)

	// NOTE: always add a blank line at the end of the input because the
	// PEG rules are line-based, so each line must end with a newline
	// (cannot use `EndLine <- EOL / EOF` in a rule as EOF does not
	// consume, so when used in conjunction with `_` which means zero
	// or more whitespace, e.g. `_ EndLine`, it is valid to not make
	// any progress and it loops "forever").
	p.src = append(p.src, '\n')

	return p
}

func (p *parser) parse(opts ...pigeon.Option) (*ast.DocBlock, error) {
	// if n is non-nil, it is guaranteed by the grammar to be
	// an *ast.DocBlock. Also, pigeon may return a value
	// and an error.
	n, err := pigeon.Parse("", p.src, opts...)
	doc, _ := n.(*ast.DocBlock)
	if doc != nil && p.optimizeAST {
		ast.Optimize(doc)
	}

	if err == nil && p.tooManyLines {
		err = errors.TooManyLines
	}
	return doc, err
}
