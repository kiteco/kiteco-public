package pythonparser

import (
	"fmt"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const undecided = pythonast.Usage(0)

var usageVisitors = sync.Pool{New: func() interface{} {
	return &usageVisitor{}
}}

// MarkUsages sets the Usage field for each NameExpr, AttributeExpr, IndexExpr,
// TupleExpr, and ListExpr. This field indicates whether the expression is being
// evaluated, assigned to, deleted, or imported.
func MarkUsages(ctx kitectx.Context, node pythonast.Node) {
	v := usageVisitors.Get().(*usageVisitor)
	*v = usageVisitor{ctx: ctx}
	pythonast.Walk(v, node)
	usageVisitors.Put(v)
}

func mark(ctx kitectx.Context, usage pythonast.Usage, node pythonast.Node) {
	v := usageVisitors.Get().(*usageVisitor)
	*v = usageVisitor{ctx: ctx, cur: usage}
	pythonast.Walk(v, node)
	usageVisitors.Put(v)
}

type usageVisitor struct {
	ctx kitectx.Context // TODO(naman) avoid storing the kitectx here by threading it through pythonast.Walk
	cur pythonast.Usage
}

func (v *usageVisitor) Visit(n pythonast.Node) pythonast.Visitor {
	v.ctx.CheckAbort()

	if n == nil {
		return nil
	}

	switch n := n.(type) {
	case *pythonast.NameExpr:
		if v.cur == undecided {
			// It makes sense to panic here because this should be impossible no matter what's in the AST
			panic(fmt.Errorf("visited %s without having a usage to assign", pythonast.String(n)))
		}
		n.Usage = v.cur
		return nil

	case *pythonast.AttributeExpr:
		if v.cur == undecided {
			// It makes sense to panic here because this should be impossible no matter what's in the AST
			panic(fmt.Errorf("visited %s without having a usage to assign", pythonast.String(n)))
		}
		n.Usage = v.cur
		mark(v.ctx, pythonast.Evaluate, n.Value)
		return nil

	case *pythonast.IndexExpr:
		if v.cur == undecided {
			// It makes sense to panic here because this should be impossible no matter what's in the AST
			panic(fmt.Errorf("visited %s without having a usage to assign", pythonast.String(n)))
		}
		n.Usage = v.cur
		mark(v.ctx, pythonast.Evaluate, n.Value)
		for _, sub := range n.Subscripts {
			mark(v.ctx, pythonast.Evaluate, sub)
		}
		return nil

	case *pythonast.TupleExpr:
		if v.cur == undecided {
			// It makes sense to panic here because this should be impossible no matter what's in the AST
			panic(fmt.Errorf("visited %s without having a usage to assign", pythonast.String(n)))
		}
		n.Usage = v.cur
		for _, expr := range n.Elts {
			mark(v.ctx, v.cur, expr)
		}
		return nil

	case *pythonast.ListExpr:
		if v.cur == undecided {
			// It makes sense to panic here because this should be impossible no matter what's in the AST
			panic(fmt.Errorf("visited %s without having a usage to assign", pythonast.String(n)))
		}
		n.Usage = v.cur
		for _, expr := range n.Values {
			mark(v.ctx, v.cur, expr)
		}
		return nil

	case *pythonast.Argument:
		if n.Name != nil {
			mark(v.ctx, pythonast.Assign, n.Name)
		}
		mark(v.ctx, pythonast.Evaluate, n.Value)
		return nil

	case *pythonast.ExprStmt:
		mark(v.ctx, pythonast.Evaluate, n.Value)
		return nil

	case *pythonast.AnnotationStmt:
		mark(v.ctx, pythonast.Assign, n.Target)
		mark(v.ctx, pythonast.Evaluate, n.Annotation)
		return nil

	case *pythonast.AssignStmt:
		for _, expr := range n.Targets {
			mark(v.ctx, pythonast.Assign, expr)
		}
		if !pythonast.IsNil(n.Annotation) {
			mark(v.ctx, pythonast.Evaluate, n.Annotation)
		}
		mark(v.ctx, pythonast.Evaluate, n.Value)
		return nil

	case *pythonast.AugAssignStmt:
		mark(v.ctx, pythonast.Assign, n.Target)
		mark(v.ctx, pythonast.Evaluate, n.Value)
		return nil

	case *pythonast.DelStmt:
		for _, expr := range n.Targets {
			mark(v.ctx, pythonast.Delete, expr)
		}
		return nil

	case *pythonast.YieldStmt:
		if n.Value != nil {
			mark(v.ctx, pythonast.Evaluate, n.Value)
		}
		return nil

	case *pythonast.AssertStmt:
		mark(v.ctx, pythonast.Evaluate, n.Condition)
		if !pythonast.IsNil(n.Message) {
			mark(v.ctx, pythonast.Evaluate, n.Message)
		}
		return nil

	case *pythonast.ExecStmt:
		if n.Body != nil {
			mark(v.ctx, pythonast.Evaluate, n.Body)
		}
		if !pythonast.IsNil(n.Globals) {
			mark(v.ctx, pythonast.Evaluate, n.Globals)
		}
		if !pythonast.IsNil(n.Locals) {
			mark(v.ctx, pythonast.Evaluate, n.Locals)
		}
		return nil

	case *pythonast.RaiseStmt:
		if !pythonast.IsNil(n.Type) {
			mark(v.ctx, pythonast.Evaluate, n.Type)
		}
		if !pythonast.IsNil(n.Instance) {
			mark(v.ctx, pythonast.Evaluate, n.Instance)
		}
		if !pythonast.IsNil(n.Traceback) {
			mark(v.ctx, pythonast.Evaluate, n.Traceback)
		}
		return nil

	case *pythonast.ReturnStmt:
		if n.Value != nil {
			mark(v.ctx, pythonast.Evaluate, n.Value)
		}
		return nil

	case *pythonast.GlobalStmt:
		for _, expr := range n.Names {
			mark(v.ctx, pythonast.Assign, expr)
		}
		return nil

	case *pythonast.NonLocalStmt:
		for _, expr := range n.Names {
			mark(v.ctx, pythonast.Assign, expr)
		}
		return nil

	case *pythonast.PrintStmt:
		for _, expr := range n.Values {
			mark(v.ctx, pythonast.Evaluate, expr)
		}
		if !pythonast.IsNil(n.Dest) {
			mark(v.ctx, pythonast.Evaluate, n.Dest)
		}
		return nil

	case *pythonast.Branch:
		if !pythonast.IsNil(n.Condition) {
			mark(v.ctx, pythonast.Evaluate, n.Condition)
		}
		for _, stmt := range n.Body {
			mark(v.ctx, undecided, stmt)
		}
		return nil

	case *pythonast.WhileStmt:
		mark(v.ctx, pythonast.Evaluate, n.Condition)
		for _, stmt := range n.Body {
			mark(v.ctx, undecided, stmt)
		}
		for _, stmt := range n.Else {
			mark(v.ctx, undecided, stmt)
		}
		return nil

	case *pythonast.ForStmt:
		for _, target := range n.Targets {
			mark(v.ctx, pythonast.Assign, target)
		}
		mark(v.ctx, pythonast.Evaluate, n.Iterable)
		for _, stmt := range n.Body {
			mark(v.ctx, undecided, stmt)
		}
		for _, stmt := range n.Else {
			mark(v.ctx, undecided, stmt)
		}
		return nil

	case *pythonast.WithItem:
		if !pythonast.IsNil(n.Target) {
			mark(v.ctx, pythonast.Assign, n.Target)
		}
		mark(v.ctx, pythonast.Evaluate, n.Value)
		return nil

	case *pythonast.ExceptClause:
		if !pythonast.IsNil(n.Target) {
			mark(v.ctx, pythonast.Assign, n.Target)
		}
		if !pythonast.IsNil(n.Type) {
			mark(v.ctx, pythonast.Evaluate, n.Type)
		}
		for _, stmt := range n.Body {
			mark(v.ctx, undecided, stmt)
		}
		return nil

	case *pythonast.ArgsParameter:
		mark(v.ctx, pythonast.Assign, n.Name)
		if !pythonast.IsNil(n.Annotation) {
			mark(v.ctx, pythonast.Evaluate, n.Annotation)
		}
		return nil

	case *pythonast.Parameter:
		mark(v.ctx, pythonast.Assign, n.Name)
		if !pythonast.IsNil(n.Annotation) {
			mark(v.ctx, pythonast.Evaluate, n.Annotation)
		}
		if !pythonast.IsNil(n.Default) {
			mark(v.ctx, pythonast.Evaluate, n.Default)
		}
		return nil

	case *pythonast.FunctionDefStmt:
		for _, dec := range n.Decorators {
			mark(v.ctx, pythonast.Evaluate, dec)
		}
		mark(v.ctx, pythonast.Assign, n.Name)
		if !pythonast.IsNil(n.Vararg) {
			mark(v.ctx, pythonast.Assign, n.Vararg)
		}
		if !pythonast.IsNil(n.Kwarg) {
			mark(v.ctx, pythonast.Assign, n.Kwarg)
		}
		for _, param := range n.Parameters {
			mark(v.ctx, pythonast.Assign, param)
		}
		if !pythonast.IsNil(n.Annotation) {
			mark(v.ctx, pythonast.Evaluate, n.Annotation)
		}
		for _, stmt := range n.Body {
			mark(v.ctx, undecided, stmt)
		}
		return nil

	case *pythonast.ClassDefStmt:
		for _, dec := range n.Decorators {
			mark(v.ctx, pythonast.Evaluate, dec)
		}
		mark(v.ctx, pythonast.Assign, n.Name)
		for _, arg := range n.Args {
			mark(v.ctx, pythonast.Evaluate, arg)
		}
		if !pythonast.IsNil(n.Vararg) {
			mark(v.ctx, pythonast.Evaluate, n.Vararg)
		}
		if !pythonast.IsNil(n.Kwarg) {
			mark(v.ctx, pythonast.Evaluate, n.Kwarg)
		}
		for _, stmt := range n.Body {
			mark(v.ctx, undecided, stmt)
		}
		return nil

	case *pythonast.ImportAsName:
		mark(v.ctx, pythonast.Import, n.External)
		if n.Internal != nil {
			mark(v.ctx, pythonast.Assign, n.Internal)
		}
		return nil

	case *pythonast.DottedAsName:
		mark(v.ctx, pythonast.Import, n.External)
		if n.Internal != nil {
			mark(v.ctx, pythonast.Assign, n.Internal)
		}
		return nil

	case *pythonast.ImportFromStmt:
		if n.Package != nil {
			mark(v.ctx, pythonast.Import, n.Package)
		}
		for _, clause := range n.Names {
			mark(v.ctx, undecided, clause)
		}
		return nil

	default:
		// keep doing whatever we're already doing
		return v
	}
}
