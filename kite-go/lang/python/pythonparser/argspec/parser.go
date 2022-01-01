package argspec

import (
	"bytes"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/argspec/internal/pigeon"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/parsing"
)

const (
	defaultMaxLines = 5
	defaultMaxExpr  = 1e5
)

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
// Default value is 5. A value of 0 means an unlimited number of
// lines will be parsed.
func MaxLines(cnt uint64) Option {
	return func(p *parser) Option {
		oldCnt := p.maxLines
		p.maxLines = cnt
		return MaxLines(oldCnt)
	}
}

// Parse parses src using the argspec parser. It returns the
// resulting *ArgSpec node and any error returned by the parser.
// It may return a partial result and an error if a limit was reached
// during parsing (e.g. a maximum number of lines).
func Parse(src []byte, opts ...Option) (*pythonimports.ArgSpec, error) {
	p := newParser(src, opts...)
	return p.parse(pigeon.MaxExpressions(p.maxExpressions))
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
	// (cannot use `EndLine <- EOL / EOF` in a rule as EOF does not
	// consume, so when used in conjunction with `_` which means zero
	// or more whitespace, e.g. `_ EndLine`, it is valid to not make
	// any progress and it loops "forever").
	p.src = append(p.src, '\n')

	return p
}

func (p *parser) parse(opts ...pigeon.Option) (*pythonimports.ArgSpec, error) {
	// if n is non-nil, it is guaranteed by the grammar to be
	// an *ArgSpec.
	n, err := pigeon.Parse("", p.src, opts...)
	spec, _ := n.(*pythonimports.ArgSpec)

	// this parser expectes the argspec to be the first non-blank line
	// in the docstring, so if there is an error with unknown reason
	// (e.g. no match found) and the source input is just whitespace,
	// return TooManyLines instead.
	if errors.ErrorReason(err) == errors.Unknown && len(bytes.TrimSpace(p.src)) == 0 {
		err = errors.TooManyLines
	}
	return spec, err
}
