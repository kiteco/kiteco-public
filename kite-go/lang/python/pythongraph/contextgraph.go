package pythongraph

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// ContextGraph encapsulates a context graph that has been propagated
type ContextGraph struct {
	nodeEmbeddingDim int
	finalNodeStates  [][]float32

	incoming nodeToNeighbors
	outgoing nodeToNeighbors

	builder *graphBuilder

	// TODO: there is probably a better spot for this...
	site pythonast.Node
}

// ContextGraphConfig ...
type ContextGraphConfig struct {
	Graph     GraphFeedConfig
	MaxHops   int
	Propagate bool
}

// ContextGraphInputs ...
type ContextGraphInputs struct {
	ModelMeta ModelMeta
	Model     *tensorflow.Model
	In        Inputs

	// we use this for pruning if it is provided
	Site pythonast.Node
}

// NewContextGraph ...
func NewContextGraph(ctx kitectx.Context, config ContextGraphConfig, in ContextGraphInputs) (*ContextGraph, error) {
	a := newAnalysis(in.In.RM, in.In.Words, in.In.RAST)
	ctx.CheckAbort()

	return newContextGraph(ctx, config, in, a)
}

func newContextGraph(ctx kitectx.Context, config ContextGraphConfig, in ContextGraphInputs, a *analysis) (*ContextGraph, error) {
	builder := newBuilder(ctx, a, false, true)
	ctx.CheckAbort()

	builder.BuildEdges(config.Graph.EdgeSet)
	ctx.CheckAbort()

	if !pythonast.IsNil(in.Site) && config.MaxHops > 0 {
		// TODO: this is just for backwards compatibility
		scope := builder.ScopeForCall(ctx, in.Site)
		if len(scope) == 0 {
			return nil, fmt.Errorf("no variables in scope")
		}
		builder.vm.ReduceTo(in.In.RAST.Root, scope)

		keep := nodeSet{
			builder.astNodes[in.Site]: true,
		}

		for _, n := range builder.ContextTokens(in.Site) {
			keep[n] = true
		}

		for _, v := range builder.vm.Variables {
			for _, ref := range v.Refs.Names() {
				rn := builder.astNodes[ref]
				keep[rn] = true
			}
		}

		builder.Prune(ctx, keep, config.MaxHops)
		ctx.CheckAbort()
	}

	var finalNodeStates [][]float32
	var nodeEmbeddingDim int
	if config.Propagate {
		feed := builder.newGraphFeed(in.ModelMeta).FeedDict("context_graph")

		fetchOp := "context_graph/final_node_states"

		res, err := in.Model.Run(feed, []string{fetchOp})
		if err != nil {
			return nil, fmt.Errorf("error propagating context graph: %v", err)
		}
		ctx.CheckAbort()

		finalNodeStates = res[fetchOp].([][]float32)
		if len(finalNodeStates) == 0 {
			// TODO: we should be able to support this case we
			// just need a way to set the node embedding dimension
			return nil, fmt.Errorf("no context graph nodes")
		}
		nodeEmbeddingDim = len(finalNodeStates[0])
	}

	incoming := make(nodeToNeighbors, len(builder.nodes))
	outgoing := make(nodeToNeighbors, len(builder.nodes))
	for _, e := range builder.edges {
		// only take forward edges to ensure that the expansion graph
		// has a well defined topological order
		if !e.Forward {
			continue
		}
		outgoing.addNeighbor(e.from, e.to, e.Type, NeighborData{})
		incoming.addNeighbor(e.to, e.from, e.Type, NeighborData{})
	}

	return &ContextGraph{
		nodeEmbeddingDim: nodeEmbeddingDim,
		incoming:         incoming,
		outgoing:         outgoing,
		builder:          builder,
		finalNodeStates:  finalNodeStates,
		site:             in.Site,
	}, nil
}
