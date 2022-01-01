package pythongraph

import (
	"fmt"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const maxNumContextTokens = 10

type astToNode map[pythonast.Node]*Node

type graphBuilder struct {
	a             *analysis
	astNodes      astToNode
	wordNodes     map[pythonscanner.Word]*Node
	wordsForNodes map[pythonast.Node][]pythonscanner.Word

	nodes []*Node
	edges []*Edge

	vm *variableManager
}

func newBuilder(ctx kitectx.Context, a *analysis, addMissingNames, instanceTypes bool) *graphBuilder {
	ctx.CheckAbort()

	defer newBuilderDuration.DeferRecord(time.Now())
	b := &graphBuilder{
		a:             a,
		astNodes:      make(astToNode),
		wordNodes:     make(map[pythonscanner.Word]*Node),
		wordsForNodes: wordsForNodes(a.RAST.Root, a.Words),
	}

	addASTNode := func(node pythonast.Node, nd NodeData) *Node {

		t := ASTInternalNode
		if pythonast.IsTerminal(node) {
			t = ASTTerminalNode
		}

		attrs := b.attributesFor(ctx, node, instanceTypes)
		attrs.Client = nd

		n := &Node{
			ID:       NodeID(len(b.nodes)),
			Type:     t,
			Attrs:    attrs,
			outgoing: make(nodeSet),
		}

		b.astNodes[node] = n
		b.nodes = append(b.nodes, n)

		return n
	}

	addWordsForASTNode := func(node *Node, ast pythonast.Node) {
		if pythonast.IsTerminal(ast) {
			// map all words for a terminal ast node to the same node
			// e.g for ellipsis expressions this means that we have one
			// terminal node for the expression, rather than 3 separate terminal nodes for each period
			for _, word := range b.wordsForNodes[ast] {
				b.wordNodes[word] = node
			}
			return
		}

		call, _ := ast.(*pythonast.CallExpr)

		for _, word := range b.wordsForNodes[ast] {
			if skipWord(word) {
				continue
			}

			var nd NodeData
			if call != nil {
				if *call.LeftParen == word {
					nd.ASTParentField = "LeftParen"
				}
				if call.RightParen != nil && *call.RightParen == word {
					nd.ASTParentField = "RightParen"
				}
				for i, comma := range call.Commas {
					if *comma == word {
						nd.ASTParentField = "Commas"
						nd.ASTParentPos = i
					}

				}
			}

			wn := &Node{
				ID:   NodeID(len(b.nodes)),
				Type: ASTTerminalNode,
				Attrs: Attributes{
					Literal: traindata.WordLiteral(word),
					Types:   []string{traindata.NAType},
					Token:   word.Token,
					Client:  nd,
				},
				outgoing: make(nodeSet),
			}

			b.wordNodes[word] = wn

			b.nodes = append(b.nodes, wn)

			b.edges = append(b.edges, makeEdges(node, wn, ASTChild)...)
		}
	}

	mod := addASTNode(b.a.RAST.Root, NodeData{ASTParentField: "ROOT"})
	addWordsForASTNode(mod, b.a.RAST.Root)

	pythonast.InspectEdges(b.a.RAST.Root, func(parent, child pythonast.Node, field string) bool {
		if pythonast.IsNil(parent) || pythonast.IsNil(child) {
			// at root or exiting child, just recurse
			return true
		}

		p := b.astNodes[parent]
		if p == nil {
			panic(fmt.Sprintf("this should enver happen got nil parent for %v", pythonast.String(parent)))
		}

		c := b.astNodes[child]
		if c == nil {
			c = addASTNode(child, NodeData{ASTParentField: field})
		}

		// if the parent is a call the child is an arg then
		// set the index for the argument
		if call, ok := parent.(*pythonast.CallExpr); ok && field == "Args" {
			for i, arg := range call.Args {
				if arg == child {
					nd := c.Attrs.Client.(NodeData)
					nd.ASTParentPos = i
					c.Attrs.Client = nd
					break
				}
			}
		}

		b.edges = append(b.edges, makeEdges(p, c, ASTChild)...)

		addWordsForASTNode(c, child)

		return true
	})

	// build next token edges

	// add "start of file" token
	prev := &Node{
		ID:   NodeID(len(b.nodes)),
		Type: ASTTerminalNode,
		Attrs: Attributes{
			Literal: traindata.SOFMarker,
			Types:   []string{traindata.NAType},
			Client:  NodeData{},
		},
		outgoing: make(nodeSet),
	}
	b.nodes = append(b.nodes, prev)

	// start of file token is child of module, just like eof token
	b.edges = append(b.edges, makeEdges(mod, prev, ASTChild)...)

	for _, word := range b.a.Words {
		wn := b.wordNodes[word]
		if wn == nil {
			continue
		}
		if prev == wn {
			// happens for terminal nodes that contain multiple words,
			// e.g StringExpr, EllipsisExpr, etc
			continue
		}

		b.edges = append(b.edges, makeEdges(prev, wn, NextToken)...)

		prev = wn
	}

	// build variable manager
	b.vm = newVariableManager(a, addMissingNames)
	return b
}

func (b *graphBuilder) BuildEdges(edges EdgeSet) {
	defer buildEdgesDuration.DeferRecord(time.Now())

	if edges.Contains(LastLexicalUse) {
		b.buildLastLexicalUseEdges()
	}

	if edges.Contains(ComputedFrom) {
		b.buildComputedFromEdges()
	}

	// TODO: just have data flow edges instead?
	if edges.Contains(LastRead) || edges.Contains(LastWrite) {
		b.buildLastReadAndWriteEdges()
	}

	if edges.Contains(DataFlow) {
		b.buildDataFlowEdges()
	}

	if edges.Contains(ReturnValueOf) {
		b.buildReturnValueOfEdges()
	}
}

func (b *graphBuilder) buildLastLexicalUseEdges() {
	for _, v := range b.vm.Variables {
		usages := v.Refs.Names()

		for i := 1; i < len(usages); i++ {
			prev := b.astNodes[usages[i-1]]
			curr := b.astNodes[usages[i]]

			b.edges = append(b.edges, makeEdges(curr, prev, LastLexicalUse)...)
		}
	}
}

func (b *graphBuilder) buildComputedFromEdges() {
	names := func(n pythonast.Node) []*pythonast.NameExpr {
		var selected []*pythonast.NameExpr
		pythonast.Inspect(n, func(nn pythonast.Node) bool {
			switch nn := nn.(type) {
			case *pythonast.AttributeExpr:
				return false
			case *pythonast.NameExpr:
				selected = append(selected, nn)
				return false
			default:
				return true
			}
		})
		return selected
	}

	pythonast.Inspect(b.a.RAST.Root, func(n pythonast.Node) bool {
		if n == nil {
			return true
		}

		assign, ok := n.(*pythonast.AssignStmt)
		if !ok {
			return true
		}

		var lhs []*pythonast.NameExpr
		for _, t := range assign.Targets {
			lhs = append(lhs, names(t)...)
		}

		rhs := names(assign.Value)

		for _, l := range lhs {
			ln := b.astNodes[l]
			for _, r := range rhs {
				rn := b.astNodes[r]
				b.edges = append(b.edges, makeEdges(ln, rn, ComputedFrom)...)
			}
		}

		return false
	})
}

func (b *graphBuilder) buildReturnValueOfEdges() {
	names := func(n pythonast.Node) []*pythonast.NameExpr {
		var selected []*pythonast.NameExpr
		pythonast.Inspect(n, func(nn pythonast.Node) bool {
			switch nn := nn.(type) {
			case *pythonast.AttributeExpr:
				return false
			case *pythonast.NameExpr:
				selected = append(selected, nn)
				return false
			default:
				return true
			}
		})
		return selected
	}

	pythonast.Inspect(b.a.RAST.Root, func(n pythonast.Node) bool {
		if n == nil {
			return true
		}

		assign, ok := n.(*pythonast.AssignStmt)
		if !ok {
			return true
		}

		var lhs []*pythonast.NameExpr
		for _, t := range assign.Targets {
			lhs = append(lhs, names(t)...)
		}

		// TODO: could handle multiple calls on RHS
		// but would need to be smarter about which
		// name we link the call to on the lhs
		call, ok := assign.Value.(*pythonast.CallExpr)
		if !ok {
			return false
		}

		// TODO: kind of hacky but we connect this to the Func node
		// instead of the call node
		fn := b.astNodes[call.Func]
		for _, l := range lhs {
			ln := b.astNodes[l]
			b.edges = append(b.edges, makeEdges(ln, fn, ReturnValueOf)...)
		}

		return false
	})
}

func (b *graphBuilder) buildLastReadAndWriteEdges() {
	for _, v := range b.vm.Variables {
		graph := b.forwardFlowGraph(v.Refs)
		for name, neighbors := range graph {
			if name.Usage == pythonast.Delete {
				continue
			}

			// the flow graph computes forward data flow,
			// so we reverse the edges to get the last
			// read and write
			dest := b.astNodes[name]

			t := LastWrite
			if name.Usage == pythonast.Evaluate {
				t = LastRead
			}

			for flowsTo := range neighbors.Set() {
				src := b.astNodes[flowsTo]
				b.edges = append(b.edges, makeEdges(src, dest, t)...)

			}
		}
	}
}

func (b *graphBuilder) buildDataFlowEdges() {
	for _, v := range b.vm.Variables {
		graph := b.forwardFlowGraph(v.Refs)
		for name, neighbors := range graph {
			src := b.astNodes[name]
			for flowsTo := range neighbors.Set() {
				dest := b.astNodes[flowsTo]
				b.edges = append(b.edges, makeEdges(src, dest, DataFlow)...)
			}
		}
	}
}

func (b *graphBuilder) attributesFor(ctx kitectx.Context, node pythonast.Node, instanceTypes bool) Attributes {
	attrs := Attributes{
		ASTNodeType: traindata.ASTNodeType(node),
		Literal:     traindata.ASTNodeLiteral(node),
	}

	switch ex := node.(type) {
	case *pythonast.EllipsisExpr:
		attrs.Types = []string{traindata.NATypeMarker}
	case pythonast.Expr:
		gvs := b.a.ResolveToGlobals(ctx, ex)
		if len(gvs) == 0 {
			attrs.Types = []string{traindata.UnknownTypeMarker}
			break
		}

		attrs.values = gvs

		attrs.Types = make([]string, 0, len(gvs))
		for _, gv := range gvs {
			if t := typeString(gv, instanceTypes); t != "" {
				attrs.Types = append(attrs.Types, t)
			}
		}

		if len(attrs.Types) == 0 {
			attrs.Types = []string{traindata.UnknownTypeMarker}
		}

	default:
		attrs.Types = []string{traindata.NATypeMarker}
	}

	// TODO: we should sort the underyling values instead
	sort.Strings(attrs.Types)

	return attrs
}

func typeString(gv pythontype.GlobalValue, instanceTypes bool) string {
	switch gv := gv.(type) {
	case pythontype.External:
		return gv.Symbol().Canonical().PathString()
	case pythontype.ExternalInstance:
		if instanceTypes {
			return gv.TypeExternal.Symbol().Canonical().Path().WithTail(traindata.InstanceTail).String()
		}
		return gv.TypeExternal.Symbol().Canonical().PathString()
	case pythontype.ExternalReturnValue:
		return gv.Func().Canonical().Path().WithTail(traindata.ReturnValueTail).String()
	default:
		return ""
	}
}

func (b *graphBuilder) Graph() *Graph {
	return &Graph{
		Nodes: b.nodes,
		Edges: b.edges,
	}
}

func makeEdges(from, to *Node, t EdgeType) []*Edge {
	from.outgoing[to] = true
	to.outgoing[from] = true

	return []*Edge{
		&Edge{
			From:    from.ID,
			To:      to.ID,
			Type:    t,
			Forward: true,
			from:    from,
			to:      to,
		},
		&Edge{
			From:    to.ID,
			To:      from.ID,
			Type:    t,
			Forward: false,
			from:    to,
			to:      from,
		},
	}
}

// adds synthetic "scope" nodes to the graph (one per variable in scope), then connects all references to
// a given variable to this node, and finally returns the ids for the added "scope" nodes.
func (b *graphBuilder) AddScopeNodeAndEdges(scope scope) []*Node {
	// connect references for variables in scope
	// to scope node for that variable
	var scopeNodes []*Node
	for _, v := range scope {
		vNode := &Node{
			ID:    NodeID(len(b.nodes)),
			Type:  ScopeNode,
			Attrs: b.astNodes[v.Origin].Attrs,
		}
		b.nodes = append(b.nodes, vNode)
		scopeNodes = append(scopeNodes, vNode)

		for _, ref := range v.Refs.Names() {
			from := b.astNodes[ref]

			// TODO: gross
			from.outgoing[vNode] = true
			b.edges = append(b.edges, &Edge{
				From:    from.ID,
				To:      vNode.ID,
				Type:    ScopeEdge,
				Forward: true,
				from:    from,
				to:      vNode,
			})
		}
	}
	return scopeNodes
}

func (b *graphBuilder) ContextTokens(ast pythonast.Node) []*Node {
	// find deepest node, may not be a terminal in some cases,
	// e.g {}.pop($) will not have a terminal node
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if !pythonast.IsNil(n) {
			ast = n
		}
		return true
	})
	node := b.astNodes[ast]

	include := func(n *Node) bool {
		if n.Type != ASTTerminalNode {
			return false
		}
		if n.Attrs.Literal == traindata.SOFMarker {
			return true
		}

		tok := n.Attrs.Token
		if tok.IsWhitespace() {
			return false
		}

		switch tok {
		case pythonscanner.Illegal, pythonscanner.BadToken,
			pythonscanner.Cursor, pythonscanner.Comment,
			pythonscanner.Lparen, pythonscanner.Lbrack,
			pythonscanner.Lbrace, pythonscanner.Comma,
			pythonscanner.Period, pythonscanner.Rparen,
			pythonscanner.Rbrack, pythonscanner.Rbrace,
			pythonscanner.Semicolon, pythonscanner.Colon,
			pythonscanner.Backtick:
			return false
		default:
			return true
		}
	}

	// ordered by BFS search
	nodes := []*Node{node}
	seen := map[*Node]bool{
		node: true,
	}
	for i := 0; i < len(nodes) && len(nodes) < maxNumContextTokens; i++ {
		n := nodes[i]
		for nn := range n.outgoing {
			if !seen[nn] && include(nn) {
				seen[nn] = true
				nodes = append(nodes, nn)
			}
		}
	}

	return nodes
}
