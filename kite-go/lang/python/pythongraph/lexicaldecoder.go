package pythongraph

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// lexicalDecoder implements a "bottom up" decoder for the ast (e.g using the lexical tokens to drive the expansion),
// this is morally similar to the way a shift reduce parser operates.
type lexicalDecoder struct {
	t     *tracer
	saver Saver
	rm    pythonresource.Manager
	cbs   ExprCallbacks
	meta  ModelMeta
	depth int

	// for debugging purposes
	buffer []byte
	ast    pythonast.Node
}

var (
	nameASTNode = traindata.ASTNodeType(&pythonast.NameExpr{})
	argASTNode  = traindata.ASTNodeType(&pythonast.Argument{})
	attrASTNode = traindata.ASTNodeType(&pythonast.AttributeExpr{})
	callASTNode = traindata.ASTNodeType(&pythonast.CallExpr{})
)

// ExpansionTask ...
type ExpansionTask string

// EgClientData ...
func (e ExpansionTask) EgClientData() string { return string(e) }

type lexicalGrammarT struct {
	Placeholder,
	Stop,
	Expr,
	ExprDone,
	ChooseExprType,
	ChooseTerminalType,
	Attr,
	AttrDone,
	Call,
	CallDone,
	ArgDone,
	Keyword,
	KeywordDone,
	Positional,
	ChooseArgType,
	InferKeywordArgName,
	InferAttr,
	InferName,
	NameDone,
	GenericNoPlaceholder,
	Propagate ExpansionTask
}

var lexicalGrammar = lexicalGrammarT{
	Placeholder:          "placeholder",
	Stop:                 "stop",
	Expr:                 "expr",
	ExprDone:             "exprDone",
	ChooseExprType:       "choose_expr_type",
	ChooseTerminalType:   "choose_terminal_type",
	Attr:                 "attr",
	AttrDone:             "attr_done",
	Call:                 "call",
	CallDone:             "call_done",
	ArgDone:              "arg_done",
	Keyword:              "keyword",
	KeywordDone:          "keyword_done",
	Positional:           "positional",
	ChooseArgType:        "choose_arg_type",
	InferKeywordArgName:  "infer_keyword_arg_name",
	InferAttr:            "infer_attr",
	InferName:            "infer_name",
	NameDone:             "name_done",
	GenericNoPlaceholder: "generic_no_placeholder",
	Propagate:            "propagate",
}

// ExpansionTasks ...
func ExpansionTasks() []ExpansionTask {
	return append([]ExpansionTask{}, expansionTaskList...)
}

var expansionTaskIDs = make(map[ExpansionTask]pythonimports.Hash)
var expansionTaskList []ExpansionTask

// ExpansionTaskIDs ...
func ExpansionTaskIDs() map[ExpansionTask]pythonimports.Hash {
	cpy := make(map[ExpansionTask]pythonimports.Hash, len(expansionTaskIDs))
	for k, v := range expansionTaskIDs {
		cpy[k] = v
	}
	return cpy
}

// ExpansionTaskID ...
func ExpansionTaskID(t ExpansionTask) pythonimports.Hash {
	id, ok := expansionTaskIDs[t]
	if !ok {
		panic(fmt.Sprintf("no id for expansion task %v", t))
	}
	return id
}

// ExpansionTaskRoot ...
const ExpansionTaskRoot = "EXPANSION_TASK"

func init() {
	t := reflect.TypeOf(lexicalGrammar)
	val := reflect.ValueOf(lexicalGrammar)
	for i := 0; i < t.NumField(); i++ {
		fv := val.Field(i)
		if fv.String() == "" {
			panic(fmt.Sprintf("Value for field %s is not set", t.Field(i).Name))
		}
		fvs := ExpansionTask(fv.String())
		expansionTaskList = append(expansionTaskList, fvs)
		expansionTaskIDs[fvs] = pythonimports.PathHash([]byte(fmt.Sprintf("%v::%v", ExpansionTaskRoot, fvs)))
	}
}

// NodeData ...
type NodeData struct {
	// We use this symbol to update the parent
	// attribute expression after an inferAttr task has been completed.
	Symbol pythonresource.Symbol

	// ASTParentField is the field in the parent ast node
	// that points to this node.
	// NOTE: this is currently not set for nodes that originate
	// from a `pythonscanner.Word`.
	ASTParentField string

	// ASTParentPos is used to mark the position in the parent ast node
	// that points to this node in the case that this node is part of a
	// slice of nodes.
	ASTParentPos int
}

// String ...
func (nd NodeData) String() string {
	return fmt.Sprintf("Symbol: %v, ASTParentField: %s, ASTParentPos: %d", nd.Symbol, nd.ASTParentField, nd.ASTParentPos)
}

// EgClientData ...
func (nd NodeData) EgClientData() string { return nd.String() }

type inferProdAttr pythonresource.Symbol

func (i inferProdAttr) EgClientData() string { return pythonresource.Symbol(i).String() }

type inferProdKeyword string

func (k inferProdKeyword) EgClientData() string { return string(k) }

func (ld lexicalDecoder) PrepareForInference(ctx kitectx.Context, cg *ContextGraph, eg *ExpansionGraph, ast pythonast.Node) (EgTaskStack, error) {
	ld.print("@ PrepareForInference")
	defer func() {
		ld.print("DONE PrepareForInference")
	}()

	var task EgTask
	var cgSite *Node
	switch ast := ast.(type) {
	case *pythonast.CallExpr:
		// Send in:
		//   - a fake "completed" `chooseExprType` task
		//   - the "site" should be the top level call expression ast node
		//   - we mark the "completed" infer prod task as a call
		//   - make sure the prediction site is actually in the expansion graph
		// NOTE:
		//   - we send in a context graph node for the `EgTask.ClientNodes` since
		//     that is what `lexicalDecoder.chooseExprTypeCompleted` expects,
		//     unclear if we should copy this node to the expansion graph first?
		cgSite = cg.builder.astNodes[ast]

		egSite := eg.AddNode(cgSite.Type, cgSite.Attrs, cg.finalNodeStates[cgSite.ID])
		task = EgTask{
			Type:            InferProductionTask,
			Site:            egSite,
			Client:          lexicalGrammar.ChooseExprType,
			ClientNodes:     []*Node{cg.builder.astNodes[ast.Func]},
			InferProdClient: []EgClientData{lexicalGrammar.Call},
		}
	case *pythonast.AttributeExpr:
		// Send in:
		//   - a fake "completed" `chooseExprType` task
		//   - the "site" should be the top level attribute expression ast node
		//   - mark the "completed" infer prod task as an attr
		//   - make sure the prediction site is actually in the expansion graph
		// NOTE:
		//   - we send in a context graph node for the `EgTask.ClientNodes` since
		//     that is what `lexicalDecoder.chooseExprTypeCompleted` expects,
		//     unclear if we should copy this node to the expansion graph first?

		cgSite = cg.builder.astNodes[ast]

		egSite := eg.AddNode(cgSite.Type, cgSite.Attrs, cg.finalNodeStates[cgSite.ID])
		task = EgTask{
			Type:            InferProductionTask,
			Site:            egSite,
			Client:          lexicalGrammar.ChooseExprType,
			ClientNodes:     []*Node{cg.builder.astNodes[ast.Value]},
			InferProdClient: []EgClientData{lexicalGrammar.Attr},
		}
	case *pythonast.NameExpr:
		// Send in:
		//   - a fake "completed" `chooseTerminalType` task
		//   - make sure the prediction site is actually in the expansion graph
		cgSite = cg.builder.astNodes[ast]

		egSite := eg.AddNode(cgSite.Type, cgSite.Attrs, cg.finalNodeStates[cgSite.ID])
		task = EgTask{
			Type:            InferProductionTask,
			Site:            egSite,
			Client:          lexicalGrammar.ChooseTerminalType,
			InferProdClient: []EgClientData{lexicalGrammar.InferName},
		}
	default:
		return nil, errors.Errorf("unsupported node type %T", ast)
	}

	ld.print("initial context graph site %v", cgSite)
	ld.print("initial expansion graph site %v", task.Site)

	// connect the prediction site to the relevant context graph nodes
	for _, n := range cg.incoming[cgSite] {
		ld.print("adding incoming edge to eg site %v %v", n.Type, n.Node)
		eg.AddEdge(n.Node, task.Site, n.Type)
	}

	for _, n := range cg.outgoing[cgSite] {
		ld.print("adding outgoing edge from eg site %v %v", n.Type, n.Node)
		// need to connect the outgoing edges back to the context graph
		// as well in order for navigation to work properly
		eg.AddNavOnlyEdge(task.Site, n.Node, n.Type)
	}

	save(ld.saver, SavedBundle{
		builder:    cg.builder,
		Label:      "context-graph",
		NodeLabels: map[NodeID]string{cgSite.ID: "site"},
		Buffer:     ld.buffer,
	})

	// there is always an exprDone task at the bottom of the remaining stack
	stack, err := ld.TaskCompleted(ctx, eg, task, EgTaskStack{{
		Type:   NoInferTask,
		Client: lexicalGrammar.ExprDone,
		Site:   task.Site,
	}})
	if err != nil {
		return nil, errors.Errorf("error preparing initial task: %v", err)
	}
	return stack, nil
}

// General idea, we use "lexical decoding" to decode the AST from the "bottom up",
// this is morally similar to the way a "Shift-reduce" parser operates.
// Notes:
//   - We use the `choose` prefix to denote tasks which only modify the structure of the ast.
//     The result of these choices is stored in `Node.Attrs.Client.(NodeData).ChooseResult`.
//   - We use the `infer` prefix to denote tasks which modify the contents of nodes in the ast, but
//     do not alter its structure. For these tasks `Node.Attrs.Client.(NodeData).ChooseResult == ""`.
//   - Rules with neither a `choose` or a `infer` prefix are internal nodes in this particular grammar.
//   - We use the `Done` suffix for rules which are markers to denote when a multi step task is completed,
//     these are typically reserved for internal nodes in the underlying python grammar.
//   - We use the `stop` rule as a catch all for signifying that a particular branch of exploration should be stopped.
//     Typically we also pop some done tasks off the stack when this is hit (see below).
//   - The additions to the stack are listed as [head, ..., tail]
//   - 0 signifies no expansions and nothing as added to the stack
// Grammar
//   - expr -> chooseTerminalType [exprDone]
//   - chooseTerminalType -> inferName | placeholder
//   - placeholder -> 0
//   - chooseExprType -> attr [attrDone] | call [callDone] | stop
//   - call -> chooseArgType [argDone]
//   - chooseArgType -> argKeyword | positional | stop
//   - keyword -> inferKeywordArgName [expr]
//   - positional -> expr
//   - exprDone -> 0
//   - inferName -> chooseExprType
//   - callDone -> chooseExprType
//   - attrDone -> chooseExprType
// NOTE:
//   - This is kind of painful to track, can we formalize this in a cleaner way, perhaps using the existing AST?
//   - We curerntly use the `*Done` task to track information about the node in the graph that originated the associated
//     multi step completion, this is kind of nasty.
//   - When we remove a node we just disconnect it, should we actually just remove it?
//   - Currently when we add more than one task to the stack we also add the relevant site nodes to the graph if needed,
//     however this can be error prone since these nodes now appear in the graph and may be accessed during the next round
//     of prediction before it is intended (e.g this hhappens with terminal nodes, they will become part of the context tokens
//     once they are added and connected even if they are not filled yet, see `lexicalDecoder.argNext`).
func (ld lexicalDecoder) TaskCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (newRemaining EgTaskStack, err error) {
	ld.depth = len(remaining)

	ld.print("TaskCompleted: %v", task)
	ld.printStack(remaining)

	ld.saveTaskCompletedStart(eg, task, remaining)
	defer func() {
		ld.saveTaskCompletedEnd(eg, newRemaining, err)
		if err != nil {
			ld.print("=> err: %v", err)
			return
		}
		ld.depth++
		ld.printStack(newRemaining)
		ld.depth--
		ld.print("=> %d items on stack", len(newRemaining))
	}()

	switch task.Client.(ExpansionTask) {
	case lexicalGrammar.Expr:
		return ld.exprCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.ChooseTerminalType:
		return ld.chooseTerminalTypeCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.InferName:
		return ld.inferNameCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.InferAttr:
		return ld.inferAttrCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.AttrDone:
		return ld.attrDoneCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.CallDone:
		return ld.callDoneCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.ChooseExprType:
		return ld.chooseExprTypeCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.ChooseArgType:
		return ld.chooseArgTypeCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.InferKeywordArgName:
		return ld.inferKeywordArgNameCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.ArgDone:
		return ld.argDoneCompleted(ctx, eg, task, remaining)
	case lexicalGrammar.KeywordDone, lexicalGrammar.Propagate, lexicalGrammar.ExprDone, lexicalGrammar.Call:
		// nothing to do here
		return remaining, nil
	}

	panic(fmt.Sprintf("unhandled case, lastTask: %v", task))
}

func (ld lexicalDecoder) exprCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ exprCompleted")

	remaining = remaining.Push(EgTask{
		Type:   NoInferTask,
		Client: lexicalGrammar.ExprDone,
		Site:   task.Site,
	})

	return ld.chooseTerminalTypeNext(ctx, eg, task, remaining)
}

func (ld lexicalDecoder) chooseTerminalTypeNext(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ chooseTerminalTypeNext")

	var targets []int32
	prodData := []EgClientData{
		lexicalGrammar.InferName, // no placeholder = name
	}

	site := task.Site
	site.Attrs.Types = []string{traindata.ChooseTerminalTypeMarker}
	site.Attrs.Literal = traindata.ChooseTerminalTypeMarker

	// TODO: this is pretty nasty, we should make all these datasets consistent
	if arg := ld.astAncestorWithASTType(ctx, 1, eg, task.Site, argASTNode); !arg.Nil() {
		fi, err := ld.funcInfoBundle(ctx, eg, arg.Node)
		if err != nil {
			return nil, errors.Errorf("unable to get func info in choose terminal type: %v", err)
		}

		argName := fi.Kw
		if argName == "" {
			argName = fi.Info.Patterns.ArgName(fi.ArgIdx)
		}

		idxs := fi.Info.ArgPlaceholderIdxs[argName]
		if len(idxs) == 2 {
			targets = []int32{
				idxs[traindata.NoPlaceholder],
				idxs[traindata.Placeholder],
			}
			prodData = append(prodData, lexicalGrammar.Placeholder)

			if arg := fi.Info.Patterns.ArgsByName[argName]; arg != nil {
				if ts := arg.Types; len(ts) > 0 {
					site.Attrs.Types = nil
					for _, t := range ts[:1] {
						site.Attrs.Types = append(site.Attrs.Types, pythonimports.NewDottedPath(t.Path).Last())
					}
				}
			}
		} else {
			targets = ld.meta.ProductionIndex.MustGetIndices(
				ExpansionTaskID(lexicalGrammar.GenericNoPlaceholder),
			)
		}
	} else {
		targets = ld.meta.ProductionIndex.MustGetIndices(
			ExpansionTaskID(lexicalGrammar.GenericNoPlaceholder),
		)
	}

	tt := InferProductionTask
	if len(targets) == 1 {
		tt = NoInferTask
	}

	remaining = remaining.Push(EgTask{
		Type:             tt,
		Site:             site,
		Client:           lexicalGrammar.ChooseTerminalType,
		InferProdTargets: targets,
		InferProdClient:  prodData,
	})

	return remaining, nil
}

func (ld lexicalDecoder) chooseTerminalTypeCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ chooseTerminalTypeCompleted")
	if task.ChosenProdData().(ExpansionTask) == lexicalGrammar.Placeholder {
		task.Site.Attrs.ASTNodeType = nameASTNode
		task.Site.Attrs.Literal = PlaceholderPlaceholder
		return remaining, nil
	}
	return ld.inferNameNext(ctx, eg, task, remaining)
}

func (ld lexicalDecoder) inferNameNext(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ inferNameNext")

	site := task.Site
	site.Attrs.Literal = traindata.InferNameMarker
	site.Attrs.Types = []string{traindata.InferNameMarker}

	remaining = remaining.Push(EgTask{
		Type:   InferNameTask,
		Client: lexicalGrammar.InferName,
		Site:   site,
	})

	return remaining, nil
}

func (ld lexicalDecoder) inferNameCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ inferNameCompleted")

	// transition to the choose expr type task
	return ld.chooseExprTypeNext(ctx, eg, task, remaining)
}

// We essentially perform the equivalent of a "shift" op in a shift reduce parser, in particular:
//   - Push an `attrDone` task onto the stack and use the `EgTask.Site` to mark the original attribute expression.
//   - Create a new terminal ast node that is the child of the `lastSite`, this represents the attribute slot to fill
//   - Assign the type of the new node based on the type of base of the attribute. This is pretty hacky, but we need
//     to do this until we do the attribute grammar graph structure.
// NOTE:
//   - Because of the way `chooseExprTypeNext` works, `task.Site` is the node associated with the full `pythonast.AttributeExpr`.
//   - We include the `pythonresource.Symbol` for each candidate in the resulting node attributes because we need it to update
//     the parent attribute expression after this task is completed, can we do something cleaner?
func (ld lexicalDecoder) inferAttrNext(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ inferAttrNext")

	lastSite := task.Site

	var origAttr *Node
	for _, n := range eg.Outgoing(lastSite) {
		if n.Type != ASTChild {
			continue
		}
		nd := n.Node.Attrs.Client.(NodeData)
		if n.Node.Type == ASTTerminalNode && !strings.Contains(n.Node.Attrs.Literal, ".") && nd.ASTParentField == "" {
			origAttr = n.Node
			break
		}
	}
	var prefix string
	if origAttr != nil {
		prefix = origAttr.Attrs.Literal
	}

	// this is the node associated with the `pythonast.AttributeExpr.Attribute` node
	nextSite := eg.AddNode(ASTTerminalNode, Attributes{
		Literal: traindata.InferAttrMarker,
		Types:   lastSite.Attrs.Types,
		Client:  NodeData{},
	}, nil)

	eg.AddEdge(lastSite, nextSite, ASTChild)

	// store a pointer to the `nextSite` in the `attrDone` task so we can
	// use it to update the parent attribute expr in `lexicalDecoder.attrDoneCompleted`
	// based on the result of the chosen attribute expression
	remaining = remaining.Push(EgTask{
		Type:        NoInferTask,
		Client:      lexicalGrammar.AttrDone,
		Site:        lastSite,
		ClientNodes: []*Node{nextSite},
	})

	// get the base of the attribute expression and get the appropriate decoder targets
	attrBase := ld.astChildForField(eg, lastSite, "Value")
	syms := ld.symbols(ctx, attrBase.Node.Attrs.values)
	if len(syms) == 0 {
		return nil, errors.Errorf("unable to resolve attr base %v", attrBase)
	}

	var targets []int32
	var cands []pythonresource.Symbol
	for _, sym := range syms {
		var err error
		targets, cands, err = ld.cbs.Attr.Candidates(ld.rm, sym)
		if err == nil {
			break
		}
	}
	var filteredTargets []int32
	var filteredCands []pythonresource.Symbol
	for i, c := range cands {
		if strings.HasPrefix(c.Canonical().PathLast(), prefix) {
			filteredCands = append(filteredCands, c)
			filteredTargets = append(filteredTargets, targets[i])
		}
	}

	if len(filteredTargets) == 0 {
		return nil, errors.Errorf("no supported types %v", ld.symsString(syms))
	}

	prodData := make([]EgClientData, 0, len(filteredCands))
	for _, cand := range filteredCands {
		prodData = append(prodData, inferProdAttr(cand))
	}

	remaining = remaining.Push(EgTask{
		Type:             InferProductionTask,
		Client:           lexicalGrammar.InferAttr,
		Site:             nextSite,
		InferProdTargets: filteredTargets,
		InferProdClient:  prodData,
	})

	return remaining, nil
}

func (ld lexicalDecoder) inferAttrCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ inferAttrCompleted")

	// NOTE: the site associated with the `attrDone` task is the location
	// of the associated `pythonast.AttributeExpr` in the graph (see `lexicalDecoder.inferAttrNext`),
	// - we need to update the site with the chosen attribute data
	// - transition to choose expr type
	sym := pythonresource.Symbol(task.ChosenProdData().(inferProdAttr))
	task.Site.Attrs.Literal = sym.Path().Last()
	task.Site.Attrs.Types = []string{traindata.NAType}

	nd := task.Site.Attrs.Client.(NodeData)
	nd.Symbol = sym
	task.Site.Attrs.Client = nd

	return remaining, nil
}

// NOTE:
//   - `task.Site` associated with the `attrDone` task is the location
//     of the associated `pythonast.AttributeExpr` in the graph so we
//     can send this into `ld.ChooseExprType` next.
//   - We need to update `site.Attrs.Types` based on the attribute
//     that was actually chosen. This is kind of gross and can be removed once we figure out
//     the attribute grammar graph structure. see `lexicalDecoder.inferAttrNext`.
func (ld lexicalDecoder) attrDoneCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ attrDoneCompleted")

	sym := task.ClientNodes[0].Attrs.Client.(NodeData).Symbol
	site := task.Site
	site.Attrs.values = []pythontype.GlobalValue{pythontype.NewExternal(sym, ld.rm)}
	site.Attrs.Types = []string{typeString(site.Attrs.values[0], true)}

	return ld.chooseExprTypeNext(ctx, eg, task, remaining)
}

// We essentially perform the equivalent of a "reduce" op in a shift reduce parser, in particular:
//   - Create a new ASTInternal node that is the parent of the last site and an AST child for
//     the the parent of the last site.
//   - Assign the types of the last site to the attributes of the new internal node.
//     This is pretty hacky but we need to do this because otherwise the inference task will not
//     be able to condition on the base of the attribute expression or the func of a call properly.
//     We can do this in a more principled manner once we use the attribute grammar graph structure.
func (ld lexicalDecoder) chooseExprTypeNext(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ chooseExprTypeNext")

	lastSite := task.Site

	// need to make sure we copy the ast parent info to leave
	// the graph in a consistent state.
	nextSite := eg.AddNode(ASTInternalNode, Attributes{
		ASTNodeType: traindata.InferExprTypeMarker,
		Types:       lastSite.Attrs.Types,
		Literal:     traindata.InferExprTypeMarker,
		Client: NodeData{
			ASTParentField: lastSite.Attrs.Client.(NodeData).ASTParentField,
			ASTParentPos:   lastSite.Attrs.Client.(NodeData).ASTParentPos,
		},
	}, nil)

	// remove the old ast edge from the parent of last site to last site
	parent := ld.mustASTParent(eg, lastSite)
	eg.RemoveEdge(parent.Node, lastSite, parent.Type)

	// parent of last site is now the ast parent of the next site
	eg.AddEdge(parent.Node, nextSite, ASTChild)
	// next site is now the ast parent of the last site
	eg.AddEdge(nextSite, lastSite, ASTChild)

	var targets []int32
	var prodData []EgClientData
	for _, v := range lastSite.Attrs.values {
		if v.Kind() == pythontype.FunctionKind {
			// TODO: this is just for backwards compatibility
			targets = ld.meta.ProductionIndex.MustGetIndices(
				ExpansionTaskID(lexicalGrammar.Call),
			)
			prodData = append(prodData, lexicalGrammar.Call)
			break
		}
	}

	if len(targets) == 0 {
		targets = ld.meta.ProductionIndex.MustGetIndices(
			ExpansionTaskID(lexicalGrammar.Stop),
		)
		prodData = append(prodData, lexicalGrammar.Stop)
	}

	tt := InferProductionTask
	if len(targets) == 1 {
		tt = NoInferTask
	}

	return remaining.Push(EgTask{
		Type:             tt,
		Client:           lexicalGrammar.ChooseExprType,
		Site:             nextSite,
		InferProdTargets: targets,
		InferProdClient:  prodData,
		// keep a pointer to the `lastSite` so that we can update it's ast parent info in `lexicalDecoder.chooseExprTypeNext`
		ClientNodes: []*Node{lastSite},
	}), nil
}

func (ld lexicalDecoder) chooseExprTypeCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ chooseExprTypeCompleted")

	site := task.Site
	child := task.ClientNodes[0]
	childNodeData := child.Attrs.Client.(NodeData)

	switch task.ChosenProdData().(ExpansionTask) {
	case lexicalGrammar.Stop:
		// remove the stop node and re-connect the old ast parent to the old ast child,
		// SEE: `lexicalDecoder.chooseExprTypeNext`
		parent := ld.mustASTParent(eg, site)

		eg.RemoveEdge(parent.Node, site, ASTChild)
		eg.RemoveEdge(site, child, ASTChild)
		eg.AddEdge(parent.Node, child, ASTChild)

		// hacky, we need to reset the ast parent info
		childNodeData.ASTParentField = site.Attrs.Client.(NodeData).ASTParentField
		childNodeData.ASTParentPos = site.Attrs.Client.(NodeData).ASTParentPos
		child.Attrs.Client = childNodeData

		// NOTE: we need to pop the "done" task off the stack
		// to avoid loops since when we hit a `callDone` or an `attrDone`
		// above we push a `chooseExpr` task onto the stack.
		_, remaining = remaining.Pop()
		return remaining, nil
	case lexicalGrammar.Attr:
		// NOTE: need to update the node for the attribute base
		// with the correct ast child information.
		childNodeData.ASTParentField = "Value"
		child.Attrs.Client = childNodeData

		return ld.inferAttrNext(ctx, eg, task, remaining)
	case lexicalGrammar.Call:
		// NOTE: need to update the node for the `Func`
		// with the correct ast child information.
		childNodeData.ASTParentField = "Func"
		child.Attrs.Client = childNodeData
		site.Attrs.ASTNodeType = callASTNode
		newNode := eg.AddNode(ASTTerminalNode, Attributes{
			Literal: traindata.WordLiteral(pythonscanner.Word{Token: pythonscanner.Lparen}),
			Types:   []string{traindata.NAType},
			Client: NodeData{
				ASTParentField: "LeftParen",
				ASTParentPos:   0,
			},
		}, nil)
		eg.AddEdge(site, newNode, ASTChild)
		// NOTE: the last site is now the node associated with the
		// pythonast.Callexpr, so we push a `callDone` node onto the stack to mark the location
		// in the graph of the original call.
		remaining = remaining.Push(EgTask{
			Type:   NoInferTask,
			Client: lexicalGrammar.CallDone,
			Site:   site,
		})
		var err error
		remaining, err = ld.chooseArgTypeNext(ctx, eg, task, remaining)
		if err != nil {
			return nil, err
		}
		remaining = remaining.Push(EgTask{
			Type:   PropagateTask,
			Site:   newNode,
			Client: lexicalGrammar.Propagate,
		})
		remaining = remaining.Push(EgTask{
			Type:   NoInferTask,
			Client: lexicalGrammar.Call,
			Site:   site,
		})
		return remaining, err
	default:
		panic(fmt.Sprintf("unsupported choose result %v", task.ChosenProdData().(ExpansionTask)))
	}
}

func (ld lexicalDecoder) callDoneCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ callDoneCompleted")

	// NOTE: the site associated with the `callDone` task is the location
	// of the associated `pythonast.CallExpr` in the graph (see `call` case below) so we can transition
	// directly to `chooseExprTypeNext`.
	return ld.chooseExprTypeNext(ctx, eg, task, remaining)
}

// Prepare for a `chooseArgType` prediction:
//   - Push an `argDone` task onto the stack, with the `task.Site` (the call node) as the site to mark the call in the expansion graph.
//   - Create a new argument node
// NOTE:
//   - this assumes that we always complete the last argument in the call
func (ld lexicalDecoder) chooseArgTypeNext(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ chooseArgTypeNext")

	callNode := task.Site
	remaining = remaining.Push(EgTask{
		Type:   NoInferTask,
		Client: lexicalGrammar.ArgDone,
		Site:   callNode,
	})

	// needs to match `lexicalDecoder.funcInfoBundle`,
	// ASTParentPos is set below
	nd := NodeData{
		ASTParentField: "Args",
	}

	// create a new argument node, client data set below
	argNode := eg.AddNode(ASTInternalNode, Attributes{
		ASTNodeType: argASTNode,
		Types:       []string{traindata.InferArgTypeMarker},
		Literal:     traindata.InferArgTypeMarker,
		Client:      nd,
	}, nil)

	eg.AddEdge(callNode, argNode, ASTChild)

	fi, err := ld.funcInfoBundle(ctx, eg, argNode)
	if err != nil {
		ld.print("unable to find func info: %v", err)
		return nil, errors.Errorf("unable to find func info: %v", err)
	}

	// -1 because we added an arg node above
	numArgs := fi.NumArgs - 1

	nd.ASTParentPos = numArgs
	argNode.Attrs.Client = nd

	// stop is always allowed
	prodData := []EgClientData{
		lexicalGrammar.Stop,
	}
	targets := []int32{
		fi.Info.ArgTypeIdxs[traindata.Stop],
	}

	if fi.KeywordAllowed() {
		prodData = append(prodData, lexicalGrammar.Keyword)
		targets = append(targets, fi.Info.ArgTypeIdxs[traindata.Keyword])
	}

	if fi.PositionalAllowed() {
		prodData = append(prodData, lexicalGrammar.Positional)
		targets = append(targets, fi.Info.ArgTypeIdxs[traindata.Positional])
	}

	tt := InferProductionTask
	if len(targets) == 1 {
		// only one target, so just make this a no infer task
		tt = NoInferTask
	}

	remaining = remaining.Push(EgTask{
		Type:             tt,
		Client:           lexicalGrammar.ChooseArgType,
		Site:             argNode,
		InferProdTargets: targets,
		InferProdClient:  prodData,
	})

	// now handle next token edges to the arg
	// TODO: this is for backwards compatibility since when we generated
	// the old training data this is what we did

	if numArgs == 0 {
		lp := ld.astChildForField(eg, callNode, "LeftParen")
		eg.AddEdge(lp.Node, argNode, NextToken)
	} else {
		comma := eg.AddNode(ASTTerminalNode, Attributes{
			Literal: traindata.WordLiteral(pythonscanner.Word{Token: pythonscanner.Comma}),
			Types:   []string{traindata.NAType},
			Client: NodeData{
				ASTParentField: "Commas",
				ASTParentPos:   numArgs - 1,
			},
		}, nil)

		eg.AddEdge(callNode, comma, ASTChild)
		eg.AddEdge(comma, argNode, NextToken)

		prevArg := ld.astChildFor(eg, callNode, "Args", numArgs-1)

		for _, n := range eg.Outgoing(prevArg.Node) {
			if n.Type == ASTChild && n.Node.Type == ASTTerminalNode {
				eg.AddEdge(n.Node, comma, NextToken)
			}
		}

		remaining = remaining.Push(EgTask{
			Type:   PropagateTask,
			Site:   comma,
			Client: lexicalGrammar.Propagate,
		})

	}

	return remaining, nil
}

func (ld lexicalDecoder) chooseArgTypeCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ chooseArgTypeCompleted")

	res := task.ChosenProdData().(ExpansionTask)
	if res == lexicalGrammar.Stop {
		// remove the stop node, needs to match `lexicalDecoder.chooseArgTypeNext`
		parent := ld.mustASTParent(eg, task.Site)
		eg.RemoveEdge(parent.Node, task.Site, parent.Type)

		// NOTE: we need to pop the "argDone" task off the stack
		// to avoid loops since when we hit a `argDone`
		// below we push a `chooseArgType` task onto the stack.
		_, remaining = remaining.Pop()
		return remaining, nil
	}

	// the last site is the node associated with `pythonast.Argument`,
	// see `lexicalDecoder.chooseArgTypeNext`
	task.Site.Attrs.Literal = argASTNode
	task.Site.Attrs.Literal = ""
	task.Site.Attrs.Types = []string{traindata.NAType}

	// remove the old next token edge that went into the argument site
	var prevToken *Node
	for _, n := range eg.Incoming(task.Site) {
		if n.Type == NextToken {
			prevToken = n.Node
			eg.RemoveEdge(n.Node, task.Site, NextToken)
			break
		}
	}

	return ld.argNext(ctx, eg, remaining, task.Site, prevToken, res)
}

// Prepare the graph for predicting a positional or keyword argument and then it's value:
//   - attach a child ast node to the argument for the value and push it onto the stack
//   - if needed, attach a child ast node to the argument for the keyword
// NOTE:
//   - we use a `ASTTerminalNode` marker for the argument value since this cannot be updated once it is set
//     and we use the `Node.Type` field when computing the context tokens.
//   - we have to set the types/literal fields of the Value node to something reasonable since it will be included
//     in the keyword task via the token edges.
//   - we currently attach the `prevToken` to either the keyword node or the value node, if it is the value node then this
//     is not quite right.
func (ld lexicalDecoder) argNext(ctx kitectx.Context, eg ExpansionGraphMutate, remaining EgTaskStack, argNode, prevToken *Node, t ExpansionTask) (EgTaskStack, error) {
	ld = ld.trace("@ argNext")

	fi, err := ld.funcInfoBundle(ctx, eg, argNode)
	if err != nil {
		return nil, errors.Errorf("unable to find func info: %v", err)
	}

	valueNode := eg.AddNode(ASTTerminalNode, Attributes{
		Literal: traindata.UnknownTokenMarker,
		Types:   []string{traindata.UnknownType},
		Client:  NodeData{ASTParentField: "Value"}, // needs to match `lexicalDecoder.funcInfoBundle`
	}, nil)

	eg.AddEdge(argNode, valueNode, ASTChild)

	remaining = remaining.Push(EgTask{
		Type:   NoInferTask,
		Client: lexicalGrammar.Expr,
		Site:   valueNode,
	})

	if t == lexicalGrammar.Positional {
		eg.AddEdge(prevToken, valueNode, NextToken)
		return remaining, nil
	}

	keywordNode := eg.AddNode(ASTTerminalNode, Attributes{
		Literal: traindata.InferKwargNameMarker,
		Types:   []string{traindata.InferKwargNameMarker},
		Client: NodeData{
			ASTParentField: "Name", // needs to match `lexicalDecoder.funcInfoBundle`
		},
	}, nil)

	eg.AddEdge(argNode, keywordNode, ASTChild)
	eg.AddEdge(prevToken, keywordNode, NextToken)

	remaining = remaining.Push(EgTask{
		Type:   NoInferTask,
		Client: lexicalGrammar.KeywordDone,
		Site:   keywordNode,
	})

	var targets []int32
	var prodData []EgClientData
	for _, ni := range fi.Info.KwargNameIdxs {
		if fi.Seen[ni.Name] {
			continue
		}
		targets = append(targets, ni.Idx)
		prodData = append(prodData, inferProdKeyword(ni.Name))
	}

	remaining = remaining.Push(EgTask{
		Type:             InferProductionTask,
		Client:           lexicalGrammar.InferKeywordArgName,
		Site:             keywordNode,
		InferProdTargets: targets,
		InferProdClient:  prodData,
	})

	return remaining, nil
}

func (ld lexicalDecoder) argDoneCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ argDoneCompleted")

	// we just finished an argument, push a new `chooseArgType` task onto the graph.
	// NOTE: the `callNode` is stored as the `EgTask.Site` for the `argDone` task,
	// SEE: `ld.chooseArgTypeNext`.
	return ld.chooseArgTypeNext(ctx, eg, task, remaining)
}

func (ld lexicalDecoder) inferKeywordArgNameCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) (EgTaskStack, error) {
	ld = ld.trace("@ inferKeywordArgNameCompleted")

	// we just finished inferring a keyword, assign the chosen keyword
	// to the site node, the value is already prepared so we just return the remaining stack
	// SEE: `lexicalDecoder.inferKeywordNext`
	task.Site.Attrs.ASTNodeType = nameASTNode
	task.Site.Attrs.Types = []string{traindata.NAType}
	task.Site.Attrs.Literal = string(task.ChosenProdData().(inferProdKeyword))

	return remaining, nil
}

func (ld lexicalDecoder) InferNameDecoderEmbeddings(ctx kitectx.Context, eg ExpansionGraphNavigate, site *Node) ([]string, []string, error) {
	ld = ld.trace("@ InferNameDecoderEmbeddings")
	defer func() {
		ld.depth--
		ld.print("DONE InferNameDecoderEmbeddings")
	}()

	arg := ld.astAncestorWithASTType(ctx, 1, eg, site, argASTNode)
	if arg.Nil() {
		// just a normal name expression so use generic decoder
		return []string{traindata.AttrBaseNameDecoder}, []string{traindata.AttrBaseNameDecoder}, nil
	}

	fi, err := ld.funcInfoBundle(ctx, eg, arg.Node)
	if err != nil {
		return nil, nil, err
	}

	types, toks := fi.Info.Patterns.Feed(fi.Kw, fi.ArgIdx)
	return types, toks, nil
}

type funcInfoBundle struct {
	ArgIdx  int
	Kw      string
	Call    NeighborInfo
	Info    *FuncInfo
	Seen    map[string]bool
	NumArgs int
}

func (f *funcInfoBundle) PositionalAllowed() bool {
	if len(f.Seen) > 0 {
		return false
	}

	return f.NumArgs < f.Info.Patterns.MaxNumArgs() && f.Info.Patterns.PositionalOK(f.ArgIdx)
}

func (f *funcInfoBundle) KeywordAllowed() bool {
	var remaining int
	for _, kw := range f.Info.KwargNameIdxs {
		if !f.Seen[kw.Name] {
			remaining++
		}
	}
	return remaining > 0
}

func (ld lexicalDecoder) funcInfoBundle(ctx kitectx.Context, eg ExpansionGraphNavigate, argNode *Node) (*funcInfoBundle, error) {
	call := ld.astAncestorWithASTType(ctx, 1, eg, argNode, callASTNode)
	if call.Nil() {
		return nil, errors.Errorf("unable to find parent call of %v", argNode)
	}

	fn := ld.astChildForField(eg, call.Node, "Func")
	if fn.Nil() {
		return nil, errors.Errorf("unable to find func for call %v", call)
	}

	syms := ld.symbols(ctx, fn.Node.Attrs.values)
	if len(syms) == 0 {
		return nil, errors.Errorf("unable to resolve func %v", fn)
	}
	ld.print("found symbols: %v", ld.symsString(syms))

	var kw string
	var argIdx int
	var numArgs int
	seen := make(map[string]bool)
	for _, n := range eg.Outgoing(call.Node) {
		if n.Type != ASTChild || n.Node.Attrs.Client.(NodeData).ASTParentField != "Args" {
			continue
		}
		numArgs++
		if n.Node == argNode {
			argIdx = n.Node.Attrs.Client.(NodeData).ASTParentPos
		}
		if name := ld.astChildForField(eg, n.Node, "Name"); !name.Nil() {
			seen[name.Node.Attrs.Literal] = true
			if n.Node == argNode {
				kw = name.Node.Attrs.Literal
			}
		}
	}

	sortSymbolByPopularity(syms, ld.rm)
	var info *FuncInfo
	for i := range syms {
		// The training is done with the last available symbol
		// We iterate over syms in reverse order to try the last symbol first
		s := syms[i]
		var err error
		info, err = ld.cbs.Call.Info(ld.rm, s)
		if err != nil {
			ld.print("error looking up func info for %s: %v", s, err)
		}
		if info != nil {
			break
		}
	}

	if info != nil {
		return &funcInfoBundle{
			ArgIdx:  argIdx,
			Kw:      kw,
			Call:    call,
			Info:    info,
			Seen:    seen,
			NumArgs: numArgs,
		}, nil
	}

	return nil, errors.Errorf("unable to find valid func info for func %v", fn.Node)
}

func (ld lexicalDecoder) astChildForField(eg ExpansionGraphNavigate, n *Node, field string) NeighborInfo {
	egCandidate := NeighborInfo{}
	for _, n := range eg.Outgoing(n) {
		if n.Type == ASTChild && n.Node.Attrs.Client.(NodeData).ASTParentField == field {
			if !eg.IsEGNode(n.Node) {
				return n
			}
			egCandidate = n
		}
	}
	return egCandidate
}

func (ld lexicalDecoder) astChildFor(eg ExpansionGraphNavigate, n *Node, field string, pos int) NeighborInfo {
	for _, n := range eg.Outgoing(n) {
		if n.Type != ASTChild {
			continue
		}

		nd := n.Node.Attrs.Client.(NodeData)
		if nd.ASTParentField != field {
			continue
		}

		if pos > -1 && nd.ASTParentPos != pos {
			continue
		}
		return n
	}
	return NeighborInfo{}
}

func (ld lexicalDecoder) mustASTParent(eg ExpansionGraphNavigate, n *Node) NeighborInfo {
	for _, p := range eg.Incoming(n) {
		if p.Type == ASTChild {
			return p
		}
	}
	ld.print("unable to find ast parent for %v")
	panic(fmt.Sprintf("unable to find ast parent %v", n))
}

func (ld lexicalDecoder) astDescendentWithASTType(ctx kitectx.Context, steps int, eg ExpansionGraphNavigate, n *Node, astType string) NeighborInfo {
	return ld.recurASTType(ctx, steps, n, astType, eg.Outgoing)
}

func (ld lexicalDecoder) astAncestorWithASTType(ctx kitectx.Context, steps int, eg ExpansionGraphNavigate, n *Node, astType string) NeighborInfo {
	return ld.recurASTType(ctx, steps, n, astType, eg.Incoming)
}

func (ld lexicalDecoder) recurASTType(ctx kitectx.Context, steps int, n *Node, astType string, neighborFn func(*Node) NeighborSet) NeighborInfo {
	var recur func(int, NeighborInfo) NeighborInfo
	recur = func(count int, ne NeighborInfo) NeighborInfo {
		ctx.CheckAbort()

		if steps > 0 && count == 0 {
			return NeighborInfo{}
		}
		count--

		switch {
		case ne.Nil() || ne.Type != ASTChild:
			return NeighborInfo{}
		case ne.Node.Attrs.ASTNodeType == astType:
			return ne
		default:
			for _, nn := range neighborFn(ne.Node) {
				if found := recur(count, nn); found.Node != nil {
					return found
				}
			}
			return NeighborInfo{}
		}
	}

	// NOTE:
	//   - need +1 because the first step just checks the current node
	//   - set the neighbor to be an ast node so we do not just stop the recursion immediately
	return recur(steps+1, NeighborInfo{Node: n, Type: ASTChild})
}

func (ld lexicalDecoder) symbols(ctx kitectx.Context, vs []pythontype.GlobalValue) []pythonresource.Symbol {
	syms := make([]pythonresource.Symbol, 0, len(vs))

	for _, v := range vs {
		if sym := symbolFor(v); !sym.Nil() {
			syms = append(syms, sym)
		}
	}
	return syms
}

func (ld lexicalDecoder) trace(fmtstr string, args ...interface{}) lexicalDecoder {
	ld.depth++
	ld.print(fmtstr, args...)
	ld.depth++
	return ld
}

func (ld lexicalDecoder) print(fmtstr string, args ...interface{}) {
	if ld.t == nil {
		return
	}

	parts := strings.Split(fmtstr, "\n")
	for i, part := range parts {
		parts[i] = strings.Repeat("  ", ld.depth) + part
	}
	fmtstr = strings.Join(parts, "\n")
	print(ld.t, fmtstr, args...)
}

func (ld lexicalDecoder) symsString(syms []pythonresource.Symbol) string {
	parts := make([]string, 0, len(syms))
	for _, s := range syms {
		parts = append(parts, s.String())
	}
	return strings.Join(parts, " | ")
}

func (ld lexicalDecoder) printStack(remaining EgTaskStack) {
	if ld.t == nil {
		return
	}

	ld.print("STACK HEAD")
	ld.print("****")
	for _, r := range remaining {
		ld.print("- %v", r)
	}
	ld.print("****")
}

func (ld lexicalDecoder) saveTaskCompletedStart(eg ExpansionGraphMutate, task EgTask, remaining EgTaskStack) {
	if ld.saver == nil {
		return
	}

	sg, labels := eg.SavedGraph(task.Site)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Completed: %v\n", task)
	fmt.Fprintln(&buf, "Remaining stack")
	for _, r := range remaining {
		fmt.Fprintf(&buf, "- %v\n", r)
	}
	fmt.Fprintln(&buf, "****")

	save(ld.saver, SavedBundle{
		Label:      fmt.Sprintf("lexical-decoder-task-completed-%v-start", task.Client),
		Graph:      sg,
		NodeLabels: labels,
		Buffer:     buf.Bytes(),
		AST:        ld.ast,
	})
}

func (ld lexicalDecoder) saveTaskCompletedEnd(eg ExpansionGraphMutate, newStack EgTaskStack, err error) {
	if ld.saver == nil {
		return
	}

	task := newStack.Peek()
	label := fmt.Sprintf("lexical-decoder-task-completed-%v-end", task.Client)
	if err != nil {
		save(ld.saver, SavedBundle{
			Label:  label,
			Buffer: []byte(fmt.Sprintf("ERROR: %v", err)),
		})
		return
	}

	sg, labels := eg.SavedGraph(task.Site)

	var buf bytes.Buffer
	fmt.Fprintln(&buf, "NewStack stack")
	for _, r := range newStack {
		fmt.Fprintf(&buf, "- %v\n", r)
	}
	fmt.Fprintln(&buf, "****")

	save(ld.saver, SavedBundle{
		Label:      label,
		Graph:      sg,
		NodeLabels: labels,
		Buffer:     buf.Bytes(),
		AST:        ld.ast,
	})
}
