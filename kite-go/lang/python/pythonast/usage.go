package pythonast

// Usage indicates whether an expression was being evaluated, assigned, deleted, or imported.
// In the following examples, "x" will have Usage=Evaluate:
//    print x
//    dosomething(**x)
//    another = x
//    import nump as x  <-- because the numpy module will be assigned to "x"
//    x.y = 3           <-- because "x" is being loaded even though "x.y" is being assigned to
// In the following examples, "x" will have Usage=Assign:
//    x = 3
//    def foo(x): pass
//    def foo(*x): pass
//    def foo(ham, (w, (x, y)), spam): pass
//    lambda x: something
//    x, y, z = something()
// In the following examples, "x" will have Usage=Delete:
//    del x
// In the following examples, "x" will have Usage=Import:
//    import x
//    from somepackage import x as y
type Usage int

const (
	// Invalid Usage
	Invalid Usage = iota
	// Evaluate is for expressions that are being evaluated
	Evaluate
	// Assign is for expressions that are being assigned to
	Assign
	// Delete is for expressions that are being deleted
	Delete
	// Import is for expressions that are being imported
	Import
)

func (u Usage) String() string {
	switch u {
	case Evaluate:
		return "Evaluate"
	case Assign:
		return "Assign"
	case Delete:
		return "Delete"
	case Import:
		return "Import"
	default:
		return "Invalid"
	}
}

// GetUsage returns the Usage for the given Expr
func GetUsage(expr Expr) Usage {
	if IsNil(expr) {
		return Invalid
	}

	switch expr := expr.(type) {
	case *NameExpr:
		return expr.Usage
	case *TupleExpr:
		return expr.Usage
	case *IndexExpr:
		return expr.Usage
	case *AttributeExpr:
		return expr.Usage
	case *ListExpr:
		return expr.Usage
	default:
		return Evaluate
	}
}
