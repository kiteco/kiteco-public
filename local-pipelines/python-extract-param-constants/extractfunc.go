package main

import (
	"fmt"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

var parseOpts = pythonparser.Options{
	ErrorMode:   pythonparser.Recover,
	Approximate: true,
}

type constType string

const (
	intConst constType = "int"
	strConst constType = "str"
)

type symbolWithInfo struct {
	Symbol       pythonresource.Symbol
	ArgConstInfo pythoncode.ArgConstInfo
}

// Symbols seen with the information of constant and counts
type symbols map[pythonimports.Hash]*symbolWithInfo

// SampleTag implements pipeline.Sample
func (symbols) SampleTag() {}

func newTypedConstInfo() pythoncode.TypedConstInfo {
	return pythoncode.TypedConstInfo{
		IntConstInfo:    make(pythoncode.ConstInfo),
		StringConstInfo: make(pythoncode.ConstInfo),
	}
}

func (s symbols) hit(symbol pythonresource.Symbol, argName string, constant string, constType constType) {
	hash := symbol.PathHash()

	if s[hash].ArgConstInfo[argName].StringConstInfo == nil {
		s[hash].ArgConstInfo[argName] = newTypedConstInfo()
	}

	switch constType {
	case intConst:
		if _, err := strconv.ParseInt(constant, 10, 64); err == nil {
			s[hash].ArgConstInfo[argName].IntConstInfo[constant]++
		}
	case strConst:
		s[hash].ArgConstInfo[argName].StringConstInfo[constant]++
	}
}

func (s symbols) newSymWithInfo(symbol pythonresource.Symbol) {
	hash := symbol.PathHash()
	if s[hash] == nil {
		symWithInfo := symbolWithInfo{
			Symbol:       symbol,
			ArgConstInfo: make(pythoncode.ArgConstInfo),
		}
		s[hash] = &symWithInfo
	}
}

// HitKeyword adds the specified symbol along with constant info for keyword argument
func (s symbols) Hit(rm pythonresource.Manager, symbols []pythonresource.Symbol, argPos int, argName string, constant string, constType constType) {
	// Always use canonical symbol, only collect the unique ones
	seen := make(map[pythonimports.Hash]bool)
	var canonicals []pythonresource.Symbol
	for _, symbol := range symbols {
		symbol = symbol.Canonical()
		if !seen[symbol.PathHash()] {
			seen[symbol.PathHash()] = true
			canonicals = append(canonicals, symbol)
		}
	}

	for _, symbol := range canonicals {
		s.newSymWithInfo(symbol)
		if argName == "" {
			spec := rm.ArgSpec(symbol)
			if spec != nil && argPos < len(spec.NonReceiverArgs()) {
				argName = spec.NonReceiverArgs()[argPos].Name
			} else if argName == "" {
				// For *vararg cases
				argName = fmt.Sprintf("vararg_%d", argPos)
			}
		}
		s.hit(symbol, argName, constant, constType)
	}
}

func extractFuncConstants(rm pythonresource.Manager, s pipeline.Sample) symbols {
	rast := s.(pythonpipeline.Resolved).RAST
	symbols := make(symbols)

	val := func(expr pythonast.Expr) pythontype.Value {
		return rast.References[expr]
	}

	pythonast.Inspect(rast.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		if call, ok := node.(*pythonast.CallExpr); ok {
			syms := toSymbols(rm, val(call.Func))
			for i, arg := range call.Args {
				var constant string
				var constType constType
				switch v := arg.Value.(type) {
				case *pythonast.NumberExpr:
					if v.Number.Token == pythonscanner.Int {
						constant = v.Number.Literal
						constType = intConst
					}
				case *pythonast.StringExpr:
					constant = v.Literal()
					constType = strConst
				}

				if constType == "" {
					continue
				}
				if name, ok := arg.Name.(*pythonast.NameExpr); ok {
					symbols.Hit(rm, syms, i, name.Ident.Literal, constant, constType)
				} else {
					symbols.Hit(rm, syms, i, "", constant, constType)
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
