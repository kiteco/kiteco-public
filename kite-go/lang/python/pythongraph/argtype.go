package pythongraph

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

// ArgTypeTrainInputs contains the necessary information to produce a training sample for the arg type model.
type ArgTypeTrainInputs struct {
	Inputs
	Symbol pythonresource.Symbol
	Hash   string
}

type argTypeSite struct {
	argument  *pythonast.Argument
	call      *pythonast.CallExpr
	node      *Node
	argType   traindata.ArgType
	seenKwarg bool
	position  int
}

// NewArgTypeTrainSample creates a new training sample based on the given inputs
func NewArgTypeTrainSample(config TrainConfig, params TrainParams, in ArgTypeTrainInputs) (*InferProductionSample, error) {
	a := newAnalysis(in.RM, in.Words, in.RAST)

	builder1 := newBuilder(kitectx.Background(), a, false, true)

	builder1.BuildEdges(config.Graph.EdgeSet)

	// always canonicalize symbol
	sym := in.Symbol.Canonical()
	site1 := builder1.randomArgTypeSite(kitectx.Background(), params.Rand, sym)

	if site1 == (argTypeSite{}) {
		return nil, fmt.Errorf("arg type site not found")
	}

	// NOTE: we do not add a node label for site1.node in case it is the final
	// site (which corresponds to the stop site and does not yet have a node).
	save(
		params.Saver,
		SavedBundle{
			Label:   "original",
			builder: builder1,
			Buffer:  in.Buffer,
		},
	)

	lm := linenumber.NewMap(in.Buffer)
	trimEnd := trimEndLineOrStmt(site1.call.End(), site1.call, lm, in.RAST.ParentStmts)

	var newSrc []byte
	// For positional and keyword arguments, just replace the selected argument with a dummy one
	if site1.argType != traindata.Stop {
		newSrc = bytes.Join([][]byte{
			in.Buffer[:site1.argument.Begin()],
			[]byte(traindata.InferArgTypeMarker),
			[]byte(")"),
			in.Buffer[trimEnd:],
		}, nil)

	} else {
		allArgs := site1.call.Args
		if len(allArgs) > 0 {
			// Add a dummy argument right after the last argument
			lastArg := site1.call.Args[len(site1.call.Args)-1]
			newSrc = bytes.Join([][]byte{
				in.Buffer[:lastArg.End()],
				[]byte(","),
				[]byte(traindata.InferArgTypeMarker),
				[]byte(")"),
				in.Buffer[trimEnd:],
			}, nil)
		} else {
			// If the func call look like foo(), creating dummy argument right after left parenthesis
			newSrc = bytes.Join([][]byte{
				in.Buffer[:site1.call.LeftParen.End],
				[]byte(traindata.InferArgTypeMarker),
				[]byte(")"),
				in.Buffer[trimEnd:],
			}, nil)
		}
	}

	save(params.Saver, bufferBundle("munged-no-graph", newSrc))

	a2, err := analyze(kitectx.Background(), in.RM, newSrc)
	if err != nil {
		return nil, err
	}

	builder2 := newBuilder(kitectx.Background(), a2, false, true)
	builder2.BuildEdges(config.Graph.EdgeSet)

	site2 := builder2.findArgTypeSiteAgain()

	if site2 == (argTypeSite{}) {
		return nil, fmt.Errorf("couldn't find arg type site again")
	}

	// Make sure the target argument is of the same position of the same call
	if site2.call.Begin() != site1.call.Begin() || site1.position != site2.position {
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

	site2.node.Attrs.Literal = traindata.InferArgTypeMarker
	site2.node.Attrs.Types = []string{traindata.InferArgTypeMarker}

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

	var targets []pythonimports.Hash
	if !site1.seenKwarg {
		targets = []pythonimports.Hash{
			traindata.IDForChooseArgType(sym.PathString(), traindata.Stop),
			traindata.IDForChooseArgType(sym.PathString(), traindata.Positional),
			traindata.IDForChooseArgType(sym.PathString(), traindata.Keyword),
		}
	} else {
		// Have seen kwarg argument, targets can only be keyword or stop
		if site1.argType != traindata.Positional {
			targets = []pythonimports.Hash{
				traindata.IDForChooseArgType(sym.PathString(), traindata.Stop),
				traindata.IDForChooseArgType(sym.PathString(), traindata.Keyword),
			}
		} else {
			return nil, fmt.Errorf("argType positional after kwarg has been seen")
		}
	}

	eg, cgToEG := builder2.ExpansionGraph(params.ModelMeta, []*Node{site2.node}, joinNodes(scopeNodes, contextNodes))

	feed := ProductionModelFeed{
		PredictionNodes: []int32{int32(site2.node.ID)},
		ScopeEncoder:    newNodeIDFeed(scopeNodes, cgToEG),
		ContextTokens:   newNodeIDFeed(contextNodes, cgToEG),
	}

	labelID := traindata.IDForChooseArgType(sym.PathString(), site1.argType)
	label := -1
	for i, t := range targets {
		tid, ok := params.ProductionIndex.Index(t)
		if !ok {
			return nil, fmt.Errorf("no decoder target found for %s", t)
		}
		feed.DecoderTargets.Indices = append(feed.DecoderTargets.Indices, tid)
		feed.DecoderTargets.SampleIDs = append(feed.DecoderTargets.SampleIDs, 0)

		if t == labelID {
			label = i
		}
	}

	if label == -1 {
		return nil, fmt.Errorf("unable to find label for %s", labelID)
	}

	feed.Labels = []int{label}
	feed.Corrupted = newCorruptedSegmented(params.Rand, label, len(targets), config.NumCorrupted)

	return &InferProductionSample{
		ContextGraph:   builder2.newGraphFeed(params.ModelMeta),
		ExpansionGraph: eg,
		Production:     feed,
	}, nil
}

func (b *graphBuilder) randomArgTypeSite(ctx kitectx.Context, rand *rand.Rand, sym pythonresource.Symbol) argTypeSite {
	ctx.CheckAbort()
	var sites []argTypeSite

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

		// Since we can't have positional arguments anymore once we've seen keyword arguments
		// We need to keep track of this
		var seenKwarg bool

		for i, arg := range call.Args {
			if pythonast.IsNil(arg.Name) && !seenKwarg {
				sites = append(sites, argTypeSite{
					argument:  arg,
					node:      b.astNodes[arg],
					argType:   traindata.Positional,
					position:  i,
					seenKwarg: seenKwarg,
					call:      call,
				})
			} else {
				sites = append(sites, argTypeSite{
					argument:  arg,
					node:      b.astNodes[arg],
					argType:   traindata.Keyword,
					seenKwarg: seenKwarg,
					position:  i,
					call:      call,
				})
				seenKwarg = true
			}
		}

		sites = append(sites, argTypeSite{
			argType:   traindata.Stop,
			seenKwarg: seenKwarg,
			position:  len(call.Args),
			call:      call,
		})

		return true
	})

	if len(sites) == 0 {
		return argTypeSite{}
	}

	return sites[rand.Intn(len(sites))]
}

func (b *graphBuilder) findArgTypeSiteAgain() argTypeSite {
	for ast := range b.astNodes {
		call, ok := ast.(*pythonast.CallExpr)
		if !ok {
			continue
		}
		for i, arg := range call.Args {
			value, ok := arg.Value.(*pythonast.NameExpr)
			if !ok || value.Ident.Literal != traindata.InferArgTypeMarker {
				continue
			}

			return argTypeSite{
				argument: arg,
				node:     b.astNodes[arg],
				position: i,
				call:     call,
			}
		}
	}

	return argTypeSite{}
}
