package pythongraph

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

// KwargNameTrainInputs contains the necessary information to produce a training sample for the kwarg_name model.
type KwargNameTrainInputs struct {
	Inputs
	Symbol   pythonresource.Symbol
	Keywords []string
	Hash     string
}

type kwargNameSite struct {
	call     *pythonast.CallExpr
	argument *pythonast.Argument
	// the string for keyword argument name
	kwargName string
	// The node for keyword argument name
	node *Node
}

// NewKwargNameTrainSample creates a kwarg_name train sample from the given inputs.
func NewKwargNameTrainSample(config TrainConfig, params TrainParams, in KwargNameTrainInputs) (*InferProductionSample, error) {
	if len(in.Keywords) <= 1 {
		return nil, fmt.Errorf("func only has zero or one candidate kwarg name")
	}

	a := newAnalysis(in.RM, in.Words, in.RAST)

	builder1 := newBuilder(kitectx.Background(), a, false, true)

	builder1.BuildEdges(config.Graph.EdgeSet)

	// always canonicalize symbol
	sym := in.Symbol.Canonical()

	site1 := builder1.randomKwargNameSite(kitectx.Background(), params.Rand, sym, in.Keywords)

	if site1 == (kwargNameSite{}) {
		return nil, fmt.Errorf("kwarg name site not found")
	}

	save(
		params.Saver,
		SavedBundle{
			Label:      "original",
			builder:    builder1,
			NodeLabels: nodeLabels(site1.node, "site"),
			Buffer:     in.Buffer,
		},
	)

	lm := linenumber.NewMap(in.Buffer)
	trimEnd := trimEndLineOrStmt(site1.call.End(), site1.call, lm, in.RAST.ParentStmts)

	newSrc := bytes.Join([][]byte{
		in.Buffer[:site1.argument.Begin()],
		[]byte(traindata.InferKwargNameMarker),
		[]byte("="),
		[]byte(traindata.KwargValuePlaceholder),
		[]byte(")"),
		in.Buffer[trimEnd:],
	}, nil)

	save(params.Saver, bufferBundle("munged-no-graph", newSrc))

	a2, err := analyze(kitectx.Background(), in.RM, newSrc)
	if err != nil {
		return nil, err
	}

	builder2 := newBuilder(kitectx.Background(), a2, false, true)
	builder2.BuildEdges(config.Graph.EdgeSet)

	site2 := builder2.findKwargNameSiteAgain()
	if site2 == (kwargNameSite{}) {
		return nil, fmt.Errorf("couldn't find kwarg name site again")
	}

	if site2.argument.Begin() != site1.argument.Begin() {
		return nil, fmt.Errorf("site mismatch")
	}

	save(
		params.Saver,
		SavedBundle{
			Label:      "munged",
			builder:    builder2,
			NodeLabels: nodeLabels(site2.node, "site"),
			Buffer:     newSrc,
		},
	)

	scope := builder2.ScopeForCall(kitectx.TODO(), site2.call)
	if len(scope) == 0 {
		return nil, fmt.Errorf("no variables in scope")
	}
	builder2.vm.ReduceTo(builder2.a.RAST.Root, scope)

	scopeNodes := builder2.AddScopeNodeAndEdges(scope)
	contextNodes := builder2.ContextTokens(site2.argument)

	site2.node.Attrs.Types = []string{traindata.InferKwargNameMarker}
	site2.node.Attrs.Literal = traindata.InferKwargNameMarker

	if config.MaxHops > 0 {
		keep := nodeSet(map[*Node]bool{
			site2.node: true,
		})
		for _, nodes := range [][]*Node{scopeNodes, contextNodes} {
			for _, n := range nodes {
				keep[n] = true
			}
		}
		// TODO: we have to add all of the nodes in the current reference set
		// because otherwise they can get pruned and then some of the variable scope
		// nodes will not have incoming edges
		for _, v := range scope {
			for _, ref := range v.Refs.Names() {
				keep[builder2.astNodes[ref]] = true
			}
		}
		builder2.Prune(kitectx.TODO(), keep, config.MaxHops)
	}

	save(
		params.Saver,
		SavedBundle{
			Label:      "pruned",
			builder:    builder2,
			NodeLabels: nodeLabels(site2.node, "site"),
			Buffer:     newSrc,
		},
	)

	if len(builder2.nodes) > maxNumNodesTrainGraph {
		return nil, fmt.Errorf("too many nodes in graph got %d > %d", len(builder2.nodes), maxNumNodesTrainGraph)
	}

	targets, label, err := params.ProductionIndex.ChildrenWithLabel(
		traindata.IDForChooseKwargParent(sym.PathString()),
		traindata.IDForChooseKwarg(sym.PathString(), site1.kwargName),
	)

	if err != nil {
		return nil, fmt.Errorf("error getting production decoders: %v", err)
	}

	eg, cgToEG := builder2.ExpansionGraph(params.ModelMeta, []*Node{site2.node}, joinNodes(scopeNodes, contextNodes))

	return &InferProductionSample{
		ContextGraph:   builder2.newGraphFeed(params.ModelMeta),
		ExpansionGraph: eg,
		Production: ProductionModelFeed{
			PredictionNodes: []int32{int32(site2.node.ID)},
			Labels:          []int{label},
			DecoderTargets:  traindata.NewSegmentedIndicesFeed(targets...),
			Corrupted:       newCorruptedSegmented(params.Rand, label, len(targets), config.NumCorrupted),
			ScopeEncoder:    newNodeIDFeed(scopeNodes, cgToEG),
			ContextTokens:   newNodeIDFeed(contextNodes, cgToEG),
		},
	}, nil
}

func (b *graphBuilder) randomKwargNameSite(ctx kitectx.Context, rand *rand.Rand, sym pythonresource.Symbol, keywords []string) kwargNameSite {
	ctx.CheckAbort()
	var sites []kwargNameSite

	pythonast.Inspect(b.a.RAST.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		call, ok := node.(*pythonast.CallExpr)
		if !ok {
			return true
		}

		fn := b.astNodes[call.Func]
		if !fn.matchesType(sym, true) {
			return true
		}

		for _, arg := range call.Args {
			if pythonast.IsNil(arg.Name) {
				continue
			}

			name, ok := arg.Name.(*pythonast.NameExpr)

			if !ok || b.astNodes[name] == nil || b.astNodes[arg] == nil {
				continue
			}

			keyword := name.Ident.Literal
			for _, k := range keywords {
				if k == keyword {
					sites = append(sites, kwargNameSite{
						call:      call,
						argument:  arg,
						kwargName: keyword,
						node:      b.astNodes[name],
					})
					break
				}
			}
		}

		return true
	})

	if len(sites) == 0 {
		return kwargNameSite{}
	}

	return sites[rand.Intn(len(sites))]
}

func (b *graphBuilder) findKwargNameSiteAgain() kwargNameSite {
	for ast := range b.astNodes {
		call, ok := ast.(*pythonast.CallExpr)
		if !ok {
			continue
		}
		for _, arg := range call.Args {
			name, ok := arg.Name.(*pythonast.NameExpr)
			if !ok || name.Ident.Literal != traindata.InferKwargNameMarker {
				continue
			}

			return kwargNameSite{
				call:      call,
				argument:  arg,
				kwargName: name.Ident.Literal,
				node:      b.wordNodes[*name.Ident],
			}
		}
	}

	return kwargNameSite{}
}
