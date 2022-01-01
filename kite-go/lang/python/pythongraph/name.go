package pythongraph

import (
	"fmt"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// nameTrainSite is a location within a graph to predict a name expression
type nameSite struct {
	// Original ast node at the site
	Original *pythonast.NameExpr

	// Variable associated with the original ast node at the site, may be nil
	Variable *variable

	// Candidates to predict for the site, one for each variable in scope
	// len(Candidates) == len(Scope)
	Candidates []candidate

	// Scope contains the variables in scope at the name site
	// len(Candidates) == len(Scope)
	Scope scope
}

func (b *graphBuilder) BuildNameSite(s scope, original *pythonast.NameExpr) (nameSite, error) {
	// may be nil
	variable := b.vm.VariableFor(original)

	var validScope scope
	var validCands []candidate
	for i, cand := range b.GetUsageCandidates(original, variable, s) {
		// TODO(Juan): this is to prevent orphaned nodes in the
		// resulting graph, these can come about because of limitations
		// to the flow set calculation or to limitations of the symbol set
		// or mismatches between the flow set calculation and the name lookup
		// process. e.g consider
		// if something():
		//   x = 1
		//   plot(x)
		// else:
		//   y = 1
		// we should not consider y as a candidate for the call
		// to plot but due to limitations in the way that
		// we calculate what symbols are in scope we believe that y is also in scope
		// at the call to plot.
		// I suspect that to correct this we will either:
		// 1) need to calculate scopes manually for the graph
		// 2) track this information in analysis.
		// NOTE: if we track what position a symbol was first and last bound
		// we can do a pretty good job approximating this.
		if len(cand.Edges) > 0 {
			validScope = append(validScope, s[i])
			validCands = append(validCands, cand)
		}
	}

	if len(validCands) == 0 {
		return nameSite{}, fmt.Errorf("no valid candidates at %v in %v", original, s)
	}

	return nameSite{
		Original:   original,
		Variable:   variable,
		Candidates: validCands,
		Scope:      validScope,
	}, nil
}

func (b *graphBuilder) UpdateForInferNameTrainTask(site nameSite, maxHops int) *Node {
	b.vm.ReduceTo(b.a.RAST.Root, site.Scope)

	// re sort the candidates so that they match the variable ordering
	sort.Slice(site.Candidates, func(i, j int) bool {
		ci, cj := site.Candidates[i], site.Candidates[j]
		return ci.Variable.ID < cj.Variable.ID
	})

	// set up the context node and setup the graph
	// for the prediction task
	// update the context node
	contextNode := b.astNodes[site.Original]
	contextNode.Attrs.Literal = traindata.InferNameMarker
	contextNode.Attrs.Types = []string{traindata.InferNameMarker}

	// disconnect context node's data flow edges
	edges := make([]*Edge, 0, len(b.edges))
	for _, edge := range b.edges {
		if edge.Type == DataFlow {
			if edge.from == contextNode || edge.to == contextNode {
				continue
			}
		}
		edges = append(edges, edge)
	}

	// update graph with new edges
	b.edges = edges

	// keep set for graph pruning
	keep := make(nodeSet)
	keep[contextNode] = true

	// connect usage nodes to the graph and add to the keep set
	for _, cand := range site.Candidates {
		keep[cand.Usage] = true

		// add usage node for candidate variable to the actual graph
		cand.Usage.ID = NodeID(len(b.nodes))
		b.nodes = append(b.nodes, cand.Usage)
		// add semantic edges connecting usage node to the rest of the graph
		for _, protoEdge := range cand.Edges {
			b.edges = append(b.edges, makeEdges(protoEdge.From, protoEdge.To, protoEdge.Type)...)
		}
	}

	// prune graph
	if maxHops > 0 {
		b.Prune(kitectx.TODO(), keep, maxHops)
	}

	return contextNode
}
