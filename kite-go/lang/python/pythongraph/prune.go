package pythongraph

import (
	"sort"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func (b *graphBuilder) Prune(ctx kitectx.Context, keep nodeSet, hops int) {
	ctx.CheckAbort()

	keep = b.Connected(ctx, keep, hops)

	b.RestrictTo(ctx, keep)
}

func (b *graphBuilder) RestrictTo(ctx kitectx.Context, keep nodeSet) {
	nodes := make([]*Node, 0, len(keep))
	for node := range keep {
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	for i, n := range nodes {
		n.ID = NodeID(i)
		for neighbor := range n.outgoing {
			if !keep[neighbor] {
				delete(n.outgoing, neighbor)
			}
		}
	}

	var edges []*Edge
	for _, edge := range b.edges {
		if keep[edge.from] && keep[edge.to] {
			edge.From = edge.from.ID
			edge.To = edge.to.ID
			edges = append(edges, edge)
		}
	}

	// update the variable manager
	for _, v := range b.vm.Variables {
		newRefs := newNameSet()
		for name, order := range v.Refs.set {
			if node := b.astNodes[name]; keep[node] {
				newRefs.Add(name, order)
			}
		}

		v.Refs = newRefs
	}

	b.nodes = nodes
	b.edges = edges

}

// `to` is NOT modified
func (b *graphBuilder) Connected(ctx kitectx.Context, to nodeSet, hops int) nodeSet {
	ctx.CheckAbort()
	connected := make(nodeSet)
	for n := range to {
		// TODO: this can be exponentially inefficient for densely connected
		// graphs in which hops is on the order of the width of the graph.
		visit(ctx, n, connected, hops)
	}

	return connected
}

func visit(ctx kitectx.Context, n *Node, connected nodeSet, hops int) {
	ctx.CheckAbort()

	connected[n] = true
	if hops == 0 {
		return
	}

	for neighbor := range n.outgoing {
		visit(ctx, neighbor, connected, hops-1)
	}
}
