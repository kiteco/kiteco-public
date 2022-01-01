package pythongraph

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
)

type protoEdge struct {
	From *Node
	To   *Node
	Type EdgeType
}

type candidate struct {
	// Variable associated with the candidate, e.g we are speculatively placing this variable
	Variable *variable
	// Usage node for the candidate, may not yet be added to the graph
	Usage *Node
	// Edges that need to be added to the graph for the candidate (data flow only)
	Edges []protoEdge
}

// GetUsageCandidates gets the candidate node and edges that need to be added to the graph if the
// NameExpr `name` were associated with one of the candidate variables.
//
// `actualVar` -- actual variable associated with the NameExpr `name` that appeared in the ast, or nil
// if the NameExpr `name` is not yet associated with a variable.
//
// `candidates` -- candidate variables to associate the NameExpr `name` with, may include `actualVar`.
func (b *graphBuilder) GetUsageCandidates(name *pythonast.NameExpr, actualVar *variable, candVars []*variable) []candidate {
	cands := make([]candidate, 0, len(candVars))
	for _, cv := range candVars {
		cands = append(cands, b.GetUsageCandidate(name, actualVar, cv))
	}
	return cands
}

// GetUsageCandidate gets the candidate node and edges that need to be added to the graph if the
// NameExpr `name` were associated with the canidate variable `candVar`.
//
// `actualVar` -- actual variable associated with the NameExpr `name` that appeared in the ast, or nil
// if the NameExpr `name` is not yet associated with a variable.
//
// `candVar` -- candidate variable to associate the NameExpr `name` with, may be equal to `actualVar`.
func (b *graphBuilder) GetUsageCandidate(name *pythonast.NameExpr, actualVar *variable, candVar *variable) candidate {
	var flow nameFlowGraph
	if candVar != actualVar {
		// temporarily add original name expression to name set
		// for candidate variable and recalculate data flow edges
		// for candidate variable including the new name node
		ns := candVar.Refs

		// we do not use the order during the flow graph calculation so we can ignore it
		ns.Add(name, -1)

		flow = b.forwardFlowGraph(ns)

		// remove original name expression
		ns.Delete(name)
	} else {
		// this is the actual name expression at the location, just
		// get the flow graph
		flow = b.forwardFlowGraph(actualVar.Refs)
	}

	// create dummy usage node for candidate variable
	// use a node id of -1 to signal that we have not added the node to the graph yet.
	// all attributes for nodes belonging to the same variable are identical
	// so we can use the attributes from the origin node
	usageNode := &Node{
		ID:       NodeID(-1),
		Type:     VariableUsageNode,
		Attrs:    b.astNodes[candVar.Origin].Attrs,
		outgoing: make(nodeSet),
	}

	cand := candidate{
		Variable: candVar,
		Usage:    usageNode,
	}

	// connect dummy usage node to real nodes for the rest of the variable
	for flowsTo := range flow[name].Set() {
		dest := b.astNodes[flowsTo]
		if flowsTo == name {
			// do not allow self loops for usage nodes
			continue
		}

		cand.Edges = append(cand.Edges, protoEdge{
			From: usageNode,
			To:   dest,
			Type: DataFlow,
		})
	}

	// TODO: kind of expensive, we iterate over all
	// nodes in the flow set and check if they contain
	// the target name as a destination
	for flowsFrom, ns := range flow {
		if !ns.Contains(name) {
			continue
		}

		src := b.astNodes[flowsFrom]
		if flowsFrom == name {
			// do not allow self loops for usage nodes
			continue
		}

		cand.Edges = append(cand.Edges, protoEdge{
			From: src,
			To:   usageNode,
			Type: DataFlow,
		})
	}

	return cand
}
