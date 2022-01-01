package main

import (
	"fmt"
	"go/token"
	"math/rand"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncall"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var edges = pythongraph.EdgeSet{
	pythongraph.DataFlow,
}

var config = pythongraph.TrainConfig{
	MaxHops:      3,
	NumCorrupted: 5,
	Graph: pythongraph.GraphFeedConfig{
		EdgeSet: edges,
	},
}

type completedTask struct {
	Samples  []*renderedSample
	MetaInfo string
}

type worker struct {
	RM pythonresource.Manager
	MI pythonexpr.MetaInfo
}

func (w worker) doTask(sc srcCursor, task string) (*completedTask, error) {
	switch task {
	case "", "graph":
		return w.newGraphSamples(sc.Src)
	case "attrbase":
		return w.newAttrBaseSamples(sc)
	case "attr":
		return w.newAttrSamples(sc)
	case "call":
		return w.newCallSamples(sc)
	case "argtype":
		return w.newArgTypeSamples(sc)
	case "kwargname":
		return w.newKwargNameSamples(sc)
	case "argplaceholder":
		return w.newArgPlaceholderSamples(sc)
	default:
		return nil, fmt.Errorf("unsupported task %s", task)
	}
}

func (w worker) newGraphSamples(src string) (*completedTask, error) {
	panic("this file will be removed in a separate pr")
	// in, err := w.getInputs(src)
	// if err != nil {
	// 	return nil, err
	// }

	// g, err := pythongraph.NewGraph(kitectx.Background(), edges, in)
	// if err != nil {
	// 	return nil, err
	// }

	// samples, err := renderSavedSamples(pythongraph.SavedBundle{
	// 	Label:  "graph",
	// 	Graph:  nil,
	// 	Graph: pythongraph.New
	// 	Buffer: []byte(src),
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// return &completedTask{
	// 	Samples: samples,
	// }, nil
}

func (w worker) newAttrBaseSamples(sc srcCursor) (*completedTask, error) {
	in, err := w.getInputs(sc.Src)
	if err != nil {
		return nil, err
	}

	// find name under cursor to get symbol
	var sym pythonresource.Symbol
	cursor := token.Pos(sc.Cursor)
	pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !sym.Nil() {
			return false
		}
		if name, ok := n.(*pythonast.NameExpr); ok {
			if cursor >= name.Begin() && cursor <= name.End() {
				sym = anySym(w.RM, in.RAST.References[name])
			}
		}
		return true
	})

	if sym.Nil() {
		return nil, fmt.Errorf("unable to resolve symbol under cursor or find name under cursor")
	}

	saver := new(saver)
	_, err = pythongraph.NewAttrBaseTrainSample(config, w.defaultTrainParams(sc, saver), pythongraph.AttrBaseTrainInputs{
		Inputs: in,
		Symbol: sym,
	})

	if err != nil {
		return nil, err
	}

	samples, err := renderSavedSamples(saver.Saved...)
	if err != nil {
		return nil, err
	}

	return &completedTask{
		Samples: samples,
	}, nil
}

func (w worker) newAttrSamples(sc srcCursor) (*completedTask, error) {
	in, err := w.getInputs(sc.Src)
	if err != nil {
		return nil, err
	}

	// find attr under cursor to get symbol
	var sym pythonresource.Symbol
	cursor := token.Pos(sc.Cursor)
	pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || !sym.Nil() {
			return false
		}
		if attr, ok := n.(*pythonast.AttributeExpr); ok {
			if cursor >= attr.Attribute.Begin && cursor <= attr.Attribute.End {
				sym = anySym(w.RM, in.RAST.References[attr])
			}
		}
		return true
	})

	if sym.Nil() {
		return nil, fmt.Errorf("unable to resolve symbol under cursor or find attr under cursor")
	}

	parent, err := w.RM.PathSymbol(sym.Path().Predecessor())
	if err != nil {
		return nil, fmt.Errorf("unable to find parent of %v: %v", sym, err)
	}

	saver := new(saver)
	sample, err := pythongraph.NewAttributeTrainSample(config, w.defaultTrainParams(sc, saver), pythongraph.AttributeTrainInputs{
		Inputs: in,
		Symbol: sym,
		Parent: parent,
		CanonicalToSym: map[string][]string{
			sym.Canonical().PathString(): {sym.PathString()},
		},
	})

	if err != nil {
		return nil, err
	}

	samples, err := renderSavedSamples(saver.Saved...)
	if err != nil {
		return nil, err
	}

	return &completedTask{
		Samples:  samples,
		MetaInfo: w.inferProductionMetaInfo(sym, sample),
	}, nil
}

func (w worker) newArgTypeSamples(sc srcCursor) (*completedTask, error) {
	in, err := w.getInputs(sc.Src)
	if err != nil {
		return nil, err
	}

	// find call under cursor to get symbol
	var sym pythonresource.Symbol
	cursor := token.Pos(sc.Cursor)
	pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
		if call, ok := n.(*pythonast.CallExpr); ok {
			if cursor >= call.LeftParen.End && cursor <= call.RightParen.Begin {
				sym = anySym(w.RM, in.RAST.References[call.Func])
			}
		}
		return true
	})

	if sym.Nil() {
		return nil, fmt.Errorf("unable to resolve symbol for call")
	}

	saver := new(saver)
	sample, err := pythongraph.NewArgTypeTrainSample(config, w.defaultTrainParams(sc, saver), pythongraph.ArgTypeTrainInputs{
		Inputs: in,
		Symbol: sym,
	})

	if err != nil {
		return nil, err
	}

	rss, err := renderSavedSamples(saver.Saved...)
	if err != nil {
		return nil, err
	}

	return &completedTask{
		Samples:  rss,
		MetaInfo: w.inferProductionMetaInfo(sym, sample),
	}, nil
}

func (w worker) newKwargNameSamples(sc srcCursor) (*completedTask, error) {
	in, err := w.getInputs(sc.Src)
	if err != nil {
		return nil, err
	}

	// find attr under cursor to get symbol
	var sym pythonresource.Symbol
	cursor := token.Pos(sc.Cursor)
	pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
		if call, ok := n.(*pythonast.CallExpr); ok {
			for _, arg := range call.Args {
				if _, ok := arg.Name.(*pythonast.NameExpr); !ok {
					continue
				}
				if _, ok := arg.Name.(*pythonast.NameExpr); !ok {
					continue
				}
				if cursor < arg.Name.Begin() || cursor > arg.Name.End() {
					continue
				}
				sym = anySym(w.RM, in.RAST.References[call.Func])
				break
			}
		}
		return true
	})

	if sym.Nil() {
		return nil, fmt.Errorf("unable to resolve symbol for call")
	}

	saver := new(saver)
	keywords, err := pythoncall.GetKwargNames(pythoncode.KeywordCountsStats, 10)

	sample, err := pythongraph.NewKwargNameTrainSample(config, w.defaultTrainParams(sc, saver), pythongraph.KwargNameTrainInputs{
		Inputs:   in,
		Symbol:   sym,
		Keywords: keywords[sym.Canonical().PathString()],
	})

	if err != nil {
		return nil, err
	}

	rss, err := renderSavedSamples(saver.Saved...)
	if err != nil {
		return nil, err
	}

	return &completedTask{
		Samples:  rss,
		MetaInfo: w.inferProductionMetaInfo(sym, sample),
	}, nil
}

func (w worker) inferProductionMetaInfo(sym pythonresource.Symbol, sample *pythongraph.InferProductionSample) string {

	var targets []string
	for _, idx := range sample.Production.DecoderTargets.Indices {
		targets = append(targets, fmt.Sprintf("%d", idx))
	}

	return fmt.Sprintf(`
Symbol: %s
Label: %d
Targets: %s
`, sym.PathString(), sample.Production.Labels[0], strings.Join(targets, " , "))
}

func (w worker) newArgPlaceholderSamples(sc srcCursor) (*completedTask, error) {

	in, err := w.getInputs(sc.Src)
	if err != nil {
		return nil, err
	}

	var symbols []pythonresource.Symbol
	pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
		if call, ok := n.(*pythonast.CallExpr); ok {
			if len(call.Args) > 0 {
				sym := anySym(w.RM, in.RAST.References[call.Func])
				if !sym.Nil() {
					symbols = append(symbols, sym)
				}
				return false
			}
		}
		return true
	})

	if len(symbols) == 0 {
		return nil, fmt.Errorf("unable to find an arg placeholder site")
	}
	sym := symbols[rand.Intn(len(symbols))]

	saver := new(saver)

	sample, err := pythongraph.NewArgPlaceholderTrainSample(config, w.defaultTrainParams(sc, saver), pythongraph.ArgPlaceholderTrainInputs{
		Inputs: in,
		Symbol: sym,
	})

	rss, err := renderSavedSamples(saver.Saved...)
	if err != nil {
		return nil, err
	}

	var targets []string
	for _, idx := range sample.Production.DecoderTargets.Indices {
		targets = append(targets, fmt.Sprintf("%d", idx))
	}

	meta := fmt.Sprintf(`
Symbol: %s
Label: %d
Targets: %s
`, sym.PathString(), sample.Production.Labels, strings.Join(targets, " , "))

	return &completedTask{
		Samples:  rss,
		MetaInfo: meta,
	}, nil
}

func (w worker) newCallSamples(sc srcCursor) (*completedTask, error) {
	in, err := w.getInputs(sc.Src)
	if err != nil {
		return nil, err
	}

	var syms []pythonresource.Symbol
	cursor := token.Pos(sc.Cursor)
	pythonast.Inspect(in.RAST.Root, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || len(syms) > 0 {
			return false
		}

		if call, ok := n.(*pythonast.CallExpr); ok {
			for _, arg := range call.Args {
				if !pythonast.IsNil(arg.Name) {
					continue
				}
				if _, ok := arg.Value.(*pythonast.NameExpr); !ok {
					continue
				}
				if cursor < arg.Value.Begin() || cursor > arg.Value.End() {
					continue
				}
				syms = python.GetExternalSymbols(kitectx.Background(), w.RM, in.RAST.References[call.Func])
				break
			}

		}
		return true
	})

	if len(syms) == 0 {
		return nil, fmt.Errorf("unable to resolve symbol for call or find valid argument under cursor")
	}
	sym := syms[0]

	saver := new(saver)
	_, err = pythongraph.NewCallTrainSample(config, w.defaultTrainParams(sc, saver), pythongraph.CallTrainInputs{
		Inputs: in,
		Symbol: sym,
	})

	if err != nil {
		return nil, err
	}

	rss, err := renderSavedSamples(saver.Saved...)
	if err != nil {
		return nil, err
	}

	return &completedTask{
		Samples: rss,
	}, nil
}
