package main

import "github.com/kiteco/kiteco/kite-golib/pipeline/sample"

type paramSummary struct {
	FuncCount int

	Count                  int
	Resolved               int
	ResolvedToGlobal       int
	ResolvedToLocal        int
	AllSubtokensRecognized int
}

func newParamSummary(rec funcRecord) paramSummary {
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
		if p.Subtokens == p.RecognizedSubtokens {
			s.AllSubtokensRecognized++
		}
	}

	return s
}

func (p paramSummary) SampleTag() {}

func (p paramSummary) Add(a sample.Addable) sample.Addable {
	o := a.(paramSummary)
	return paramSummary{
		Count:                  p.Count + o.Count,
		FuncCount:              p.FuncCount + o.FuncCount,
		Resolved:               p.Resolved + o.Resolved,
		ResolvedToGlobal:       p.ResolvedToGlobal + o.ResolvedToGlobal,
		ResolvedToLocal:        p.ResolvedToLocal + o.ResolvedToLocal,
		AllSubtokensRecognized: p.AllSubtokensRecognized + o.AllSubtokensRecognized,
	}
}
