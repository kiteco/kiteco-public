package pythongraph

import (
	"fmt"
	"path"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// ExpansionGraphBaseFeed ...
type ExpansionGraphBaseFeed struct {
	Edges              EdgeFeed `json:"edges"`
	ContextToExpansion []int32  `json:"context_to_expansion"`
	LookupNodes        NodeFeed `json:"lookup_nodes"`
	LookupToExpansion  []int32  `json:"lookup_to_expansion"`
}

// FeedDict ...
func (e ExpansionGraphBaseFeed) FeedDict(prefix string) map[string]interface{} {
	path := func(n string) string {
		if n == "" {
			return path.Join(prefix, "placeholders")
		}
		return path.Join(prefix, "placeholders", n)
	}

	fd := map[string]interface{}{
		path("context_to_expansion"): e.ContextToExpansion,
		path("lookup_to_expansion"):  e.LookupToExpansion,
	}

	for k, v := range e.Edges.FeedDict(path("")) {
		fd[k] = v
	}

	for k, v := range e.LookupNodes.FeedDict(path("")) {
		fd[k] = v
	}

	return fd
}

// ExpansionGraphTrainFeed ...
type ExpansionGraphTrainFeed struct {
	ExpansionGraphBaseFeed

	ContextGraphNodes []NodeID `json:"context_graph_nodes"`

	numNodes int
}

// NumNodes ...
func (egf ExpansionGraphTrainFeed) NumNodes() int {
	return egf.numNodes
}

// NOTE:
//   - we have a separate offset for the context graph nodes (`contextOffset`) and the expansion graph nodes (`expansionOffset`) since they
//     are separate graphs with separate node indices.
func (egf ExpansionGraphTrainFeed) append(other ExpansionGraphTrainFeed, contextOffset, expansionOffset NodeID) ExpansionGraphTrainFeed {
	egf.Edges = egf.Edges.append(other.Edges, expansionOffset)

	for i, ocg := range other.ContextGraphNodes {
		egf.ContextGraphNodes = append(egf.ContextGraphNodes, ocg+contextOffset)
		egf.ContextToExpansion = append(egf.ContextToExpansion, int32(expansionOffset)+other.ContextToExpansion[i])
	}

	for _, oid := range other.LookupToExpansion {
		egf.LookupToExpansion = append(egf.LookupToExpansion, int32(expansionOffset)+oid)
	}

	// TODO: this is kind of nasty, but basically we need these nodes to be indexed from 0 to num_total_lookup_nodes
	// so that when we do the segment ops to compute the node representations we get a resulting tensor that has
	// num_total_lookup_nodes. If we do not do this then the resulting tensor that represents the lookup node embeddings
	// with have num_total_expansion_graph_nodes... which is not what we want.
	var lookupNodeOffset NodeID
	if len(egf.LookupNodes.Subtokens.SampleIDs) > 0 {
		lookupNodeOffset = NodeID(max(egf.LookupNodes.Subtokens.SampleIDs...) + 1)
	}

	egf.LookupNodes = egf.LookupNodes.append(other.LookupNodes, lookupNodeOffset)

	egf.numNodes += other.numNodes
	return egf
}

// TODO: gross
// NOTE:
//  - the underlying graph is modified, all nodes in egOnlyNodes are removed and added to the context graph
//  - the nodes in egOnlyNodes are modified
//  - edges coming into a node in egOnlyNodes are the only edges added to the expansion graph
func (g *graphBuilder) ExpansionGraph(m ModelMeta, egOnlyNodes []*Node, cgNodes []*Node) (ExpansionGraphTrainFeed, nodeIDFn) {
	// sanity check
	for _, eg := range egOnlyNodes {
		for _, cg := range cgNodes {
			if eg == cg {
				panic("bad times, expansion graph only node in context graph")
			}
		}
	}

	for _, cg := range cgNodes {
		for _, eg := range egOnlyNodes {
			if eg == cg {
				panic("bad times, context graph node in expansion graph only")
			}
		}
	}

	isEGOnlyNode := newNodeSet(egOnlyNodes)
	isCGNode := newNodeSet(cgNodes)

	// expansion graph edges are just the edges coming into the egOnlyNodes
	// the source is either another egOnlyNode or a context graph node
	var egEdges []*Edge
	for _, edge := range g.edges {
		if edge.Forward && isEGOnlyNode[edge.to] {
			egEdges = append(egEdges, edge)
		}
	}

	if len(egEdges) == 0 {
		panic("no edges in expansion graph")
	}

	// context graph is everything except the egOnlyNodes
	keep := make(nodeSet)
	for _, n := range g.nodes {
		if isEGOnlyNode[n] {
			continue
		}
		keep[n] = true
	}
	g.RestrictTo(kitectx.Background(), keep)

	// sanity check
	for _, e := range egEdges {
		for _, ee := range g.edges {
			if e == ee {
				panic("bad times expansion graph edge still in context graph")
			}
		}
	}

	// add all the context graph nodes

	// context graph nodes to add as specified by the caller
	var cgNodeIDs []NodeID
	var contextToExpansion []int32
	cgToEG := make(map[*Node]NodeID)
	for _, cg := range cgNodes {
		// id in the expansion graph for the context node
		egID := NodeID(len(cgNodeIDs))
		cgNodeIDs = append(cgNodeIDs, cg.ID)
		cgToEG[cg] = egID

		contextToExpansion = append(contextToExpansion, int32(egID))
	}

	// now add the context graph nodes that have an edge into the egOnlyNode set
	for _, e := range egEdges {
		if !isEGOnlyNode[e.from] && !isCGNode[e.from] {
			// the from node is NOT an egOnlyNode and has
			// NOT already been added to the context nodes, so add it
			// id in the expansion graph for the context node
			egID := NodeID(len(cgNodeIDs))
			cgNodeIDs = append(cgNodeIDs, e.from.ID)
			cgToEG[e.from] = egID

			contextToExpansion = append(contextToExpansion, int32(egID))
		}
	}

	// add all of the expansion graph only nodes
	// after the context graph nodes
	// and update their IDs
	var lookupToExpansion []int32
	egToEG := make(map[*Node]NodeID)
	for i, eg := range egOnlyNodes {
		egID := NodeID(len(cgNodeIDs) + i)
		egToEG[eg] = egID
		eg.ID = egID
		lookupToExpansion = append(lookupToExpansion, int32(egID))
	}

	mustNewID := func(n *Node) NodeID {
		if id, ok := egToEG[n]; ok {
			return id
		}
		if id, ok := cgToEG[n]; ok {
			return id
		}
		panic(fmt.Sprintf("bad times, no new node id for %v", n))
	}

	// fix the edges
	for _, e := range egEdges {
		e.From = mustNewID(e.from)
		e.To = mustNewID(e.to)
	}

	// sanity checks
	for _, n := range g.nodes {
		if isEGOnlyNode[n] {
			panic("bad times, context graph node is in the expansion graph")
		}
	}

	for _, cgNode := range cgNodeIDs {
		var ok bool
		for _, n := range g.nodes {
			if n.ID == cgNode {
				ok = true
				break
			}
		}
		if !ok {
			panic("bad times, unable to find context graph node")
		}
	}

	if len(cgNodeIDs) != len(contextToExpansion) {
		panic("bad times len(cgNodeIDs) != len(contextToExpansion)")
	}

	nEGNodes := NodeID(len(cgNodeIDs) + len(egOnlyNodes))
	for _, e := range egEdges {
		if e.From >= nEGNodes || e.To >= nEGNodes {
			panic("bad times, invalid expansion graph edge")
		}
	}

	return ExpansionGraphTrainFeed{
		ExpansionGraphBaseFeed: ExpansionGraphBaseFeed{
			Edges:              newEdgeFeed(egEdges),
			ContextToExpansion: contextToExpansion,
			LookupNodes:        newNodeFeed(m, egOnlyNodes),
			LookupToExpansion:  lookupToExpansion,
		},
		ContextGraphNodes: cgNodeIDs,
		numNodes:          int(nEGNodes),
	}, mustNewID
}

// ExpansionGraphTestFeed ...
type ExpansionGraphTestFeed struct {
	ExpansionGraphBaseFeed
	ContextNodeEmbeddings [][]float32
}

// FeedDict ...
func (e ExpansionGraphTestFeed) FeedDict(prefix string) map[string]interface{} {
	fd := e.ExpansionGraphBaseFeed.FeedDict(prefix)

	name := path.Join(prefix, "test_placeholders", "context_node_embeddings")
	fd[name] = e.ContextNodeEmbeddings

	return fd
}

// ExpansionGraphFeedBuilder ...
// The feed sent into tensorflow is typically a subgraph of the full context graph + expansion graph,
// the feed builder manages this subgraph and can translate between a node and its sub graph id and vice versa.
type ExpansionGraphFeedBuilder struct {
	base         ExpansionGraphBaseFeed
	nodesToEmbed []*Node

	// maps from an id in the subgraph back to the node
	subgraphIDToNode map[NodeID]*Node

	// maps from a node pointer to its id in the subgraph
	nodeToSubgraphID nodeIDFn

	// maps from node to embedding for that node
	embedNode func(*Node) []float32
}

// The actual graph we build and send into tensorflow is a subgraph of the
// full expansion + context graph so we need to renumber everything we send into tensorflow to
// make sure we just operate on this subgraph and that everything is consistent.
// The other option is to always include the full context graph + expansion graph
// in each prediction but this means we might spend alot of CPU copying nodes
// to and from tensorflow that we do not need.
// Parameters:
// - `lookup` nodes must always be in the expansion graph, these are the nodes
//   that we will "lookup" embeddings for and then propagate to.
// - `contextNodes` can be either in the expansion graph or in the context graph,
//   for the purposes of inference all nodes that are not lookup nodes
//   are condsidered context graph nodes. TODO: better names?
// Return:
// - the feed builder
// - a function mapping from to in the selected subgraph to the embedding for that node,
//   used for building the test feed, see `ExpansionGraphFeedBuilder.TestFeed`
// - a function mapping from node in the selected subgraph to the ID for the node
//   in the selected subgraph.
func newExpansionGraphFeedBuilder(eg *ExpansionGraph, lookup []*Node, contextNodes ...[]*Node) ExpansionGraphFeedBuilder {
	nodeToID := make(map[*Node]NodeID)
	mustNodeID := mustNodeIDFunc(nodeToID)

	idToNode := make(map[NodeID]*Node)

	var allContextNodes []*Node
	var contextNodeToID []int32
	addContextNode := func(n *Node) {
		if _, ok := nodeToID[n]; ok {
			return
		}

		// this node could either be in the expansion graph or in the context graph,
		// but either way nodes that are not being looked up are considered "context nodes"
		egID := len(nodeToID)
		nodeToID[n] = NodeID(egID)
		contextNodeToID = append(contextNodeToID, int32(egID))
		allContextNodes = append(allContextNodes, n)

		idToNode[NodeID(egID)] = n
	}

	embedNode := func(n *Node) []float32 {
		if eg.state.isEgNode(n) {
			return eg.state.egNodeStates[n.ID]
		}
		return eg.state.cgNodeStates[n.ID]
	}

	var lookupNodeToID []int32
	edgeFeed := make(EdgeFeed)

	// start by getting nodes that have an outgoing edge to a lookup node
	for _, l := range lookup {
		if !eg.state.isEgNode(l) {
			// bad times, we only support lookup nodes that
			// are in the expansion graph, we are NOT allowed to
			// lookup context graph nodes
			panic(fmt.Sprintf("%v is a context graph node, cannot be a lookup node", l))
		} else if _, ok := nodeToID[l]; ok {
			// bad times, this means that a lookup node
			// was also a neighbor for one of the other lookup nodes
			panic(fmt.Sprintf("%v is a neighbor of another lookup node", l))
		}

		// generate a new subgraph id for the lookup node,
		// lookup nodes get a separate ID slice
		egID := len(nodeToID)
		nodeToID[l] = NodeID(egID)
		lookupNodeToID = append(lookupNodeToID, int32(egID))
		idToNode[NodeID(egID)] = l

		// get edges coming into the lookup node
		for _, in := range eg.state.egIncoming[l] {
			addContextNode(in.Node)

			// at this point we know that the new node ids for the from and to nodes are set
			// so we can safely add the edge to the final feed
			edgeKey := EdgeKey(in.Type, true)
			edgeFeed[edgeKey] = append(edgeFeed[edgeKey], [2]int32{
				int32(mustNodeID(in.Node)),
				int32(mustNodeID(l)),
			})
		}
	}

	// next add the context nodes requested by the client
	for _, cns := range contextNodes {
		for _, cn := range cns {
			addContextNode(cn)
		}
	}

	return ExpansionGraphFeedBuilder{
		base: ExpansionGraphBaseFeed{
			Edges:              edgeFeed,
			ContextToExpansion: contextNodeToID,
			LookupNodes:        newNodeFeed(eg.meta.modelMeta, lookup),
			LookupToExpansion:  lookupNodeToID,
		},
		nodesToEmbed:     allContextNodes,
		subgraphIDToNode: idToNode,
		nodeToSubgraphID: mustNodeID,
		embedNode:        embedNode,
	}
}

// NumSubgraphNodes ...
func (e ExpansionGraphFeedBuilder) NumSubgraphNodes() int {
	return len(e.subgraphIDToNode)
}

// SubgraphID for the provided node
func (e ExpansionGraphFeedBuilder) SubgraphID(n *Node) NodeID {
	return e.nodeToSubgraphID(n)
}

// SubgraphNode for the provided id
func (e ExpansionGraphFeedBuilder) SubgraphNode(id NodeID) *Node {
	n, ok := e.subgraphIDToNode[id]
	if !ok {
		panic(fmt.Sprintf("unable to find sugraph node for id %v", id))
	}
	return n
}

// TestFeed ...
func (e ExpansionGraphFeedBuilder) TestFeed() ExpansionGraphTestFeed {
	nodeEmbeddings := make([][]float32, 0, len(e.nodesToEmbed))
	for _, n := range e.nodesToEmbed {
		nodeEmbeddings = append(nodeEmbeddings, e.embedNode(n))
	}
	return ExpansionGraphTestFeed{
		ExpansionGraphBaseFeed: e.base,
		ContextNodeEmbeddings:  nodeEmbeddings,
	}
}

// SavedGraph for the feed
func (e ExpansionGraphFeedBuilder) SavedGraph() *SavedGraph {
	var nodes []*SavedNode
	for i := NodeID(0); i < NodeID(e.NumSubgraphNodes()); i++ {
		// NOTE: we need to make a deep copy of the node and then update
		// the id so that we can get the edges right in the insight builder
		// since the edges are for the subgraph while the original node
		// could be in the full expansion grph or in the context graph
		orig := e.SubgraphNode(i)
		s := &SavedNode{
			Node:  orig.deepCopy(),
			Level: -1,
		}
		s.Node.ID = i
		nodes = append(nodes, s)
	}

	var savedEdges []*SavedEdge
	for k, edges := range e.base.Edges {
		t, forward := typeFromEdgeKey(k)
		for _, edge := range edges {
			savedEdges = append(savedEdges, &SavedEdge{
				From:    nodes[edge[0]],
				To:      nodes[edge[1]],
				Type:    t,
				Forward: forward,
			})
		}
	}

	return &SavedGraph{
		Nodes: nodes,
		Edges: savedEdges,
	}
}
