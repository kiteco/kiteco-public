package pigeon

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

const (
	parenDepthKey             = "parenDepth"
	initialArgBadExprDepthKey = "initialArgBadExprDepth"
)

var (
	// ErrNoCallExpr is returned by the parser when it can't find any
	// call expression.
	ErrNoCallExpr = errors.New("no call expression")
)

type exprListAndCommas struct {
	exprs  []pythonast.Expr
	commas []*pythonscanner.Word
}

func initState(c *current) error {
	c.state[parenDepthKey] = 0
	c.state[initialArgBadExprDepthKey] = 0
	return nil
}

func grammarArgsOnlyAction(c *current, call *pythonast.CallExpr) (*pythonast.CallExpr, error) {
	// the AST walker panics if Func is nil, so create a NameExpr with an empty identifier
	// (in ParseArguments, the *CallExpr is transferred to an *Arguments struct).
	call.Func = &pythonast.NameExpr{
		Ident: makeWordWithLiteral(c, pythonscanner.Ident, ""),
		Usage: pythonast.Evaluate,
	}
	return call, nil
}

func classDefAction(c *current, class *pythonscanner.Word, id *pythonast.NameExpr, args *pythonast.CallExpr) (*pythonast.ClassDefStmt, error) {
	if args == nil {
		args = &pythonast.CallExpr{}
	}

	// Body cannot be nil, it is required to get the End of the ClassDefStmt. Set it
	// to a single BadStmt node, at the end of this Class definition.
	end := c.pos.offset + len(c.text)
	body := []pythonast.Stmt{
		&pythonast.BadStmt{
			From: token.Pos(end),
			To:   token.Pos(end),
		},
	}

	// TODO(mna): support for Decorators?
	return &pythonast.ClassDefStmt{
		Class:  class,
		Name:   id,
		Args:   args.Args,
		Vararg: args.Vararg,
		Kwarg:  args.Kwarg,
		Body:   body,
	}, nil
}

func functionDefAction(c *current, def *pythonscanner.Word, id *pythonast.NameExpr, callTrailer *pythonast.CallExpr) (*pythonast.FunctionDefStmt, error) {
	if callTrailer == nil {
		callTrailer = &pythonast.CallExpr{}
	}

	// Body cannot be nil, it is required to get the End of the FunctionDefStmt. Set it
	// to a single BadStmt node, at the end of this Function definition.
	end := c.pos.offset + len(c.text)
	funcDef := &pythonast.FunctionDefStmt{
		Def:        def,
		Name:       id,
		LeftParen:  callTrailer.LeftParen,
		RightParen: callTrailer.RightParen,
		Body: []pythonast.Stmt{
			&pythonast.BadStmt{
				From: token.Pos(end),
				To:   token.Pos(end),
			},
		},
	}

	nameOrBadExpr := func(expr pythonast.Expr) pythonast.Expr {
		switch v := expr.(type) {
		case *pythonast.NameExpr:
			return v
		case *pythonast.BadExpr:
			return v
		default:
			return &pythonast.BadExpr{
				From:          v.Begin(),
				To:            v.End(),
				Approximation: []pythonast.Expr{v},
			}
		}
	}

	// convert the call trailer's arguments to parameters
	var params []*pythonast.Parameter
	for _, arg := range callTrailer.Args {
		if arg.Name != nil {
			// name=value form, can be copied over as-is to a Parameter, name is
			// guaranteed to be a NameExpr
			params = append(params, &pythonast.Parameter{
				Name:    arg.Name,
				Default: arg.Value,
			})
			continue
		}

		// otherwise arg.Value must be a NameExpr, if not turn it into a BadExpr
		// unless it is already one
		params = append(params, &pythonast.Parameter{
			Name: nameOrBadExpr(arg.Value),
		})
	}

	if callTrailer.Vararg != nil {
		// function's Vararg can only be a NameExpr, no way to set it to
		// a BadExpr, so add it as a BadExpr to the list of parameters if
		// it is not a NameExpr
		expr := nameOrBadExpr(callTrailer.Vararg)
		if ne, ok := expr.(*pythonast.NameExpr); ok {
			funcDef.Vararg = &pythonast.ArgsParameter{Name: ne}
		} else {
			params = append(params, &pythonast.Parameter{Name: expr})
		}
	}
	if callTrailer.Kwarg != nil {
		// same as for Vararg
		expr := nameOrBadExpr(callTrailer.Kwarg)
		if ne, ok := expr.(*pythonast.NameExpr); ok {
			funcDef.Kwarg = &pythonast.ArgsParameter{Name: ne}
		} else {
			params = append(params, &pythonast.Parameter{Name: expr})
		}
	}
	funcDef.Parameters = params

	return funcDef, nil
}

func ifStmtAction(c *current, ifWord *pythonscanner.Word, cond pythonast.Expr) (*pythonast.IfStmt, error) {
	// IfStmt must have at least one branch and each branch, one body stmt
	if cond == nil {
		// missing condition, set it to BadExpr
		cond = &pythonast.BadExpr{
			From: ifWord.End,
			To:   ifWord.End,
		}
	}

	end := c.pos.offset + len(c.text)
	branch := &pythonast.Branch{
		Condition: cond,
		Body: []pythonast.Stmt{
			&pythonast.BadStmt{
				From: token.Pos(end),
				To:   token.Pos(end),
			},
		},
	}
	return &pythonast.IfStmt{
		If:       ifWord,
		Branches: []*pythonast.Branch{branch},
	}, nil
}

func whileStmtAction(c *current, while *pythonscanner.Word, expr pythonast.Expr) (*pythonast.WhileStmt, error) {
	// Body cannot be nil, it is required to get the End of the WhileStmt. Set it
	// to a single BadStmt node, at the end of this While statement.
	end := c.pos.offset + len(c.text)
	return &pythonast.WhileStmt{
		While:     while,
		Condition: expr,
		Body: []pythonast.Stmt{
			&pythonast.BadStmt{
				From: token.Pos(end),
				To:   token.Pos(end),
			},
		},
	}, nil
}

func withStmtAction(c *current, with *pythonscanner.Word, items []*pythonast.WithItem) (*pythonast.WithStmt, error) {
	// Body cannot be nil, it is required to get the End of the WithStmt. Set it
	// to a single BadStmt node, at the end of this With statement.
	end := c.pos.offset + len(c.text)
	return &pythonast.WithStmt{
		With:  with,
		Items: items,
		Body: []pythonast.Stmt{
			&pythonast.BadStmt{
				From: token.Pos(end),
				To:   token.Pos(end),
			},
		},
	}, nil
}

func maybeWithItemsAction(c *current, first *pythonast.WithItem, rest []interface{}) ([]*pythonast.WithItem, error) {
	items := []*pythonast.WithItem{first}
	for _, next := range rest {
		elems := toIfaceSlice(next)
		// [0]: whitespace, [1]: comma, [2]: whitespace, [3]: item
		items = append(items, elems[3].(*pythonast.WithItem))
	}
	return items, nil
}

func maybeWithItemAction(c *current, value pythonast.Expr, trailer []interface{}) (*pythonast.WithItem, error) {
	var target pythonast.Expr
	if trailer != nil {
		// [0]: whitespace, [1]: "as", [2]: whitespace, [3]: expr
		target = trailer[3].(pythonast.Expr)
	}
	return &pythonast.WithItem{
		Value:  value,
		Target: target,
	}, nil
}

func forStmtAction(c *current, forWord *pythonscanner.Word, targets []*pythonast.NameExpr, in []interface{}) (*pythonast.ForStmt, error) {
	targetExprs := make([]pythonast.Expr, len(targets))
	for i, name := range targets {
		targetExprs[i] = name
	}

	end := c.pos.offset + len(c.text)

	var iterable pythonast.Expr
	if in == nil {
		// Iterable cannot be nil, set it to a BadExpr
		iterable = &pythonast.BadExpr{
			From: token.Pos(end),
			To:   token.Pos(end),
		}
	} else {
		// [0]: whitespace, [1]: "in", [2]: whitespace, [3]: slice of exprs
		// ForStmt only supports a single Expr as Iterable, we store multiple
		// expressions as a paren-less Tuple.
		exprs := in[3].(*exprListAndCommas)
		if len(exprs.commas) == 0 {
			if len(exprs.exprs) > 1 {
				panic("got more than one expression with no commas")
			}
			iterable = exprs.exprs[0]
		} else {
			iterable = &pythonast.TupleExpr{
				Elts:   exprs.exprs,
				Commas: exprs.commas,
				Usage:  pythonast.Evaluate,
			}
		}
	}

	// Body cannot be nil, it is required to get the End of the ForStmt. Set it
	// to a single BadStmt node, at the end of this For statement.
	return &pythonast.ForStmt{
		For:      forWord,
		Targets:  targetExprs,
		Iterable: iterable,
		Body: []pythonast.Stmt{
			&pythonast.BadStmt{
				From: token.Pos(end),
				To:   token.Pos(end),
			},
		},
	}, nil
}

func maybeIDListAction(c *current, first *pythonast.NameExpr, rest []interface{}) ([]*pythonast.NameExpr, error) {
	ids := []*pythonast.NameExpr{first}
	for _, next := range rest {
		elems := toIfaceSlice(next)
		// [0]: whitespace, [1]: comma, [2]: whitespace, [3]: id
		ids = append(ids, elems[3].(*pythonast.NameExpr))
	}
	return ids, nil
}

func exprListAction(c *current, first pythonast.Expr, rest []interface{}, lastComma *pythonscanner.Word) (*exprListAndCommas, error) {
	// pre-allocate all the needed space (+1 for the first expression)
	exprs := make([]pythonast.Expr, 0, len(rest)+1)
	exprs = append(exprs, first)

	// same for commas
	commas := make([]*pythonscanner.Word, 0, len(rest)+1)

	for _, commaExpr := range rest {
		commaExprSlice := toIfaceSlice(commaExpr)
		// [0]: whitespace, [1]: comma, [2]: whitespace, [3]: expr
		commas = append(commas, commaExprSlice[1].(*pythonscanner.Word))
		exprs = append(exprs, commaExprSlice[3].(pythonast.Expr))
	}

	if lastComma != nil {
		commas = append(commas, lastComma)
	}

	return &exprListAndCommas{
		exprs:  exprs,
		commas: commas,
	}, nil
}

func assignStmtAction(c *current, lhs []interface{}, op *pythonscanner.Word, expr pythonast.Expr) (pythonast.Stmt, error) {
	// lhs: [0]: ID, [1]: slice of *AttributeExpr
	target := lhs[0].(pythonast.Expr)
	dots := toIfaceSlice(lhs[1])
	for _, dot := range dots {
		ae := dot.(*pythonast.AttributeExpr)
		ae.Value = target
		target = ae
	}

	if op.Token == pythonscanner.Assign {
		return &pythonast.AssignStmt{
			Targets: []pythonast.Expr{target},
			Value:   expr,
		}, nil
	}
	return &pythonast.AugAssignStmt{
		Target: target,
		Op:     op,
		Value:  expr,
	}, nil
}

func atomExprAction(c *current, stars interface{}, atom pythonast.Expr, trailers []interface{}) (pythonast.Expr, error) {
	val := atom
	for _, expr := range trailers {
		switch expr := expr.(type) {
		case *pythonast.AttributeExpr:
			// From DotTrailer rule
			expr.Value = val
			val = expr
		case *pythonast.CallExpr:
			// From CallTrailer rule
			expr.Func = val
			val = expr
		default:
			panic(fmt.Errorf("unexpected atom trailer type: %T", expr))
		}
	}

	if stars != nil {
		starsAr := toIfaceSlice(stars)
		if len(starsAr) != 2 {
			panic("expected len(starsAr) == 2")
		}
		// wrap val into an UnaryExpr, set the Op to either Mul ("*") or
		// Pow ("**").
		lit := string(starsAr[0].([]byte))
		t := pythonscanner.Mul
		if len(lit) == 2 {
			t = pythonscanner.Pow
		}
		op := makeNonLiteralWord(c, t)
		val = &pythonast.UnaryExpr{
			Op:    op,
			Value: val,
		}
	}

	return val, nil
}

func maybeAtomExprAction(c *current, atom pythonast.Expr) (pythonast.Expr, error) {
	if atom == nil {
		// missing atom, return BadExpr
		return &pythonast.BadExpr{
			From: token.Pos(c.pos.offset),
			To:   token.Pos(c.pos.offset),
		}, nil
	}
	return atom, nil
}

func dotTrailerAction(c *current, dot *pythonscanner.Word, id *pythonast.NameExpr) (*pythonast.AttributeExpr, error) {
	return &pythonast.AttributeExpr{
		Dot:       dot,
		Attribute: id.Ident,
		Usage:     pythonast.Evaluate,
	}, nil
}

func callTrailerAction(c *current, lp *pythonscanner.Word, call *pythonast.CallExpr, rp *pythonscanner.Word) (*pythonast.CallExpr, error) {
	call.LeftParen = lp
	call.RightParen = rp

	// if there is a right parenthesis, and the last argument is an empty
	// one, remove the last argument (e.g. if the input is `fn(a,)`,
	// remove the second, BadExpr argument, but leave it if the input
	// is `fn(a,` - without a closing paren).
	if call.RightParen != nil && len(call.Args) > 0 {
		last := call.Args[len(call.Args)-1]
		if isEmptyArgument(last) {
			call.Args = call.Args[:len(call.Args)-1]
		}
	}

	// Extract the vararg/kwarg from the arguments list to the Vararg and Kwarg fields.
	// This must be done after the removal of the last empty argument if there's a
	// closing paren (see above), otherwise valid var/kwarg would be marked as BadExpr.

	convertToBadExpr := func(ue *pythonast.UnaryExpr, arg *pythonast.Argument) {
		arg.Value = &pythonast.BadExpr{
			From:          ue.Begin(),
			To:            ue.End(),
			Approximation: []pythonast.Expr{ue},
		}
	}

	var starDone bool
	for i := len(call.Args) - 1; i >= 0; i-- {
		arg := call.Args[i]
		ue, ok := arg.Value.(*pythonast.UnaryExpr)
		if !ok || ue.Op == nil || arg.Name != nil || arg.Equals != nil {
			// not a vararg/kwarg, means that there are no more valid such arguments
			starDone = true
			continue
		}

		if ue.Op.Token == pythonscanner.Pow {
			// kwarg, move to CallExpr and remove from args if valid, otherwise convert
			// to BadExpr argument
			if starDone || call.Kwarg != nil {
				// kwarg in invalid position, store as BadExpr
				convertToBadExpr(ue, arg)
				continue
			}
			// NOTE: we unwrap the UnaryExpr here to keep only the UnaryExpr.Value
			// expression, as the "**" part is implied by setting the Kwarg.
			call.Kwarg = ue.Value
			call.Args = call.Args[:len(call.Args)-1]

		} else if ue.Op.Token == pythonscanner.Mul {
			// vararg, move to CallExpr and remove from args if valid, otherwise convert
			// to BadExpr argument
			if starDone || call.Vararg != nil {
				convertToBadExpr(ue, arg)
				continue
			}
			// NOTE: we unwrap the UnaryExpr here to keep only the UnaryExpr.Value
			// expression, as the "*" part is implied by setting the Vararg.
			call.Vararg = ue.Value
			call.Args = call.Args[:len(call.Args)-1]
			starDone = true // after the Vararg, done with valid star arguments

		} else {
			// no more vararg/kwarg to process
			starDone = true
		}
	}

	return call, nil
}

func maybeArgListAction(c *current, first *pythonast.Argument, rest []interface{}) (*pythonast.CallExpr, error) {
	if isEmptyArgument(first) && len(rest) == 0 {
		// no arguments, no commas, return an empy CallExpr
		return &pythonast.CallExpr{}, nil
	}

	var commas []*pythonscanner.Word

	args := []*pythonast.Argument{first}
	for _, commaArg := range rest {
		commaArgAr := toIfaceSlice(commaArg)
		for _, elem := range commaArgAr {
			switch elem := elem.(type) {
			case *pythonscanner.Word:
				commas = append(commas, elem)
			case *pythonast.Argument:
				args = append(args, elem)
			}
		}
	}

	return &pythonast.CallExpr{
		Commas: commas,
		Args:   args,
	}, nil
}

func maybeArgumentKeywordAtomAction(c *current, kw *pythonast.Argument, atom pythonast.Expr) (*pythonast.Argument, error) {
	arg := kw
	if arg == nil {
		// return an argument with only Value set
		arg = &pythonast.Argument{}
	}
	arg.Value = atom

	return arg, nil
}

func rparenPredicate(c *current) (bool, error) {
	n := c.state[parenDepthKey].(int)
	initial := c.state[initialArgBadExprDepthKey].(int)
	return n > initial, nil
}

func enterArgumentBadExprState(c *current) error {
	n := c.state[parenDepthKey].(int)
	c.state[initialArgBadExprDepthKey] = n
	return nil
}

func maybeArgumentBadExprAction(c *current) (*pythonast.Argument, error) {
	// TODO(naman) makeNonLiteralWord?
	w := makeLiteralWord(c, pythonscanner.BadToken)
	return &pythonast.Argument{
		Value: &pythonast.BadExpr{
			From: w.Begin,
			To:   w.End,
			Word: w,
		},
	}, nil
}

func keywordAction(c *current, id *pythonast.NameExpr, eq *pythonscanner.Word) (*pythonast.Argument, error) {
	return &pythonast.Argument{
		Name:   id,
		Equals: eq,
	}, nil
}

func tupleExprAction(c *current, lp *pythonscanner.Word, items *exprListAndCommas, rp *pythonscanner.Word) (pythonast.Expr, error) {
	var exprs []pythonast.Expr
	var commaCount int
	if items != nil {
		exprs = items.exprs
		commaCount = len(items.commas)
	}

	// special-case: if there is only a single expression and no comma,
	// return the single expression, not a tuple. To make a single-value
	// tuple, the value must be followed by a trailing comma.
	// E.g.:
	//   ( "not a tuple" )
	//   ( "totally a tuple", )
	if len(exprs) == 1 && commaCount == 0 {
		return exprs[0], nil
	}

	return &pythonast.TupleExpr{
		LeftParen:  lp,
		Elts:       exprs,
		RightParen: rp,
		Usage:      pythonast.Evaluate,
	}, nil
}

func listExprAction(c *current, lb *pythonscanner.Word, items *exprListAndCommas, rb *pythonscanner.Word) (*pythonast.ListExpr, error) {
	var exprs []pythonast.Expr
	if items != nil {
		exprs = items.exprs
	}
	return &pythonast.ListExpr{
		LeftBrack:  lb,
		Values:     exprs,
		Usage:      pythonast.Evaluate,
		RightBrack: rb,
	}, nil
}

func dictOrSetExprAction(c *current, lb *pythonscanner.Word, items interface{}, rb *pythonscanner.Word) (pythonast.Expr, error) {
	switch items := items.(type) {
	case nil:
		// an empty {} defaults to a dictionary, not a set
		return &pythonast.DictExpr{
			LeftBrace:  lb,
			RightBrace: rb,
		}, nil

	case []*pythonast.KeyValuePair:
		return &pythonast.DictExpr{
			LeftBrace:  lb,
			Items:      items,
			RightBrace: rb,
		}, nil

	case *exprListAndCommas:
		return &pythonast.SetExpr{
			LeftBrace:  lb,
			Values:     items.exprs,
			RightBrace: rb,
		}, nil

	default:
		panic(fmt.Sprintf("unexpected items type in dictOrSetExprAction: %T", items))
	}
}

func dictListAction(c *current, first *pythonast.KeyValuePair, rest []interface{}) ([]*pythonast.KeyValuePair, error) {
	// pre-allocate all the needed space (+1 for the first expression)
	pairs := make([]*pythonast.KeyValuePair, 0, len(rest)+1)
	pairs = append(pairs, first)

	for _, commaPair := range rest {
		commaPairAr := toIfaceSlice(commaPair)
		for _, pair := range commaPairAr {
			switch pair := pair.(type) {
			case *pythonast.KeyValuePair:
				pairs = append(pairs, pair)
			}
		}
	}
	return pairs, nil
}

func dictKeyValAction(c *current, key pythonast.Expr, val pythonast.Expr) (*pythonast.KeyValuePair, error) {
	return &pythonast.KeyValuePair{
		Key:   key,
		Value: val,
	}, nil
}

func isKeywordPredicate(c *current, id []interface{}) (bool, error) {
	var buf bytes.Buffer

	// build the ID string from its parts:
	// [0]: IDStart, [1]: slice of IDContinue
	buf.Write(id[0].([]byte))
	elems := toIfaceSlice(id[1])
	for _, elem := range elems {
		buf.Write(elem.([]byte))
	}

	_, ok := pythonscanner.Keywords[buf.String()]
	return ok, nil
}

func idAction(c *current) (*pythonast.NameExpr, error) {
	return &pythonast.NameExpr{
		Ident: makeLiteralWord(c, pythonscanner.Ident),
		Usage: pythonast.Evaluate,
	}, nil
}

func integerAction(c *current, long interface{}) (*pythonast.NumberExpr, error) {
	if long == nil {
		return &pythonast.NumberExpr{Number: makeLiteralWord(c, pythonscanner.Int)}, nil
	}
	return &pythonast.NumberExpr{Number: makeLiteralWord(c, pythonscanner.Long)}, nil
}

func floatAction(c *current) (*pythonast.NumberExpr, error) {
	w := makeLiteralWord(c, pythonscanner.Float)
	return &pythonast.NumberExpr{Number: w}, nil
}

func imaginaryAction(c *current) (*pythonast.NumberExpr, error) {
	w := makeLiteralWord(c, pythonscanner.Imag)
	return &pythonast.NumberExpr{Number: w}, nil
}

func stringsAction(c *current, first *pythonscanner.Word, rest []interface{}) (*pythonast.StringExpr, error) {
	var strings []*pythonscanner.Word
	strings = append(strings, first)

	for _, spaceString := range rest {
		spaceStringAr := toIfaceSlice(spaceString)
		for _, elem := range spaceStringAr {
			switch elem := elem.(type) {
			case *pythonscanner.Word:
				strings = append(strings, elem)
			}
		}
	}

	se := &pythonast.StringExpr{
		Strings: strings,
	}
	return se, nil
}

func ellipsisAction(c *current, dots []interface{}) (*pythonast.EllipsisExpr, error) {
	if len(dots) != 3 {
		panic("expected len(dots) == 3")
	}
	var ellipsis pythonast.EllipsisExpr
	for i, v := range dots {
		ellipsis.Periods[i] = v.(*pythonscanner.Word)
	}
	return &ellipsis, nil
}

func lparenState(c *current) error {
	n := c.state[parenDepthKey].(int)
	c.state[parenDepthKey] = n + 1
	return nil
}

func rparenState(c *current) error {
	n := c.state[parenDepthKey].(int)
	c.state[parenDepthKey] = n - 1
	return nil
}

// create a *pythonscanner.Word from the current match and the specified token.
func makeLiteralWord(c *current, t pythonscanner.Token) *pythonscanner.Word {
	return makeWordWithLiteral(c, t, string(c.text))
}

func makeWordWithLiteral(c *current, t pythonscanner.Token, lit string) *pythonscanner.Word {
	return &pythonscanner.Word{
		Token:   t,
		Begin:   token.Pos(c.pos.offset),
		End:     token.Pos(c.pos.offset + len(lit)),
		Literal: lit,
	}
}

// create a *pythonscanner.Word with an empty literal using the current match as the word length
func makeNonLiteralWord(c *current, t pythonscanner.Token) *pythonscanner.Word {
	return &pythonscanner.Word{
		Token:   t,
		Begin:   token.Pos(c.pos.offset),
		End:     token.Pos(c.pos.offset + len(c.text)),
		Literal: "",
	}
}

func isEmptyArgument(arg *pythonast.Argument) bool {
	if arg == nil {
		return true
	}
	if arg.Name != nil || arg.Equals != nil {
		return false
	}
	if arg.Value == nil {
		return true
	}
	be, ok := arg.Value.(*pythonast.BadExpr)
	if !ok {
		return false // there's *something* in there
	}
	return be.From == be.To
}

// toIfaceSlice is a helper function for the PEG grammar parser. It converts
// v to a slice of empty interfaces.
func toIfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}
