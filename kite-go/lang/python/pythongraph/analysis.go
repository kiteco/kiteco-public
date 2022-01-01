package pythongraph

import (
	"fmt"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	scanOpts = pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}

	parseOpts = pythonparser.Options{
		Approximate: true,
		ErrorMode:   pythonparser.Recover,
	}
)

type analysis struct {
	RAST *pythonanalyzer.ResolvedAST

	Words []pythonscanner.Word

	RM pythonresource.Manager
}

func analyze(ctx kitectx.Context, rm pythonresource.Manager, src []byte) (*analysis, error) {
	words, err := pythonscanner.Lex(src, scanOpts)

	if err != nil {
		return nil, fmt.Errorf("unable to lex file: %v", err)
	}

	ast, err := pythonparser.ParseWords(ctx, src, words, parseOpts)
	if ast == nil {
		return nil, fmt.Errorf("unable to parse ast: %v", err)
	}

	rast, err := pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{
		Path: "/src.py",
	}).Resolve(ast)

	if err != nil {
		return nil, fmt.Errorf("unable to resolve ast: %v", err)
	}

	return newAnalysis(rm, words, rast), nil
}

func newAnalysis(rm pythonresource.Manager, words []pythonscanner.Word, rast *pythonanalyzer.ResolvedAST) *analysis {
	// set all names to be consistent with the appropriate symbol
	// this is to ensure that variables all have the same type
	// TODO: this is pretty hacky....
	// NOTE: this safe because we we get a copy of rast.References
	for expr := range rast.References {
		if name, ok := expr.(*pythonast.NameExpr); ok {
			table, _ := rast.TableAndScope(name)
			if table == nil {
				continue
			}

			sym := table.Find(name.Ident.Literal)
			if sym == nil {
				continue
			}
			rast.References[name] = sym.Value
		}
	}

	return &analysis{
		RAST:  rast,
		Words: words,
		RM:    rm,
	}
}

func (a *analysis) TableForName(name *pythonast.NameExpr) *pythontype.SymbolTable {
	table, _ := a.RAST.TableAndScope(name)

	return table
}

func (a *analysis) Resolve(ctx kitectx.Context, expr pythonast.Expr) pythontype.Value {
	ctx.CheckAbort()

	return a.RAST.References[expr]
}

func (a *analysis) SetResolved(ctx kitectx.Context, expr pythonast.Expr, val pythontype.Value) func() {
	ctx.CheckAbort()

	oldVal, ok := a.RAST.References[expr]

	a.RAST.References[expr] = val

	return func() {
		if !ok {
			delete(a.RAST.References, expr)
		} else {
			a.RAST.References[expr] = oldVal
		}
	}
}

func (a *analysis) ResolveToGlobals(ctx kitectx.Context, expr pythonast.Expr) []pythontype.GlobalValue {
	val := a.Resolve(ctx, expr)
	if val == nil {
		return nil
	}

	return a.translate(ctx, val)
}

func (a *analysis) ResolveToSymbols(ctx kitectx.Context, expr pythonast.Expr) []pythonresource.Symbol {
	ctx.CheckAbort()

	vals := a.ResolveToGlobals(ctx, expr)
	if len(vals) == 0 {
		return nil
	}

	syms := make([]pythonresource.Symbol, 0, len(vals))
	for _, val := range vals {
		// technically should not need this check but just to be safe we add it
		if sym := symbolFor(val); !sym.Nil() {
			syms = append(syms, sym)
		}
	}
	syms = sortSymbolByPopularity(syms, a.RM)
	return syms
}

func (a *analysis) ResolveToPaths(ctx kitectx.Context, expr pythonast.Expr, canonicalize bool) []pythonimports.DottedPath {
	ctx.CheckAbort()

	vals := a.ResolveToGlobals(ctx, expr)
	if len(vals) == 0 {
		return nil
	}

	// need to track paths we have seen since we only use dotted paths and not the full symbol
	seen := make(map[pythonimports.Hash]pythonimports.DottedPath, len(vals))
	for _, val := range vals {
		if sym := symbolFor(val); !sym.Nil() {
			// technically should not need this check but just to be safe we add it
			if canonicalize {
				sym = sym.Canonical()
			}
			seen[sym.PathHash()] = sym.Path()
		}
	}

	paths := make([]pythonimports.DottedPath, 0, len(seen))
	for _, p := range seen {
		paths = append(paths, p)
	}

	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Less(paths[j])
	})

	return paths
}

func (a *analysis) translate(ctx kitectx.Context, val pythontype.Value) []pythontype.GlobalValue {
	ctx.CheckAbort()
	if val == nil {
		return nil
	}

	var gvs []pythontype.GlobalValue

disjuncts:
	for _, dv := range pythontype.Disjuncts(ctx, val) {
		gv := pythontype.TranslateGlobal(pythontype.WidenConstants(dv), a.RM)
		if gv == nil {
			continue
		}

		for _, g := range gvs {
			if pythontype.Equal(ctx, g, gv) {
				continue disjuncts
			}
		}

		gvs = append(gvs, gv)
	}

	return gvs
}

func symbolFor(val pythontype.Value) pythonresource.Symbol {
	switch val := val.(type) {
	case pythontype.External:
		return val.Symbol()
	case pythontype.ExternalInstance:
		return val.TypeExternal.Symbol()
	default:
		return pythonresource.Symbol{}
	}
}
