package main

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

var parseOpts = pythonparser.Options{
	ErrorMode:   pythonparser.Recover,
	Approximate: true,
}

type keywordCountsWithSymbol struct {
	Symbol        pythonresource.Symbol
	KeywordCounts pythoncode.KeywordCounts
}

// Symbols seen with map of keyword arguments to counts
type Symbols map[pythonimports.Hash]*keywordCountsWithSymbol

// SampleTag implements pipeline.Sample
func (Symbols) SampleTag() {}

func (s Symbols) hit(symb pythonresource.Symbol, keyword string) {

	// Always use canonical symbol
	symb = symb.Canonical()
	counts := s[symb.PathHash()]
	if counts == nil {
		counts = &keywordCountsWithSymbol{
			Symbol:        symb,
			KeywordCounts: make(pythoncode.KeywordCounts),
		}
		s[symb.PathHash()] = counts
	}

	counts.KeywordCounts[keyword]++
}

// Hit adds the specified symbol along with the appeared keyword
func (s Symbols) Hit(symbols []pythonresource.Symbol, keyword string) {
	for _, symb := range symbols {
		s.hit(symb, keyword)
	}
}

// Extract keyword arguments appeared in function calls from file
func Extract(rm pythonresource.Manager, ks pipeline.Keyed) Symbols {
	rast := ks.Sample.(pythonpipeline.Resolved).RAST
	symbols := make(Symbols)

	val := func(expr pythonast.Expr) pythontype.Value {
		return rast.References[expr]
	}

	pythonast.Inspect(rast.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		if call, ok := node.(*pythonast.CallExpr); ok {
			syms := toSymbols(rm, val(call.Func))
			for _, arg := range call.Args {
				if name, ok := arg.Name.(*pythonast.NameExpr); ok {
					symbols.Hit(syms, name.Ident.Literal)
				}
			}
		}
		return true
	})

	return symbols
}

// extracts "symbols" that a value references, see README.md.
func toSymbols(rm pythonresource.Manager, val pythontype.Value) []pythonresource.Symbol {
	if val == nil {
		return nil
	}

	var gvs []pythontype.Value
	for _, dv := range pythontype.DisjunctsNoCtx(val) {
		dv = pythontype.WidenConstants(dv)
		if dv := pythontype.TranslateGlobal(dv, rm); dv != nil {
			gvs = append(gvs, dv)
		}
	}

	gvs = pythontype.DisjunctsNoCtx(pythontype.UniteNoCtx(gvs...))

	symbs := make([]pythonresource.Symbol, 0, len(gvs))
	seen := make(map[pythonimports.Hash]bool, len(gvs))
	for _, val := range gvs {
		if val == nil {
			continue
		}

		var symb pythonresource.Symbol
		switch val := val.(type) {
		case pythontype.External:
			symb = val.Symbol()
		case pythontype.ExternalInstance:
			symb = val.TypeExternal.Symbol()
		}

		if symb.Nil() || seen[symb.PathHash()] {
			continue
		}

		seen[symb.PathHash()] = true
		symbs = append(symbs, symb)
	}
	return symbs
}
