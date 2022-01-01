package pythonast

import "fmt"

// CountNodes counts the number of nodes in an AST
func CountNodes(node Node) int {
	var count int
	InspectEdges(node, func(parent, child Node, field string) bool {
		if !IsNil(child) {
			count++
		}
		return true
	})
	return count
}

// ConstructParentTable creates a map from nodes to their parents.
// Nodecount is the number of nodes in the AST, which is used to pre-allocate
// the map. This parameter can be set to zero, in which case the map will grow
// automatically, but note that this will incur additional heap allocations.
func ConstructParentTable(node Node, nodecount int) map[Node]Node {
	parents := make(map[Node]Node, nodecount)
	InspectEdges(node, func(parent, child Node, field string) bool {
		if !IsNil(parent) && !IsNil(child) {
			parents[child] = parent
		}
		return true
	})
	return parents
}

type stmtTableVisitor struct {
	out  map[Expr]Stmt
	stmt Stmt
}

// Visit implements EdgeVisitor
func (v stmtTableVisitor) VisitEdge(parent, node Node, field string) (w EdgeVisitor) {
	if IsNil(node) {
		return nil
	}
	if stmt, ok := node.(Stmt); ok {
		return stmtTableVisitor{v.out, stmt}
	}
	if expr, ok := node.(Expr); ok && v.stmt != nil {
		v.out[expr] = v.stmt
	}
	return v
}

// ConstructStmtTable creates a map from expressions to the most deeply
// nested statement contain them.
// Nodecount is the number of nodes in the AST, which is used to pre-allocate
// the map. This parameter can be set to zero, in which case the map will grow
// automatically, but note that this will incur additional heap allocations.
func ConstructStmtTable(node Node, nodecount int) map[Expr]Stmt {
	out := make(map[Expr]Stmt, nodecount)
	WalkEdges(stmtTableVisitor{out, nil}, node)
	return out
}

// ConstructScopeTable creates a map from expressions to the deepest containing lexical scope, in which name resolution would begin.
// All names in the module have a well defined lexical scope.
// For other expressions, this is equivalent to the scope of a hypothetical name at the expression's position
// NOTES:
//   - For comprehension expressions we match the python 3 behavior and start the scope lookup
//     in the comprehension itself, this matches the behavior in pythonstatic which assigns a new
//     symbol table for each comprehension expression.
func ConstructScopeTable(mod *Module) map[Expr]Scope {
	temp := make(map[Node]Scope)

	out := make(map[Expr]Scope)

	InspectEdges(mod, func(parent, child Node, field string) bool {
		if parent == nil {
			// must be at module
			temp[mod] = mod
			return true
		}

		if child == nil {
			return false
		}

		// determine current scope
		var current Scope
		switch parent := parent.(type) {
		case *ClassDefStmt:
			switch field {
			case "Body":
				// when we recurse into the body of the class we
				// always resolve names in the class scope
				current = parent
			case "Name", "Args", "Decorators", "Vararg", "Kwarg":
				// for all other fields in the class (name, args, decorators) we resolve
				// names in the lexical scope that contains the class def.
				current = temp[parent]
			default:
				panic(fmt.Errorf("unhandled class def field %s", field))
			}
		case *FunctionDefStmt:
			switch field {
			case "Name", "Decorators", "Annotation":
				// name of function, names in decorators decorators, and names in the annotation
				// are all resolved in the scope that contains the parent function def
				current = temp[parent]
			case "Parameters", "Vararg", "Kwarg", "Body":
				// names in all other fields (parameters, body) are resolved in the function def scope
				current = parent
			default:
				panic(fmt.Errorf("unhandled function def field %s", field))
			}
		case *LambdaExpr:
			// names in all fields of the lambda are resolved in the lambda scope
			current = parent
		case Comprehension:
			// names in all fields of a comprehension are resolved in the comprehension scope,
			// NOTE: see comment above, this matches the python 3 behavior.
			current = parent
		case *Module:
			// back in module scope, reset parent
			current = parent
		case *Parameter, *ArgsParameter:
			switch field {
			case "Annotation", "Default":
				// names in the annotation and default for function parameters are resolved in
				// the scope that contains the function def. Here that is
				// the parent of the Parameter since in the function def
				// clause above we resolve the args in the function scope.
				current = temp[temp[parent]]
			case "Name":
				// name of the function parameter is evaluated
				// in the function scope, which is set to the
				// function def in the above switch clause
				current = temp[parent]
			default:
				panic(fmt.Errorf("unhandled field %s for %T", field, parent))
			}
		default:
			current = temp[parent]
		}

		temp[child] = current
		if expr, ok := child.(Expr); ok {
			out[expr] = current
		}
		return true
	})
	return out
}
