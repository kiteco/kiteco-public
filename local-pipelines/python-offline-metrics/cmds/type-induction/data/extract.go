package data

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

type candidate struct {
	Return []pythonresource.Symbol
	Func   []pythonresource.Symbol
	Name   *pythonast.NameExpr
	Sym    *pythontype.Symbol
}

type symToNames map[*pythontype.Symbol][]*pythonast.NameExpr

// Extractor  ...
type Extractor struct {
	packages map[string]bool
	rm       pythonresource.Manager
}

// NewExtractor creates an extractor with rm and packages
func NewExtractor(rm pythonresource.Manager, pkgs []string) Extractor {
	pm := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		pm[p] = true
	}
	return Extractor{
		packages: pm,
		rm:       rm,
	}
}

// Extract the samples from ResolvedAST
func (e Extractor) Extract(rast *pythonanalyzer.ResolvedAST) []pipeline.Sample {
	cands, names := e.candidates(rast)

	if len(cands) == 0 {
		return nil
	}

	var samples []pipeline.Sample
	for _, cand := range cands {
		s := e.newSample(cand, names, rast)
		if s.Func.Path.Empty() {
			continue
		}
		samples = append(samples, s)
	}

	return samples
}

func (e Extractor) newSample(cand candidate, names symToNames, rast *pythonanalyzer.ResolvedAST) Sample {
	// check to make sure the function is part of a package in the package list
	fs := e.validFuncSym(cand.Func)
	if fs.Nil() {
		return Sample{}
	}

	var attrs []string
	for _, name := range names[cand.Sym] {
		p := rast.Parent[name]
		attr, ok := p.(*pythonast.AttributeExpr)
		if !ok {
			continue
		}
		if attr.Begin() < cand.Name.End() {
			continue
		}

		attrs = append(attrs, attr.Attribute.Literal)
	}

	if len(attrs) == 0 {
		return Sample{}
	}

	ret := make([]Symbol, 0, len(cand.Return))
	for _, r := range cand.Return {
		ret = append(ret, NewSymbol(r))
	}

	return Sample{
		Pkg:    fs.Path().Head(),
		Func:   NewSymbol(fs),
		Return: ret,
		Attrs:  attrs,
	}
}

func (e Extractor) validFuncSym(fs []pythonresource.Symbol) pythonresource.Symbol {
	for _, f := range fs {
		p := f.Path().Head()
		if e.packages[p] {
			return f
		}
	}
	return pythonresource.Symbol{}
}

func (e Extractor) candidates(rast *pythonanalyzer.ResolvedAST) ([]candidate, symToNames) {
	symbolForName := func(name *pythonast.NameExpr) *pythontype.Symbol {
		table, _ := rast.TableAndScope(name)
		if table == nil {
			return nil
		}
		return table.Find(name.Ident.Literal)
	}

	var cands []candidate
	names := make(symToNames)
	pythonast.Inspect(rast.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		switch node := node.(type) {
		case *pythonast.AssignStmt:
			// check if LHS is a single name expression
			if len(node.Targets) != 1 {
				return true
			}

			name, ok := node.Targets[0].(*pythonast.NameExpr)
			if !ok {
				return true
			}

			sym := symbolForName(name)
			if sym == nil {
				return true
			}

			// check if rhs is a call that resolves to a global func
			call, ok := node.Value.(*pythonast.CallExpr)
			if !ok {
				return true
			}

			syms := e.symbols(rast.References[call.Func])
			if len(syms) == 0 {
				return true
			}

			cands = append(cands, candidate{
				Func:   syms,
				Name:   name,
				Return: e.symbols(rast.References[call]),
				Sym:    sym,
			})
			return true
		case *pythonast.NameExpr:
			sym := symbolForName(node)
			if sym == nil {
				return false
			}
			names[sym] = append(names[sym], node)
			return false
		default:
			return true
		}
	})
	return cands, names
}

func (e Extractor) symbols(val pythontype.Value) []pythonresource.Symbol {
	if val == nil {
		return nil
	}

	var gvs []pythontype.Value
	for _, dv := range pythontype.DisjunctsNoCtx(val) {
		dv = pythontype.WidenConstants(dv)
		if dv := pythontype.TranslateGlobal(dv, e.rm); dv != nil {
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
		case pythontype.ExternalReturnValue:
			continue
		case pythontype.External:
			symb = val.Symbol().Canonical()
		case pythontype.ExternalInstance:
			symb = val.TypeExternal.Symbol().Canonical()
		}

		if symb.Nil() || seen[symb.PathHash()] {
			continue
		}

		seen[symb.PathHash()] = true
		symbs = append(symbs, symb)
	}
	return symbs
}
