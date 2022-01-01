package pythongraph

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/kiteco/kiteco/kite-golib/linenumber"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	maxNumNodesTrainGraph = 3000
)

// CallTrainInputs groups the inputs required to
// compute training sample for call completion.
type CallTrainInputs struct {
	// Hash of input source file, used for marking bad
	// hashes during batch building
	Hash string
	Inputs
	// Symbol requested by the user, this
	// will be canonicalized since the pythongraph operates
	// on canonical symbols
	Symbol pythonresource.Symbol
}

// NewCallTrainSample builds a new call train sample
// from the specified source.
func NewCallTrainSample(config TrainConfig, params TrainParams, in CallTrainInputs) (*InferNameSample, error) {
	// always canonicalize symbol
	sym := in.Symbol.Canonical()

	patterns := traindata.NewCallPatterns(in.RM, sym)
	if patterns == nil {
		return nil, fmt.Errorf("no patterns for %v", sym)
	}

	// first pass, build graph, get call sites
	call, arg, err := func() (callSite, callSiteArg, error) {

		a := newAnalysis(in.RM, in.Words, in.RAST)

		builder := newBuilder(kitectx.Background(), a, false, true)

		builder.BuildEdges(config.Graph.EdgeSet)

		// get valid call sites and select an argument
		calls := builder.BuildCallSites(kitectx.Background(), sym, patterns)
		if len(calls) == 0 {
			return callSite{}, callSiteArg{}, fmt.Errorf("unable to find valid call site for %s", sym.PathString())
		}

		idx := rand.Int() % len(calls)
		call := calls[idx]
		argIdx := rand.Int() % len(call.Args)

		save(
			params.Saver,
			SavedBundle{
				Label:      "original",
				builder:    builder,
				NodeLabels: nodeLabels(builder.astNodes[call.Call.Args[argIdx].Value], "site"),
				Buffer:     in.Buffer,
			},
		)

		return call, call.Args[argIdx], nil
	}()
	if err != nil {
		return nil, err
	}

	// trim source and refind the call
	arg, builder, newBuffer, err := func() (callSiteArg, *graphBuilder, []byte, error) {
		trimEnd := trimEndLineOrStmt(arg.NameSite.Original.End(), call.Call, linenumber.NewMap(in.Buffer), in.RAST.ParentStmts)

		// TODO: do better, does not handle nested calls or basically anything except the simplest case
		newBuffer := bytes.Join([][]byte{
			in.Buffer[:arg.NameSite.Original.End()],
			[]byte(")"),
			in.Buffer[trimEnd:],
		}, nil)

		save(params.Saver, bufferBundle("munged-no-graph", newBuffer))

		a, err := analyze(kitectx.Background(), in.RM, newBuffer)
		if err != nil {
			return callSiteArg{}, nil, nil, fmt.Errorf("error re-analyzing new buffer: %v", err)
		}

		var newCall *pythonast.CallExpr
		pythonast.Inspect(a.RAST.Root, func(n pythonast.Node) bool {
			if pythonast.IsNil(n) || !pythonast.IsNil(newCall) {
				return false
			}

			if c, ok := n.(*pythonast.CallExpr); ok && c.LeftParen.Begin == call.Call.LeftParen.Begin {
				newCall = c
			}
			return true
		})

		if pythonast.IsNil(newCall) {
			return callSiteArg{}, nil, nil, fmt.Errorf("unable find call expression again")
		}

		if l := len(newCall.Args); arg.Position >= l {
			return callSiteArg{}, nil, nil, fmt.Errorf("arg position is %d but new call only has %d args", arg.Position, l)
		}

		newArg := newCall.Args[arg.Position]
		name, ok := newArg.Value.(*pythonast.NameExpr)
		if !ok {
			return callSiteArg{}, nil, nil, fmt.Errorf("new arg is no longer a name expression, go %T instead", newArg.Value)
		}

		if name.Ident.Literal != arg.NameSite.Original.Ident.Literal {
			return callSiteArg{}, nil, nil, fmt.Errorf("new arg name %s != original arg name %s", name.Ident.Literal, arg.NameSite.Original.Ident.Literal)
		}

		// NOTE: we set the value of the resulting call expression to be nil
		// since at inference time we do not set the value of the call expression
		// during beam search.
		// NOTE: we could update the beam search to set the value for the new call expression
		// that we insert but this is probably not worth the hassle atm.
		// a.SetResolved(kitectx.Background(), newCall, nil)

		builder := newBuilder(kitectx.Background(), a, false, true)

		builder.BuildEdges(config.Graph.EdgeSet)

		nameSite, err := builder.BuildNameSite(builder.ScopeForCall(kitectx.Background(), newCall), name)
		if err != nil {
			return callSiteArg{}, nil, nil, fmt.Errorf("error building new name site: %v", err)
		}

		if !nameSite.Scope.Contains(builder.vm.VariableFor(name)) {
			return callSiteArg{}, nil, nil, fmt.Errorf("new name %s is not in scope %s", name.Ident.Literal, nameSite.Scope.String())
		}

		if len(nameSite.Scope) < 2 {
			return callSiteArg{}, nil, nil, fmt.Errorf("new scope only has %d entries", len(nameSite.Scope))
		}

		save(
			params.Saver,
			SavedBundle{
				Label:      "munged",
				builder:    builder,
				NodeLabels: nodeLabels(builder.astNodes[nameSite.Original], "site"),
				Buffer:     newBuffer,
			},
		)

		newArgSite := callSiteArg{
			Position: arg.Position,
			Name:     arg.Name,
			NameSite: nameSite,
		}

		return newArgSite, builder, newBuffer, nil
	}()
	if err != nil {
		return nil, fmt.Errorf("error refinding call: %v", err)
	}

	contextNode := builder.UpdateForInferNameTrainTask(arg.NameSite, config.MaxHops)

	save(
		params.Saver,
		SavedBundle{
			Label:      "pruned",
			builder:    builder,
			NodeLabels: nodeLabels(contextNode, "site"),
			Buffer:     newBuffer,
		},
	)

	// check if graph is still too large after pruning
	if len(builder.nodes) > maxNumNodesTrainGraph {
		return nil, fmt.Errorf("too many nodes in graph got %d > %d", len(builder.nodes), maxNumNodesTrainGraph)
	}

	// sanity check
	var labelIdx int
	for i, v := range arg.NameSite.Scope {
		if v == arg.NameSite.Variable {
			labelIdx = i
		}
	}

	assertTrue(VariableID(labelIdx) == arg.NameSite.Variable.ID, "bad times")

	types, subtokens := patterns.Feed(arg.Name, arg.Position)

	var typeFeed traindata.SegmentedIndicesFeed
	for _, t := range types {
		typeFeed.Indices = append(typeFeed.Indices, int32(params.TypeSubtokenIndex.Index(t)))
		typeFeed.SampleIDs = append(typeFeed.SampleIDs, 0)
	}

	var subtokenFeed traindata.SegmentedIndicesFeed
	for _, st := range subtokens {
		subtokenFeed.Indices = append(subtokenFeed.Indices, int32(params.NameSubtokenIndex.Index(st)))
		subtokenFeed.SampleIDs = append(subtokenFeed.SampleIDs, 0)
	}

	egOnlyNodes := []*Node{contextNode}
	for _, cand := range arg.NameSite.Candidates {
		egOnlyNodes = append(egOnlyNodes, cand.Usage)
	}

	eg, _ := builder.ExpansionGraph(params.ModelMeta, egOnlyNodes, nil)

	return &InferNameSample{
		ContextGraph:   builder.newGraphFeed(params.ModelMeta),
		ExpansionGraph: eg,
		Name: NameModelFeed{
			PredictionNodes: []int32{int32(contextNode.ID)},
			Corrupted:       newCorruptedSegmented(params.Rand, labelIdx, len(arg.NameSite.Scope), config.NumCorrupted),
			Labels:          []VariableID{arg.NameSite.Variable.ID},
			Types:           typeFeed,
			Subtokens:       subtokenFeed,
			Names:           newNameEncoderFeedFromNameSite(builder.astNodes, arg.NameSite),
		},
	}, nil
}

type callSiteArg struct {
	Name string
	// Position in the call (e.g 0 for the first argument, 1 for the second, etc)
	Position int

	// NameSite for placing a name expression (e.g variable) for the positional argument
	NameSite nameSite
}

type callSite struct {
	Call *pythonast.CallExpr
	Args []callSiteArg
}

func reduceScopeForCall(ctx kitectx.Context, a *analysis, s scope) scope {
	return filterScopeByKinds(ctx, a, s, pythontype.ModuleKind, pythontype.FunctionKind)
}

func (b *graphBuilder) ScopeForCall(ctx kitectx.Context, node pythonast.Node) scope {
	return reduceScopeForCall(ctx, b.a, b.vm.InScope(node, true))
}

func (b *graphBuilder) BuildCallSites(ctx kitectx.Context, canonical pythonresource.Symbol, patterns *traindata.CallPatterns) []callSite {
	ctx.CheckAbort()
	var calls []callSite
	pythonast.Inspect(b.a.RAST.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		call, ok := node.(*pythonast.CallExpr)
		if !ok {
			return true
		}

		fn := b.astNodes[call.Func]
		if !fn.matchesType(canonical, true) {
			return true
		}

		if !patterns.Matches(call) {
			return true
		}

		scope := b.ScopeForCall(ctx, call)

		var args []callSiteArg
		for i, arg := range call.Args {
			var kw string
			if name, ok := arg.Name.(*pythonast.NameExpr); ok {
				kw = name.Ident.Literal
			}

			name, ok := arg.Value.(*pythonast.NameExpr)
			if !ok {
				continue
			}

			nameSite, err := b.BuildNameSite(scope, name)
			if err != nil {
				continue
			}

			variable := b.vm.VariableFor(name)
			if !nameSite.Scope.Contains(variable) {
				// this can happen because we limit the variables that we consider
				// "in scope", see variable.go and name.go
				continue
			}

			if len(nameSite.Scope) < 2 {
				continue
			}

			args = append(args, callSiteArg{
				Name:     kw,
				Position: i,
				NameSite: nameSite,
			})
		}

		if len(args) > 0 {
			calls = append(calls, callSite{
				Call: call,
				Args: args,
			})
		}

		return true
	})
	return calls
}
