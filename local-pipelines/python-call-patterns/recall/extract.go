package main

import (
	"math/rand"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

type call struct {
	Call *pythonast.CallExpr
	RAST *pythonanalyzer.ResolvedAST
	Src  string
	Sym  pythonresource.Symbol
}

func (call) SampleTag() {}

func extractCalls(s pythonpipeline.AnalyzedEvent) []pipeline.Sample {
	var calls []pipeline.Sample
	pythonast.Inspect(s.Context.Resolved.Root, func(n pythonast.Node) bool {
		switch n := n.(type) {
		case *pythonast.BadStmt:
			return false
		case *pythonast.CallExpr:
			if !valid(s.Context.Importer.Global, s.Context.Resolved, n) {
				break
			}

			val := s.Context.Resolved.References[n.Func]
			syms := python.GetExternalSymbols(kitectx.Background(), s.Context.Importer.Global, val)
			if len(syms) == 0 {
				break
			}

			calls = append(calls, call{
				Call: n,
				RAST: s.Context.Resolved,
				Src:  s.Event.Buffer,
				Sym:  syms[0],
			})
		}
		return true
	})

	if len(calls) > maxCallsPerFile {
		var subSampled []pipeline.Sample
		for _, i := range rand.Perm(len(calls))[:maxCallsPerFile] {
			subSampled = append(subSampled, calls[i])
		}
		return subSampled
	}

	return calls
}

func valid(rm pythonresource.Manager, rast *pythonanalyzer.ResolvedAST, call *pythonast.CallExpr) bool {
	keywords := make(map[string]bool)
	for _, arg := range call.Args {
		name, _ := arg.Name.(*pythonast.NameExpr)
		if name == nil && len(keywords) > 0 {
			return false
		}
		if name != nil {
			keywords[name.Ident.Literal] = true
		}

		syms := python.GetExternalSymbols(kitectx.Background(), rm, rast.References[arg.Value])
		if len(syms) == 0 {
			// TODO: this could cause weird biases in the results
			return false
		}
	}
	return true
}
