package pythonparser

import (
	"fmt"
	"go/token"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	pyscan "github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	maxRecoverCount = 10
)

// ErrorMode determines how the parser behaves when
// a parser error is encountered.
type ErrorMode int

const (
	// FailFast causes the parser to return on the first error.
	// In this mode the returned AST is guaranteed to be either nil or contain only
	// valid AST nodes, e.g no BadStmt or BadToken nodes.
	FailFast ErrorMode = iota

	// Recover causes the parser to sync to the next valid
	// statement on error and continue parsing.
	// In this mode the returned AST may contain BadStmt nodes.
	Recover

	// ParseTimeout limits how long a parse can take before it is skipped
	ParseTimeout = 60 * time.Millisecond
)

var (
	errMaxRecover = errors.New("max num recoveries")
	errWrongToken = errors.New("unexpected token")
)

// Options repersents configuration for parsing
type Options struct {
	Trace       bool           // Trace determines whether the parse tree is printed to stdout
	MaxDepth    int            // MaxDepth is a threshold on the parse tree depth (only has effect if Trace=true)
	ErrorMode   ErrorMode      // ErrorMode determines what happens when there is a parse error
	TraceWriter io.Writer      // TraceWriter receives tracing output
	ScanOptions pyscan.Options // ScanOptions contains options for the lexer
	Approximate bool           // Approximate switches on regex-based approximation mode for BadStmts and BadExprs
	Cursor      *token.Pos     // Cursor is the position of the cursor in the file.
}

// A Parser processes a token stream into a syntax tree
type parser struct {
	// we violate the standard guideline of not storing ctx in another object to avoid threading this everywhere
	ctx kitectx.Context

	lexer pyscan.Lexer
	word  *pyscan.Word
	opts  Options

	// for error recovery
	recoverCount int
	recoverPos   token.Pos

	// Tracing
	indent int

	cursor     *pyscan.Word
	cursorSeen bool
	prevWord   *pyscan.Word

	errs errors.Errors
}

// newParser constructs a parser that reads tokens from the given lexer
func newParser(ctx kitectx.Context, lexer pyscan.Lexer, opts Options) *parser {
	ctx.CheckAbort()

	if opts.TraceWriter == nil {
		opts.TraceWriter = os.Stdout
	}
	opts.ScanOptions.ScanComments = false
	parser := &parser{
		ctx:   ctx,
		lexer: lexer,
		opts:  opts,
	}
	if opts.Cursor != nil {
		parser.cursor = &pyscan.Word{
			Begin: *opts.Cursor,
			End:   *opts.Cursor,
			Token: pyscan.Cursor,
		}
	}
	parser.next()
	return parser
}

func (p *parser) printTrace(a ...interface{}) {
	p.printTraceSymbol("  ", a...)
}

func (p *parser) printTraceSymbol(symbol string, a ...interface{}) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	fmt.Fprintf(p.opts.TraceWriter, "%s%9d: ", symbol, p.word.Begin)
	i := 2 * p.indent
	for i > len(dots) {
		fmt.Fprint(p.opts.TraceWriter, dots)
		i -= len(dots)
	}
	fmt.Fprint(p.opts.TraceWriter, dots[:i])
	fmt.Fprintln(p.opts.TraceWriter, a...)
}

func trace(p *parser, msg string) *parser {
	p.printTrace(msg, "(")
	p.indent++
	if p.opts.MaxDepth > 0 && p.indent > p.opts.MaxDepth {
		panic("maximum depth exceeded")
	}
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p *parser) {
	p.indent--
	p.printTrace(")")
}

// recoverStmt recovers from an error in parsing a statement
func (p *parser) recoverStmt(begin token.Pos, ex interface{}) *pythonast.BadStmt {
	if ex != nil {
		switch ex {
		case errWrongToken:
			if p.opts.ErrorMode == FailFast {
				panic(ex)
			}
			// sync recover
			p.syncStmt()

			// remove any stmts we may have added since they are all techincally
			// bad e.g foo(); bar(); car(): -> only one bad stmt, no good stmts.
			return &pythonast.BadStmt{
				From: begin,
				To:   p.word.Begin,
			}
		default:
			panic(ex)
		}
	}
	return nil
}

func (p *parser) recoverParse(err *error) {
	if ex := recover(); ex != nil {
		switch ex {
		case errMaxRecover, errWrongToken:
		default:
			panic(ex)
		}
	}
	*err = p.errs
}

// advance moves the lexer forward one token
func (p *parser) advance() {
	p.ctx.CheckAbort()

	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is token.ILLEGAL), so don't print it .
	if p.opts.Trace && p.word != nil {
		s := p.word.Token.String()
		switch {
		case p.word.Token.IsLiteral():
			if len(p.word.Literal) > 50 || strings.Contains(p.word.Literal, "\n") {
				p.printTraceSymbol(" -", s, fmt.Sprintf("<%d chars not shown>", len(p.word.Literal)))
			} else {
				p.printTraceSymbol(" -", s, p.word.Literal)
			}
		case p.word.Token.IsOperator(), p.word.Token.IsKeyword():
			p.printTraceSymbol(" -", "\""+s+"\"")
		default:
			p.printTraceSymbol(" -", s)
		}
	}
	p.prevWord = p.word
	p.word = p.lexer.Next()
	// TODO: deal with lexer errors here
}

// next moves the lexer forward to the next non-comment token
func (p *parser) next() {
	p.advance()
	for p.word.Token == pyscan.Comment || p.word.Token == pyscan.Magic {
		p.advance()
	}
}

// error adds an error to the list
func (p *parser) error(pos token.Pos, msg string) {
	if p.opts.Trace {
		p.printTraceSymbol("**", "ERROR:", msg)
	}
	p.errs = errors.Append(p.errs, pyscan.PosError{Pos: pos, Msg: msg})

	panic(errWrongToken)
}

// errorExpected adds an error of the form "expected <X>" to the list
func (p *parser) errorExpected(pos token.Pos, expected string) {
	p.error(pos, fmt.Sprintf("expected '%s' (got '%s')", expected, p.word.String()))
}

func tokenStrings(toks []pyscan.Token) []string {
	var s []string
	for _, tok := range toks {
		s = append(s, tok.String())
	}
	return s
}

// expect raises an error if the current token is not tok,
// this method always removes a token from the stream or panics.
func (p *parser) expect(tok ...pyscan.Token) *pyscan.Word {
	word := p.word
	if !p.at(tok...) {
		p.error(p.word.Begin, "expected "+strings.Join(tokenStrings(tok), " or "))
	}
	p.next()
	return word
}

func (p *parser) expectOrCursor(toks ...pyscan.Token) *pyscan.Word {
	word := p.word
	if p.at(toks...) {
		p.next()
		return word
	}

	if !p.cursorSeen && p.prevWord != nil && p.cursor != nil && p.prevWord.End == p.cursor.Begin {
		p.cursorSeen = true
		return p.cursor
	}

	p.error(word.Begin, "expected "+strings.Join(tokenStrings(toks), " or ")+" or cursor")
	return nil
}

// at return true if the next token is one of the specified tokens. Does not consume
// any tokens
func (p *parser) at(toks ...pyscan.Token) bool {
	p.ctx.CheckAbort()

	for _, tok := range toks {
		if p.word.Token == tok {
			return true
		}
	}
	return false
}

// consume a token if it matches one of a list, otherwise do not consume anything and return false
func (p *parser) take(toks ...pyscan.Token) *pyscan.Word {
	cur := p.word
	if p.at(toks...) {
		p.next()
		return cur
	}
	return nil
}

// consume a token if it matches one of a list, otherwise do not consume anything and return false
func (p *parser) has(toks ...pyscan.Token) bool {
	return p.take(toks...) != nil
}

// atTest returns true if the next token could be the beginning of a test. Does not consume
// any tokens.
func (p *parser) atTest() bool {
	// An expression can start in one of the following ways:
	// - an identifier
	// - a literal (string or number)
	// - a unary operator: "+", "-", or "~""
	// - a list or dict comprehension: "[" or "{"
	// - a backtick (for an old-style repr)
	// - "lambda"
	// - "not"
	// - a . for an elipsis
	// - "await"
	return p.at(
		pyscan.Ident,
		pyscan.Int,
		pyscan.Long,
		pyscan.Float,
		pyscan.Imag,
		pyscan.String,
		pyscan.Add,
		pyscan.Sub,
		pyscan.BitNot,
		pyscan.Lparen,
		pyscan.Lbrack,
		pyscan.Lbrace,
		pyscan.Backtick,
		pyscan.Not,
		pyscan.Lambda,
		pyscan.Period,
		pyscan.Await)
}

// atSubscript returns true if the next token could be the beginning of a subscript. This is
// the same as atTest except '.' and ':' can also be the start of a subscript.
func (p *parser) atSubscript() bool {
	return p.at(pyscan.Period, pyscan.Colon) || p.atTest()
}

// syncStmt advances to the next statement.
// Used for synchronization after an error.
func (p *parser) syncStmt() {
	if p.opts.Trace {
		defer un(trace(p, "<syncstmt>"))
	}

	// check how many recoveries we have made with no progress
	if p.word.Begin == p.recoverPos {
		if p.recoverCount >= maxRecoverCount {
			panic(errMaxRecover)
		}
		p.recoverCount++
	} else {
		p.recoverCount = 0
		p.recoverPos = p.word.Begin
	}

	for {
		switch p.word.Token {
		case pyscan.Break, pyscan.Continue, pyscan.Return, pyscan.Raise,
			pyscan.Yield, pyscan.While, pyscan.Try, pyscan.With,
			pyscan.Def, pyscan.Class, pyscan.At,
			pyscan.Del, pyscan.Pass, pyscan.Import, pyscan.From,
			pyscan.Global, pyscan.Assert, pyscan.EOF,
			pyscan.Dedent, pyscan.NonLocal, pyscan.Async:
			return
		}
		p.next()
	}
}

// Parse a name expression
func (p *parser) parseName() *pythonast.NameExpr {
	if p.opts.Trace {
		defer un(trace(p, "Name"))
	}

	ident := p.expect(pyscan.Ident)
	return &pythonast.NameExpr{
		Ident: ident,
	}
}

// Parse a dotted name
func (p *parser) parseDottedExpr() *pythonast.DottedExpr {
	if p.opts.Trace {
		defer un(trace(p, "DottedExpr"))
	}

	names := []*pythonast.NameExpr{p.parseName()}
	var dots []*pyscan.Word
	dot := p.take(pyscan.Period)
	for dot != nil {
		names = append(names, p.parseName())
		dots = append(dots, dot)
		dot = p.take(pyscan.Period)
	}

	return &pythonast.DottedExpr{
		Names: names,
		Dots:  dots,
	}
}

// Parse a string literal
func (p *parser) parseStringLiteral() *pythonast.StringExpr {
	if p.opts.Trace {
		defer un(trace(p, "StringLiteral"))
	}

	strs := []*pyscan.Word{}
	for s := p.expect(pyscan.String); s != nil; s = p.take(pyscan.String) {
		strs = append(strs, s)
	}
	return &pythonast.StringExpr{
		Strings: strs,
	}
}

// Parse a number literal
func (p *parser) parseNumberLiteral() *pythonast.NumberExpr {
	if p.opts.Trace {
		defer un(trace(p, "NumberLiteral"))
	}

	w := p.expect(pyscan.Int, pyscan.Long, pyscan.Float, pyscan.Imag)
	return &pythonast.NumberExpr{
		Number: w,
	}
}

// Parse a generator of the form "for a in b if c"
func (p *parser) parseGenerator() *pythonast.Generator {
	if p.opts.Trace {
		defer un(trace(p, "Generator"))
	}

	var async *pyscan.Word
	if p.at(pyscan.Async) {
		async = p.expect(pyscan.Async)
	}

	forTok := p.expect(pyscan.For)
	vars := p.parseExprList()
	p.expect(pyscan.In)
	val := p.parseOrExpr()

	var filters []pythonast.Expr
	for p.has(pyscan.If) {
		filters = append(filters, p.parseOrExpr())
	}

	return &pythonast.Generator{
		Async:    async,
		For:      forTok,
		Vars:     vars,
		Iterable: val,
		Filters:  filters,
	}
}

// Parse a generator chain of the form "for a in b for x in y"
func (p *parser) parseGeneratorChain() []*pythonast.Generator {
	if p.opts.Trace {
		defer un(trace(p, "GeneratorChain"))
	}

	generators := []*pythonast.Generator{p.parseGenerator()}
	for p.at(pyscan.For, pyscan.Async) {
		generators = append(generators, p.parseGenerator())
	}
	return generators
}

// Parse a "listmaker" node - either a list literal or a list comprehension
//   []
//   [1, 2, 3]
//   [x+1 for x in y if z]
func (p *parser) parseListMaker() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "ListMaker"))
	}

	lbrack := p.expect(pyscan.Lbrack)

	if rbrack := p.take(pyscan.Rbrack); rbrack != nil {
		// Case 0: empty list literal: []
		return &pythonast.ListExpr{
			LeftBrack:  lbrack,
			RightBrack: rbrack,
		}
	}

	value := p.parseTestExpr()
	if p.at(pyscan.For, pyscan.Async) {
		// Case 1: a list comrpehension
		generators := p.parseGeneratorChain()
		rbrack := p.expect(pyscan.Rbrack)
		return &pythonast.ListComprehensionExpr{
			LeftBrack: lbrack,
			BaseComprehension: &pythonast.BaseComprehension{
				Value:      value,
				Generators: generators,
			},
			RightBrack: rbrack,
		}
	}

	// Case 2: a non-empty list literal
	values := []pythonast.Expr{value}
	for p.has(pyscan.Comma) && p.atTest() {
		values = append(values, p.parseTestExpr())
	}
	rbrack := p.expect(pyscan.Rbrack)
	return &pythonast.ListExpr{
		LeftBrack:  lbrack,
		Values:     values,
		RightBrack: rbrack,
	}
}

// Parse a "dictorsetmaker" node - either a dict/set literal or a dict/set comprehension
//   {}
//   {a, b, c}
//   {a for x in y}
//   {foo:bar, ham: spam}
//   {foo:bar for foo in xyz}
func (p *parser) parseDictOrSetMaker() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "DictOrSetMaker"))
	}

	lbrace := p.expect(pyscan.Lbrace)

	if rbrace := p.take(pyscan.Rbrace); rbrace != nil {
		// Case 0: an empty dict comprehension: {}
		return &pythonast.DictExpr{
			LeftBrace:  lbrace,
			RightBrace: rbrace,
		}
	}

	value := p.parseTestExpr()
	if p.at(pyscan.For, pyscan.Async) {
		// Case 1: set comprehension
		generators := p.parseGeneratorChain()
		rbrace := p.expect(pyscan.Rbrace)
		return &pythonast.SetComprehensionExpr{
			LeftBrace: lbrace,
			BaseComprehension: &pythonast.BaseComprehension{
				Value:      value,
				Generators: generators,
			},
			RightBrace: rbrace,
		}
	}

	if p.has(pyscan.Colon) {
		// Case 2+3: either a dict literal or a dict comprehension
		key := value
		value = p.parseTestExpr()
		if p.at(pyscan.For, pyscan.Async) {
			// Case 2: dictionary comprehension
			generators := p.parseGeneratorChain()
			rbrace := p.expect(pyscan.Rbrace)
			return &pythonast.DictComprehensionExpr{
				LeftBrace: lbrace,
				BaseComprehension: &pythonast.BaseComprehension{
					Key:        key,
					Value:      value,
					Generators: generators,
				},
				RightBrace: rbrace,
			}
		}

		// Case 3: dict literal
		items := []*pythonast.KeyValuePair{&pythonast.KeyValuePair{
			Key:   key,
			Value: value,
		}}
		for p.has(pyscan.Comma) && p.atTest() {
			key := p.parseTestExpr()
			p.expect(pyscan.Colon)
			value := p.parseTestExpr()
			items = append(items, &pythonast.KeyValuePair{
				Key:   key,
				Value: value,
			})
		}

		rbrace := p.expect(pyscan.Rbrace)
		return &pythonast.DictExpr{
			LeftBrace:  lbrace,
			Items:      items,
			RightBrace: rbrace,
		}
	}

	// Case 4: set expr
	vals := []pythonast.Expr{value}
	for p.has(pyscan.Comma) && p.atTest() {
		vals = append(vals, p.parseTestExpr())
	}
	rbrace := p.expect(pyscan.Rbrace)
	return &pythonast.SetExpr{
		LeftBrace:  lbrace,
		Values:     vals,
		RightBrace: rbrace,
	}
}

// Parse a "testlist_comp" node; lparen must be non-nil
func (p *parser) parseTestListComprehensionWithLParen(lparen *pyscan.Word) pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "TestListComprehension"))
	}

	if rparen := p.take(pyscan.Rparen); rparen != nil {
		// We have an empty tuple. TODO: deal with begin/end position here
		return &pythonast.TupleExpr{
			LeftParen:  lparen,
			RightParen: rparen,
		}
	}

	value := p.parseTestExpr()
	if p.at(pyscan.For, pyscan.Async) {
		// We have a comprehension like "(x*2 for x in z)"
		generators := p.parseGeneratorChain()
		rparen := p.expect(pyscan.Rparen)
		return &pythonast.ComprehensionExpr{
			LeftParen: lparen,
			BaseComprehension: &pythonast.BaseComprehension{
				Value:      value,
				Generators: generators,
			},
			RightParen: rparen,
		}
	}

	// We have a bracketed subexpression like "(a, b, c)"
	values := []pythonast.Expr{value}
	var commas []*pyscan.Word
	for {
		comma := p.take(pyscan.Comma)
		if comma == nil {
			break
		}
		commas = append(commas, comma)
		if p.atTest() {
			values = append(values, p.parseTestExpr())
		} else {
			break
		}
	}
	rparen := p.expect(pyscan.Rparen)
	if len(commas) == 0 {
		return value
	}
	return &pythonast.TupleExpr{
		LeftParen:  lparen,
		Elts:       values,
		Commas:     commas,
		RightParen: rparen,
	}
}

// Parse a yield expression
func (p *parser) parseYieldExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "YieldExpr"))
	}

	yield := p.expect(pyscan.Yield)
	var value pythonast.Expr
	if p.atTest() {
		value = p.parseTestList()
	}

	return &pythonast.YieldExpr{
		Yield: yield,
		Value: value,
	}
}

// Parse an old-style backticked repr expression, e.g. `foo`
func (p *parser) parseReprExpr() pythonast.Expr {
	ltick := p.expect(pyscan.Backtick)
	expr := p.parseTestExpr()
	rtick := p.expect(pyscan.Backtick)
	return &pythonast.ReprExpr{
		LeftBacktick:  ltick,
		Value:         expr,
		RightBacktick: rtick,
	}
}

// Parse an atom
func (p *parser) parseAtomExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "Atom"))
	}

	switch p.word.Token {
	case pyscan.Lparen:
		lparen := p.word
		p.next()

		if p.at(pyscan.Yield) {
			expr := p.parseYieldExpr()
			p.expect(pyscan.Rparen)
			return expr
		}
		return p.parseTestListComprehensionWithLParen(lparen)
	case pyscan.Lbrack:
		return p.parseListMaker()
	case pyscan.Lbrace:
		return p.parseDictOrSetMaker()
	case pyscan.Backtick:
		return p.parseReprExpr()
	case pyscan.Ident:
		return p.parseName()
	case pyscan.Int, pyscan.Long, pyscan.Float, pyscan.Imag:
		return p.parseNumberLiteral()
	case pyscan.String:
		return p.parseStringLiteral()
	case pyscan.Period:
		dot1 := p.take(pyscan.Period)
		dot2 := p.expect(pyscan.Period)
		dot3 := p.expect(pyscan.Period)
		return &pythonast.EllipsisExpr{
			Periods: [3]*pyscan.Word{dot1, dot2, dot3},
		}
	}

	p.errorExpected(p.word.Begin, "atom")
	// we never reach this code path, since errorExpected panics
	return nil
}

// Parse a call expr given that the function being called has already been consumed, e.g.:
//   ()
//   (x, y+1, z=3, *foo, ham=spam, **bar)
func (p *parser) parseCallExprAfterFunc(fun pythonast.Expr) *pythonast.CallExpr {
	if p.opts.Trace {
		defer un(trace(p, "CallExprAfterFunc"))
	}

	var args []*pythonast.Argument
	var stararg, kwarg pythonast.Expr
	var commas []*pyscan.Word

	lparen := p.expect(pyscan.Lparen)
	if !p.at(pyscan.Rparen) {
		args, stararg, kwarg, commas = p.parseArgumentList()
	}

	rparen := p.expect(pyscan.Rparen)

	return &pythonast.CallExpr{
		Func:       fun,
		LeftParen:  lparen,
		Args:       args,
		Vararg:     stararg,
		Kwarg:      kwarg,
		Commas:     commas,
		RightParen: rparen,
	}
}

// Parse a subscript, e.g.:
//   ...
//   x
//   x:
//   :x
//   x:y
//   :
//   x:y:step
//   ::step
func (p *parser) parseSubscript() pythonast.Subscript {
	if p.opts.Trace {
		defer un(trace(p, "Subscript"))
	}

	if dot1 := p.take(pyscan.Period); dot1 != nil {
		// Parse an ellipsis
		dot2 := p.expect(pyscan.Period)
		dot3 := p.expect(pyscan.Period)
		return &pythonast.EllipsisExpr{
			Periods: [3]*pyscan.Word{dot1, dot2, dot3},
		}
	}

	var lower, upper, step pythonast.Expr
	if !p.at(pyscan.Colon) {
		lower = p.parseTestExpr()

		// If still not at a colon then we have just a single index, no colons
		if !p.at(pyscan.Colon) {
			return &pythonast.IndexSubscript{
				Value: lower,
			}
		}
	}

	// Now we must be at a colon
	firstColon := p.expect(pyscan.Colon)

	if p.atTest() {
		upper = p.parseTestExpr()
	}
	secondColon := p.take(pyscan.Colon)
	if secondColon != nil && p.atTest() {
		step = p.parseTestExpr()
	}

	return &pythonast.SliceSubscript{
		Lower:       lower,
		FirstColon:  firstColon,
		Upper:       upper,
		SecondColon: secondColon,
		Step:        step,
	}
}

// Parse a list of subscripts
func (p *parser) parseSubscriptList() []pythonast.Subscript {
	if p.opts.Trace {
		defer un(trace(p, "SubscriptList"))
	}

	subs := []pythonast.Subscript{p.parseSubscript()}
	for p.has(pyscan.Comma) && p.atSubscript() {
		subs = append(subs, p.parseSubscript())
	}

	return subs
}

// Parse an index expr given that the object being indexed has already been consume
func (p *parser) parseIndexExprAfterValue(value pythonast.Expr) *pythonast.IndexExpr {
	if p.opts.Trace {
		defer un(trace(p, "IndexExprAfterValue"))
	}

	lbrack := p.expect(pyscan.Lbrack)
	subscripts := p.parseSubscriptList()
	rbrack := p.expect(pyscan.Rbrack)

	return &pythonast.IndexExpr{
		LeftBrack:  lbrack,
		Value:      value,
		Subscripts: subscripts,
		RightBrack: rbrack,
	}
}

// Parse an attribute expr given that the object being indexed has already been consume
func (p *parser) parseAttributeExprAfterValue(value pythonast.Expr) *pythonast.AttributeExpr {
	if p.opts.Trace {
		defer un(trace(p, "AttributeExprAfterValue"))
	}

	dot := p.expect(pyscan.Period)

	attr := p.expectOrCursor(pyscan.Ident)

	return &pythonast.AttributeExpr{
		Value:     value,
		Dot:       dot,
		Attribute: attr,
	}
}

// Parse a power expression
func (p *parser) parsePowerExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "PowerExpr"))
	}

	// atom expr may start with 'await' - if this is the case,
	// wrap the full atom expr in an AwaitExpr.
	var await *pyscan.Word
	if p.at(pyscan.Await) {
		await = p.expect(pyscan.Await)
	}

	left := p.parseAtomExpr()
	for p.at(pyscan.Lparen, pyscan.Lbrack, pyscan.Period) {
		switch p.word.Token {
		case pyscan.Lparen:
			left = p.parseCallExprAfterFunc(left)
		case pyscan.Lbrack:
			left = p.parseIndexExprAfterValue(left)
		case pyscan.Period:
			left = p.parseAttributeExprAfterValue(left)
		}
	}

	if await != nil {
		left = &pythonast.AwaitExpr{
			Await: await,
			Value: left,
		}
	}

	if op := p.take(pyscan.Pow); op != nil {
		right := p.parseFactorExpr()
		return &pythonast.BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
		}
	}

	return left
}

// Parse a factor
func (p *parser) parseFactorExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "FactorExpr"))
	}

	if op := p.take(pyscan.Add, pyscan.Sub, pyscan.BitNot); op != nil {
		expr := p.parseFactorExpr()
		return &pythonast.UnaryExpr{
			Op:    op,
			Value: expr,
		}
	}

	return p.parsePowerExpr()
}

type exprParser func() pythonast.Expr

// Parse a left-associative binary expression
func (p *parser) parseBinaryExpr(parseLHS, parseRHS exprParser, toks ...pyscan.Token) pythonast.Expr {
	left := parseLHS()
	if op := p.take(toks...); op != nil {
		right := parseRHS()
		return &pythonast.BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
		}
	}

	return left
}

// Parse a term
func (p *parser) parseTermExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "TermExpr"))
	}
	return p.parseBinaryExpr(p.parseFactorExpr, p.parseTermExpr, pyscan.Mul, pyscan.Div, pyscan.Pct, pyscan.Truediv)
}

// Parse an expression
func (p *parser) parseArithmeticExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "ArithmeticExpr"))
	}
	return p.parseBinaryExpr(p.parseTermExpr, p.parseArithmeticExpr, pyscan.Add, pyscan.Sub)
}

// Parse an "or_test" expression
func (p *parser) parseShiftExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "ShiftExpr"))
	}
	return p.parseBinaryExpr(p.parseArithmeticExpr, p.parseShiftExpr, pyscan.BitLshift, pyscan.BitRshift)
}

// Parse an "or_test" expression
func (p *parser) parseBitAndExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "BitAndExpr"))
	}
	return p.parseBinaryExpr(p.parseShiftExpr, p.parseBitAndExpr, pyscan.BitAnd)
}

// Parse an "or_test" expression
func (p *parser) parseBitXorExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "BitXorExpr"))
	}
	return p.parseBinaryExpr(p.parseBitAndExpr, p.parseBitXorExpr, pyscan.BitXor)
}

// Parse an "expr" node in the python grammar. Confusingly, this is not actually a fully general
// expression but is instead one of:
//   "a OP b"       where OP can be + - * / % | ^ & but NOT "and" "or"
//   "(EXPR)"       where p is a fully general "test" expression
//   "foo(...)"
func (p *parser) parseExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "Expr"))
	}
	return p.parseBinaryExpr(p.parseBitXorExpr, p.parseExpr, pyscan.BitOr)
}

// Parse a comparison operator, or return nil without consuming anything if there is not one here
func (p *parser) tryComparisonOp() *pyscan.Word {
	if op := p.take(pyscan.Not); op != nil {
		if p.has(pyscan.In) {
			// TODO: set op for "not in"
		}
		return op
	} else if op := p.take(pyscan.Is); op != nil {
		if p.has(pyscan.Not) {
			// TODO: set op for "is not"
		}
		return op
	} else if op := p.take(pyscan.Lt,
		pyscan.Gt,
		pyscan.Eq,
		pyscan.Ge,
		pyscan.Le,
		pyscan.Lg,
		pyscan.Ne,
		pyscan.In,
		pyscan.Not,
		pyscan.Is); op != nil {
		return op
	}
	return nil
}

// Parse a "comparison" node
func (p *parser) parseComparison() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "Comparison"))
	}

	left := p.parseExpr()
	if op := p.tryComparisonOp(); op != nil {
		right := p.parseComparison()
		return &pythonast.BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
		}
	}

	return left
}

// Parse a "not_test" node
func (p *parser) parseNotExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "NotExpr"))
	}

	if op := p.take(pyscan.Not); op != nil {
		expr := p.parseNotExpr()
		return &pythonast.UnaryExpr{
			Op:    op,
			Value: expr,
		}
	}

	return p.parseComparison()
}

// Parse an "and_test" node
func (p *parser) parseAndExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "AndExpr"))
	}
	return p.parseBinaryExpr(p.parseNotExpr, p.parseAndExpr, pyscan.And)
}

// Parse an "or_test" node
func (p *parser) parseOrExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "OrExpr"))
	}
	return p.parseBinaryExpr(p.parseAndExpr, p.parseOrExpr, pyscan.Or)
}

// Parse a lambda expression
func (p *parser) parseLambdaExpr() *pythonast.LambdaExpr {
	if p.opts.Trace {
		defer un(trace(p, "LambdaExpr"))
	}

	lambda := p.expect(pyscan.Lambda)
	params, vararg, kwarg := p.parseParameterList(false)
	p.expect(pyscan.Colon)
	body := p.parseTestExpr()

	return &pythonast.LambdaExpr{
		Lambda:     lambda,
		Parameters: params,
		Vararg:     vararg,
		Kwarg:      kwarg,
		Body:       body,
	}
}

// Parse an "x if y else z" expression given that the "x" part has already been parsed
func (p *parser) parseIfExprAfterBody(body pythonast.Expr) pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "IfExprAfterBody"))
	}

	p.expect(pyscan.If)
	condition := p.parseTestExpr()
	p.expect(pyscan.Else)
	els := p.parseTestExpr()

	return &pythonast.IfExpr{
		Body:      body,
		Condition: condition,
		Else:      els,
	}
}

// Parse a general expression. This is different to "parseExpr" because this function can
// also accomodate expressions like:
//  - lambda foo: bar
//  - a if b else c
//  - ham [and|or] spam
//  - not foo
func (p *parser) parseTestExpr() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "TestExpr"))
	}

	if p.at(pyscan.Lambda) {
		return p.parseLambdaExpr()
	}

	expr := p.parseOrExpr()
	if p.at(pyscan.If) {
		return p.parseIfExprAfterBody(expr)
	}

	return expr
}

// Parse a list of expressions
func (p *parser) parseExprList() []pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "ExprList"))
	}

	exprs := []pythonast.Expr{p.parseExpr()}
	for p.has(pyscan.Comma) && p.atTest() {
		exprs = append(exprs, p.parseExpr())
	}

	return exprs
}

// Parse a comma separated list of high-level expressions, and check for a trailing comma
func (p *parser) parseTestList() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "TestList"))
	}

	exprs := []pythonast.Expr{p.parseTestExpr()}
	var commas []*pyscan.Word
	for {
		comma := p.take(pyscan.Comma)
		if comma == nil {
			break
		}
		commas = append(commas, comma)
		if p.atTest() {
			exprs = append(exprs, p.parseTestExpr())
		} else {
			break
		}
	}

	if len(commas) > 0 {
		return &pythonast.TupleExpr{
			Elts:   exprs,
			Commas: commas,
		}
	}
	// assert len(exprs) == 1
	return exprs[0]
}

// Parse "print" statement
func (p *parser) parsePrintStmt() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "PrintStmt"))
	}

	// the print keyword is scanned as an identifier
	printFunc := p.expect(pyscan.Ident)
	if printFunc != nil && printFunc.Literal != "print" {
		p.error(printFunc.Begin, "print statement started with an identifier other than 'print'")
	}

	if p.at(pyscan.Lparen) {
		// Parse it as a python3-style function call
		expr := p.parseCallExprAfterFunc(&pythonast.NameExpr{Ident: printFunc})

		// Ignore everything until the end of the line (TODO: do properly with backtracking)
		for !p.at(pyscan.NewLine, pyscan.EOF) {
			p.next()
		}

		return &pythonast.ExprStmt{
			Value: expr,
		}
	}

	var dest pythonast.Expr
	if p.has(pyscan.BitRshift) {
		dest = p.parseTestExpr()
		p.has(pyscan.Comma)
	}
	newline := true
	var values []pythonast.Expr
	for p.atTest() {
		values = append(values, p.parseTestExpr())
		if p.has(pyscan.Comma) {
			newline = false
		} else {
			newline = true
			break
		}
	}

	return &pythonast.PrintStmt{
		Print:   printFunc,
		Dest:    dest,
		Values:  values,
		NewLine: newline,
	}
}

// Parse "del" statement
func (p *parser) parseDelStmt() *pythonast.DelStmt {
	if p.opts.Trace {
		defer un(trace(p, "DelStmt"))
	}

	del := p.expect(pyscan.Del)
	targets := p.parseExprList()
	return &pythonast.DelStmt{
		Del:     del,
		Targets: targets,
	}
}

// Parse "pass" statement
func (p *parser) parsePassStmt() *pythonast.PassStmt {
	if p.opts.Trace {
		defer un(trace(p, "PassStmt"))
	}

	pass := p.expect(pyscan.Pass)
	return &pythonast.PassStmt{
		Pass: pass,
	}
}

// Parse "foo as spam"
func (p *parser) parseImportAsName() *pythonast.ImportAsName {
	external := p.parseName()
	var internal *pythonast.NameExpr
	if p.has(pyscan.As) {
		internal = p.parseName()
	}
	return &pythonast.ImportAsName{
		External: external,
		Internal: internal,
	}
}

// Parse "foo.bar.baz as spam"
func (p *parser) parseDottedAsName() *pythonast.DottedAsName {
	external := p.parseDottedExpr()
	var internal *pythonast.NameExpr
	if p.has(pyscan.As) {
		internal = p.parseName()
	}
	return &pythonast.DottedAsName{
		External: external,
		Internal: internal,
	}
}

// Parse "import bar" statement
func (p *parser) parseImportNameStmt() *pythonast.ImportNameStmt {
	if p.opts.Trace {
		defer un(trace(p, "ImportNameStmt"))
	}

	importTok := p.expect(pyscan.Import)

	// We must have at least one name
	names := []*pythonast.DottedAsName{p.parseDottedAsName()}
	var commas []*pyscan.Word
	for comma := p.take(pyscan.Comma); comma != nil; comma = p.take(pyscan.Comma) {
		names = append(names, p.parseDottedAsName())
		commas = append(commas, comma)
	}

	return &pythonast.ImportNameStmt{
		Import: importTok,
		Names:  names,
		Commas: commas,
	}
}

// Parse "from foo import bar" statement
func (p *parser) parseImportFromStmt() *pythonast.ImportFromStmt {
	if p.opts.Trace {
		defer un(trace(p, "ImportFromStmt"))
	}

	from := p.expect(pyscan.From)

	var dots []*pyscan.Word
	for dot := p.take(pyscan.Period); dot != nil; dot = p.take(pyscan.Period) {
		dots = append(dots, dot)
	}

	var pkg *pythonast.DottedExpr
	if len(dots) == 0 || p.at(pyscan.Ident) {
		pkg = p.parseDottedExpr()
	}
	imp := p.expect(pyscan.Import)

	var names []*pythonast.ImportAsName
	var commas []*pyscan.Word

	wildcard := p.take(pyscan.Mul)
	var lparen, rparen *pyscan.Word
	if wildcard == nil {
		lparen = p.take(pyscan.Lparen) // for python3

		// We must have at least one name
		names = append(names, p.parseImportAsName())
		for comma := p.take(pyscan.Comma); comma != nil; comma = p.take(pyscan.Comma) {
			if !p.at(pyscan.Ident) {
				if lparen == nil {
					p.errorExpected(comma.Begin, "trailing comma not allowed without parentheses in ImportFromStmt")
				} else {
					break
				}
			}

			names = append(names, p.parseImportAsName())
			commas = append(commas, comma)
		}

		// If there was an opening paren then there must be a closing paren
		if lparen != nil {
			rparen = p.expect(pyscan.Rparen) // for python3
		}
	}

	return &pythonast.ImportFromStmt{
		From:       from,
		Package:    pkg,
		Dots:       dots,
		Import:     imp,
		LeftParen:  lparen,
		Names:      names,
		Commas:     commas,
		Wildcard:   wildcard,
		RightParen: rparen,
	}
}

// Parse "import" statement
func (p *parser) parseImportStmt() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "ImportStmt"))
	}

	if p.at(pyscan.From) {
		return p.parseImportFromStmt()
	}
	return p.parseImportNameStmt()
}

// Parse "global" statement
func (p *parser) parseGlobalStmt() *pythonast.GlobalStmt {
	if p.opts.Trace {
		defer un(trace(p, "GlobalStmt"))
	}

	global := p.expect(pyscan.Global)
	names := []*pythonast.NameExpr{p.parseName()}
	for p.has(pyscan.Comma) {
		names = append(names, p.parseName())
	}
	return &pythonast.GlobalStmt{
		Global: global,
		Names:  names,
	}
}

// Parse "nonlocal" statement
func (p *parser) parseNonLocalStmt() *pythonast.NonLocalStmt {
	if p.opts.Trace {
		defer un(trace(p, "NonLocalStmt"))
	}

	nonlocal := p.expect(pyscan.NonLocal)
	names := []*pythonast.NameExpr{p.parseName()}
	for p.has(pyscan.Comma) {
		names = append(names, p.parseName())
	}
	return &pythonast.NonLocalStmt{
		NonLocal: nonlocal,
		Names:    names,
	}
}

// Parse "exec" statement/function
func (p *parser) parseExecStmt() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "ExecStmt"))
	}

	execFunc := p.expect(pyscan.Ident)
	if execFunc != nil && execFunc.Literal != "exec" {
		p.error(execFunc.Begin, "exec statement started with an identifier other than 'exec'")
	}

	if p.at(pyscan.Lparen) {
		// Parse it as a python3-style function call
		expr := p.parseCallExprAfterFunc(&pythonast.NameExpr{Ident: execFunc})

		// Ignore everything until the end of the line (TODO: do properly with backtracking)
		for !p.at(pyscan.NewLine, pyscan.EOF) {
			p.next()
		}

		return &pythonast.ExprStmt{
			Value: expr,
		}
	}

	body := p.parseExpr()
	var locals, globals pythonast.Expr
	if p.has(pyscan.In) {
		locals = p.parseTestExpr()
		if p.has(pyscan.Comma) {
			globals = p.parseTestExpr()
		}
	}
	return &pythonast.ExecStmt{
		Exec:    execFunc,
		Body:    body,
		Locals:  locals,
		Globals: globals,
	}
}

// Parse an "assert" statement
func (p *parser) parseAssertStmt() *pythonast.AssertStmt {
	if p.opts.Trace {
		defer un(trace(p, "AssertStmt"))
	}

	assert := p.expect(pyscan.Assert)
	condition := p.parseTestExpr()
	var msg pythonast.Expr
	if p.has(pyscan.Comma) {
		msg = p.parseTestExpr()
	}
	return &pythonast.AssertStmt{
		Assert:    assert,
		Condition: condition,
		Message:   msg,
	}
}

// Parse "break" statement
func (p *parser) parseBreakStmt() *pythonast.BreakStmt {
	if p.opts.Trace {
		defer un(trace(p, "BreakStmt"))
	}

	breakTok := p.expect(pyscan.Break)
	return &pythonast.BreakStmt{
		Break: breakTok,
	}
}

// Parse continue statement
func (p *parser) parseContinueStmt() *pythonast.ContinueStmt {
	if p.opts.Trace {
		defer un(trace(p, "ContinueStmt"))
	}

	continueTok := p.expect(pyscan.Continue)
	return &pythonast.ContinueStmt{
		Continue: continueTok,
	}
}

// Parse a "return" statement
func (p *parser) parseReturnStmt() *pythonast.ReturnStmt {
	if p.opts.Trace {
		defer un(trace(p, "ReturnStmt"))
	}

	returnTok := p.expect(pyscan.Return)

	// The return value is optional
	var value pythonast.Expr
	if p.atTest() {
		value = p.parseTestList()
	}

	return &pythonast.ReturnStmt{
		Return: returnTok,
		Value:  value,
	}
}

// Parse "raise" statement
func (p *parser) parseRaiseStmt() *pythonast.RaiseStmt {
	if p.opts.Trace {
		defer un(trace(p, "RaiseStmt"))
	}

	raise := p.expect(pyscan.Raise)
	var typ, instance, traceback pythonast.Expr
	if p.atTest() {
		typ = p.parseTestExpr()
		if p.has(pyscan.Comma) {
			instance = p.parseTestExpr()
			if p.has(pyscan.Comma) {
				traceback = p.parseTestExpr()
			}
		} else {
			// If there is only one expression then it is the instance
			instance = typ
			typ = nil
		}
	}

	return &pythonast.RaiseStmt{
		Raise:     raise,
		Type:      typ,
		Instance:  instance,
		Traceback: traceback,
	}
}

// Parse "yield" statement
func (p *parser) parseYieldStmt() *pythonast.YieldStmt {
	if p.opts.Trace {
		defer un(trace(p, "YieldStmt"))
	}

	yield := p.expect(pyscan.Yield)
	var value pythonast.Expr
	if p.atTest() {
		value = p.parseTestList()
	}

	return &pythonast.YieldStmt{
		Yield: yield,
		Value: value,
	}
}

// Parse an indented block
func (p *parser) parseSuite() []pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "Suite"))
	}

	if p.has(pyscan.NewLine) {
		p.expect(pyscan.Indent)
		stmts := p.parseStmt()
		// Need to check that not at EOF incase we had to sync and remove rest of token stream.
		for !p.has(pyscan.Dedent) && !p.at(pyscan.EOF) {
			stmts = append(stmts, p.parseStmt()...)
		}
		return stmts
	}
	return p.parseSimpleStmt()
}

// Parse a single branch of an "if" statement
func (p *parser) parseBranch() *pythonast.Branch {
	if p.opts.Trace {
		defer un(trace(p, "Branch"))
	}

	condition := p.parseTestExpr()
	p.expect(pyscan.Colon)
	body := p.parseSuite()
	return &pythonast.Branch{
		Condition: condition,
		Body:      body,
	}
}

// Parse "if" statement
func (p *parser) parseIfStmt() *pythonast.IfStmt {
	if p.opts.Trace {
		defer un(trace(p, "IfStmt"))
	}

	// Parse the "if" branch
	ifTok := p.expect(pyscan.If)
	branches := []*pythonast.Branch{p.parseBranch()}

	// Parse the "elif" branches, if any
	for p.has(pyscan.Elif) {
		branches = append(branches, p.parseBranch())
	}

	// Parse the "else" branch, if present
	var elseBody []pythonast.Stmt
	if p.has(pyscan.Else) {
		p.expect(pyscan.Colon)
		elseBody = p.parseSuite()
	}

	return &pythonast.IfStmt{
		If:       ifTok,
		Branches: branches,
		Else:     elseBody,
	}
}

// Parse "while" statement
func (p *parser) parseWhileStmt() *pythonast.WhileStmt {
	if p.opts.Trace {
		defer un(trace(p, "WhileStmt"))
	}

	// Parse the "while" body
	while := p.expect(pyscan.While)
	cond := p.parseTestExpr()
	p.expect(pyscan.Colon)
	body := p.parseSuite()

	// Parse the "else" part, if present
	var elseBody []pythonast.Stmt
	if p.has(pyscan.Else) {
		p.expect(pyscan.Colon)
		elseBody = p.parseSuite()
	}

	return &pythonast.WhileStmt{
		While:     while,
		Condition: cond,
		Body:      body,
		Else:      elseBody,
	}
}

// Parse "for" statement
func (p *parser) parseForStmt() *pythonast.ForStmt {
	if p.opts.Trace {
		defer un(trace(p, "ForStmt"))
	}

	// Parse the "for" body
	forTok := p.expect(pyscan.For)
	targets := p.parseExprList()
	p.expect(pyscan.In)
	iterable := p.parseTestList()
	p.expect(pyscan.Colon)
	body := p.parseSuite()

	// Parse the "else" part, if present
	var elseBody []pythonast.Stmt
	if p.has(pyscan.Else) {
		p.expect(pyscan.Colon)
		elseBody = p.parseSuite()
	}

	return &pythonast.ForStmt{
		For:      forTok,
		Targets:  targets,
		Iterable: iterable,
		Body:     body,
		Else:     elseBody,
	}
}

// Parse an "except" clause
func (p *parser) parseExceptClause() *pythonast.ExceptClause {
	if p.opts.Trace {
		defer un(trace(p, "ExceptClause"))
	}

	except := p.expect(pyscan.Except)
	var exceptionType, target pythonast.Expr
	if p.atTest() {
		exceptionType = p.parseTestExpr()
		if p.has(pyscan.As, pyscan.Comma) {
			target = p.parseTestExpr()
		}
	}

	p.expect(pyscan.Colon)
	body := p.parseSuite()

	return &pythonast.ExceptClause{
		Except: except,
		Type:   exceptionType,
		Target: target,
		Body:   body,
	}
}

// Parse "try" statement
func (p *parser) parseTryStmt() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "TryStmt"))
	}

	// Parse "try" block
	try := p.expect(pyscan.Try)
	p.expect(pyscan.Colon)
	body := p.parseSuite()

	// Parse "except" blocks
	var excepts []*pythonast.ExceptClause
	for p.at(pyscan.Except) {
		excepts = append(excepts, p.parseExceptClause())
	}

	// Parse "else" block, if present
	// Note that technically the else block should not be permitted if there are no excepts
	var elseBlock []pythonast.Stmt
	if len(excepts) > 0 && p.has(pyscan.Else) {
		p.expect(pyscan.Colon)
		elseBlock = p.parseSuite()
	}

	// Parse "finally" block, if present
	var finallyBlock []pythonast.Stmt
	if p.has(pyscan.Finally) {
		p.expect(pyscan.Colon)
		finallyBlock = p.parseSuite()
	}

	return &pythonast.TryStmt{
		Try:      try,
		Body:     body,
		Handlers: excepts,
		Else:     elseBlock,
		Finally:  finallyBlock,
	}
}

// Parse a with_item
func (p *parser) parseWithItem() *pythonast.WithItem {
	if p.opts.Trace {
		defer un(trace(p, "WithItem"))
	}

	value := p.parseTestExpr()
	var name pythonast.Expr
	if p.has(pyscan.As) {
		name = p.parseExpr()
	}
	return &pythonast.WithItem{
		Value:  value,
		Target: name,
	}
}

// Parse "with" statement
func (p *parser) parseWithStmt() *pythonast.WithStmt {
	if p.opts.Trace {
		defer un(trace(p, "WithStmt"))
	}

	with := p.expect(pyscan.With)
	items := []*pythonast.WithItem{p.parseWithItem()}
	for p.has(pyscan.Comma) {
		items = append(items, p.parseWithItem())
	}
	p.expect(pyscan.Colon)
	suite := p.parseSuite()
	return &pythonast.WithStmt{
		With:  with,
		Items: items,
		Body:  suite,
	}
}

// Parse an "expr_statement" node from the grammar, which may be:
// - an assignment, e.g.:         x = foo()
// - an "aug" assignment, e.g.:   x += foo()
// - an expression, e.g.:         foo()
func (p *parser) parseExprStmt() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "ExprStmt"))
	}

	lhs := p.parseTestList()
	var annotation pythonast.Expr
	if _, ok := lhs.(*pythonast.TupleExpr); !ok {
		// If we have exactly one assignable expression, we're allowed to have an annotation
		annotation = p.parseAnnotation()
	}

	if p.at(pyscan.Assign) {
		// Case 1: we have an assignment
		// Make sure to deal with "a = b = c = d"
		items := []pythonast.Expr{lhs}
		for p.has(pyscan.Assign) {
			if p.at(pyscan.Yield) {
				items = append(items, p.parseYieldExpr())
			} else {
				items = append(items, p.parseTestList())
			}
		}

		if annotation != nil && len(items) > 2 {
			// We're not allowed `a: foo = b = ...`
			p.error(items[2].Begin(), "annotations not allowed in chained assignments")
		}

		// The value is the last item; the rest are targets
		return &pythonast.AssignStmt{
			Targets:    items[:len(items)-1],
			Annotation: annotation,
			Value:      items[len(items)-1],
		}
	}

	if op := p.take(pyscan.AddAssign,
		pyscan.SubAssign,
		pyscan.MulAssign,
		pyscan.DivAssign,
		pyscan.PctAssign,
		pyscan.BitAndAssign,
		pyscan.BitOrAssign,
		pyscan.BitXorAssign,
		pyscan.BitLshiftAssign,
		pyscan.BitRshiftAssign,
		pyscan.PowAssign,
		pyscan.TruedivAssign); op != nil {
		// Case 2: we have an "aug" assignment
		var rhs pythonast.Expr
		if p.at(pyscan.Yield) {
			rhs = p.parseYieldExpr()
		} else {
			rhs = p.parseTestList()
		}
		return &pythonast.AugAssignStmt{
			Target: lhs, // TODO: check lhs is not TupleExpr
			Op:     op,
			Value:  rhs,
		}
	}

	if annotation != nil {
		// Case 3: we have an uninitialized variable annotation `foo: bar`
		return &pythonast.AnnotationStmt{
			Target:     lhs,
			Annotation: annotation,
		}
	}

	// Case 4: we have an expression
	return &pythonast.ExprStmt{
		Value: lhs,
	}
}

// Parse an argument in a function call (not a function definition), e.g.:
//   foo
//   foo=bar
//   a+1 for a in foo    [which is not a general expression]
func (p *parser) parseArgument() *pythonast.Argument {
	if p.opts.Trace {
		defer un(trace(p, "Argument"))
	}

	var name pythonast.Expr
	var equals *pyscan.Word

	value := p.parseTestExpr()
	if p.at(pyscan.Assign) {
		// in the foo=bar form, the "foo" part (the keyword) must be a NAME,
		// not any expression.
		name = value
		if _, ok := name.(*pythonast.NameExpr); !ok {
			// raising error with "expected comma" because this is not a keyword,
			// if it was raised with "expecting name", the error message doesn't
			// make sense because it will mention "expecting name, got '='" as
			// at this point we are on the "=" sign.
			p.errorExpected(name.Begin(), "comma")
		}
		equals = p.take(pyscan.Assign)
		value = p.parseTestExpr()
	} else if p.at(pyscan.For, pyscan.Async) {
		generators := p.parseGeneratorChain()
		value = &pythonast.ComprehensionExpr{
			BaseComprehension: &pythonast.BaseComprehension{
				Value:      value,
				Generators: generators,
			},
		}
	}

	return &pythonast.Argument{
		Name:   name,
		Equals: equals,
		Value:  value,
	}
}

// Parse an argument list (in a function call not a function definition)
func (p *parser) parseArgumentList() (args []*pythonast.Argument, vararg pythonast.Expr, kwarg pythonast.Expr, commas []*pyscan.Word) {
	if p.opts.Trace {
		defer un(trace(p, "ArgumentsList"))
	}

	// Here we really need a do...while loop
	keepgoing := true
	for keepgoing {
		switch {
		case p.has(pyscan.Pow):
			if kwarg != nil {
				p.error(p.word.Begin, "only one argument can be expanded with a **")
			}
			kwarg = p.parseTestExpr()
		case p.has(pyscan.Mul):
			if vararg != nil {
				p.error(p.word.Begin, "only one argument can be expanded with a *")
			}
			if kwarg != nil {
				p.error(p.word.Begin, "*args cannot appear after **args")
			}
			vararg = p.parseTestExpr()
		default:
			if kwarg != nil {
				p.error(p.word.Begin, "argument cannot appear after **args")
			}
			args = append(args, p.parseArgument())
		}

		comma := p.take(pyscan.Comma)
		if comma != nil {
			commas = append(commas, comma)
		}

		// if we do not have a comma next then we are at the end of the argument list
		keepgoing = comma != nil && (p.at(pyscan.Pow, pyscan.Mul) || p.atTest())
	}
	return
}

// Parse a function parameter, which might be a nested list of params, e.g.:
//    a
//    (x, y, z)
//    (x, (y, z), (ham, spam))
func (p *parser) parseParameter() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "FunctionParam"))
	}

	if lparen := p.take(pyscan.Lparen); lparen != nil {
		elts := []pythonast.Expr{p.parseParameter()}
		var commas []*pyscan.Word
		for {
			comma := p.take(pyscan.Comma)
			if comma == nil {
				break
			}
			commas = append(commas, comma)
			if !p.at(pyscan.Ident, pyscan.Lparen) {
				break
			}
			elts = append(elts, p.parseParameter())
		}

		rparen := p.expect(pyscan.Rparen)
		return &pythonast.TupleExpr{
			LeftParen:  lparen,
			Elts:       elts,
			Commas:     commas,
			RightParen: rparen,
		}
	}

	return p.parseName()
}

// Optionally parse a type annotation (a colon `:` followed by an expression)
//  returning nil otherwise
func (p *parser) parseAnnotation() pythonast.Expr {
	if p.has(pyscan.Colon) {
		return p.parseTestExpr()
	}
	return nil
}

// Parse an argument list, with optional type annotations. This is used in the following places:
//  - in function definitions, where annotations=true
//  - in lambda definitions, where annotations=false
// This is not function calls. Examples:
//   a
//   a,
//   a, b=1, *args, **kwargs
//   a, *b, c, d               [keyword-only args only in python 3]
//   a:list                    [annotations only in python 3]
//   a, b:foo()=0, c,          [annotations only in python 3]
//   a, *, b                   [anonymous varargs only in python 3]
//   <EMPTY>
func (p *parser) parseParameterList(annotations bool) (params []*pythonast.Parameter, vararg, kwarg *pythonast.ArgsParameter) {
	if p.opts.Trace {
		defer un(trace(p, "ParameterList"))
	}

	// Parse as many ordinary args as we have
	var hasVararg bool
	for p.at(pyscan.Ident, pyscan.Lparen, pyscan.Mul) {
		// *args can appear anywhere in the list of parameters as of python 3, but only once
		if p.has(pyscan.Mul) {
			if hasVararg {
				p.error(p.word.Begin, "multiple *args not permitted")
			}
			// the vararg can be anonymous in python 3
			if p.at(pyscan.Ident) {
				name := p.parseName()
				var annotation pythonast.Expr
				if annotations {
					annotation = p.parseAnnotation()
				}
				vararg = &pythonast.ArgsParameter{Name: name, Annotation: annotation}
			}
			hasVararg = true
		} else {
			name := p.parseParameter()
			var annotation pythonast.Expr
			if annotations {
				annotation = p.parseAnnotation()
			}
			var def pythonast.Expr
			if p.has(pyscan.Assign) {
				def = p.parseTestExpr()
			}
			params = append(params, &pythonast.Parameter{
				Name:        name,
				Annotation:  annotation,
				Default:     def,
				KeywordOnly: hasVararg,
			})
		}
		if !p.has(pyscan.Comma) {
			break
		}
	}

	// Parse **kwargs if present (which can only appear as the last argument)
	if p.has(pyscan.Pow) {
		name := p.parseName()
		var annotation pythonast.Expr
		if annotations {
			annotation = p.parseAnnotation()
		}
		kwarg = &pythonast.ArgsParameter{Name: name, Annotation: annotation}
	}

	// Python3 allows a trailing commas following **kwargs, even though **kwargs must be the final parameter
	_ = p.has(pyscan.Comma)

	return
}

// Parse a function definition, e.g. "def foo(a, b=1, *x): return 1"
func (p *parser) parseFunctionDef() *pythonast.FunctionDefStmt {
	if p.opts.Trace {
		defer un(trace(p, "FunctionDef"))
	}

	// Parse the function name
	def := p.expect(pyscan.Def)
	name := p.parseName()

	// Parse the arguments
	lparen := p.expect(pyscan.Lparen)
	params, vararg, kwarg := p.parseParameterList(true)
	rparen := p.expect(pyscan.Rparen)

	// Parse the return annotation if present
	var annotation pythonast.Expr
	if p.has(pyscan.Arrow) {
		annotation = p.parseTestExpr()
	}

	// Parse the body
	p.expect(pyscan.Colon)
	body := p.parseSuite()

	return &pythonast.FunctionDefStmt{
		Def:        def,
		Name:       name,
		LeftParen:  lparen,
		Parameters: params,
		Vararg:     vararg,
		Kwarg:      kwarg,
		Annotation: annotation,
		RightParen: rparen,
		Body:       body,
	}
}

// Parse an async statement, e.g. "async def foo(): pass"
func (p *parser) parseAsyncStmt(funcOnly bool) pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "AsyncStmt"))
	}

	// parse the "async" word
	async := p.expect(pyscan.Async)

	// then parse the associated statement
	switch p.word.Token {
	case pyscan.Def:
		decl := p.parseFunctionDef()
		decl.Async = async
		return decl

	case pyscan.With:
		if funcOnly {
			break
		}
		decl := p.parseWithStmt()
		decl.Async = async
		return decl

	case pyscan.For:
		if funcOnly {
			break
		}
		decl := p.parseForStmt()
		decl.Async = async
		return decl
	}

	if funcOnly {
		p.errorExpected(p.word.Begin, "function def")
	} else {
		p.errorExpected(p.word.Begin, "function def, with or for statement")
	}
	return &pythonast.BadStmt{
		From: p.word.Begin,
		To:   p.word.End,
	}
}

// Parse a class definition, e.g. "class C(base, foo): pass"
func (p *parser) parseClassDef() *pythonast.ClassDefStmt {
	if p.opts.Trace {
		defer un(trace(p, "ClassDef"))
	}

	// Parse the class name
	class := p.expect(pyscan.Class)
	name := p.parseName()

	// Parse the base classes
	var args []*pythonast.Argument
	var vararg, kwarg pythonast.Expr
	if p.has(pyscan.Lparen) {
		if !p.at(pyscan.Rparen) {
			args, vararg, kwarg, _ = p.parseArgumentList()
		}
		p.expect(pyscan.Rparen)
	}

	// Parse the body
	p.expect(pyscan.Colon)
	body := p.parseSuite()

	return &pythonast.ClassDefStmt{
		Class:  class,
		Name:   name,
		Args:   args,
		Vararg: vararg,
		Kwarg:  kwarg,
		Body:   body,
	}
}

// Parse a decorator, e.g. "@foo.bar(1)"
func (p *parser) parseDecorator() pythonast.Expr {
	if p.opts.Trace {
		defer un(trace(p, "Decorator"))
	}

	p.expect(pyscan.At)
	dotted := p.parseDottedExpr()

	// Convert the flat list of names to a nested AttributeExpr
	var dec pythonast.Expr = &pythonast.NameExpr{Ident: dotted.Names[0].Ident}
	for i, part := range dotted.Names[1:] {
		dec = &pythonast.AttributeExpr{
			Value:     dec,
			Dot:       dotted.Dots[i],
			Attribute: part.Ident,
		}
	}

	if p.at(pyscan.Lparen) {
		dec = p.parseCallExprAfterFunc(dec) // will consume Lparen and Rparen
	}
	p.expect(pyscan.NewLine)
	return dec
}

// Parse a decorated statement, e.g.:
//   @foo
//   def bar(): pass
func (p *parser) parseDecoratedStmt() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "DecoratedStmt"))
	}

	begin := p.word.Begin

	decorators := []pythonast.Expr{p.parseDecorator()}
	for p.at(pyscan.At) {
		decorators = append(decorators, p.parseDecorator())
	}
	switch p.word.Token {
	case pyscan.Async:
		// only function definitions are allowed here, in the context of decorators
		decl := p.parseAsyncStmt(true)
		if fd, ok := decl.(*pythonast.FunctionDefStmt); ok {
			fd.Decorators = decorators
		}
		return decl
	case pyscan.Def:
		decl := p.parseFunctionDef()
		decl.Decorators = decorators
		return decl
	case pyscan.Class:
		decl := p.parseClassDef()
		decl.Decorators = decorators
		return decl
	default:
		p.errorExpected(p.word.Begin, "function or class def")
		return &pythonast.BadStmt{
			From: begin,
			// mark to starting at the begining of the current word since it
			// will not be pulled off the stream as part of this method.
			To: p.word.Begin,
		}
	}
}

func (p *parser) parseSmallStmtImpl() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "SmallStmt"))
	}

	// we scan the print & exec keywords as identifiers for python 3 compatibility,
	// since in python 3 they can be used as function names, parameters, variables,
	// etc. But to handle python 2 we check explicitly for statements that start
	// with the these identifier and parse it in a special way.
	// TODO: (hrysoula) how does this change now that we're getting rid of python 2?
	if p.word.Token == pyscan.Ident {
		switch p.word.Literal {
		case "print":
			return p.parsePrintStmt()
		case "exec":
			return p.parseExecStmt()
		}
	}

	switch p.word.Token {
	// small_stmt:
	case pyscan.Del:
		return p.parseDelStmt()
	case pyscan.Pass:
		return p.parsePassStmt()
	case pyscan.Import, pyscan.From:
		return p.parseImportStmt()
	case pyscan.Global:
		return p.parseGlobalStmt()
	case pyscan.Assert:
		return p.parseAssertStmt()
	case pyscan.NonLocal:
		return p.parseNonLocalStmt()

	// flow_stmt:
	case pyscan.Break:
		return p.parseBreakStmt()
	case pyscan.Continue:
		return p.parseContinueStmt()
	case pyscan.Return:
		return p.parseReturnStmt()
	case pyscan.Raise:
		return p.parseRaiseStmt()
	case pyscan.Yield:
		return p.parseYieldStmt()

	// expr_stmt:
	default:
		if !p.atTest() {
			p.errorExpected(p.word.Begin, "statement")
			return &pythonast.BadStmt{
				From: p.word.Begin,
				To:   p.word.End,
				Word: p.word,
			}
		}
		return p.parseExprStmt()
	}
}

// Parse a simple statement, e.g. "print 123"
func (p *parser) parseSmallStmt() pythonast.Stmt {
	// check if at dedent, need this becuase we can sync to a dedent
	if p.has(pyscan.Dedent) {
		p.error(p.word.Begin, "unexpected Dedent parsing small stmt")
	}

	stmt := p.parseSmallStmtImpl()

	return stmt
}

// Parse a simple statement, e.g. "foo(); a += b; print a"
func (p *parser) parseSimpleStmt() (stmts []pythonast.Stmt) {
	if p.opts.Trace {
		defer un(trace(p, "SimpleStmt"))
	}

	// in case we encounter error parsing stmt
	begin := p.word.Begin
	defer func() {
		if bad := p.recoverStmt(begin, recover()); bad != nil {
			stmts = []pythonast.Stmt{bad}
		}
	}()

	stmts = []pythonast.Stmt{p.parseSmallStmt()}
	for p.has(pyscan.Semicolon) && !p.at(pyscan.NewLine, pyscan.EOF) {
		stmts = append(stmts, p.parseSmallStmt())
	}
	// do not "consume" an EOF
	if !p.at(pyscan.EOF) {
		p.expect(pyscan.NewLine)
	}
	return stmts
}

func (p *parser) parseCompoundStmtImpl() (stmt pythonast.Stmt) {
	if p.opts.Trace {
		defer un(trace(p, "CompoundStmt"))
	}

	switch p.word.Token {
	case pyscan.If:
		return p.parseIfStmt()
	case pyscan.While:
		return p.parseWhileStmt()
	case pyscan.For:
		return p.parseForStmt()
	case pyscan.Try:
		return p.parseTryStmt()
	case pyscan.With:
		return p.parseWithStmt()
	case pyscan.Def:
		return p.parseFunctionDef()
	case pyscan.Class:
		return p.parseClassDef()
	case pyscan.At:
		return p.parseDecoratedStmt()
	case pyscan.Async:
		return p.parseAsyncStmt(false)
	}

	p.errorExpected(p.word.Begin, "statement")
	return &pythonast.BadStmt{
		From: p.word.Begin,
		To:   p.word.End,
	}
}

// Parse a compound statement
func (p *parser) parseCompoundStmt() (stmt pythonast.Stmt) {
	// in case we encounter error parsing stmt
	begin := p.word.Begin
	defer func() {
		if bad := p.recoverStmt(begin, recover()); bad != nil {
			stmt = bad
		}
	}()

	// try to parse
	stmt = p.parseCompoundStmtImpl()

	return stmt
}

// Parse a statement
func (p *parser) parseStmt() []pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "Stmt"))
	}

	switch p.word.Token {
	case pyscan.If,
		pyscan.While,
		pyscan.For,
		pyscan.Try,
		pyscan.With,
		pyscan.Def,
		pyscan.Class,
		pyscan.At,
		pyscan.Async:
		return []pythonast.Stmt{p.parseCompoundStmt()}
	default:
		return p.parseSimpleStmt()
	}
}

// Parse a small or compound statement followed immediately by EOF
func (p *parser) parseStmtEOF() pythonast.Stmt {
	if p.opts.Trace {
		defer un(trace(p, "StmtEOF"))
	}

	var stmt pythonast.Stmt
	switch p.word.Token {
	case pyscan.If,
		pyscan.While,
		pyscan.For,
		pyscan.Try,
		pyscan.With,
		pyscan.Def,
		pyscan.Class,
		pyscan.At,
		pyscan.Async:
		stmt = p.parseCompoundStmt()
	default:
		stmt = p.parseSmallStmt()
	}
	p.expect(pyscan.EOF)
	return stmt
}

// Parse a complete source file
func (p *parser) parseModule() *pythonast.Module {
	if p.opts.Trace {
		defer un(trace(p, "Module"))
	}

	var body []pythonast.Stmt

	for !p.at(pyscan.EOF) {
		if p.at(pyscan.NewLine) {
			// Skip empty lines
			p.next()
		} else {
			body = append(body, p.parseStmt()...)
		}
	}

	return &pythonast.Module{
		Body: body,
	}
}

func parse(ctx kitectx.Context, src []byte, words []pyscan.Word, opts Options) (mod *pythonast.Module, err error) {
	ctx.CheckAbort()

	// approximation mode only works when we have comments, newlines, and recovery
	if opts.Approximate {
		// need to scan comment tokens so that we can remove them before applying regexes
		opts.ScanOptions.ScanComments = true
		opts.ScanOptions.ScanNewLines = true
		opts.ErrorMode = Recover
	}

	// create the parser
	lexer := pyscan.NewListLexer(words)
	parser := newParser(ctx, lexer, opts)

	// run the recursive descent parser
	defer parser.recoverParse(&err)

	mod = parser.parseModule()
	if mod == nil {
		return nil, parser.errs
	}

	// run approximations if there were syntax errors and approximation is enabled
	if parser.errs != nil && opts.Approximate {
		ApproximateBadRegions(ctx, mod, src, words)
	}

	// mark evaluate/assign/delete/import usages on lvalue nodes
	MarkUsages(ctx, mod)
	return mod, parser.errs
}

// Parse translates a python source file to a syntax tree.
func Parse(ctx kitectx.Context, src []byte, opts Options) (*pythonast.Module, error) {
	ctx.CheckAbort()

	// check for cached parse
	parseEntry, ok := getCachedParse(src)
	if ok {
		return parseEntry.mod, parseEntry.err
	}

	// run the lexer
	words, _ := pyscan.Lex(src, opts.ScanOptions)

	// parse
	p, err := parse(ctx, src, words, opts)
	cacheParse(src, p, err)
	return p, err
}

// ParseWords translates a python source file to a syntax tree using the provided words
// as the token stream instead of lexing src directly.
func ParseWords(ctx kitectx.Context, src []byte, words []pyscan.Word, opts Options) (*pythonast.Module, error) {
	ctx.CheckAbort()

	return parse(ctx, src, words, opts)
}

// ParseStatement parses a single statement
func ParseStatement(ctx kitectx.Context, src []byte, opts Options) (stmt pythonast.Stmt, err error) {
	ctx.CheckAbort()

	// run the recursive descent parser
	lexer := pyscan.NewStreamLexer(src, opts.ScanOptions)
	parser := newParser(ctx, lexer, opts)

	defer parser.recoverParse(&err)

	stmt = parser.parseStmtEOF()
	if stmt == nil {
		return
	}

	// mark evaluate/assign/delete/import usages on lvalue nodes
	MarkUsages(ctx, stmt)
	return
}
