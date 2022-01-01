package main

import (
	"fmt"
	"hash/fnv"
	"math/rand"

	"github.com/kiteco/kiteco/kite-exp/localcode-analysis/index"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
)

type fileRecord struct {
	LocalIndex index.LocalIndex
	File       sample.FileInfo
	Buffer     []byte

	RAST *pythonanalyzer.ResolvedAST
}

func (fileRecord) SampleTag() {}

type callRecord struct {
	CorpusID string
	Type     string
	Cursor   int64
	Address  string
	Params   []paramRecord
}

func (callRecord) SampleTag() {}

type paramRecord struct {
	Name string
	Type string

	NameSubtokens           int
	RecognizedNameSubtokens int

	Local     bool
	LocalName string
	LocalType string

	Global     bool
	GlobalName string
	GlobalType string
}

func sourceFunctionParams(res resources, sf *pythontype.SourceFunction) []paramRecord {
	prs := make([]paramRecord, 0, len(sf.Parameters))

	for i, p := range sf.Parameters {
		if i == 0 && (sf.HasClassReceiver || sf.HasReceiver) {
			// ignore receivers
			continue
		}

		nameSubtokens := traindata.SplitNameLiteral(p.Name)
		var rns int
		for _, s := range nameSubtokens {
			if _, found := res.mi.NameSubtokenIndex[s]; found {
				rns++
			}
		}

		pr := paramRecord{
			Name:                    p.Name,
			Type:                    fmt.Sprintf("%T", p.Symbol.Value),
			NameSubtokens:           len(nameSubtokens),
			RecognizedNameSubtokens: rns,
		}

		val := pythontype.Translate(kitectx.Background(), pythontype.WidenConstants(p.Symbol.Value), res.rm)
		if val == nil {
			prs = append(prs, pr)
			continue
		}

		for _, c := range pythontype.DisjunctsNoCtx(val) {
			switch c := c.(type) {
			case pythontype.SourceValue:
				pr.Local = true
				pr.LocalName = fmt.Sprintf("%s", c)
				pr.LocalType = fmt.Sprintf("%T", c)
			case pythontype.GlobalValue:
				pr.Global = true
				pr.GlobalName = fmt.Sprintf("%s", c)
				pr.GlobalType = fmt.Sprintf("%T", c)
			}
		}

		prs = append(prs, pr)
	}

	return prs
}

func getSourceFunction(rm pythonresource.Manager, val pythontype.Value) (*pythontype.SourceFunction, error) {
	origVal := val
	val = pythontype.Translate(kitectx.Background(), val, rm)
	if val == nil {
		return nil, pipeline.WrapErrorAsError(
			"unresolved translated val", fmt.Errorf("original val: %T", origVal))
	}

	switch val := val.(type) {
	case *pythontype.SourceFunction:
		return val, nil
	case *pythontype.SourceClass:
		initSym, ok := val.Members.Table["__init__"]
		if !ok {
			return nil, pipeline.NewErrorAsError("no __init__ in SourceClass")
		}
		return initSym.Value.(*pythontype.SourceFunction), nil
	case pythontype.SourceInstance:
		initSym, ok := val.Class.Members.Table["__init__"]
		if !ok {
			return nil, pipeline.NewErrorAsError("no __init__ in SourceInstance")
		}
		return initSym.Value.(*pythontype.SourceFunction), nil

	case pythontype.External:
		return nil, pipeline.WrapErrorAsError("non-local value", fmt.Errorf("%T", val))
	case pythontype.ExternalInstance:
		return nil, pipeline.WrapErrorAsError("non-local value", fmt.Errorf("%T", val))
	case pythontype.ExternalReturnValue:
		return nil, pipeline.WrapErrorAsError("non-local value", fmt.Errorf("%T", val))

	case pythontype.Union:
		for _, c := range pythontype.DisjunctsNoCtx(val) {
			sf, err := getSourceFunction(rm, c)
			if err == nil {
				return sf, nil
			}
		}
		return nil, pipeline.NewErrorAsError("union with no SourceFunction constituents")

	default:
		return nil, pipeline.WrapErrorAsError("unrecognized value", fmt.Errorf("%T", val))
	}

}

func newCallRecord(res resources, fr fileRecord, funcExpr pythonast.Expr, val pythontype.Value) (callRecord, error) {
	cr := callRecord{
		CorpusID: fr.LocalIndex.Corpus.ID(),
		Cursor:   int64(funcExpr.Begin()),
		Type:     fmt.Sprintf("%T", val),
	}
	if val == nil {
		return callRecord{}, pipeline.NewErrorAsError("unresolved val")
	}

	sf, err := getSourceFunction(res.rm, val)
	if err != nil {
		return callRecord{}, pipeline.WrapErrorAsError("getSourceFunction error", err)
	}
	cr.Address = sf.Address().String()

	cr.Params = sourceFunctionParams(res, sf)
	if len(cr.Params) == 0 {
		return callRecord{}, pipeline.NewErrorAsError("no params")
	}
	return cr, nil
}

func getCallRecords(res resources, fr fileRecord) []pipeline.Sample {
	var out []pipeline.Sample
	var recs []callRecord

	for expr := range fr.RAST.References {
		call, ok := expr.(*pythonast.CallExpr)
		if !ok {
			continue
		}

		val := fr.RAST.References[call.Func]

		rec, err := newCallRecord(res, fr, call.Func, val)
		if err != nil {
			out = append(out, pipeline.WrapError("newCallRecord error", err))
			continue
		}
		recs = append(recs, rec)
	}

	if len(recs) == 0 {
		return []pipeline.Sample{pipeline.NewError("no source functions found")}
	}

	// if there are too many calls, pseudorandomly choose a subset and discard the rest
	count := len(recs)
	if count > maxCallsPerFile {
		count = maxCallsPerFile
	}
	h := fnv.New64()
	h.Write([]byte(fr.Buffer))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	for _, i := range r.Perm(len(recs))[:count] {
		out = append(out, recs[i])
	}
	for i := 0; i < len(recs)-count; i++ {
		out = append(out, pipeline.NewError("call discarded"))
	}

	return out
}
