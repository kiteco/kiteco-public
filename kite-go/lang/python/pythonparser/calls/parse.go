// Package calls implements a robust python parser for call
// expressions. It parses a subset of the python grammar, as required
// for the most common call expressions as documented in
// Robust Parsing of Python Call Expressions [0].
//
// It supports parsing of partial call expressions, for example
// unclosed parenthesis, missing argument values, unclosed string
// literals and such.
//
// [0]: https://kite.quip.com/vNE3AOhV6mmW/Robust-parsing-of-python-call-expressions
//
package calls

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/calls/internal/pigeon"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/parsing"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

const (
	defaultMaxLines = 3
	defaultMaxExpr  = 1e5
)

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
	return p
}

func (p *parser) parseReturnCallExpr(opts ...pigeon.Option) (*pythonast.CallExpr, error) {
	// if n is non-nil, it is guaranteed by the grammar to be
	// a *pythonast.CallExpr. Also, pigeon may return a value
	// and an error.
	n, err := p.parse(opts...)
	ce, _ := n.(*pythonast.CallExpr)

	// if the input was trimmed because there was too many lines, return
	// errTooManyLines if the call expression is not complete.
	if err == nil && p.tooManyLines && ce.RightParen == nil {
		err = errors.TooManyLines
	}
	return ce, err
}

func (p *parser) parse(opts ...pigeon.Option) (pythonast.Node, error) {
	n, err := pigeon.Parse("", p.src, opts...)
	node, _ := n.(pythonast.Node)
	return node, err
}

func (p *parser) parseCall() (*pythonast.CallExpr, error) {
	return p.parseReturnCallExpr(pigeon.MaxExpressions(p.maxExpressions))
}

func (p *parser) parseArguments() (*pythonast.CallExpr, error) {
	return p.parseReturnCallExpr(pigeon.Entrypoint("GrammarArgsOnly"),
		pigeon.MaxExpressions(p.maxExpressions))
}

func (p *parser) parseStmt() (pythonast.Stmt, error) {
	n, err := p.parse(pigeon.Entrypoint("GrammarStmt"),
		pigeon.MaxExpressions(p.maxExpressions))
	stmt, _ := n.(pythonast.Stmt)
	return stmt, err
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

// MaxLines that the parser will accept before returning a partial node and stopping with an error.
// Default value is 3. A value of 0 means an unlimited number of lines will be parsed.
func MaxLines(cnt uint64) Option {
	return func(p *parser) Option {
		oldCnt := p.maxLines
		p.maxLines = cnt
		return MaxLines(oldCnt)
	}
}

// Parse parses src using the python call expression parser.
// It returns the resulting *pythonast.CallExpr node and
// any error returned by the parser. It may return a partial
// result and an error if a limit was reached during parsing
// (e.g. a maximum number of lines).
func Parse(src []byte, opts ...Option) (*pythonast.CallExpr, error) {
	return newParser(src, opts...).parseCall()
}

// Arguments in a function or method call. It is returned by calls to
// ParseArguments, when only the arguments part of a call is provided.
type Arguments struct {
	Args   []*pythonast.Argument
	Vararg pythonast.Expr
	Kwarg  pythonast.Expr
	Commas []*pythonscanner.Word
}

// ParseArguments using the python call expression parser. It expects
// src to start with an opening parenthesis and tries to parse it
// as the arguments part of a call expression. It returns an *Arguments
// struct and any error returned by the parser. It may return a partial
// result and an error if a limit was reached during parsing
// (e.g. a maximum number of lines).
func ParseArguments(src []byte, opts ...Option) (*Arguments, error) {
	ce, err := newParser(src, opts...).parseArguments()
	if ce == nil {
		return nil, err
	}
	return &Arguments{
		Args:   ce.Args,
		Vararg: ce.Vararg,
		Kwarg:  ce.Kwarg,
		Commas: ce.Commas,
	}, err
}

// ParseStmt parses src using the python statement parser. It returns the
// resulting pythonast.Stmt and any error returned by the parser.
func ParseStmt(src []byte, opts ...Option) (pythonast.Stmt, error) {
	return newParser(src, opts...).parseStmt()
}
