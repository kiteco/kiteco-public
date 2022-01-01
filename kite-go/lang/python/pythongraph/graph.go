package pythongraph

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// NodeID for nodes in the graph
type NodeID int

// VariableID for a variable in the graph
type VariableID int

// EdgeType for edges in the graph
type EdgeType string

const (
	// Back bone edges

	// ASTChild edge connects ast nodes
	ASTChild EdgeType = "ast_child"
	// NextToken connects nodes that appear in sequence
	NextToken EdgeType = "next_token"

	// ASTChildAttrValue connects an attribute expression to the node associated with Value of the attribute expression
	ASTChildAttrValue = "ast_child_attr_value"

	// ASTChildArgValue connects an argument node to the node associated with the Value of the Argument
	ASTChildArgValue = "ast_child_arg_value"

	// ASTChildAssignRHS connects the rhs of an assignment statement to the parent assignment statement node
	ASTChildAssignRHS = "ast_child_assign_rhs"

	// Lexical edges

	// LastLexicalUse of a particular variable
	LastLexicalUse EdgeType = "last_lexical_use"

	// Semantic edges

	// ComputedFrom connects a variable node to the other variables that were used to compute it
	ComputedFrom EdgeType = "computed_from"

	// LastRead connects a variable node to one of the last places that it was read from
	LastRead EdgeType = "last_read"

	// LastWrite connects a variable node to one of the last places it was written to
	LastWrite EdgeType = "last_write"

	// DataFlow connects a variable node to one of the next places that it will be written to or read from
	DataFlow EdgeType = "data_flow"

	// ReturnValueOf connects a variable node to the call expression that it was returned from
	ReturnValueOf = "return_value_of"

	// ScopeEdge connects name expression nodes for the variables in scope to the
	// scope node and the scope node to the prediction site
	// TODO: we should rename these to "variable" edges or "reference edges".
	ScopeEdge EdgeType = "scope"
)

// Valid returns nil if the edge is valid
func (e EdgeType) Valid() error {
	switch e {
	case ASTChild, NextToken, ASTChildAttrValue, ASTChildArgValue,
		ASTChildAssignRHS, LastLexicalUse, ComputedFrom,
		LastRead, LastWrite, DataFlow, ReturnValueOf, ScopeEdge:
		return nil
	default:
		return fmt.Errorf("invalid edge type %s", string(e))
	}
}

// Edge in the graph
type Edge struct {
	From    NodeID   `json:"from"`
	To      NodeID   `json:"to"`
	Type    EdgeType `json:"type"`
	Forward bool     `json:"forward"`

	from *Node
	to   *Node
}

// NodeType for nodes in the graph
type NodeType string

const (
	// ASTInternalNode in the graph
	ASTInternalNode NodeType = "ast_internal_node"
	// ASTTerminalNode in the graph
	ASTTerminalNode NodeType = "ast_terminal_node"
	// VariableUsageNode marks a dummy node used to
	// compute the usage representation for a candidate variable
	VariableUsageNode NodeType = "variable_usage_node"
	// ScopeNode marks a dummy node associated with each variable in scope
	// TODO: we should rename these to "variable" nodes.
	ScopeNode NodeType = "scope_node"
	// PlaceholderPlaceholder represents placeholder in the graph
	PlaceholderPlaceholder = "KITE_PLACEHOLDER"
)

// Attributes for nodes in the graph
type Attributes struct {
	ASTNodeType string `json:"ast_node_type"`
	Literal     string `json:"literal"`

	// TODO: replace `string` with an interface that exposes the methods to
	// convert the set of types to []int32, and maybe a few other things?
	Types  []string `json:"types"`
	values []pythontype.GlobalValue
	Token  pythonscanner.Token

	// set in the expansion graph to track information about predictions
	Client EgClientData
}

// String implements Stringer
func (a Attributes) String() string {
	return fmt.Sprintf("{%s %s %s %v}", a.ASTNodeType, a.Literal, strings.Join(a.Types, ":"), a.Client)
}

type nodeSet map[*Node]bool

func newNodeSet(nodes []*Node) nodeSet {
	ns := make(nodeSet)
	for _, n := range nodes {
		ns[n] = true
	}
	return ns
}

// Node in the graph
type Node struct {
	ID NodeID `json:"id"`
	// TODO: move Type into the attributes?
	Type     NodeType   `json:"type"`
	Attrs    Attributes `json:"attrs"`
	outgoing nodeSet
}

func (n *Node) deepCopy() *Node {
	return &Node{
		ID:   n.ID,
		Type: n.Type,
		Attrs: Attributes{
			ASTNodeType: n.Attrs.ASTNodeType,
			Literal:     n.Attrs.Literal,
			Types:       append([]string{}, n.Attrs.Types...),
			Client:      n.Attrs.Client,
			values:      append([]pythontype.GlobalValue{}, n.Attrs.values...),
		},
		outgoing: make(nodeSet),
	}
}

// String implements Stringer
func (n *Node) String() string {
	if n == nil {
		return "NIL!"
	}
	return fmt.Sprintf("{%v %s Attrs: %v}", n.ID, n.Type, n.Attrs)
}
func (n *Node) matchesType(sym pythonresource.Symbol, canonicalize bool) bool {
	var h pythonimports.Hash
	if canonicalize {
		h = sym.Canonical().PathHash()
	} else {
		h = sym.PathHash()
	}
	for _, v := range n.Attrs.values {
		var s pythonresource.Symbol
		switch v := v.(type) {
		case pythontype.External:
			s = v.Symbol()
		case pythontype.ExternalInstance:
			s = v.TypeExternal.Symbol()
		case pythontype.ExternalReturnValue:
			continue
		}
		if canonicalize {
			if s.Canonical().PathHash() == h {
				return true
			}
		} else {
			if s.PathHash() == h {
				return true
			}
		}

	}
	return false
}

// Graph is an (edge and node) labelled multi-graph representing
// an abstraction around the outputs from lexing, parsing, and static analysis.
type Graph struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// Inputs for building a graph
type Inputs struct {
	RM     pythonresource.Manager
	RAST   *pythonanalyzer.ResolvedAST
	Words  []pythonscanner.Word
	Buffer []byte
}

// NewGraph builds a new graph from the resolved AST.
func NewGraph(ctx kitectx.Context, edges EdgeSet, in Inputs) (*Graph, error) {
	ctx.CheckAbort()

	a := newAnalysis(in.RM, in.Words, in.RAST)
	b := newBuilder(ctx, a, false, true)
	b.BuildEdges(edges)
	return &Graph{
		Nodes: b.nodes,
		Edges: b.edges,
	}, nil
}

func makeSingleEdge(from, to *Node, t EdgeType, forward bool) *Edge {
	from.outgoing[to] = true

	return &Edge{
		From:    from.ID,
		To:      to.ID,
		Type:    t,
		Forward: forward,
		from:    from,
		to:      to,
	}
}

// DeepCopy of g along with a map from old to new node
func (g *Graph) DeepCopy() (*Graph, map[*Node]*Node) {
	nodes := make([]*Node, 0, len(g.Nodes))

	oldToNew := make(map[*Node]*Node, len(g.Nodes))
	for _, n := range g.Nodes {
		new := n.deepCopy()
		oldToNew[n] = new
		nodes = append(nodes, new)
	}

	edges := make([]*Edge, 0, len(g.Edges))
	for _, e := range g.Edges {
		from := oldToNew[e.from]
		to := oldToNew[e.to]
		edges = append(edges, makeSingleEdge(from, to, e.Type, e.Forward))
	}
	return &Graph{
		Nodes: nodes,
		Edges: edges,
	}, oldToNew
}
