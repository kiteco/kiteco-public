package pythongraph

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
)

func (b *graphBuilder) forwardFlowGraph(names *nameSet) nameFlowGraph {
	if len(b.a.RAST.Root.Body) == 0 {
		return nil
	}

	return newNameGraphBuilder(names, b.a.RAST.Root).Build()
}

// nameFlowGraph represents a mapping from a name expression (typically corresponding to a variable) in an ast to
// the set of possible ast locations (name expressions) at which the value
// of the variable will next be written to or read from.
// Intuitively, for each name expression we track the possible locations to which the value of
// the name expression could possibly flow to next.
// Consider the following simple example:
// 1) x = 1
// 2) x = x + 1
// Let x1 denote the name expression in line 1, x2 denote the name expression on the rhs of the assignment in line 2, and
// let x3 denote the name expression on the lhs of the assignment in line 2.
// for this example we get the following flow graph: x1 -> x2 -> x3
// Intuitively, x1 flows to x2 because x2 represents the next location in the ast at which the value of the variable of x will be read from,
// similarly x2 flows to x3 because after x is read at location x2, x3 is the next location in the ast at which the value of the variable x will be written to.
// NOTE: this is an approximation, additionally the analogy is not quite correct
// since in python the name x is really bound to the new value x + 1 on the LHS of line 2). But for our purposes
// we ignore this distinction.
type nameFlowGraph map[*pythonast.NameExpr]*nameSet

type nameFlowGraphBuilder struct {
	graph nameFlowGraph
	names *nameSet
	mod   *pythonast.Module
}

func newNameGraphBuilder(names *nameSet, mod *pythonast.Module) nameFlowGraphBuilder {
	return nameFlowGraphBuilder{
		graph: make(nameFlowGraph, names.Len()),
		names: names,
		mod:   mod,
	}
}

func (b nameFlowGraphBuilder) flowsTo(src, dest *nameSet) {
	for s := range src.Set() {
		neighbors := b.graph[s]
		if neighbors == nil {
			b.graph[s] = dest.Copy()
			continue
		}

		for d, order := range dest.Set() {
			neighbors.Add(d, order)
		}
	}
}

func (b nameFlowGraphBuilder) Build() nameFlowGraph {

	b.flowSuite(b.mod.Body)

	return b.graph
}

func (b nameFlowGraphBuilder) blocksFlow(stmt pythonast.Stmt) bool {
	if pythonast.IsNil(stmt) {
		return false
	}

	switch stmt := stmt.(type) {
	case *pythonast.IfStmt:
		// the first condition of the first branch
		// is the only thing that we know must be evaluated
		return b.containsName(stmt.Branches[0].Condition)
	case *pythonast.ForStmt:
		// we are guaranteed that the iterables are evaluated
		// atleast once
		return b.containsName(stmt.Iterable)
	case *pythonast.WhileStmt:
		// we are guaranted that the while condition is evaluated atleast
		// once
		return b.containsName(stmt.Condition)
	case *pythonast.WithStmt:
		// we are only guaranteed that the first value will be evaluated
		return b.containsName(stmt.Items[0].Value)
	case *pythonast.ImportNameStmt, *pythonast.ImportFromStmt:
		// entry and exit set the same so can just check one
		return b.addEntrySet(nil, stmt) > 0 // passing a nil nameSet returns a positive value if names *would* be added -- the exact count is wrong
	case *pythonast.ClassDefStmt, *pythonast.FunctionDefStmt:
		return false
	case *pythonast.TryStmt:
		// TODO: do better here
		return false
	default:
		// for any other statement if the variable of interest
		// is referenced then the statement blocks.
		return b.containsName(stmt)
	}
}

func (b nameFlowGraphBuilder) flowSuite(suite []pythonast.Stmt) {
	lastExit := newNameSet()
	for _, stmt := range suite {
		if !b.containsName(stmt) {
			continue
		}

		// flow statement
		entry, exit := b.flowStmt(stmt)

		// every thing from previous exit
		// set flows into the current
		// entry set
		b.flowsTo(lastExit, entry)

		if b.blocksFlow(stmt) {
			// current statement blocks flow of the variable
			lastExit = exit
		} else {
			// variable escapes current statement so unite
			// current exit set with last exit set.
			for ne, order := range exit.Set() {
				lastExit.Add(ne, order)
			}
		}
	}
}

func (b nameFlowGraphBuilder) flowStmt(stmt pythonast.Stmt) (*nameSet, *nameSet) {
	if pythonast.IsNil(stmt) {
		return nil, nil
	}

	entry, exit := newNameSet(), newNameSet()
	b.addEntrySet(entry, stmt)
	b.addExitSet(exit, stmt)

	switch stmt := stmt.(type) {
	case *pythonast.AssignStmt:
		// need to check equality
		// since for a statement like
		// x = 1
		// the entry and exit sets are the
		// same but it does not make sense
		// to say x flows to itself here
		if !entry.Equals(exit) {
			b.flowsTo(entry, exit)
		}

	case *pythonast.AugAssignStmt:
		if !entry.Equals(exit) {
			b.flowsTo(entry, exit)
		}

	case *pythonast.IfStmt:
		//
		// general ideas:
		// - last condition that contains variable of interest
		//   always flows into child block and into subsequent branch
		//   blocks unless the subsequent branch is guarded by an expression
		//   that contains the variable of interest.
		// - last condition that contains variable of interest
		//   always flows into else body.
		// - last condition that contains variable of interest always flows into
		//   the next branch condition that contains the variable of interest.
		// - the exit set from a branch body never flows into another branch body (at this level,
		//   a parent statement may be a loop in which case this can happen but it does not happen
		//   when we are flowing the if statement).

		var lastCondition *nameSet
		for _, branch := range stmt.Branches {
			condition := newNameSet()
			b.addExprNames(condition, branch.Condition)

			// last condition always flows into current condition
			b.flowsTo(lastCondition, condition)

			if !condition.Empty() {
				lastCondition = condition
			}

			entryBody := newNameSet()
			b.addSuiteEntrySet(entryBody, branch.Body)
			b.flowsTo(lastCondition, entryBody)

			b.flowSuite(branch.Body)
		}

		entryElse := newNameSet()
		b.addSuiteEntrySet(entryElse, stmt.Else)
		b.flowsTo(lastCondition, entryElse)

		b.flowSuite(stmt.Else)

	case *pythonast.ForStmt:
		//
		// handle iterables and targets
		//
		iters, targets := newNameSet(), newNameSet()
		b.addExprNames(iters, stmt.Iterable)
		b.addExprNames(targets, stmt.Targets...)

		b.flowsTo(iters, targets)

		//
		// handle body of for loop
		//
		entryBody, exitBody := newNameSet(), newNameSet()
		b.addSuiteEntrySet(entryBody, stmt.Body)
		b.addSuiteExitSet(exitBody, stmt.Body)

		if !targets.Empty() {
			b.flowsTo(targets, entryBody)
		} else {
			b.flowsTo(iters, entryBody)
		}

		b.flowSuite(stmt.Body)

		//
		// handle else body
		//
		entryElse := newNameSet()
		b.addSuiteEntrySet(entryElse, stmt.Else)
		if !iters.Empty() {
			// iterables are always evaluated on each iteration,
			// thus if the variable of interest is present
			// it flows into the else body
			b.flowsTo(iters, entryElse)
		} else {
			// if the variable of interest is not present in the iterables,
			// then the targets flow into the else body, this happens
			// because we cannot guarantee that the variable of interest
			// is always evaluated even if it is present in the body of the for loop.
			b.flowsTo(targets, entryElse)
		}

		// the exit set of the body always flows into the else
		b.flowsTo(exitBody, entryElse)

		b.flowSuite(stmt.Else)

		//
		// handle loops
		//
		switch {
		case exitBody.Empty():
			if !iters.Empty() {
				b.flowsTo(iters, iters)
			} else {
				b.flowsTo(targets, targets)
			}
		case !iters.Empty():
			b.flowsTo(exitBody, iters)
		case !targets.Empty():
			b.flowsTo(exitBody, targets)
		default:
			b.flowsTo(exitBody, entryBody)
		}

	case *pythonast.WhileStmt:
		condition := newNameSet()
		b.addExprNames(condition, stmt.Condition)

		// handle body
		entryBody, exitBody := newNameSet(), newNameSet()
		b.addSuiteEntrySet(entryBody, stmt.Body)
		b.addSuiteExitSet(exitBody, stmt.Body)

		b.flowsTo(condition, entryBody)

		b.flowSuite(stmt.Body)

		// handle else body
		entryElse := newNameSet()
		b.addSuiteEntrySet(entryElse, stmt.Else)

		b.flowsTo(condition, entryElse)
		b.flowsTo(exitBody, entryElse)

		b.flowSuite(stmt.Else)

		// handle loops
		switch {
		case exitBody.Empty():
			b.flowsTo(condition, condition)
		case !condition.Empty():
			b.flowsTo(exitBody, condition)
		default:
			b.flowsTo(exitBody, entryBody)
		}

	case *pythonast.FunctionDefStmt:
		// TODO(juan): flow from parameters to body?
		params := newNameSet()

		for _, param := range stmt.Parameters {
			for _, name := range unpackParamName(param) {
				b.maybeAddName(params, name)
			}
		}

		if !params.Empty() {
			entryBody := newNameSet()
			b.addSuiteEntrySet(entryBody, stmt.Body)
			b.flowsTo(params, entryBody)
		}

		b.flowSuite(stmt.Body)

	case *pythonast.ClassDefStmt:
		// TODO(juan): more to do here?
		b.flowSuite(stmt.Body)

	case *pythonast.WithStmt:
		exitItems := newNameSet()
		for _, item := range stmt.Items {
			if numAdded := b.addExprNames(exitItems, item.Target); numAdded == 0 {
				b.addExprNames(exitItems, item.Value)
			}
		}

		entryBody := newNameSet()
		b.addSuiteEntrySet(entryBody, stmt.Body)

		b.flowsTo(exitItems, entryBody)

	case *pythonast.TryStmt:
		b.flowSuite(stmt.Body)

		exitBody := newNameSet()
		b.addSuiteExitSet(exitBody, stmt.Body)

		exit := newNameSet()
		for _, handler := range stmt.Handlers {
			entryBody := newNameSet()
			b.addSuiteEntrySet(entryBody, handler.Body)

			entry := newNameSet()
			if numAdded := b.addExprNames(entry, handler.Type, handler.Target); numAdded > 0 {
				b.flowsTo(exitBody, entry)
				b.flowsTo(entry, entryBody)
			} else {
				b.flowsTo(exitBody, entryBody)
			}

			b.flowSuite(handler.Body)

			b.addSuiteExitSet(exit, handler.Body)
		}

		// all exit sets flow into the finally
		entryFinal := newNameSet()
		b.addSuiteEntrySet(entryFinal, stmt.Finally)
		b.flowsTo(exit, entryFinal)

		// exit set of body always flows into else
		entrySuite := newNameSet()
		b.addSuiteEntrySet(entrySuite, stmt.Else)
		b.flowsTo(exitBody, entrySuite)
	}
	return entry, exit
}

func (b nameFlowGraphBuilder) addSuiteEntrySet(ns *nameSet, suite []pythonast.Stmt) int {
	for _, stmt := range suite {
		if n := b.addEntrySet(ns, stmt); n > 0 {
			return n
		}
	}
	return 0
}

func (b nameFlowGraphBuilder) addEntrySet(ns *nameSet, stmt pythonast.Stmt) (numAdded int) {
	if pythonast.IsNil(stmt) {
		return
	}

	switch stmt := stmt.(type) {
	case *pythonast.AssignStmt:
		if n := b.addExprNames(ns, stmt.Value); n > 0 {
			numAdded += n
			return
		}
		// If the variable was not referenced in the values of the
		// statement (RHS) then the first point at which the variable
		// could be read from or written to must be in the LHS of the
		// assignment statement, if it is present at all.
		// Consider a simple case such as:
		// 1) x = 1
		// 2) x = 2
		// the entry set for x in line 2 is simply the x on the lhs since
		// this is the only place that the variable x could be read from or written to.
		// NOTE: we account for the case in which the entry and exit set are equal
		// explicitly in `flowStmt`
		for _, target := range stmt.Targets {
			numAdded += b.addExprNames(ns, target)
		}

	case *pythonast.AugAssignStmt:
		if n := b.addExprNames(ns, stmt.Value); n > 0 {
			numAdded += n
			return
		}
		numAdded += b.addExprNames(ns, stmt.Target)

	case *pythonast.IfStmt:
		var foundCondition bool
		for _, branch := range stmt.Branches {
			if n := b.addExprNames(ns, branch.Condition); n > 0 {
				numAdded += n
				foundCondition = true
				break
			}

			numAdded += b.addSuiteEntrySet(ns, branch.Body)
		}
		if !foundCondition {
			numAdded += b.addSuiteEntrySet(ns, stmt.Else)
		}

	case *pythonast.ForStmt:
		// iterables are always evaluated atleast once,
		// thus if the variable is referenced
		// in the iterables then we are done
		if n := b.addExprNames(ns, stmt.Iterable); n > 0 {
			numAdded += n
			return
		}
		numAdded += b.addSuiteEntrySet(ns, stmt.Else)
		if n := b.addExprNames(ns, stmt.Targets...); n > 0 {
			numAdded += n
			return
		}
		numAdded += b.addSuiteEntrySet(ns, stmt.Body)

	case *pythonast.WhileStmt:
		// condition is always evaluated atleast once,
		// thus if the variable is referenced
		// in the condition then we are done
		if n := b.addExprNames(ns, stmt.Condition); n > 0 {
			numAdded += n
			return
		}
		numAdded += b.addSuiteEntrySet(ns, stmt.Body)
		numAdded += b.addSuiteEntrySet(ns, stmt.Else)

	case *pythonast.ImportNameStmt:
		for _, name := range stmt.Names {
			if name.Internal != nil {
				numAdded += b.maybeAddName(ns, name.Internal)
				continue
			}
			if len(name.External.Names) > 0 {
				// the root is always added to the
				// local namespace.
				numAdded += b.maybeAddName(ns, name.External.Names[0])
			}
		}

	case *pythonast.ImportFromStmt:
		for _, clause := range stmt.Names {
			if clause.Internal != nil {
				numAdded += b.maybeAddName(ns, clause.Internal)
			} else {
				numAdded += b.maybeAddName(ns, clause.External)
			}
		}

	case *pythonast.WithStmt:
		// the first value is always evaluated, thus
		// if the first value contains the item of interest
		// then we are done, we cannot guarantee anymore of
		// the values are evaluated.
		if n := b.addExprNames(ns, stmt.Items[0].Value); n > 0 {
			numAdded += n
			return
		}
		for _, item := range stmt.Items {
			// the value of an item is evaluated before the target
			if n := b.addExprNames(ns, item.Value); n > 0 {
				numAdded += n
			} else {
				numAdded += b.addExprNames(ns, item.Target)
			}
		}
		// the items are certainly evaluated before the body,
		// so if we have any references then we are done
		if numAdded > 0 {
			return
		}
		numAdded += b.addSuiteEntrySet(ns, stmt.Body)

	case *pythonast.ClassDefStmt:
		numAdded += b.addExprNames(ns, stmt.Decorators...)
		numAdded += b.addExprNames(ns, stmt.Kwarg, stmt.Vararg)
		numAdded += b.addArgValueNames(ns, stmt.Args...)
		numAdded += b.maybeAddName(ns, stmt.Name)

	case *pythonast.FunctionDefStmt:
		for _, param := range stmt.Parameters {
			numAdded += b.addExprNames(ns, param.Default, param.Annotation)
		}
		if stmt.Kwarg != nil {
			numAdded += b.addExprNames(ns, stmt.Kwarg.Annotation)
		}
		if stmt.Vararg != nil {
			numAdded += b.addExprNames(ns, stmt.Vararg.Annotation)
		}
		numAdded += b.addExprNames(ns, stmt.Decorators...)
		numAdded += b.addExprNames(ns, stmt.Annotation)
		numAdded += b.maybeAddName(ns, stmt.Name)

	case *pythonast.TryStmt:
		if n := b.addSuiteEntrySet(ns, stmt.Body); n > 0 {
			// approximation, the reference in
			// the body may not have executed
			numAdded += n
			return
		}
		for _, clause := range stmt.Handlers {
			if n := b.addExprNames(ns, clause.Type, clause.Target); n > 0 {
				numAdded += n
			} else {
				numAdded += b.addSuiteEntrySet(ns, clause.Body)
			}
		}
		numAdded += b.addSuiteEntrySet(ns, stmt.Else)
		numAdded += b.addSuiteEntrySet(ns, stmt.Finally)

	case pythonast.Stmt:
		numAdded += b.addStmtNames(ns, stmt)
	}
	return
}

func (b nameFlowGraphBuilder) addSuiteExitSet(ns *nameSet, suite []pythonast.Stmt) int {
	for i := len(suite) - 1; i >= 0; i-- {
		if n := b.addExitSet(ns, suite[i]); n > 0 {
			return n
		}
	}
	return 0
}

func (b nameFlowGraphBuilder) addExitSet(ns *nameSet, stmt pythonast.Stmt) (numAdded int) {
	if pythonast.IsNil(stmt) {
		return
	}

	switch stmt := stmt.(type) {
	case *pythonast.AssignStmt:
		for _, target := range stmt.Targets {
			numAdded += b.addExprNames(ns, target)
		}
		if numAdded > 0 {
			return
		}
		numAdded += b.addExprNames(ns, stmt.Value)

	case *pythonast.AugAssignStmt:
		if n := b.addExprNames(ns, stmt.Target); n > 0 {
			numAdded += n
			return
		}
		numAdded += b.addExprNames(ns, stmt.Value)

	case *pythonast.IfStmt:
		// exit set always includes all branches
		// and the else body, because
		// we do not know which branch will be executed.
		for _, branch := range stmt.Branches {
			// exit set always includes conditional because
			// even if the variable is referenced in the body
			// of the branch there is no guarantee that it
			// will actually be evaluated, see unit tests.
			numAdded += b.addExprNames(ns, branch.Condition)
			numAdded += b.addSuiteExitSet(ns, branch.Body)
		}
		numAdded += b.addSuiteExitSet(ns, stmt.Else)

	case *pythonast.ForStmt:
		// the exit set of the else body
		// always exit since we cannot guarantee
		// that the else will not be evaluated
		//
		// the iterables always exit because
		// we cannot guarantee
		// targets or body of for loop body is evaluated
		//
		// the targets always exit because the first
		// line of the for loop could be a break statement
		//
		// the body always exits because we could reference
		// the variable of interest in the first line then break
		// immediately.
		numAdded += b.addSuiteExitSet(ns, stmt.Else)
		numAdded += b.addExprNames(ns, stmt.Iterable)
		numAdded += b.addSuiteExitSet(ns, stmt.Body)
		numAdded += b.addExprNames(ns, stmt.Targets...)

	case *pythonast.WhileStmt:
		// the exit set of the else body
		// always exit since we cannot guarantee
		// that the else will not be evaluated
		//
		// the condition always exits because
		// we cannot guarantee that the body will be evaluated
		//
		// the body always exits because we could reference
		// the variable of interest in the first line then break
		// immediately.
		numAdded += b.addSuiteExitSet(ns, stmt.Else)
		numAdded += b.addExprNames(ns, stmt.Condition)
		numAdded += b.addSuiteExitSet(ns, stmt.Body)

	case *pythonast.ImportNameStmt:
		for _, name := range stmt.Names {
			if name.Internal != nil {
				numAdded += b.maybeAddName(ns, name.Internal)
				continue
			}
			if len(name.External.Names) > 0 {
				// the root is always added to the
				// local namespace.
				numAdded += b.maybeAddName(ns, name.External.Names[0])
			}
		}

	case *pythonast.ImportFromStmt:
		for _, clause := range stmt.Names {
			if clause.Internal != nil {
				numAdded += b.maybeAddName(ns, clause.Internal)
			} else {
				numAdded += b.maybeAddName(ns, clause.External)
			}
		}

	case *pythonast.WithStmt:
		// we have no guarantee that anything after the
		// first item is ever evaluated so we include everything in the exit set
		for _, item := range stmt.Items {
			numAdded += b.addExprNames(ns, item.Target, item.Value)
		}
		numAdded += b.addSuiteExitSet(ns, stmt.Body)

	case *pythonast.ClassDefStmt:
		numAdded += b.addExprNames(ns, stmt.Decorators...)
		numAdded += b.addExprNames(ns, stmt.Kwarg, stmt.Vararg)
		numAdded += b.addArgValueNames(ns, stmt.Args...)
		numAdded += b.maybeAddName(ns, stmt.Name)

	case *pythonast.FunctionDefStmt:
		for _, param := range stmt.Parameters {
			numAdded += b.addExprNames(ns, param.Default, param.Annotation)
		}
		if stmt.Kwarg != nil {
			numAdded += b.addExprNames(ns, stmt.Kwarg.Annotation)
		}
		if stmt.Vararg != nil {
			numAdded += b.addExprNames(ns, stmt.Vararg.Annotation)
		}
		numAdded += b.addExprNames(ns, stmt.Decorators...)
		numAdded += b.addExprNames(ns, stmt.Annotation)
		numAdded += b.maybeAddName(ns, stmt.Name)

	case *pythonast.TryStmt:
		if n := b.addSuiteExitSet(ns, stmt.Finally); n > 0 {
			numAdded += n
			return
		}
		// approximation, could do better here...
		numAdded += b.addSuiteExitSet(ns, stmt.Else)
		numAdded += b.addSuiteExitSet(ns, stmt.Body)

		for _, handler := range stmt.Handlers {
			if n := b.addSuiteExitSet(ns, handler.Body); n > 0 {
				numAdded += n
			} else {
				numAdded += b.addExprNames(ns, handler.Target, handler.Type)
			}
		}

	case pythonast.Stmt:
		numAdded += b.addStmtNames(ns, stmt)
	}
	return
}

func (b nameFlowGraphBuilder) addArgValueNames(ns *nameSet, args ...*pythonast.Argument) (numAdded int) {
	for _, arg := range args {
		numAdded += b.addExprNames(ns, arg.Value)
	}
	return
}

func (b nameFlowGraphBuilder) addStmtNames(ns *nameSet, stmt pythonast.Stmt) (numAdded int) {
	if pythonast.IsNil(stmt) {
		return
	}
	pythonast.Inspect(stmt, func(node pythonast.Node) bool {
		if expr, ok := node.(pythonast.Expr); ok {
			numAdded += b.addExprNames(ns, expr)
			return false
		}
		return true
	})
	return
}

func (b nameFlowGraphBuilder) addExprNames(ns *nameSet, exprs ...pythonast.Expr) (numAdded int) {
	for _, expr := range exprs {
		if pythonast.IsNil(expr) {
			continue
		}

		pythonast.Inspect(expr, func(n pythonast.Node) bool {
			name, ok := n.(*pythonast.NameExpr)
			if ok {
				numAdded += b.maybeAddName(ns, name)
			}
			return true
		})
	}
	return
}

func (b nameFlowGraphBuilder) maybeAddName(ns *nameSet, name *pythonast.NameExpr) int {
	if order, ok := b.names.Get(name); ok {
		// if ns == nil, consider the name added, and assume the caller just wants to check if a name is added
		// note that the total count may be wrong, since the same name added multiple times would be counted multiple times
		if ns == nil || ns.Add(name, order) {
			return 1
		}
	}
	return 0
}

func (b nameFlowGraphBuilder) containsName(node pythonast.Node) bool {
	if pythonast.IsNil(node) {
		return false
	}

	var found bool
	pythonast.Inspect(node, func(n pythonast.Node) bool {
		name, ok := n.(*pythonast.NameExpr)
		if ok {
			if b.names.Contains(name) {
				found = true
			}
		}
		return !found
	})
	return found
}
