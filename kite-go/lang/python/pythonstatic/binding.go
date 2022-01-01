package pythonstatic

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonast"

// functionBinding determines whether a function is a static method/class method by
// looking for the "@staticmethod" and "@classmethod" decorators. This is a heuristic
// because it is possible to redefine these symbols, or even to implement custom
// versions.
func functionBinding(stmt *pythonast.FunctionDefStmt) string {
	for _, dec := range stmt.Decorators {
		if name, isname := dec.(*pythonast.NameExpr); isname {
			if name.Ident.Literal == "classmethod" || name.Ident.Literal == "staticmethod" {
				return name.Ident.Literal
			}
		}
	}
	return ""
}
