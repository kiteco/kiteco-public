package mtacconf

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/threshold"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// GetMixData returns the MixData for a completion completing/replacing expr
func GetMixData(ctx kitectx.Context, rm pythonresource.Manager, sel data.Selection, words []pythonscanner.Word, rast *pythonanalyzer.ResolvedAST, expr pythonast.Expr) MixData {
	sym := GetContainingCallSym(ctx, rm, rast, expr)
	if !sym.Sym.Nil() {
		return MixData{Call: sym, Scenario: threshold.InCall}
	}

	for i, w := range words {
		if int(w.End) >= sel.Begin {
			words = words[:i+1]
			break
		}
	}

	switch {
	case inIf(words):
		return MixData{Scenario: threshold.InIf}
	case inWhile(words):
		return MixData{Scenario: threshold.InWhile}
	case inFor(words):
		return MixData{Scenario: threshold.InFor}
	}

	return MixData{Scenario: threshold.Other}
}

// GetContainingCallSym returns the CallSym for the call containing expr
func GetContainingCallSym(ctx kitectx.Context, rm pythonresource.Manager, rast *pythonanalyzer.ResolvedAST, expr pythonast.Expr) CallSym {
	ctx.CheckAbort()

	arg, _ := rast.Parent[expr].(*pythonast.Argument)
	if arg == nil {
		return CallSym{}
	}

	call, _ := rast.Parent[arg].(*pythonast.CallExpr)
	if call == nil {
		return CallSym{}
	}

	val := rast.References[call.Func]
	if val == nil {
		return CallSym{}
	}

	sym, err := pythontype.ChooseExternal(ctx, rm, val)
	if err != nil {
		return CallSym{}
	}

	// find argIdx
	argIdx := -1
	for i, a := range call.Args {
		if a == arg {
			argIdx = i
			break
		}
	}
	if argIdx < 0 {
		rollbar.Error(errors.Errorf("could not find index of argument in parent call expression"))
		return CallSym{}
	}

	// argIdx is checked when generating features that it's not equal to -1
	return CallSym{
		Sym: sym,
		Pos: argIdx,
	}
}

func inIf(words []pythonscanner.Word) bool {
	pos := len(words) - 1
	for pos > 0 {
		switch words[pos].Token {
		case pythonscanner.Ident:
			pos--
		case pythonscanner.If, pythonscanner.Elif:
			return true
		default:
			return false
		}
	}
	return false
}

func inWhile(words []pythonscanner.Word) bool {
	pos := len(words) - 1
	for pos > 0 {
		switch words[pos].Token {
		case pythonscanner.Ident:
			pos--
		case pythonscanner.While:
			return true
		default:
			return false
		}
	}
	return false
}

func inFor(words []pythonscanner.Word) bool {
	pos := len(words) - 1
	var seenIn bool
	for pos > 0 {
		switch words[pos].Token {
		case pythonscanner.Ident:
			pos--
		case pythonscanner.In:
			seenIn = true
			pos--
		case pythonscanner.For:
			return seenIn
		default:
			return false
		}
	}
	return false
}
