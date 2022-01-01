package pythonast

// A StatementTransformer inputs a statement and outputs zero or more replacement statements.
type StatementTransformer interface {
	// Transform inputs a statement and outputs zero or more replacement statements.
	Transform(stmt Stmt) []Stmt
}

// TransformStatements replaces each statement in the AST with the result of
// calling the transformer. The node is modified in-place.
func TransformStatements(f StatementTransformer, n Node) {
	Walk(transformVisitor{n, f}, n)
}

// transformVisitor adapts a StatementTransformer to a Visitor
type transformVisitor struct {
	n Node
	f StatementTransformer
}

func (t transformVisitor) Visit(n Node) Visitor {
	if n != nil {
		return transformVisitor{n, t.f}
	}

	switch n := t.n.(type) {
	case *Module:
		n.Body = transformStmtList(n.Body, t.f)
	case *ClassDefStmt:
		n.Body = transformStmtList(n.Body, t.f)
	case *FunctionDefStmt:
		n.Body = transformStmtList(n.Body, t.f)
	case *Branch:
		n.Body = transformStmtList(n.Body, t.f)
	case *IfStmt:
		n.Else = transformStmtList(n.Else, t.f)
	case *ForStmt:
		n.Body = transformStmtList(n.Body, t.f)
		n.Else = transformStmtList(n.Else, t.f)
	case *WhileStmt:
		n.Body = transformStmtList(n.Body, t.f)
		n.Else = transformStmtList(n.Else, t.f)
	case *ExceptClause:
		n.Body = transformStmtList(n.Body, t.f)
	case *TryStmt:
		n.Body = transformStmtList(n.Body, t.f)
		n.Else = transformStmtList(n.Body, t.f)
		n.Finally = transformStmtList(n.Body, t.f)
	case *WithStmt:
		n.Body = transformStmtList(n.Body, t.f)
	}
	return nil
}

func transformStmtList(stmts []Stmt, f StatementTransformer) []Stmt {
	var out []Stmt
	for _, stmt := range stmts {
		out = append(out, f.Transform(stmt)...)
	}
	return out
}
