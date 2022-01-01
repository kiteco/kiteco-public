package main

import (
	"fmt"

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

type countsAndSymbol struct {
	Symbol pythonresource.Symbol
	Counts pythoncode.SymFileCounts
}

// Symbols seen in a file with the specified hash
type Symbols struct {
	Hash string

	// CanonCounts maps from the hash of a canonical symbol to
	// its counts
	CanonCounts map[pythonimports.Hash]*countsAndSymbol

	// NonCanonCounts maps from the hash of a non canonical symbol
	// to its counts
	NonCanonCounts map[pythonimports.Hash]*countsAndSymbol
}

// SampleTag implements pipeline.Sample
func (Symbols) SampleTag() {}

func (s *Symbols) hit(symb pythonresource.Symbol, ctxs ...pythoncode.SymbolContext) {
	counts := s.NonCanonCounts[symb.PathHash()]
	if counts == nil {
		counts = &countsAndSymbol{
			Symbol: symb,
		}
		s.NonCanonCounts[symb.PathHash()] = counts
	}

	cs := symb.Canonical()
	canonCounts := s.CanonCounts[cs.PathHash()]
	if canonCounts == nil {
		canonCounts = &countsAndSymbol{
			Symbol: cs,
		}
		s.CanonCounts[cs.PathHash()] = canonCounts
	}

	for _, ctx := range ctxs {
		switch ctx {
		case pythoncode.SymbolContextAttribute:
			canonCounts.Counts.Attribute++
			counts.Counts.Attribute++
		case pythoncode.SymbolContextCallFunc:
			canonCounts.Counts.CallFunc++
			counts.Counts.CallFunc++
		case pythoncode.SymbolContextExpr:
			canonCounts.Counts.Expr++
			counts.Counts.Expr++
		case pythoncode.SymbolContextImport:
			canonCounts.Counts.Import++
			counts.Counts.Import++
		case pythoncode.SymbolContextName:
			canonCounts.Counts.Name++
			counts.Counts.Name++
		default:
			alias, isAlias := ctx.ImportAlias()
			if !isAlias {
				panic(fmt.Sprintf("unsupported symbol context %v", ctx))
			}
			if alias == "" {
				continue
			}
			if canonCounts.Counts.ImportAliases == nil {
				canonCounts.Counts.ImportAliases = make(map[string]int32)
			}
			canonCounts.Counts.ImportAliases[alias]++
			if counts.Counts.ImportAliases == nil {
				counts.Counts.ImportAliases = make(map[string]int32)
			}
			counts.Counts.ImportAliases[alias]++
		}
	}
}

// Hit adds the specified symbols in the specified contexts
func (s *Symbols) Hit(rm pythonresource.Manager, val pythontype.Value, ctxs ...pythoncode.SymbolContext) {
	if val == nil {
		return
	}

	seen := make(map[pythonimports.Hash]bool)
	for _, val := range pythontype.DisjunctsNoCtx(val) {
		switch v := val.(type) {
		case pythontype.ConstantValue:
			val = pythontype.WidenConstant(v, rm)
		default:
			val = pythontype.TranslateGlobal(v, rm)
		}
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

		s.hit(symb, ctxs...)
	}
}

// Extract modules from file
func Extract(rm pythonresource.Manager, ks pipeline.Keyed) Symbols {
	rast := ks.Sample.(pythonpipeline.Resolved).RAST
	symbols := Symbols{
		Hash:           ks.Key,
		NonCanonCounts: make(map[pythonimports.Hash]*countsAndSymbol, len(rast.References)),
		CanonCounts:    make(map[pythonimports.Hash]*countsAndSymbol, len(rast.References)),
	}

	val := func(expr pythonast.Expr) pythontype.Value {
		return rast.References[expr]
	}

	pythonast.Inspect(rast.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		switch node := node.(type) {
		case *pythonast.ImportNameStmt:
			for _, name := range node.Names {
				var ctxs []pythoncode.SymbolContext
				if name.Internal == nil {
					ctxs = []pythoncode.SymbolContext{pythoncode.SymbolContextImport}
				} else {
					ctxs = []pythoncode.SymbolContext{
						pythoncode.SymbolContextImport,
						pythoncode.SymbolContextImportAlias(name.Internal.Ident.Literal),
					}
				}
				symbols.Hit(rm, val(name.External), ctxs...)
			}
			return false

		case *pythonast.ImportFromStmt:
			if node.Package == nil || len(node.Dots) > 0 {
				return false
			}

			symbols.Hit(rm, val(node.Package), pythoncode.SymbolContextImport)

			for _, name := range node.Names {
				var ctxs []pythoncode.SymbolContext
				if name.Internal == nil {
					ctxs = []pythoncode.SymbolContext{pythoncode.SymbolContextImport}
				} else {
					ctxs = []pythoncode.SymbolContext{
						pythoncode.SymbolContextImport,
						pythoncode.SymbolContextImportAlias(name.Internal.Ident.Literal),
					}
				}
				symbols.Hit(rm, val(name.External), ctxs...)
			}
			return false
		case *pythonast.NameExpr:
			symbols.Hit(rm, val(node),
				pythoncode.SymbolContextName, pythoncode.SymbolContextExpr)
		case *pythonast.CallExpr:
			symbols.Hit(rm, val(node.Func), pythoncode.SymbolContextCallFunc)
			symbols.Hit(rm, val(node), pythoncode.SymbolContextExpr)
		case *pythonast.AttributeExpr:
			symbols.Hit(rm, val(node),
				pythoncode.SymbolContextAttribute, pythoncode.SymbolContextExpr)
		case pythonast.Expr:
			symbols.Hit(rm, val(node), pythoncode.SymbolContextExpr)
		}
		return true
	})

	return symbols
}
