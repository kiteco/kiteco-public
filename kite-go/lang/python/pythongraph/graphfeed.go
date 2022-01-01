package pythongraph

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

const (
	maxNumSubtokensPerNode = 9
	maxNumTypesPerNode     = 9
)

// ModelMeta bundles meta information for a model
type ModelMeta struct {
	NameSubtokenIndex traindata.SubtokenIndex
	TypeSubtokenIndex traindata.SubtokenIndex
	ProductionIndex   traindata.ProductionIndex
}

// EdgeSet defines a set of edge types
type EdgeSet []EdgeType

// Contains returns true if the specified edge type is in the set
func (e EdgeSet) Contains(t EdgeType) bool {
	for _, tt := range e {
		if tt == t {
			return true
		}
	}
	return false
}

// Valid returns nil if the set is valid
func (e EdgeSet) Valid() error {
	for _, t := range e {
		if err := t.Valid(); err != nil {
			return err
		}
	}
	return nil
}

// EdgeFeed ...
type EdgeFeed map[string][][2]int32

func newEdgeFeed(edges []*Edge) EdgeFeed {
	feed := make(EdgeFeed)
	for _, edge := range edges {
		key := EdgeKey(edge.Type, edge.Forward)
		feed[key] = append(feed[key], [2]int32{int32(edge.From), int32(edge.To)})
	}
	for k, s := range feed {
		sort.Slice(s, func(i, j int) bool {
			if s[i][0] == s[j][0] {
				return s[i][1] < s[j][1]
			}
			return s[i][0] < s[j][0]
		})
		feed[k] = s
	}

	return feed
}

func (e EdgeFeed) append(other EdgeFeed, nodeOffset NodeID) EdgeFeed {
	var eks []string
	for ek := range other {
		eks = append(eks, ek)
	}

	for _, ek := range eks {
		edges := e[ek]
		for _, edge := range other[ek] {
			edges = append(edges, [2]int32{
				edge[0] + int32(nodeOffset),
				edge[1] + int32(nodeOffset),
			})
		}
		e[ek] = edges
	}

	return e
}

// FeedDict ...
func (e EdgeFeed) FeedDict(prefix string) map[string]interface{} {
	fd := make(map[string]interface{})
	for key, edges := range e {
		key = path.Join(prefix, key)
		fd[key] = edges
	}
	return fd
}

// NodeFeed ...
type NodeFeed struct {
	Types     traindata.SegmentedIndicesFeed `json:"types"`
	Subtokens traindata.SegmentedIndicesFeed `json:"subtokens"`
	numNodes  int
}

// FeedDict ...
func (nf NodeFeed) FeedDict(prefix string) map[string]interface{} {
	fd := nf.Types.FeedDict(path.Join(prefix, "types"))
	for k, v := range nf.Subtokens.FeedDict(path.Join(prefix, "subtokens")) {
		fd[k] = v
	}
	return fd
}

func (nf NodeFeed) append(other NodeFeed, nodeOffset NodeID) NodeFeed {
	nf.Types = nf.Types.Append(other.Types, int32(nodeOffset), 0)
	nf.Subtokens = nf.Subtokens.Append(other.Subtokens, int32(nodeOffset), 0)
	nf.numNodes += other.numNodes
	return nf
}

func newNodeFeed(m ModelMeta, nodes []*Node) NodeFeed {
	subtokens := traindata.NewSegmentedIndicesFeed()
	types := traindata.NewSegmentedIndicesFeed()
	for i, node := range nodes {
		for _, s := range getNodeSubtokens(node, maxNumSubtokensPerNode) {
			subtokens.Indices = append(subtokens.Indices, int32(m.NameSubtokenIndex.Index(s)))
			subtokens.SampleIDs = append(subtokens.SampleIDs, int32(i))
		}

		for _, t := range getNodeTypes(node, maxNumTypesPerNode) {
			types.Indices = append(types.Indices, int32(m.TypeSubtokenIndex.Index(t)))
			types.SampleIDs = append(types.SampleIDs, int32(i))
		}
	}
	return NodeFeed{
		Types:     types,
		Subtokens: subtokens,
		numNodes:  len(nodes),
	}
}

// GraphFeedConfig defines how the graph feed is created.
type GraphFeedConfig struct {
	// EdgeSet defines the edge types for which Edges will be guaranteed to have entries.
	EdgeSet EdgeSet `json:"edge_set"`
}

// Valid returns nil if the configuration is valid
func (c GraphFeedConfig) Valid() error {
	return c.EdgeSet.Valid()
}

// EdgeKey contains type and direction info
func EdgeKey(typ EdgeType, forward bool) string {
	if forward {
		return string(typ) + "_forward"
	}
	return string(typ) + "_backward"
}

func typeFromEdgeKey(edgeKey string) (EdgeType, bool) {
	if i := strings.Index(edgeKey, "_forward"); i > -1 {
		return EdgeType(edgeKey[:i]), true
	}
	return EdgeType(strings.TrimSuffix(edgeKey, "_backward")), false
}

// GraphFeed represents a graph in a format that can directly be used by the GraphEncoder subgraph of a
// Tensorflow model.
type GraphFeed struct {
	// NodeSubtokens contains, for each node, a slice of subtokens representing the node's literal.
	NodeSubtokens traindata.SegmentedIndicesFeed `json:"node_subtokens"`
	// NodeTypes contains, for each node, a slice of ints representing the node's types.
	NodeTypes traindata.SegmentedIndicesFeed `json:"node_types"`

	// Edges are in the format of: edge_key => list of edges in [from node ID, to node ID]
	// ...where edge_key is <edge type>_<edge direction>
	Edges EdgeFeed `json:"edges"`

	numNodes int
}

func (b *graphBuilder) newGraphFeed(m ModelMeta) GraphFeed {
	nf := newNodeFeed(m, b.nodes)
	return GraphFeed{
		NodeSubtokens: nf.Subtokens,
		NodeTypes:     nf.Types,
		Edges:         newEdgeFeed(b.edges),
		numNodes:      len(b.nodes),
	}
}

func (g GraphFeed) append(other GraphFeed, offset NodeID) GraphFeed {
	g.NodeSubtokens = g.NodeSubtokens.Append(other.NodeSubtokens, int32(offset), 0)
	g.NodeTypes = g.NodeTypes.Append(other.NodeTypes, int32(offset), 0)

	g.Edges = g.Edges.append(other.Edges, offset)

	g.numNodes += other.numNodes

	return g
}

// NumNodes in the graph feed
func (g GraphFeed) NumNodes() int {
	return g.numNodes
}

// FeedDict returns the feeds that can be fed directly into the model.
func (g GraphFeed) FeedDict(prefix string) map[string]interface{} {
	p := func(parts ...string) string {
		if prefix != "" {
			parts = append([]string{prefix, "placeholders"}, parts...)
			return path.Join(parts...)
		}
		parts = append([]string{"placeholders"}, parts...)
		return path.Join(parts...)
	}

	feeds := g.NodeTypes.FeedDict(p("nodes", "types"))
	for k, v := range g.NodeSubtokens.FeedDict(p("nodes", "subtokens")) {
		feeds[k] = v
	}

	for k, v := range g.Edges.FeedDict(p("edges")) {
		feeds[k] = v
	}

	return feeds
}

// Valid returns nil if the graph feed is valid
func (g GraphFeed) Valid(config GraphFeedConfig) error {
	numNodes := g.NumNodes()
	if is := intSet(g.NodeSubtokens.SampleIDs); len(is) != numNodes {
		return fmt.Errorf("num subtokens %d != num nodes %d", len(is), numNodes)
	}

	if is := intSet(g.NodeTypes.SampleIDs); len(is) != numNodes {
		return fmt.Errorf("num node types %d != num nodes %d", len(is), numNodes)
	}

	nodeStr := func(ni int) string {
		var ts []string
		for i, nid := range g.NodeTypes.SampleIDs {
			if nid == int32(ni) {
				ts = append(ts, fmt.Sprintf("%d", g.NodeTypes.Indices[i]))
			}
		}

		if len(ts) == 0 {
			ts = append(ts, "CANNOT FIND NODE TYPES")
		}

		var sts []string
		for i, nid := range g.NodeSubtokens.SampleIDs {
			if nid == int32(ni) {
				sts = append(ts, fmt.Sprintf("%d", g.NodeSubtokens.Indices[i]))
			}
		}

		if len(sts) == 0 {
			sts = append(ts, "CANNOT FIND NODE SUBTOKENS")
		}

		return fmt.Sprintf("{ID: %d, Types: %s, Subtokens: %s}",
			ni, strings.Join(ts, ":"), strings.Join(sts, ":"),
		)
	}

	validNode := func(i int32) bool {
		return i >= 0 && i < int32(numNodes)
	}

	edgeStr := func(key string, edge [2]int32) string {
		fs, ts := "NA", "NA"
		if validNode(edge[0]) {
			fs = nodeStr(int(edge[0]))
		}

		if validNode(edge[1]) {
			ts = nodeStr(int(edge[1]))
		}

		return fmt.Sprintf("edge %s: %d (%s) -> %d (%s)", key, edge[0], fs, edge[1], ts)
	}

	for key, edges := range g.Edges {
		for _, edge := range edges {
			from, to := edge[0], edge[1]
			if !validNode(from) {
				return fmt.Errorf("edge.from not in [0,%d-1]: %s", numNodes, edgeStr(key, edge))
			}

			if !validNode(to) {
				return fmt.Errorf("edge.to not in [0,%d-1]: %s", numNodes, edgeStr(key, edge))
			}
		}
	}

	for i := 0; i < numNodes; i++ {
		var foundType bool
		for _, nid := range g.NodeTypes.SampleIDs {
			if nid == int32(i) {
				foundType = true
				break
			}
		}

		if !foundType {
			return fmt.Errorf("no type found for node %s", nodeStr(i))
		}

		var foundSubtok bool
		for _, nid := range g.NodeSubtokens.SampleIDs {
			if nid == int32(i) {
				foundSubtok = true
				break
			}
		}

		if !foundSubtok {
			return fmt.Errorf("no subtok found for node %s", nodeStr(i))
		}
	}

	return nil
}

var nameExprTypeStr = traindata.ASTNodeType(&pythonast.NameExpr{})

func getNodeSubtokens(node *Node, numSubtokens int) []string {
	var sts []string
	switch {
	case node.Attrs.Literal != "":
		if node.Attrs.ASTNodeType == nameExprTypeStr {
			sts = traindata.SplitNameLiteral(node.Attrs.Literal)
			break
		}
		sts = []string{node.Attrs.Literal}
	case node.Attrs.ASTNodeType != "":
		sts = []string{node.Attrs.ASTNodeType}
	default:
		panic(fmt.Sprintf("unhandled case getting node subtokens (num subtokens %d): %v", numSubtokens, node))
	}
	if len(sts) > numSubtokens {
		sts = sts[:numSubtokens]
	}
	if len(sts) == 0 {
		panic("bad times no node subtokens")
	}
	return sts
}

func getNodeTypes(node *Node, numTypes int) []string {
	ts := make([]string, 0, numTypes)
	tempMap := make(map[string]struct{})
	for _, t := range node.Attrs.Types {
		for _, tt := range TypeToSubtokens(t) {
			if _, ok := tempMap[tt]; !ok {
				ts = append(ts, tt)
			}
			tempMap[tt] = struct{}{}
		}
	}
	sort.Slice(ts, func(i, j int) bool {
		return ts[i] < ts[j]
	})
	if len(ts) > numTypes {
		ts = ts[:numTypes]
	}
	if len(ts) == 0 {
		panic("bad times no node types")
	}
	return ts
}

func intSet(is []int32) map[int32]bool {
	s := make(map[int32]bool)
	for _, i := range is {
		s[i] = true
	}
	return s
}

// TypeToSubtokens converts a type name to a sequence of subtokens
func TypeToSubtokens(typeName string) []string {
	// don't split special type markers
	if traindata.IsSpecialToken(typeName) {
		return []string{typeName}
	}

	p := pythonimports.NewDottedPath(typeName)
	// TODO: do we want to use the full path?
	var symName string
	switch {
	case p.Last() == traindata.ReturnValueTail:
		symName = p.Predecessor().Last() + "_" + traindata.ReturnValueTail
	case p.Last() == traindata.InstanceTail:
		symName = p.Predecessor().Last() + "_" + traindata.InstanceTail
	default:
		symName = p.Last()
	}
	return traindata.SplitNameLiteral(symName)
}
