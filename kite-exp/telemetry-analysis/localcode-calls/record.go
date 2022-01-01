package main

import (
	"fmt"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"hash/fnv"
	"math/rand"
)

type paramRecord struct {
	Name                string
	Type                string
	Subtokens           int
	RecognizedSubtokens int

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

		subtokens := traindata.SplitNameLiteral(p.Name)
		var rs int
		for _, s := range subtokens {
			if _, found := res.sti[s]; found {
				rs++
			}
		}

		pr := paramRecord{
			Name:                p.Name,
			Type:                fmt.Sprintf("%T", p.Symbol.Value),
			Subtokens:           len(subtokens),
			RecognizedSubtokens: rs,
		}

		val := pythontype.Translate(kitectx.Background(), pythontype.WidenConstants(p.Symbol.Value), res.rm)
		if val == nil {
			prs = append(prs, pr)
			continue
		}

		for _, c := range pythontype.DisjunctsNoCtx(val) {
			if sv, ok := c.(pythontype.SourceValue); ok {
				pr.Local = true
				pr.LocalName = fmt.Sprintf("%s", sv)
				pr.LocalType = fmt.Sprintf("%T", sv)
			}
			if gv, ok := c.(pythontype.GlobalValue); ok {
				pr.Global = true
				pr.GlobalName = fmt.Sprintf("%s", gv)
				pr.GlobalType = fmt.Sprintf("%T", gv)
			}
		}

		prs = append(prs, pr)
	}

	return prs
}

type funcRecord struct {
	MessageID string
	Type      string
	Cursor    int64
	Address   string
	Params    []paramRecord
}

func (funcRecord) SampleTag() {}

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

func newFuncRecord(res resources, messageID string, fnc pythonast.Expr, val pythontype.Value) (funcRecord, error) {
	fr := funcRecord{
		MessageID: messageID,
		Cursor:    int64(fnc.Begin()),
		Type:      fmt.Sprintf("%T", val),
	}
	if val == nil {
		return funcRecord{}, pipeline.NewErrorAsError("unresolved val")
	}

	sf, err := getSourceFunction(res.rm, val)
	if err != nil {
		return funcRecord{}, pipeline.WrapErrorAsError("getSourceFunction error", err)
	}
	fr.Address = sf.Address().String()

	fr.Params = sourceFunctionParams(res, sf)
	if len(fr.Params) == 0 {
		return funcRecord{}, pipeline.NewErrorAsError("no params")
	}
	return fr, nil
}

func getRecords(res resources, ev pythonpipeline.AnalyzedEvent) []pipeline.Sample {
	var out []pipeline.Sample
	var recs []funcRecord

	for expr := range ev.Context.Resolved.References {
		call, ok := expr.(*pythonast.CallExpr)
		if !ok {
			continue
		}

		val := ev.Context.Resolved.References[call.Func]

		rec, err := newFuncRecord(res, ev.Event.Meta.ID.String(), call.Func, val)
		if err != nil {
			out = append(out, pipeline.WrapError("newFuncRecord error", err))
			continue
		}
		recs = append(recs, rec)
	}

	if len(recs) == 0 {
		return []pipeline.Sample{pipeline.NewError("no source functions found")}
	}

	// if there are too many calls, pseudorandomly choose a subset and discard the rest
	count := len(recs)
	if count > maxCallsPerEvent {
		count = maxCallsPerEvent
	}
	h := fnv.New64()
	h.Write([]byte(ev.Event.Buffer))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	for _, i := range r.Perm(len(recs))[:count] {
		out = append(out, recs[i])
	}
	for i := 0; i < len(recs)-count; i++ {
		out = append(out, pipeline.NewError("call discarded"))
	}

	return out
}
