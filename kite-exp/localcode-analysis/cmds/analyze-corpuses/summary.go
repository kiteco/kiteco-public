package main

import "github.com/kiteco/kiteco/kite-golib/pipeline/sample"

type paramSummary struct {
	FuncCount int

	Count                      int
	Resolved                   int
	ResolvedToGlobal           int
	ResolvedToLocal            int
	AllNameSubtokensRecognized int
}

func newParamSummary(rec callRecord) paramSummary {
	s := paramSummary{FuncCount: 1}
	for _, p := range rec.Params {
		s.Count++
		if p.Global || p.Local {
			s.Resolved++
		}
		if p.Global {
			s.ResolvedToGlobal++
		}
		if p.Local {
			s.ResolvedToLocal++
		}
		if p.NameSubtokens == p.RecognizedNameSubtokens {
			s.AllNameSubtokensRecognized++
		}
	}

	return s
}

func (p paramSummary) SampleTag() {}

func (p paramSummary) Add(a sample.Addable) sample.Addable {
	o := a.(paramSummary)
	return paramSummary{
		Count:                      p.Count + o.Count,
		FuncCount:                  p.FuncCount + o.FuncCount,
		Resolved:                   p.Resolved + o.Resolved,
		ResolvedToGlobal:           p.ResolvedToGlobal + o.ResolvedToGlobal,
		ResolvedToLocal:            p.ResolvedToLocal + o.ResolvedToLocal,
		AllNameSubtokensRecognized: p.AllNameSubtokensRecognized + o.AllNameSubtokensRecognized,
	}
}
