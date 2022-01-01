package pythongraph

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

const (
	// the string to which the attribute is renamed to when munging the buffer before re-parsing/re-analyzing - this
	// has no bearing on the output unless it happens to collide with the name of a real attribute
	renamedAttribute = "GUESS_ME__34578032"
)

type attributeSite struct {
	expr   *pythonast.AttributeExpr
	node   *Node
	parent *Node
}

// AttributeTrainInputs contains the necessary information to produce a training sample for the attribute GGNN model.
type AttributeTrainInputs struct {
	Inputs
	CanonicalToSym map[string][]string
	Parent         pythonresource.Symbol
	Symbol         pythonresource.Symbol
	Hash           string
}

// NewAttributeTrainSample creates an attribute train sample from the given inputs.
func NewAttributeTrainSample(config TrainConfig, params TrainParams, in AttributeTrainInputs) (*InferProductionSample, error) {
	a1 := newAnalysis(in.RM, in.Words, in.RAST)

	builder1 := newBuilder(kitectx.Background(), a1, false, true)

	builder1.BuildEdges(config.Graph.EdgeSet)

	site1 := builder1.randomAttributeSite(params.Rand, in.Symbol.PathString(), in.CanonicalToSym)
	if site1 == (attributeSite{}) {
		return nil, fmt.Errorf("attribute site not found")
	}

	save(
		params.Saver,
		SavedBundle{
			Label:      "original",
			builder:    builder1,
			NodeLabels: nodeLabels(site1.node, "site"),
			Buffer:     in.Buffer,
		},
	)

	lm := linenumber.NewMap(in.Buffer)
	trimEnd := trimEndLineOrStmt(site1.expr.Dot.End, site1.expr, lm, in.RAST.ParentStmts)

	newSrc := bytes.Join([][]byte{
		in.Buffer[:site1.expr.Dot.End],
		[]byte(renamedAttribute),
		in.Buffer[trimEnd:],
	}, nil)

	save(params.Saver, bufferBundle("munged-no-graph", newSrc))

	a2, err := analyze(kitectx.Background(), in.RM, newSrc)
	if err != nil {
		return nil, err
	}

	builder2 := newBuilder(kitectx.Background(), a2, false, true)
	builder2.BuildEdges(config.Graph.EdgeSet)

	site2 := builder2.findAttributeSiteAgain()
	if site2 == (attributeSite{}) {
		return nil, fmt.Errorf("couldn't find attrib site again")
	}

	if site2.expr.Dot.End != site1.expr.Dot.End {
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

	scope := builder2.ScopeForAttr(kitectx.TODO(), site2.expr)
	if len(scope) == 0 {
		return nil, fmt.Errorf("no variables in scope")
	}
	builder2.vm.ReduceTo(builder2.a.RAST.Root, scope)

	scopeNodes := builder2.AddScopeNodeAndEdges(scope)
	contextNodes := builder2.ContextTokens(site2.expr)

	site2.parent.Attrs.Types = []string{traindata.InferAttrMarker}
	site2.node.Attrs.Literal = traindata.InferAttrMarker

	if config.MaxHops > 0 {
		keep := nodeSet(map[*Node]bool{
			site2.node:   true,
			site2.parent: true,
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

	children, label, err := params.ProductionIndex.ChildrenWithLabel(in.Parent.PathHash(), in.Symbol.PathHash())
	if err != nil {
		return nil, fmt.Errorf("unable to find children: %v", err)
	}

	eg, cgToEG := builder2.ExpansionGraph(params.ModelMeta, []*Node{site2.node}, joinNodes(scopeNodes, contextNodes))

	return &InferProductionSample{
		ContextGraph:   builder2.newGraphFeed(params.ModelMeta),
		ExpansionGraph: eg,
		Production: ProductionModelFeed{
			PredictionNodes: []int32{int32(site2.node.ID)},
			Labels:          []int{label},
			DecoderTargets:  traindata.NewSegmentedIndicesFeed(children...),
			Corrupted:       newCorruptedSegmented(params.Rand, label, len(children), config.NumCorrupted),
			ScopeEncoder:    newNodeIDFeed(scopeNodes, cgToEG),
			ContextTokens:   newNodeIDFeed(contextNodes, cgToEG),
		},
	}, nil
}

func (b *graphBuilder) ScopeForAttr(ctx kitectx.Context, node pythonast.Node) scope {
	return b.vm.InScope(node, true)
}

func (b *graphBuilder) randomAttributeSite(rand *rand.Rand, sym string, canonToSym map[string][]string) attributeSite {
	var sites []attributeSite

	for ast, node := range b.astNodes {
		attr, ok := ast.(*pythonast.AttributeExpr)
		if !ok {
			continue
		}

		// TODO: this is pretty nasty, but we cannot really guarantee
		// that analysis does not canonicalize the symbols
		// so for now this is all we can do...
		var matches bool
	match:
		for _, gv := range node.Attrs.values {
			t := symbolFor(gv).Canonical().PathString()
			for _, s := range canonToSym[t] {
				if s == sym {
					matches = true
					break match
				}
			}
		}

		if !matches {
			continue
		}

		// see https://github.com/kiteco/kiteco/issues/6636
		if b.wordNodes[*attr.Attribute] == nil {
			continue
		}

		sites = append(sites, attributeSite{
			expr:   attr,
			node:   b.wordNodes[*attr.Attribute],
			parent: node,
		})
	}

	if len(sites) == 0 {
		return attributeSite{}
	}

	sort.Slice(sites, func(i, j int) bool { return sites[i].expr.Begin() < sites[j].expr.Begin() })

	return sites[rand.Intn(len(sites))]
}

func (b *graphBuilder) findAttributeSiteAgain() attributeSite {
	for ast, node := range b.astNodes {
		attr, ok := ast.(*pythonast.AttributeExpr)
		if !ok {
			continue
		}

		if attr.Attribute.Literal != renamedAttribute {
			continue
		}

		// see https://github.com/kiteco/kiteco/issues/6636
		if b.wordNodes[*attr.Attribute] == nil {
			continue
		}

		return attributeSite{
			expr:   attr,
			node:   b.wordNodes[*attr.Attribute],
			parent: node,
		}
	}

	return attributeSite{}
}
