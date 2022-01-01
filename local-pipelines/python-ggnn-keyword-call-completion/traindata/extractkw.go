package main

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type keywordsCount map[string]int

type symbols map[string]keywordsCount

func extract(symbolName string, rm pythonresource.Manager, ti *typeinduction.Client, content []byte) (string, keywordsCount, error) {
	ast, err := pythonparser.Parse(kitectx.Background(), content, pythonparser.Options{})
	if err != nil {
		return symbolName, nil, err
	}

	resolved, err := resolve(rm, ti, ast)
	if err != nil {
		return symbolName, nil, err
	}

	kwct := make(keywordsCount)
	pythonast.Inspect(ast, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}
		switch node := node.(type) {
		case *pythonast.CallExpr:
			var matches bool
			for _, path := range toPaths(rm, resolved[node.Func]) {
				if path.Equals(symbolName) {
					matches = true
					break
				}
			}

			if matches {
				for _, arg := range node.Args {

					if name, ok := arg.Name.(*pythonast.NameExpr); ok {
						kwct[name.Ident.Literal]++
					}
				}
			}
			return true
		default:
			return true
		}
	})

	return symbolName, kwct, nil
}

// this function merges the keywords set together
func merge(symbolName string, entry keywordsCount, all symbols) {
	if _, ok := all[symbolName]; !ok {
		all[symbolName] = entry
		return
	}
	kwcnts := all[symbolName]
	for kw, cnt := range entry {
		kwcnts[kw] += cnt
	}
}

func toPaths(rm pythonresource.Manager, val pythontype.Value) []pythonimports.DottedPath {
	if val == nil {
		return nil
	}

	var gvs []pythontype.Value
	for _, dv := range pythontype.Disjuncts(kitectx.Background(), val) {
		dv = pythontype.WidenConstants(dv)
		if dv := pythontype.TranslateGlobal(dv, rm); dv != nil {
			gvs = append(gvs, dv)
		}
	}

	var paths []pythonimports.DottedPath
	seen := make(map[pythonimports.Hash]bool)
	for _, val := range pythontype.Disjuncts(kitectx.Background(), pythontype.Unite(kitectx.Background(), gvs...)) {
		if val == nil {
			continue
		}

		var path pythonimports.DottedPath
		switch val := val.(type) {
		case pythontype.External:
			path = val.Symbol().Canonical().Path()
		case pythontype.ExternalInstance:
			path = val.TypeExternal.Symbol().Canonical().Path()
		}

		if !path.Empty() && !seen[path.Hash] {
			seen[path.Hash] = true
			paths = append(paths, path)
		}
	}
	return paths
}

func resolve(rm pythonresource.Manager, ti *typeinduction.Client, ast *pythonast.Module) (map[pythonast.Expr]pythontype.Value, error) {
	delegate := make(delegate)
	_, err := pythonstatic.AnalyzeGlobal(kitectx.Background(), pythonstatic.AssemblerInputs{
		Graph:    rm,
		Delegate: delegate,
	}, ast)

	if err != nil {
		return nil, err
	}
	return delegate, nil
}

type delegate map[pythonast.Expr]pythontype.Value

func (d delegate) Pass(curr, total int) {}

func (d delegate) Resolved(expr pythonast.Expr, value pythontype.Value) {
	d[expr] = value
}
