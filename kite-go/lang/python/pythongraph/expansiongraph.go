package pythongraph

import (
	"fmt"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// EgTaskType ...
type EgTaskType string

const (
	// InferNameTask ...
	InferNameTask = EgTaskType("infer_name_task")
	// InferNameTaskCompleted ...
	InferNameTaskCompleted = EgTaskType("infer_name_task_completed")
	// InferProductionTask ...
	InferProductionTask = EgTaskType("infer_production_task")
	// InferProductionTaskCompleted ...
	InferProductionTaskCompleted = EgTaskType("infer_production_task_completed")
	// PropagateTask runs a single round of propagation
	PropagateTask = EgTaskType("propagate_task")
	// PropagateTaskCompleted ...
	PropagateTaskCompleted = EgTaskType("propagate_task_completed")
	// NoInferTask is a special task in which no infernece is performed and
	// instead the task is simply popped off the stack and `EgCallbacks.TaskCompleted`
	// is called.
	NoInferTask = EgTaskType("no_infer_task")
	// NoInferTaskCompleted ...
	NoInferTaskCompleted = EgTaskType("no_infer_task_completed")
)

// Completed ...
func (e EgTaskType) Completed() bool {
	switch e {
	case InferNameTaskCompleted, InferProductionTaskCompleted,
		PropagateTaskCompleted, NoInferTaskCompleted:
		return true
	default:
		return false
	}
}

// EgTask is a task to perform (see below).
type EgTask struct {
	Type EgTaskType

	// Site for the prediciton task, the state of this node
	// is always determined by looking up its type/subtokens
	// and then propagating to the node.
	Site *Node

	// InferProdTargets if this is an InferProductionTask
	InferProdTargets []int32

	// InferProdClient is used by the client to store meta information
	// for the result of each infer prod target. The client can leave this empty
	// if desired.
	InferProdClient []EgClientData

	// InferProdChosen is the index of the result of the infer prod task,
	// we will always have `0 <= InferProdChosen < len(InferProdTargets)`
	InferProdChosen int

	// Client data that is sent along with the task
	Client EgClientData

	// ClientNodes are nodes the client can store in the task
	// that can then be examined when the task is completed.
	ClientNodes []*Node
}

// ChosenProdData returns the production data associated with the infer prod result
func (e EgTask) ChosenProdData() EgClientData {
	return e.InferProdClient[e.InferProdChosen]
}

// String ...
func (e EgTask) String() string {
	return fmt.Sprintf("Type: %v, Client: %v, Site: %v", e.Type, e.Client, e.Site)
}

func (e EgTask) deepCopy(oldToNew func(*Node) *Node) EgTask {
	ee := EgTask{
		Type:             e.Type,
		Site:             oldToNew(e.Site),
		InferProdTargets: e.InferProdTargets,
		InferProdClient:  e.InferProdClient,
		InferProdChosen:  e.InferProdChosen,
		Client:           e.Client,
		ClientNodes:      make([]*Node, 0, len(e.ClientNodes)),
	}

	for _, old := range e.ClientNodes {
		ee.ClientNodes = append(ee.ClientNodes, oldToNew(old))
	}
	return ee
}

// EgTaskStack is a `stack` of `EgTask`s.
type EgTaskStack []EgTask

func (e EgTaskStack) deepCopy(oldToNew func(*Node) *Node) EgTaskStack {
	ee := make(EgTaskStack, 0, len(e))
	for _, t := range e {
		ee = append(ee, t.deepCopy(oldToNew))
	}
	return ee
}

// Empty ...
func (e EgTaskStack) Empty() bool {
	return len(e) == 0
}

// Push ...
func (e EgTaskStack) Push(t EgTask) EgTaskStack {
	return append(EgTaskStack{t}, e...)
}

// Pop ...
func (e EgTaskStack) Pop() (EgTask, EgTaskStack) {
	t := e[0]
	if len(e) == 1 {
		return t, nil
	}
	return t, e[1:]
}

// Peek ...
func (e EgTaskStack) Peek() EgTask {
	if e.Empty() {
		return EgTask{}
	}
	return e[0]
}

// ExpansionGraphNavigate api encapsulates the api that clients can use
// to navigate the current `ExpansionGraph`.
// NOTES:
//   - Clients should NEVER modify the values returned by this API.
type ExpansionGraphNavigate interface {
	SavedGraph(*Node) (*SavedGraph, NodeLabels)
	// Incoming returns the set of nodes that have an outgoing edge that points to
	// `n`, e.g these are the edges incoming to `n`.
	// NOTE:
	//   - The returned nodes can be from the expansion graph or the context graph.
	//   - Clients should NOT modify the resulting set or the neighbors within the set.
	Incoming(n *Node) NeighborSet

	// Outgoing returns the set of nodes that have an incoming edge from
	// `n`, e.g these are the edges outgoing from `n`.
	// NOTE:
	//   - The returned nodes can be from the expansion graph or the context graph.
	//   - Clients should NOT modify the resulting set or the neighbors within the set.
	Outgoing(n *Node) NeighborSet

	// IsEGNode test if a node is in the ExpansionGraph (true) or the ContextGraph (false)
	IsEGNode(n *Node) bool
}

// ExpansionGraphMutate api encapsulates the api that clients can use
// to mutate the current `ExpansionGraph`.
// NOTES:
//   - Clients should never call this API from multiple go routines
type ExpansionGraphMutate interface {
	ExpansionGraphNavigate
	// AddEdge to the graph.
	// NOTE:
	//   - This is NOT safe to call from multiple go routines.
	AddEdge(from, to *Node, t EdgeType)

	// AddNode to the graph.
	// NOTE:
	//   - This is NOT safe to call from multiple go routines.
	//   - The initial state of the node embedding is specified by `initState`, if it has length 0 (or is nil)
	//     then the node state is initialized to the all 0 state.
	AddNode(t NodeType, attrs Attributes, initState []float32) *Node

	// RemoveEdge from the graph.
	// NOTE:
	//   - This is NOT safe to call from multiple go routines.
	//   - If the edge is not found then the graph is not modified.
	RemoveEdge(from, to *Node, t EdgeType)
}

// EgCallbacks provide the main mechanism for client's to prepare the graph for inference and inject relevant parameters.
// To navigate the current `ExpansionGraph` clients should use the `ExpansionGraphNavigate` api.
// To modify the current `ExpansionGraph` clients should use the `ExpansionGraphMutate` api.
// The `EgCallbacks` are used in conjunction with the `EgUpdate`s (see below).
// NOTES:
//   - Implementors of `EgCallbacks` should NEVER keep pointers to the `ExpansionGraph` that a callback function was invoked with.
type EgCallbacks interface {
	// TaskCompleted is called after an inference task has been completed,
	//   - `ctx` is the context that was called with `EgUpdate.Expand`.
	// 	 - `eg` is a pointer to the updated expansion graph that results from
	//     applying the prediction at `site`, clients can inspect and modify this as needed.
	//   - `t` is the task that was completed. `t.Site` is the prediction site node, if the task was an infer
	//    name task then the attributes of this node will be set to match the chosen variable,
	//    if the task was an infer production task then this node will be unmodified and clients should update its attributes
	//    as desired.
	//   - `remaining` tasks on the stack.
	// The return values are:
	//   - The new set of tasks (`EgTaskStack`) that will be used to create a new `EgUpdate` (along with the modified graph)
	//     that will be returned from the call to `EgUpdate.Expand`. To simply continue with the existing stack
	//     of completions clients should return `remaining`, to stop inference immediately with no errors clients should return nil.
	//   - An error if inference should stop immediately.
	TaskCompleted(ctx kitectx.Context, eg ExpansionGraphMutate, t EgTask, remaining EgTaskStack) (EgTaskStack, error)
	// InfernameDecoderEmbeddings is called at a particular prediction `site` in the graph,
	// in order to get the appropriate name decoder embeddings.
	// The return values are:
	//   - The `[]string` types representing the types in the name decoder embedding
	//   - The `[]string` subtokens representing the subtokens in the name decoder embedding.
	//   - An error if inference should stop immediately.
	// NOTE:
	//   - If either the returned types or subtokens `[]string` is empty for the particular site then the underlying `EgUpdate.Expand`
	//     will return an error and no inference will be performed (because it is not clear what the appropriate state transition should be).
	InferNameDecoderEmbeddings(ctx kitectx.Context, eg ExpansionGraphNavigate, site *Node) ([]string, []string, error)
}

// EgUpdate encapsulates an update to a base expansion graph along with information required to perform another round of inference
// using the updated graph.
// The basic lifecycle of inference works as follows:
//   1) `EgUpdate.Expand()` is called to perform a round of inference, a slice of updated `EgUpdate`s (call theses `updates`) are returned,
//      each corresponds to a possible expansion of the underlying expansion graph. During the course of performing inference
//      `EgCallbacks.InferNameDecoderEmbeddings` may be called if this is an infer name task.
//   2) For each returned `EgUpdate` in `updates` (from 1) the client can access the state of the prediction `site` node via
//      `EgUpdate.Peek().Node`.
//   3) If the client calls `updates[i].Expand()` then `EgCallbacks.TaskCompleted` will be called next
//      with the updated expansion graph, the updated prediction site where inference was performed and the appropriate task.
//      Clients should use this callback to inspect/modify the updated graph and return a NEW task stack that contains the tasks that should be performed next.
//      The new task stack along with the modified graph will be used to create a new `EgUpdate` that is returned from the call to `Expand`.
//   4) Repeat 1) - 3) until the underlying task stack in `EgUpdate` is empty in which case the call to `Expand` will return `nil,nil`.
// NOTE:
//   - For a particular `EgUpdate` it is never safe to call `Expand` from multiple go routines, however each call to `Expand` will return
//     a new set of `EgUpdates` (`updates`) which are deep copies of the original `EgUpdate` (with the appropriate updates) and it is safe to call
//     `updates[i].Expand()` and `updates[j].Expand()` from multiple go routines for `i != j`.
//   - Implementors of `EgCallbacks` should NEVER keep pointers to the `ExpansionGraph` that a callback function was invoked with.
//   - There must ALWAYS be atleast one variable in scope when the initial update is created.
//   - We do not allow clients to store data in the edge because it is difficult to develop an api around the expected behavior that
//     is easy to reason about.
// TODO:
//   - Move this into a separate package.
//   - We should have `Node` api.
//   - Do we need to keep `EgUpdate.Expand` idempotent (leads to alot of deep copies even for "transition tasks")?
//   - Provide a better way for the `TaskCompleted` return values to update node states, the key idea would be that there is
//     some sort of attrs map that is used to allow node states to be updated once the task is completed, depending on the results
//     of the task. Right now we get around this by stuffing data in `Node.Attrs.Client` (see `lexicalDecoder.inferAttrNext`).
//   - Add the variables in scope to `EgTask` to allow clients to specialize the scope. Requires abstracting away the `variable` type. At this point
//     we can also just return `Attributes` we want to set for the resulting variable we chose.
//   - We can be much smarter about when we copy the expansion graph state and when we do not need to.
//   - We could make the expansion graph state modifications return copies of the graph.
//   - Could make the `ExpansionGraph` private.
//   - Deal with `NextToken` edges, right now we don't even add them at inference because they are too painful to hook up,
//     instead we just rely on the context token gathering to only look for terminal nodes and ignore the edge type.
//   - We should probably remove the notion of an id from the node struct because the id changes depending on the actual context,
//     e.g is it a context graph node in the expansion graph is it an expansion graph node in a subgraph etc
//   - Remove the deep copy in infer production after the prediction has been made (I think this should be ok).
type EgUpdate struct {
	graph *ExpansionGraph

	meta expansionGraphMeta

	// the probability for the particular update.
	prob float32

	// stack of tasks to expand
	stack EgTaskStack
}

// EgUpdates ...
type EgUpdates []EgUpdate

// ByProb sorts the updates in order of decreasing probability
func (es EgUpdates) ByProb() {
	sort.Slice(es, func(i, j int) bool { return es[i].prob > es[j].prob })
}

// Peek at the head of the task stack for the update.
func (e EgUpdate) Peek() EgTask {
	return e.stack.Peek()
}

// Prob of the update
func (e EgUpdate) Prob() float32 {
	return e.prob
}

// Expand the graph by performing another round of inference based on the task stack,
// if the stack is empty `nil,nil` is returned, otherwise the list of possible expansions
// is returned.
// NOTE:
//   - It is not safe to call this from multiple go routines.
func (e EgUpdate) Expand(ctx kitectx.Context) (EgUpdates, error) {
	if e.stack.Empty() {
		return nil, nil
	}

	var updates EgUpdates
	var err error

	switch t, remaining := e.stack.Pop(); t.Type {
	case InferNameTask:
		updates, err = e.inferName(ctx, t, remaining)
	case InferProductionTask:
		updates, err = e.inferProduction(ctx, t, remaining)
	case PropagateTask:
		updates, err = e.propagate(ctx, t, remaining)
	case NoInferTask, InferProductionTaskCompleted, InferNameTaskCompleted, PropagateTaskCompleted:
		updates, err = e.taskCompleted(ctx, t, remaining)
	default:
		err = errors.Errorf("unsupported task type %v", t.Type)
	}

	if err != nil {
		return nil, err
	}
	updates.ByProb()

	return updates, nil
}

func (e EgUpdate) taskCompleted(ctx kitectx.Context, t EgTask, remaining EgTaskStack) (EgUpdates, error) {
	// deep copy of the base so clients can modify it if desired,
	// also need to do a deep copy to make things idempotent
	cpy, oldToNew := e.graph.deepCopy()
	t = t.deepCopy(oldToNew)
	remaining = remaining.deepCopy(oldToNew)

	newStack, err := e.meta.cb.TaskCompleted(ctx, cpy, t, remaining)
	if err != nil {
		e.print("error getting new stack of tasks: %v", err)
		return nil, errors.Errorf("error getting new stack of tasks: %v", err)
	}

	prob := e.prob
	switch t.Type {
	case NoInferTask, PropagateTaskCompleted:
		// no infer and propagate tasks always transition with prob 1
		prob = 1
	}

	return EgUpdates{{
		graph: cpy,
		meta:  e.meta,
		prob:  prob,
		stack: newStack,
	}}, nil
}

func (e EgUpdate) propagate(ctx kitectx.Context, t EgTask, remaining EgTaskStack) (EgUpdates, error) {
	eg, oldToNew := e.graph.deepCopy()
	t = t.deepCopy(oldToNew)
	remaining = remaining.deepCopy(oldToNew)
	ctx.CheckAbort()

	if err := eg.propagateTF(ctx, t); err != nil {
		return nil, errors.Errorf("error propagating: %v", err)
	}

	return EgUpdates{{
		graph: eg,
		meta:  e.meta,
		prob:  1,
		stack: remaining,
	}}, nil

}

func (e EgUpdate) inferProduction(ctx kitectx.Context, t EgTask, remaining EgTaskStack) (EgUpdates, error) {
	// deep copy of the base so that we can modify it for prediction,
	// also need to do a deep copy to make things idempotent
	eg, oldToNew := e.graph.deepCopy()
	t = t.deepCopy(oldToNew)
	remaining = remaining.deepCopy(oldToNew)
	ctx.CheckAbort()

	if len(t.InferProdTargets) == 0 {
		e.print("invalid candidates targets %d at site %v", len(t.InferProdTargets), t.Site)
		return nil, errors.Errorf("invalid targets %d at site %v", len(t.InferProdTargets), t.Site)
	}

	probs, err := eg.inferProdTF(ctx, t)
	if err != nil {
		e.print("infer production error for task %v: %v", t, err)
		return nil, errors.Errorf("infer production error: %v", err)
	}
	ctx.CheckAbort()

	var updates EgUpdates
	for i, p := range probs {
		// we need to make deep copies of the graph here
		// because we modify the graph state based on the
		// results of the infer production
		cpy, oldToNew := eg.deepCopy()

		cpyTask := t.deepCopy(oldToNew)

		cpyTask.Type = InferProductionTaskCompleted
		cpyTask.InferProdChosen = i

		// need to make a deep copy of the stack to make sure the
		// node references are updated
		tasks := remaining.deepCopy(oldToNew).Push(cpyTask)

		updates = append(updates, EgUpdate{
			graph: cpy,
			meta:  e.meta,
			stack: tasks,
			prob:  p,
		})

		ctx.CheckAbort()
	}

	return updates, nil
}

func (e EgUpdate) inferName(ctx kitectx.Context, t EgTask, remaining EgTaskStack) (EgUpdates, error) {
	// deep copy of the base so that we can modify it for prediction,
	// also need to do a deep copy to make things idempotent
	preds, attrs, err := func() ([]inferredNameData, []Attributes, error) {
		eg, oldToNew := e.graph.deepCopy()
		task := t.deepCopy(oldToNew)

		ctx.CheckAbort()

		// create new "usage" nodes for each variable
		usages := make([]*Node, 0, len(eg.state.variables))
		attrs := make([]Attributes, 0, len(eg.state.variables))
		for _, v := range eg.state.variables {
			usage := eg.AddNode(ASTTerminalNode, v.Latest.Attrs, nil)
			// TODO: we just add data flow edges for all of the name references
			for _, ref := range v.Refs {
				eg.AddEdge(ref, usage, DataFlow)
			}
			usages = append(usages, usage)
			attrs = append(attrs, v.Latest.Attrs)
		}

		types, subtokens, err := e.meta.cb.InferNameDecoderEmbeddings(ctx, eg, task.Site)
		if err != nil {
			e.print("error getting name decoder embeddings at %v: %v", t.Site, err)
			return nil, nil, errors.Errorf("error getting name decoder embeddings at %v: %v", task.Site, err)
		}
		if len(types) == 0 || len(subtokens) == 0 {
			e.print("missing types %d or subtokens %d for site %v", len(types), len(subtokens), t.Site)
			return nil, nil, errors.Errorf("missing types %d or subtokens %d for site %v", len(types), len(subtokens), task.Site)
		}

		preds, err := eg.inferNameTF(ctx, task, usages, types, subtokens)
		if err != nil {
			e.print("infer name error: %v", err)
			return nil, nil, errors.Errorf("infer name error: %v", err)
		}
		ctx.CheckAbort()
		return preds, attrs, err
	}()

	if err != nil {
		return nil, err
	}

	var updates EgUpdates
	for i, p := range preds {
		newAttrs := attrs[i]
		// We use the original graph as the base for the copy so we do not have
		// to worry about removing edges and then
		// add the appropriate nodes and edges for the chosen variable.

		// TODO: this is kind of nasty because it means the prediction node state is overwritten.
		cpy, oldToNew := e.graph.deepCopy()
		cpyTask := t.deepCopy(oldToNew)
		cpyTask.Type = InferNameTaskCompleted

		// TODO: this is pretty nasty, we copy the client data separately
		// to avoid overwriting it...
		cd := cpyTask.Site.Attrs.Client
		cpyTask.Site.Attrs = newAttrs
		cpyTask.Site.Attrs.Client = cd

		cpyVariable := cpy.state.variables[i]

		cpy.state.addEdge(cpyVariable.Latest, cpyTask.Site, DataFlow, NeighborData{})
		cpyVariable.Latest = cpyTask.Site
		cpyVariable.Refs = append(cpyVariable.Refs, cpyTask.Site)
		cpy.state.variables[i] = cpyVariable

		// set usage state for the variable
		// TODO: is this the correct node id?
		cpy.state.egNodeStates[cpyTask.Site.ID] = p.UsageNodeState

		// need to make a deep copy of the stack to make sure the
		// node references are updated
		cpyStack := remaining.deepCopy(oldToNew).Push(cpyTask)

		updates = append(updates, EgUpdate{
			graph: cpy,
			meta:  e.meta,
			stack: cpyStack,
			prob:  p.Prob,
		})

		ctx.CheckAbort()
	}

	return updates, nil
}

func (e EgUpdate) print(fmt string, args ...interface{}) {
	print(e.meta.tracer, fmt, args...)
}

type expansionGraphState struct {
	nodeEmbeddingDim int

	// will be shared across all expansion graphs that are children of the root context graph
	cgNodeStates [][]float32
	cgIncoming   nodeToNeighbors
	cgOutgoing   nodeToNeighbors

	// separate for each expansion graph
	variables egVariables

	egNodes []*Node

	// edge structure for the expansion graph, all edges must have one of the following forms:
	// 1) `from` and `to` nodes are both in the expansion graph
	// 2) `from` node is in the context graph, `to` node is in the expansion graph
	egIncoming nodeToNeighbors
	egOutgoing nodeToNeighbors

	egNodeStates [][]float32
}

func newExpansionGraphState(cg *ContextGraph) *expansionGraphState {
	egs := &expansionGraphState{
		nodeEmbeddingDim: cg.nodeEmbeddingDim,
		cgNodeStates:     cg.finalNodeStates,
		cgIncoming:       cg.incoming,
		cgOutgoing:       cg.outgoing,
		egIncoming:       make(nodeToNeighbors),
		egOutgoing:       make(nodeToNeighbors),
		variables:        make(egVariables, 0, len(cg.builder.vm.Variables)),
	}

	for _, v := range cg.builder.vm.Variables {
		egv := egVariable{
			Refs: make([]*Node, 0, len(v.Refs.set)),
		}
		var latest *pythonast.NameExpr
		for ref := range v.Refs.set {
			// TODO: this is pretty nasty, could use proper data flow here? could use lexical usage?
			rn := cg.builder.astNodes[ref]
			if egv.Latest == nil {
				egv.Latest = rn
				latest = ref
			} else if ref.Begin() > latest.End() && ref.End() < cg.site.Begin() {
				egv.Latest = rn
				latest = ref
			}
			egv.Refs = append(egv.Refs, rn)
		}

		egs.variables = append(egs.variables, egv)
	}
	return egs
}

func (s *expansionGraphState) deepCopy() (*expansionGraphState, func(*Node) *Node) {
	oldToNew := make(map[*Node]*Node, len(s.egNodes))

	egNodes := make([]*Node, 0, len(s.egNodes))
	for _, old := range s.egNodes {
		new := old.deepCopy()
		oldToNew[old] = new
		egNodes = append(egNodes, new)
	}

	mustGetNode := func(old *Node) *Node {
		if new, ok := oldToNew[old]; ok {
			return new
		}
		// at this point we know that the old node must have
		// been in the context graph so we can just
		// return it directly since these are never modified
		return old
	}

	variables := s.variables.deepCopy(mustGetNode)

	incoming := s.egIncoming.deepCopy(mustGetNode)
	outgoing := s.egOutgoing.deepCopy(mustGetNode)

	egNodeStates := make([][]float32, len(s.egNodeStates))
	copy(egNodeStates, s.egNodeStates)

	return &expansionGraphState{
		nodeEmbeddingDim: s.nodeEmbeddingDim,
		cgNodeStates:     s.cgNodeStates,
		cgIncoming:       s.cgIncoming,
		cgOutgoing:       s.cgOutgoing,
		egNodes:          egNodes,
		egIncoming:       incoming,
		egOutgoing:       outgoing,
		variables:        variables,
		egNodeStates:     egNodeStates,
	}, mustGetNode
}

func (s *expansionGraphState) removeEdge(from, to *Node, t EdgeType) {
	s.egOutgoing.removeNeighbor(from, to, t)
	s.egIncoming.removeNeighbor(to, from, t)
}

func (s *expansionGraphState) addEdge(from, to *Node, t EdgeType, nd NeighborData) {
	if !nd.NavOnly && !s.isEgNode(to) {
		// we are only allowed to have edges going from the context
		// graph to the expansion graph that are not nav only
		panic(fmt.Sprintf("trying to create non nav only edge %v -> %v (%v) but to node is in the context graph", from, to, t))
	}
	s.egOutgoing.addNeighbor(from, to, t, nd)
	s.egIncoming.addNeighbor(to, from, t, nd)
}

func (s *expansionGraphState) isEgNode(n *Node) bool {
	for _, egn := range s.egNodes {
		if n == egn {
			return true
		}
	}
	return false
}

func (s *expansionGraphState) addNode(t NodeType, attrs Attributes, initState []float32) *Node {
	n := &Node{
		ID:    NodeID(len(s.egNodes)),
		Type:  t,
		Attrs: attrs,
	}

	if len(initState) == 0 {
		// initialize to all zero state
		initState = make([]float32, s.nodeEmbeddingDim)
	}

	if len(initState) != s.nodeEmbeddingDim {
		panic(fmt.Sprintf("initial state has len %d expected %d", len(initState), s.nodeEmbeddingDim))
	}

	s.egNodeStates = append(s.egNodeStates, initState)
	s.egNodes = append(s.egNodes, n)

	return n
}

type expansionGraphMeta struct {
	useUncompressedModel bool

	model     *tensorflow.Model
	modelMeta ModelMeta

	cb EgCallbacks

	tracer *tracer

	saver Saver

	// ONLY FOR DEBUGGING
	buffer []byte
	ast    pythonast.Node
}

// ExpansionGraph represents a proposed expansion of the underlying context graph.
// The `ExpansionGraph` owns the graph structure and provides a way for clients to access and traverse
// the graph. Typically clients get a pointer to the `ExpansionGraph` via the `EgCallbacks` api.
// Internally the `ExpansionGraph` handles interacting with tensorflow for a round of inference and updating the underlying graph state
// as a result of inference.
// NOTE:
//   - Clients should never modify any of the values returned from the `ExpansionGraph` api.
//   - From the clients perspective there should be no distinction between the underlying context graph and the expansion graph,
//     the only exception is when an initial `EgUpdate` is being created and the client has a handle on both the context graph and
//     the expansion graph.
//   - Implementors of `EgCallbacks` should NEVER keep pointers to the `ExpansionGraph` that a callback function was invoked with.
//   - The `ExpansionGraph` api is NOT go routine safe.
type ExpansionGraph struct {
	meta expansionGraphMeta

	state *expansionGraphState
}

// Incoming returns the set of nodes that have an outgoing edge that points to
// `n`, e.g these are the edges incoming to `n`.
// NOTE:
//   - The returned nodes can be from the expansion graph or the context graph.
//   - Clients should NOT modify the resulting set or the neighbors within the set.
func (e *ExpansionGraph) Incoming(n *Node) NeighborSet {
	return joinNeighborSets(e.state.cgIncoming[n], e.state.egIncoming[n])
}

// Outgoing returns the set of nodes that have an incoming edge from
// `n`, e.g these are the edges outgoing from `n`.
// NOTE:
//   - The returned nodes can be from the expansion graph or the context graph.
//   - Clients should NOT modify the resulting set or the neighbors within the set.
func (e *ExpansionGraph) Outgoing(n *Node) NeighborSet {
	return joinNeighborSets(e.state.cgOutgoing[n], e.state.egOutgoing[n])
}

// AddEdge to the graph.
// NOTE:
//   - This is NOT safe to call from multiple go routines.
func (e *ExpansionGraph) AddEdge(from, to *Node, t EdgeType) {
	e.state.addEdge(from, to, t, NeighborData{})
}

// IsEGNode test is a node is in the expansion part of the graph
func (e *ExpansionGraph) IsEGNode(n *Node) bool {
	return e.state.isEgNode(n)
}

// AddNavOnlyEdge to the graph.
// NOTE:
//   - This is NOT safe to call from multiple go routines.
func (e *ExpansionGraph) AddNavOnlyEdge(from, to *Node, t EdgeType) {
	e.state.addEdge(from, to, t, NeighborData{NavOnly: true})
}

// RemoveEdge from the graph.
// NOTE:
//   - This is NOT safe to call from multiple go routines.
func (e *ExpansionGraph) RemoveEdge(from, to *Node, t EdgeType) {
	e.state.removeEdge(from, to, t)
}

// AddNode to the graph.
// NOTE:
//   - This is NOT safe to call from multiple go routines.
//   - The initial state of the node embedding is specified by `initState`, if it has length 0 (or is nil)
//     then the node state is initialized to the all 0 state.
func (e *ExpansionGraph) AddNode(t NodeType, attrs Attributes, initState []float32) *Node {
	return e.state.addNode(t, attrs, initState)
}

const (
	egNodeStatesOp   = "test/expansion_graph/graph/node_states"
	egFeedDictPrefix = "test/expansion_graph"
)

func (e *ExpansionGraph) propagateTF(ctx kitectx.Context, task EgTask) error {
	fb := newExpansionGraphFeedBuilder(e, []*Node{task.Site})

	fd := fb.TestFeed().FeedDict(egFeedDictPrefix)

	e.savePropagateData(task, fb, fd)

	res, err := e.meta.model.Run(fd, []string{egNodeStatesOp})
	if err != nil {
		return err
	}
	ctx.CheckAbort()

	newEgNodeStates := res[egNodeStatesOp].([][]float32)
	e.updateEgNodeStates([]*Node{task.Site}, newEgNodeStates, fb)

	return nil
}

type inferredNameData struct {
	Prob           float32
	UsageNodeState []float32
}

func (e *ExpansionGraph) inferNameTF(ctx kitectx.Context, task EgTask, usages []*Node, types, subtokens []string) ([]inferredNameData, error) {
	lookupNodes := append([]*Node{task.Site}, usages...)

	fb := newExpansionGraphFeedBuilder(e, lookupNodes)

	name := e.nameModelFeed(task.Site, usages, types, subtokens, fb)

	fd := fb.TestFeed().FeedDict(egFeedDictPrefix)
	for k, v := range name.FeedDict("test/infer_name") {
		fd[k] = v
	}
	ctx.CheckAbort()

	e.saveInferNameData(task, fb, name, fd)

	const predOp = "test/infer_name/prediction/pred"

	res, err := e.meta.model.Run(fd, []string{predOp, egNodeStatesOp})
	if err != nil {
		return nil, err
	}
	ctx.CheckAbort()

	newEgNodeStates := res[egNodeStatesOp].([][]float32)

	e.updateEgNodeStates([]*Node{task.Site}, newEgNodeStates, fb)

	inferred := make([]inferredNameData, 0, len(usages))
	for i, p := range res[predOp].([]float32) {
		usage := usages[i]

		ns := make([]float32, e.state.nodeEmbeddingDim)
		copy(ns, newEgNodeStates[fb.SubgraphID(usage)])

		inferred = append(inferred, inferredNameData{
			Prob:           p,
			UsageNodeState: ns,
		})
	}

	return inferred, nil
}

func (e *ExpansionGraph) nameModelFeed(site *Node, usages []*Node, decoderTypes, decoderSubtokens []string, fb ExpansionGraphFeedBuilder) NameModelFeed {
	typeFeed := traindata.SegmentedIndicesFeed{
		Indices:   make([]int32, 0, len(decoderTypes)),
		SampleIDs: make([]int32, len(decoderTypes)),
	}
	for _, t := range decoderTypes {
		typeFeed.Indices = append(typeFeed.Indices, int32(e.meta.modelMeta.TypeSubtokenIndex.Index(t)))
	}

	subtokenFeed := traindata.SegmentedIndicesFeed{
		Indices:   make([]int32, 0, len(decoderSubtokens)),
		SampleIDs: make([]int32, len(decoderSubtokens)),
	}
	for _, st := range decoderSubtokens {
		subtokenFeed.Indices = append(subtokenFeed.Indices, int32(e.meta.modelMeta.NameSubtokenIndex.Index(st)))
	}

	usagesFeed := traindata.SegmentedIndicesFeed{
		Indices:   make([]int32, 0, len(usages)),
		SampleIDs: make([]int32, len(usages)),
	}
	for _, u := range usages {
		usagesFeed.Indices = append(usagesFeed.Indices, int32(fb.SubgraphID(u)))
	}

	return NameModelFeed{
		PredictionNodes: []int32{int32(fb.SubgraphID(site))},
		Types:           typeFeed,
		Subtokens:       subtokenFeed,
		Names: NameEncoderFeed{
			Usages: usagesFeed,
		},
	}
}

func (e *ExpansionGraph) inferProdTF(ctx kitectx.Context, task EgTask) ([]float32, error) {
	scopeNodes := e.scopeNodes()
	contextTokens := e.contextTokens(task.Site)

	fb := newExpansionGraphFeedBuilder(e, []*Node{task.Site}, scopeNodes, contextTokens)

	feed := fb.TestFeed()

	fd := feed.FeedDict(egFeedDictPrefix)
	pf := ProductionModelFeed{
		PredictionNodes: []int32{int32(fb.SubgraphID(task.Site))},
		DecoderTargets: traindata.SegmentedIndicesFeed{
			Indices:   task.InferProdTargets,
			SampleIDs: make([]int32, len(task.InferProdTargets)),
		},
		ScopeEncoder:  newNodeIDFeed(scopeNodes, fb.SubgraphID),
		ContextTokens: newNodeIDFeed(contextTokens, fb.SubgraphID),
	}

	for k, v := range pf.FeedDict("test/infer_production") {
		fd[k] = v
	}

	e.saveInferProdData(task, fb, pf, fd)

	const predOp = "test/infer_production/prediction/pred"

	res, err := e.meta.model.Run(fd, []string{predOp, egNodeStatesOp})
	if err != nil {
		return nil, err
	}
	ctx.CheckAbort()

	newEgNodeStates := res[egNodeStatesOp].([][]float32)
	e.updateEgNodeStates([]*Node{task.Site}, newEgNodeStates, fb)

	return res[predOp].([]float32), nil
}

func (e *ExpansionGraph) updateEgNodeStates(lookupNodes []*Node, allStates [][]float32, fb ExpansionGraphFeedBuilder) {
	// TODO: this currently includes all nodes in the expansion sub graph
	// but we could limit it to just the lookup nodes to make it faster
	// TODO: can we avoid this copy?

	newStates := make([][]float32, len(lookupNodes)+len(e.state.egNodeStates))
	copy(newStates, e.state.egNodeStates)

	for _, ln := range lookupNodes {
		ns := newStates[ln.ID]
		if len(ns) == 0 {
			ns = make([]float32, e.state.nodeEmbeddingDim)
			newStates[ln.ID] = ns
		}
		copy(ns, allStates[fb.SubgraphID(ln)])
	}
}

func (e *ExpansionGraph) scopeNodes() []*Node {
	var nodes []*Node
	for _, v := range e.state.variables {
		if len(nodes) == 0 {
			nodes = make([]*Node, 0, len(v.Refs)*len(e.state.variables))
		}
		nodes = append(nodes, v.Refs...)
	}
	return nodes
}

func (e *ExpansionGraph) contextTokens(node *Node) []*Node {
	var nodes []*Node

	toVisit := []*Node{node}
	seen := make(map[*Node]bool)
	for i := 0; i < len(toVisit) && len(nodes) < maxNumContextTokens; i++ {
		n := toVisit[i]
		if seen[n] {
			continue
		}
		seen[n] = true
		if n.Type == ASTTerminalNode {
			nodes = append(nodes, n)
		}

		neighborSets := [4]NeighborSet{
			e.state.cgOutgoing[n],
			e.state.egOutgoing[n],
			e.state.cgIncoming[n],
			e.state.egIncoming[n],
		}

		for _, nns := range neighborSets {
			for _, nn := range nns {
				if !seen[nn.Node] {
					toVisit = append(toVisit, nn.Node)
				}
			}
		}
	}

	if len(nodes) == 0 {
		nodes = append(nodes, node)
	}

	return nodes
}

func (e *ExpansionGraph) deepCopy() (*ExpansionGraph, func(*Node) *Node) {
	state, oldToNew := e.state.deepCopy()

	return &ExpansionGraph{
		meta:  e.meta,
		state: state,
	}, oldToNew
}

func (e *ExpansionGraph) saveInferNameData(task EgTask, fb ExpansionGraphFeedBuilder, nf NameModelFeed, feed map[string]interface{}) {
	if e.meta.saver == nil {
		return
	}

	sg := fb.SavedGraph()
	savedSite := sg.Nodes[fb.SubgraphID(task.Site)]
	labels := NodeLabels{
		savedSite.Node.ID: "SITE",
	}

	var usages []*SavedNode
	for _, id := range nf.Names.Usages.Indices {
		usages = append(usages, sg.Nodes[id])
		labels[NodeID(id)] = "USAGE"
	}

	ib := newInsightBuilder(insightBuilderParams{
		Model:                e.meta.model,
		Graph:                sg,
		Edges:                fb.base.Edges,
		Feed:                 feed,
		MaxHops:              1,
		Labels:               labels,
		UseUncompressedModel: e.meta.useUncompressedModel,
	}, inferNameInsights)

	for i, t := range []string{"expansion-graph", "insights"} {
		it := finalPropagationStep
		if i == 1 {
			it = allPropagationSteps
		}

		sb := ib.InferNameInsights(it, usages, savedSite)

		sb.Label = fmt.Sprintf("infer-name-%s", t)
		if task.Client != nil {
			sb.Label += fmt.Sprintf("-%s", task.Client.EgClientData())
		}

		sb.AST = e.meta.ast
		sb.Buffer = e.meta.buffer

		save(e.meta.saver, sb)
	}
}

func (e *ExpansionGraph) savePropagateData(task EgTask, fb ExpansionGraphFeedBuilder, feed map[string]interface{}) {
	if e.meta.saver == nil {
		return
	}

	sg := fb.SavedGraph()
	savedSite := sg.Nodes[fb.SubgraphID(task.Site)]
	labels := NodeLabels{
		savedSite.Node.ID: "SITE",
	}

	ib := newInsightBuilder(insightBuilderParams{
		Model:                e.meta.model,
		Graph:                sg,
		Edges:                fb.base.Edges,
		MaxHops:              1,
		Feed:                 feed,
		Labels:               labels,
		UseUncompressedModel: e.meta.useUncompressedModel,
	}, graphInsights)

	sb := ib.GraphInsights(finalPropagationStep, savedSite)

	sb.Label = fmt.Sprintf("infer-propagate-%s", task.Client.EgClientData())
	sb.AST = e.meta.ast
	sb.Buffer = e.meta.buffer

	save(e.meta.saver, sb)
}

func (e *ExpansionGraph) saveInferProdData(task EgTask, fb ExpansionGraphFeedBuilder, pf ProductionModelFeed, feed map[string]interface{}) {
	if e.meta.saver == nil {
		return
	}

	sg := fb.SavedGraph()
	savedSite := sg.Nodes[fb.SubgraphID(task.Site)]
	labels := NodeLabels{
		savedSite.Node.ID: "SITE",
	}

	for i, ids := range [][]int32{pf.ScopeEncoder.Indices, pf.ContextTokens.Indices} {
		label := "SCOPE"
		if i == 1 {
			label = "CONTEXT_TOKEN"
		}

		for _, id := range ids {
			labels[NodeID(id)] = label
		}
	}

	var targets []string
	for _, attr := range task.InferProdClient {
		targets = append(targets, attr.EgClientData())
	}

	ib := newInsightBuilder(insightBuilderParams{
		Model:                e.meta.model,
		Graph:                sg,
		Edges:                fb.base.Edges,
		Feed:                 feed,
		MaxHops:              1,
		Labels:               labels,
		UseUncompressedModel: e.meta.useUncompressedModel,
	}, inferProdInsights)

	for i, t := range []string{"expansion-graph", "insights"} {
		it := finalPropagationStep
		if i == 1 {
			it = allPropagationSteps
		}

		sb := ib.InferProdInsights(it, targets, savedSite)

		sb.Label = fmt.Sprintf("infer-prod-%s", t)
		if task.Client != nil {
			sb.Label += fmt.Sprintf("-%s", task.Client.EgClientData())
		}

		sb.AST = e.meta.ast
		sb.Buffer = e.meta.buffer

		save(e.meta.saver, sb)
	}
}

// SavedGraph to render
func (e *ExpansionGraph) SavedGraph(site *Node) (*SavedGraph, NodeLabels) {
	var nodes []*SavedNode
	saved := make(map[*Node]*SavedNode)

	// have to use pointers since we re number the node ids below
	labels := make(map[*Node]string)

	addNode := func(n *Node, extra string) *SavedNode {
		label := "EG"
		if !e.state.isEgNode(n) {
			label = "CG"
		}

		if old := labels[n]; old != "" {
			label = old
		}

		if extra != "" {
			label += "::" + extra
		}
		labels[n] = label

		if s, ok := saved[n]; ok {
			return s
		}

		new := &SavedNode{
			Node:  n.deepCopy(),
			Level: -1,
		}
		new.Node.ID = NodeID(len(nodes))

		saved[n] = new
		nodes = append(nodes, new)

		return new
	}

	if site != nil {
		addNode(site, "SITE")
	}

	var edges []*SavedEdge
	for _, n := range e.state.egNodes {
		s := addNode(n, "")
		for _, nn := range e.Incoming(n) {
			ss := addNode(nn.Node, "")
			edges = append(edges, &SavedEdge{
				From:    ss,
				To:      s,
				Type:    nn.Type,
				Forward: true,
			})
		}
		for _, nn := range e.Outgoing(n) {
			ss := addNode(nn.Node, "")
			edges = append(edges, &SavedEdge{
				From:    s,
				To:      ss,
				Type:    nn.Type,
				Forward: true,
			})
		}
	}

	var contextTokens []*Node
	if site != nil {
		contextTokens = e.contextTokens(site)
	}

	// add the scope/context/site nodes as disconnected
	for i, ns := range [][]*Node{e.scopeNodes(), contextTokens} {
		extra := "SCOPE"
		if i == 1 {
			extra = "CONTEXT"
		}
		for _, n := range ns {
			addNode(n, extra)
		}
	}

	newLabels := make(map[NodeID]string)
	for n, l := range labels {
		s := saved[n]
		newLabels[s.Node.ID] = l
	}

	return &SavedGraph{
		Nodes: nodes,
		Edges: edges,
	}, newLabels
}

func (e *ExpansionGraph) print(fmt string, args ...interface{}) {
	print(e.meta.tracer, fmt, args...)
}

type egVariable struct {
	Refs   []*Node
	Latest *Node
}

func (e egVariable) deepCopy(oldToNew func(*Node) *Node) egVariable {
	ee := egVariable{
		Refs:   make([]*Node, 0, len(e.Refs)),
		Latest: oldToNew(e.Latest),
	}

	for _, ref := range e.Refs {
		ee.Refs = append(ee.Refs, oldToNew(ref))
	}
	return ee
}

type egVariables []egVariable

func (e egVariables) deepCopy(oldToNew func(*Node) *Node) egVariables {
	ee := make(egVariables, 0, len(e))
	for _, v := range e {
		ee = append(ee, v.deepCopy(oldToNew))
	}
	return ee
}

// EgClientData is an optional struct used by clients
// to include data in nodes or edges
// NOTE:
//   - These labels are copied by value when we need to make copies of the graph so they should be immutable.
type EgClientData interface {
	EgClientData() string
}

// NeighborData is data associated with a particular neighbor,
// it should be considered immutable as the neighbor relation
// is defined solely by the Node and the Type.
type NeighborData struct {
	NavOnly bool
}

// NeighborInfo represents a neighbor with a pointer to the node, the type of the edge and the data associated to this edge
type NeighborInfo struct {
	Node *Node
	Type EdgeType
	data NeighborData
}

// Nil ...
func (n NeighborInfo) Nil() bool {
	return n.Node == nil
}

// NeighborSet is a set of neighbor nodes in an expansion graph.
type NeighborSet []NeighborInfo

func (e NeighborSet) deepCopy(oldToNew func(*Node) *Node) NeighborSet {
	ee := make(NeighborSet, 0, len(e))
	for _, old := range e {
		ee = append(ee, NeighborInfo{
			Node: oldToNew(old.Node),
			Type: old.Type,
			data: NeighborData{old.data.NavOnly},
		})
	}
	return ee
}

func (e NeighborSet) addNeighbor(n *Node, t EdgeType, data NeighborData) NeighborSet {
	for i, nn := range e {
		if nn.Node == n && nn.Type == t {
			e[i].data = data
			return e
		}
	}

	en := NeighborInfo{
		Node: n,
		Type: t,
		data: data,
	}
	return append(e, en)
}

func (e NeighborSet) removeNeighbor(n *Node, t EdgeType) NeighborSet {
	for i, ne := range e {
		if ne.Node == n && ne.Type == t {
			return append(e[:i], e[i+1:]...)
		}
	}
	return e
}

func joinNeighborSets(ss ...NeighborSet) NeighborSet {
	switch len(ss) {
	case 0:
		return nil
	case 1:
		return ss[0]
	case 2:
		// special case the common use case
		s0, s1 := ss[0], ss[1]
		if len(s0) == 0 {
			return s1
		}
		if len(s1) == 0 {
			return s0
		}
	}

	var cap int
	for _, s := range ss {
		cap += len(s)
	}

	if cap == 0 {
		return nil
	}

	joined := make(NeighborSet, cap)
	for _, s := range ss {
		for k, v := range s {
			joined[k] = v
		}
	}

	return joined
}

type nodeToNeighbors map[*Node]NeighborSet

func (e nodeToNeighbors) deepCopy(oldToNew func(*Node) *Node) nodeToNeighbors {
	ee := make(nodeToNeighbors, len(e))
	for old, oldNs := range e {
		ee[oldToNew(old)] = oldNs.deepCopy(oldToNew)
	}
	return ee
}

func (e nodeToNeighbors) addNeighbor(n, neighbor *Node, t EdgeType, data NeighborData) {
	e[n] = e[n].addNeighbor(neighbor, t, data)
}

func (e nodeToNeighbors) removeNeighbor(n, neighbor *Node, t EdgeType) {
	ns := e[n]
	if ns == nil {
		return
	}
	e[n] = ns.removeNeighbor(neighbor, t)
}
