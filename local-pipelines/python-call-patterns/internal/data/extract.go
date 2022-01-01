package data

import (
	"math/rand"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Extract calls from the rast
func Extract(ctx kitectx.Context, rm pythonresource.Manager, maxPerSym int, src []byte, rast *pythonanalyzer.ResolvedAST) []Call {
	byHash := make(map[pythonimports.Hash][]Call)
	addCall := func(call *pythonast.CallExpr) {
		if !valid(call) {
			return
		}

		fss := symbolsForFunc(ctx, rm, rast.References[call.Func])
		if len(fss) == 0 {
			return
		}

		var positional []pythonpatterns.ExprSummary
		keyword := make(map[string]pythonpatterns.ExprSummary)
		for _, arg := range call.Args {
			v := pythonpatterns.ExprSummary{
				Syms:  symbolsForArgValue(ctx, rm, rast.References[arg.Value]),
				Count: 1,
				SrcStrs: pythonpatterns.StrCount{
					string(src[arg.Value.Begin():arg.Value.End()]): 1,
				},
				ASTTypes: pythonpatterns.StrCount{
					pythonpipeline.TypeName(arg.Value): 1,
				},
			}
			name, _ := arg.Name.(*pythonast.NameExpr)
			if name == nil {
				positional = append(positional, v)
			} else {
				keyword[name.Ident.Literal] = v
			}
		}

		ch := pythoncode.CodeHash(src)
		for _, fs := range fss {
			h := fs.Hash()
			byHash[h] = append(byHash[h], Call{
				Func:       fs,
				Positional: positional,
				Keyword:    keyword,
				Hash:       ch,
			})
		}
	}

	pythonast.Inspect(rast.Root, func(n pythonast.Node) bool {
		switch n := n.(type) {
		case *pythonast.BadExpr, *pythonast.BadStmt:
			// only consider calls in syntactically valid regions of code
			return false
		case *pythonast.CallExpr:
			addCall(n)
			return true
		default:
			return true
		}
	})

	var all []Call
	for _, calls := range byHash {
		if maxPerSym == 0 || len(calls) <= maxPerSym {
			all = append(all, calls...)
			continue
		}

		for _, i := range rand.Perm(len(calls))[:maxPerSym] {
			all = append(all, calls[i])
		}
	}

	return all
}

func valid(call *pythonast.CallExpr) bool {
	keywords := make(map[string]bool)
	for _, arg := range call.Args {
		name, _ := arg.Name.(*pythonast.NameExpr)
		if name == nil && len(keywords) > 0 {
			return false
		}
		if name != nil {
			keywords[name.Ident.Literal] = true
		}
	}
	return true
}

func symbolsForFunc(ctx kitectx.Context, rm pythonresource.Manager, v pythontype.Value) []pythonpatterns.Symbol {
	var vs []pythontype.Value
	for _, elem := range pythontype.Disjuncts(ctx, v) {
		// we only support calls on functions or types atm
		if elem.Kind() != pythontype.FunctionKind && elem.Kind() != pythontype.TypeKind {
			continue
		}

		// try mapping to __init__ for types otherwise skip
		if elem.Kind() == pythontype.TypeKind {
			init, _ := pythontype.Attr(ctx, elem, "__init__")
			if !init.ExactlyOne() {
				continue
			}
			elem = init.Value()
		}

		// translate to globals since we cannot serialize specialized
		// values easily atm
		elem = pythontype.Translate(ctx, elem, rm)
		if elem != nil {
			vs = append(vs, elem)
		}
	}

	// unite to dedupe
	v = pythontype.Unite(ctx, vs...)

	var syms []pythonpatterns.Symbol
	for _, v := range pythontype.Disjuncts(ctx, v) {
		if s := symbol(v); !s.Nil() {
			syms = append(syms, s)
		}
	}

	return syms

}

func symbolsForArgValue(ctx kitectx.Context, rm pythonresource.Manager, v pythontype.Value) []pythonpatterns.Symbol {
	var vs []pythontype.Value
	for _, elem := range pythontype.Disjuncts(ctx, v) {
		// widen constants since we track the source literal for args
		elem = pythontype.WidenConstants(elem)
		// translate to globals since we cannot serialize specialized
		// values easily atm
		elem = pythontype.Translate(ctx, elem, rm)
		if elem != nil {
			vs = append(vs, elem)
		}
	}

	// unite to dedupe
	v = pythontype.Unite(ctx, vs...)

	var syms []pythonpatterns.Symbol
	for _, v := range pythontype.Disjuncts(ctx, v) {
		if s := symbol(v); !s.Nil() {
			syms = append(syms, s)
		}
	}

	return syms
}

func symbol(val pythontype.Value) pythonpatterns.Symbol {
	switch val := val.(type) {
	case pythontype.External:
		cs := val.Symbol().Canonical()
		return pythonpatterns.Symbol{
			Dist: cs.Dist(),
			Path: cs.Path(),
			Kind: pythonpatterns.External,
		}
	case pythontype.ExternalInstance:
		cs := val.TypeExternal.Symbol().Canonical()
		return pythonpatterns.Symbol{
			Dist: cs.Dist(),
			Path: cs.Path(),
			Kind: pythonpatterns.ExternalInstance,
		}
	case pythontype.ExternalReturnValue:
		cs := val.Func().Canonical()
		return pythonpatterns.Symbol{
			Dist: cs.Dist(),
			Path: cs.Path(),
			Kind: pythonpatterns.ExternalReturnValue,
		}
	default:
		return pythonpatterns.Symbol{}
	}
}
