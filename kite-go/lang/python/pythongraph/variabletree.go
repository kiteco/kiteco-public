package pythongraph

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
)

type block struct {
	// header variables that are associated with the
	// body but not part of it,
	// e.g
	// for x in []:
	//   y = 1 # x is in scope here but not part of any statements in the body
	header []*variable
	body   []pythonast.Stmt
	// variables that were defined with the execution of
	// each statement in body
	// e.g suppose
	// x = 1 # body[0], variables[0] = [x]
	// y = 2 # body[1], variables[1] = [y]
	variables [][]*variable
}

func newBlock(header []*variable, body []pythonast.Stmt, nameToVariable nameToVariable) block {
	getVariable := func(name *pythonast.NameExpr) *variable {
		v := nameToVariable[name]
		if v != nil && v.Origin == name {
			return v
		}
		return nil
	}

	var getVariables func(pythonast.Stmt) []*variable
	getVariables = func(stmt pythonast.Stmt) []*variable {
		var vs []*variable
		switch stmt := stmt.(type) {
		case *pythonast.BadStmt:
			for _, s := range stmt.Approximation {
				vs = append(vs, getVariables(s)...)
			}
		case *pythonast.FunctionDefStmt:
			if v := getVariable(stmt.Name); v != nil {
				vs = append(vs, v)
			}
		case *pythonast.ClassDefStmt:
			if v := getVariable(stmt.Name); v != nil {
				vs = append(vs, v)
			}
		default:
			// APPROXIMATION: after any other statement besides the above is executed
			// then all variables defined in that statement will be in scope.
			pythonast.Inspect(stmt, func(n pythonast.Node) bool {
				if name, ok := n.(*pythonast.NameExpr); ok {
					if v := getVariable(name); v != nil {
						vs = append(vs, v)
					}
				}
				return true
			})
		}
		return vs
	}

	var vars [][]*variable
	for _, stmt := range body {

		vs := getVariables(stmt)

		sortVars(vs)

		vars = append(vars, vs)
	}

	return block{
		header:    header,
		body:      body,
		variables: vars,
	}
}

func (b block) variablesUpTo(node pythonast.Node, includeHeader bool) []*variable {
	var vars []*variable
	if includeHeader {
		vars = b.header
	}

	i := 0
	for ; i < len(b.body); i++ {
		stmt := b.body[i]

		// node intersects statement
		if node.Begin() >= stmt.Begin() && node.End() <= stmt.End() {
			break
		}

		// statement is past node
		if stmt.Begin() >= node.End() {
			break
		}
	}

	for j, vs := range b.variables {
		if i == j {
			break
		}

		vars = append(vars, vs...)
	}

	return vars
}

func (b block) covers(node pythonast.Node) bool {
	first := b.body[0]
	if node.End() <= first.Begin() {
		return false
	}

	last := b.body[len(b.body)-1]
	if node.Begin() >= last.End() {
		return false
	}

	return true
}

func headerVars(node pythonast.Node, nameToVariable nameToVariable) []*variable {
	var headerNames []*pythonast.NameExpr
	switch header := node.(type) {
	case *pythonast.ForStmt:
		headerNames = append(headerNames, namesFromExprs(header.Targets...)...)
	case *pythonast.WithStmt:
		for _, item := range header.Items {
			headerNames = append(headerNames, namesFromExprs(item.Target)...)
		}
	case *pythonast.FunctionDefStmt:
		for _, param := range header.Parameters {
			headerNames = append(headerNames, namesFromExprs(param.Name)...)
		}

		if header.Vararg != nil {
			headerNames = append(headerNames, header.Vararg.Name)
		}

		if header.Kwarg != nil {
			headerNames = append(headerNames, header.Kwarg.Name)
		}
	}

	var vars []*variable
	for _, name := range headerNames {
		v := nameToVariable[name]
		if v != nil && v.Origin == name {
			vars = append(vars, v)
		}
	}

	sortVars(vars)
	return vars
}

// variableTreeNode is a node in a variable tree
// which allows us to approximate the answer to questions
// of the form which variables are in scope at a given ast node.
// Each node in the tree stores two things:
//
// 1) a block (aka a sequence of statements),
// these blocks are simple in the sense that they have no
// children, for each statement in a block we store which variables
// come into scope after execution of the statement.
// Each block also comes equipped with a "header", which allows
// us to talk about variables that come into scope as part of the
// execution of a block of code but are not defined in the sequence of statements
// associated with the block.
// An example is the 'x' in
// for x in []:
//   pass
// SEE: headerVars for more details.
//
// 2) children nodes, we create these children each time we
// encounter a statement in the block that potentially introduces new variables that
// have special scoping rules. Of particular interest are For/While loops, If statements
// and function and class definitions.
//
// NOTE: a variable may be defined in multiple blocks
// e.g consider:
// y = 1
// for x in [1]:
//   z = 1
//   print(x + z)
// The node associated with the top level module has 2 statements
// in its block, after the first statement (y = 1) the variable y is in scope,
// after the second statement (the for statement) the variables x and z come into scope.
// Note that the top level node also has a single child node c associated with the for loop,
// the header for the block of the child node c contains the variable x, and the first statement
// in the block of the child node c contains the variable z.
//
// NOTE: blocks are disjoint in the sense that each statement is associated with exactly one block,
// BUT an interval of source code may be covered by more than one node, again consider the example
// above, the statement z = 1 is associated with only the block in child c, but the interval
// of source code contain z =1 is sovered by both the node associated with the module and the node
// associated with the child node c.
type variableTreeNode struct {
	// ast node associated with the variable tree node, may be nil
	ast      pythonast.Node
	blk      block
	children []*variableTreeNode
}

func newVariableTree(mod *pythonast.Module, nameToVariable nameToVariable) *variableTreeNode {
	return newVariableTreeNode(mod, mod.Body, nameToVariable)
}

func newVariableTreeNode(ast pythonast.Node, suite []pythonast.Stmt, nameToVariable nameToVariable) *variableTreeNode {
	header := headerVars(ast, nameToVariable)

	vtn := &variableTreeNode{
		ast: ast,
		blk: newBlock(header, suite, nameToVariable),
	}

	for _, stmt := range suite {
		switch stmt := stmt.(type) {
		case *pythonast.BadStmt: // TODO
		case *pythonast.TryStmt: // TODO
		case *pythonast.ClassDefStmt:
			child := newVariableTreeNode(stmt, stmt.Body, nameToVariable)
			vtn.children = append(vtn.children, child)
		case *pythonast.FunctionDefStmt:
			child := newVariableTreeNode(stmt, stmt.Body, nameToVariable)
			vtn.children = append(vtn.children, child)
		case *pythonast.IfStmt:
			// TODO: unclear what the ast node should be here for each branch and the else block
			for _, branch := range stmt.Branches {
				vtn.children = append(vtn.children, newVariableTreeNode(nil, branch.Body, nameToVariable))
			}

			if len(stmt.Else) > 0 {
				vtn.children = append(vtn.children, newVariableTreeNode(nil, stmt.Else, nameToVariable))
			}
		case *pythonast.WithStmt:
			child := newVariableTreeNode(stmt, stmt.Body, nameToVariable)
			vtn.children = append(vtn.children, child)
		case *pythonast.ForStmt:
			// TODO: kind of nasty, we want to make sure that all
			// of the variables defined in the for block are in scope in the else
			// block, this is in contrast to the structure in the if/else statement
			// in which we want separate clauses to have "separate scopes" until after the
			// statement is "executed".
			body := append(stmt.Body, stmt.Else...)
			child := newVariableTreeNode(stmt, body, nameToVariable)
			vtn.children = append(vtn.children, child)
		case *pythonast.WhileStmt:
			// TODO: kind of nasty, we want to make sure that all
			// of the variables defined in the for block are in scope in the else
			// block, this is in contrast to the structure in the if/else statement
			// in which we want separate clauses to have "separate scopes" until after the
			// statement is "executed".
			body := append(stmt.Body, stmt.Else...)
			child := newVariableTreeNode(stmt, body, nameToVariable)
			vtn.children = append(vtn.children, child)
		}
	}

	return vtn
}

func (vtn *variableTreeNode) coveringNodes(node pythonast.Node) []*variableTreeNode {
	if !vtn.blk.covers(node) {
		return nil
	}

	nodes := []*variableTreeNode{vtn}
	for _, child := range vtn.children {
		nodes = append(nodes, child.coveringNodes(node)...)
	}

	return nodes
}

func (vtn *variableTreeNode) variablesUpTo(ast pythonast.Node, stopAtFunc bool) []*variable {

	nodes := vtn.coveringNodes(ast)

	var vars []*variable

	// start from deepest node
	// and stop at first function scope found
	for i := len(nodes) - 1; i > -1; i-- {
		node := nodes[i]
		if _, ok := node.ast.(*pythonast.ClassDefStmt); ok {
			// class member variables are not
			// exposed to child scopes
			// TODO: pretty hacky
			// TODO: do we need this,
			// pretty sure that these name expressions
			// will not be associated with a symbol anyways
			continue
		}

		vars = append(vars, node.blk.variablesUpTo(ast, true)...)

		if stopAtFunc {
			if _, ok := node.ast.(*pythonast.FunctionDefStmt); ok {
				break
			}
		}
	}
	return vars
}

func (vtn *variableTreeNode) InScope(at pythonast.Node, stopAtFunc bool) scope {
	vars := vtn.variablesUpTo(at, stopAtFunc)
	sortVars(vars)
	return scope(vars)
}
