package pythongraph

import (
	"bytes"
	"fmt"
	"go/token"
	"math/rand"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func reduceScopeForAttrBase(ctx kitectx.Context, a *analysis, scope scope) scope {
	return filterScopeByKinds(ctx, a, scope)
}

func (b *graphBuilder) ScopeForAttrBase(ctx kitectx.Context, node pythonast.Node) scope {
	return reduceScopeForAttrBase(ctx, b.a, b.vm.InScope(node, false))
}

func (b *graphBuilder) BuildAttrBaseSites(ctx kitectx.Context, sym pythonresource.Symbol) []*pythonast.AttributeExpr {
	ctx.CheckAbort()
	var sites []*pythonast.AttributeExpr
	pythonast.Inspect(b.a.RAST.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		attr, ok := node.(*pythonast.AttributeExpr)
		if !ok {
			return true
		}

		baseName, ok := attr.Value.(*pythonast.NameExpr)
		if !ok {
			return true
		}

		baseNode := b.astNodes[baseName]
		if !baseNode.matchesType(sym, true) {
			// base is a name so no need to recurse further
			return false
		}

		nameSite, err := b.BuildNameSite(b.ScopeForAttrBase(ctx, attr), baseName)
		if err != nil {
			return false
		}

		if len(nameSite.Scope) < 2 {
			// base is a name so no need to rescurse further
			return false
		}

		// make sure acutal variable is in scope
		variable := b.vm.VariableFor(baseName)
		var foundInScope bool
		for _, cand := range nameSite.Scope {
			if variable == cand {
				foundInScope = true
				break
			}
		}

		if !foundInScope {
			// this can happen because we limit the variables that we consider
			// "in scope", see variable.go
			return false
		}

		sites = append(sites, attr)

		// base is a name so no need to recurse further
		return false
	})

	return sites
}

// AttrBaseTrainInputs contains the inputs needed for creating a training sample
// for the base of an attribute
type AttrBaseTrainInputs struct {
	// Hash of input source file, used for marking bad
	// hashes during batch building
	Hash string

	Inputs

	// Symbol for the name at the base of the attribute expression
	Symbol pythonresource.Symbol
}

// NewAttrBaseTrainSample builds a new attribute base training sample from the inputs
func NewAttrBaseTrainSample(config TrainConfig, params TrainParams, in AttrBaseTrainInputs) (*InferNameSample, error) {
	// always canonicalize symbol
	sym := in.Symbol.Canonical()

	// find attr base site in full ast and edit the source code to remove
	// it and replace it with a dummy name expression
	edited, label, err := func() ([]byte, string, error) {

		a := newAnalysis(in.RM, in.Words, in.RAST)

		builder := newBuilder(kitectx.Background(), a, false, true)

		builder.BuildEdges(config.Graph.EdgeSet)

		attrs := builder.BuildAttrBaseSites(kitectx.Background(), sym)
		if len(attrs) == 0 {
			return nil, "", fmt.Errorf("unable to find valid attr base site for %s", sym.PathString())
		}

		idx := rand.Int() % len(attrs)
		attr := attrs[idx]

		save(
			params.Saver,
			SavedBundle{
				Label:      "original",
				builder:    builder,
				NodeLabels: nodeLabels(builder.astNodes[attr.Value], "site"),
				Buffer:     in.Buffer,
			},
		)

		// remove attr from source code
		edited := removeAttrFromSource(a, in.Buffer, attr)
		if len(edited) == 0 {
			return nil, "", fmt.Errorf("unable to remove attribute expr from source code")
		}

		save(params.Saver, bufferBundle("munged-no-graph", edited))

		return edited,
			attr.Value.(*pythonast.NameExpr).Ident.Literal,
			nil
	}()

	if err != nil {
		return nil, err
	}

	// re analyze source code and build new graph
	a, err := analyze(kitectx.Background(), in.RM, edited)
	if err != nil {
		return nil, fmt.Errorf("error re analyzing source code: %v", err)
	}

	builder := newBuilder(kitectx.Background(), a, false, true)

	builder.BuildEdges(config.Graph.EdgeSet)

	// find dummy name expression
	var target *pythonast.NameExpr
	pythonast.Inspect(a.RAST.Root, func(n pythonast.Node) bool {
		name, ok := n.(*pythonast.NameExpr)
		if ok {
			if name.Ident.Literal == renamedAttribute {
				target = name
			}
		}
		return target == nil
	})

	if target == nil {
		return nil, fmt.Errorf("unable to find renamed attribute")
	}

	save(
		params.Saver,
		SavedBundle{
			Label:      "munged",
			builder:    builder,
			NodeLabels: nodeLabels(builder.astNodes[target], "site"),
			Buffer:     edited,
		},
	)

	// build name site
	nameSite, err := func() (nameSite, error) {
		scope := builder.ScopeForAttrBase(kitectx.TODO(), target)

		for _, v := range scope {
			// TODO: why does this happen?
			if v.Origin.Ident.Literal == renamedAttribute {
				return nameSite{}, fmt.Errorf("dummy target variable should not be in scope for %s, %s", in.Hash, in.Symbol.PathString())
			}
		}

		// since we do not add names that are not found in a symbol table
		// the target variable should never be found.
		// TODO: why does this happen?
		if builder.vm.VariableFor(target) != nil {
			return nameSite{}, fmt.Errorf("variable for dummy target is not nil for %s, %s", in.Hash, in.Symbol.PathString())
		}

		return builder.BuildNameSite(scope, target)
	}()

	if err != nil {
		return nil, fmt.Errorf("error building target name site: %v", err)
	}

	if len(nameSite.Scope) < 2 {
		return nil, fmt.Errorf("not enough variables in scope")
	}

	// find the label variable
	var variable *variable
	var labelIdx int
	for i, v := range nameSite.Scope {
		if v.Origin.Ident.Literal == label {
			variable = v
			labelIdx = i
			break
		}
	}

	if variable == nil {
		return nil, fmt.Errorf("unable to find label variable after removal")
	}

	nameSite.Variable = variable

	contextNode := builder.UpdateForInferNameTrainTask(nameSite, config.MaxHops)

	assertTrue(labelIdx == int(variable.ID), fmt.Sprintf("label %d != variable id %d", labelIdx, variable.ID))

	save(
		params.Saver,
		SavedBundle{
			Label:      "pruned",
			builder:    builder,
			NodeLabels: nodeLabels(builder.astNodes[target], "site"),
			Buffer:     edited,
		},
	)

	// check if graph is still too large after pruning
	if len(builder.nodes) > maxNumNodesTrainGraph {
		return nil, fmt.Errorf("too many nodes in graph got %d > %d", len(builder.nodes), maxNumNodesTrainGraph)
	}

	egOnlyNodes := []*Node{contextNode}
	for _, cand := range nameSite.Candidates {
		egOnlyNodes = append(egOnlyNodes, cand.Usage)
	}

	eg, _ := builder.ExpansionGraph(params.ModelMeta, egOnlyNodes, nil)

	return &InferNameSample{
		ContextGraph:   builder.newGraphFeed(params.ModelMeta),
		ExpansionGraph: eg,
		Name: NameModelFeed{
			PredictionNodes: []int32{int32(contextNode.ID)},
			Corrupted:       newCorruptedSegmented(params.Rand, labelIdx, len(nameSite.Scope), config.NumCorrupted),
			Labels:          []VariableID{variable.ID},
			Types:           traindata.NewSegmentedIndicesFeed(int32(params.TypeSubtokenIndex.Index(traindata.AttrBaseNameDecoder))),
			Subtokens:       traindata.NewSegmentedIndicesFeed(int32(params.NameSubtokenIndex.Index(traindata.AttrBaseNameDecoder))),
			Names:           newNameEncoderFeedFromNameSite(builder.astNodes, nameSite),
		},
	}, nil
}

// TODO: unit test
// TODO: use this for predicting the attr portion of attributes as well
// TODO: should really be informed by user edit data to determine the correct conditioning context
func removeAttrFromSource(a *analysis, src []byte, attr *pythonast.AttributeExpr) []byte {
	// begin/end of source code to remove
	begin, end := token.Pos(-1), token.Pos(-1)

	setBeginEnd := func(node pythonast.Node) bool {
		if isChildOf(node, attr) {
			begin, end = node.Begin(), node.End()
			return true
		}
		return false
	}

	setBeginEndAny := func(exprs ...pythonast.Expr) bool {
		for _, expr := range exprs {
			if isChildOf(expr, attr) {
				setBeginEnd(expr)
				return true
			}
		}
		return false
	}

	parent := a.RAST.ParentStmts[attr]
	switch parent := parent.(type) {
	case *pythonast.BadStmt:
		// TODO
	case *pythonast.ExprStmt:
		setBeginEnd(parent)
	case *pythonast.AnnotationStmt:
		if setBeginEnd(parent.Annotation) {
			break
		}
		setBeginEnd(parent)
	case *pythonast.AssignStmt:
		if setBeginEnd(parent.Value) {
			break
		}

		if isChildOf(parent.Annotation, attr) {
			begin, end = parent.Annotation.Begin(), parent.End()
			break
		}

		setBeginEnd(parent)
	case *pythonast.AugAssignStmt:
		if setBeginEnd(parent.Value) {
			break
		}
		setBeginEnd(parent)
	case *pythonast.ClassDefStmt:
		if setBeginEndAny(parent.Decorators...) {
			break
		}

		if setBeginEndAny(parent.Kwarg, parent.Vararg) {
			break
		}

		for _, arg := range parent.Args {
			if setBeginEnd(arg) {
				break
			}
		}

	case *pythonast.FunctionDefStmt:
		if setBeginEndAny(parent.Decorators...) {
			break
		}

		if setBeginEnd(parent.Annotation) {
			break
		}

		if parent.Kwarg != nil && setBeginEnd(parent.Kwarg.Annotation) {
			break
		}

		if parent.Vararg != nil && setBeginEnd(parent.Vararg.Annotation) {
			break
		}

		for _, param := range parent.Parameters {
			if setBeginEndAny(param.Annotation, param.Default) {
				break
			}
		}

	case *pythonast.AssertStmt:
		if setBeginEnd(parent.Message) {
			break
		}

		// must be a part of the condition, wipe out the message too
		begin, end = parent.Condition.Begin(), parent.End()
	case *pythonast.ContinueStmt:
		// should not happen
	case *pythonast.BreakStmt:
		// should not happen
	case *pythonast.DelStmt:
		// wipe out all the targets,
		// TODO: could just wipe out specific target and any trailers...
		begin, end = parent.Targets[0].Begin(), parent.End()
	case *pythonast.ExecStmt:
		setBeginEndAny(parent.Body, parent.Locals, parent.Globals)
	case *pythonast.PassStmt:
		// should not happen
	case *pythonast.PrintStmt:
		if setBeginEnd(parent.Dest) {
			break
		}
		// just wipe out print statement for now, could
		// just remove specific value and trailers
		begin, end = parent.Values[0].Begin(), parent.End()
	case *pythonast.RaiseStmt:
		setBeginEndAny(parent.Instance, parent.Traceback, parent.Type)
	case *pythonast.ReturnStmt:
		// just wipe out entire statement for now,
		// could check tuple expression and wipe out partial
		setBeginEnd(parent.Value)
	case *pythonast.YieldStmt:
		// just wipe out entire statement for now
		// could check tuple expression
		setBeginEnd(parent.Value)
	case *pythonast.GlobalStmt, *pythonast.NonLocalStmt:
		// should not happen
	case *pythonast.IfStmt:
		for _, branch := range parent.Branches {
			if setBeginEnd(branch.Condition) {
				break
			}
		}
	case *pythonast.ForStmt:
		if setBeginEnd(parent.Iterable) {
			break
		}

		// remove all the targets but leave the iterable
		// TODO: probably want to remove iterable too
		last := parent.Targets[len(parent.Targets)-1]
		begin, end = parent.Targets[0].Begin(), last.End()
	case *pythonast.WhileStmt:
		// we must be in the condition portion so just remove that
		setBeginEnd(parent.Condition)
	case *pythonast.TryStmt:
		for _, h := range parent.Handlers {
			if setBeginEndAny(h.Type, h.Target) {
				break
			}
		}
	case *pythonast.WithStmt:
		for _, item := range parent.Items {
			if setBeginEnd(item) {
				break
			}
		}
	}

	if begin == -1 {
		return nil
	}

	return bytes.Join([][]byte{
		src[:begin],
		[]byte(renamedAttribute),
		src[end:],
	}, nil)
}
