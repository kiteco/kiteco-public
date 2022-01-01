"""
package pythonast

import (
	"bytes"
	"fmt"
	"go/token"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// Node represents an item in the syntax tree
type Node interface {
	Begin() token.Pos // TODO(naman) implement using Iterate?
	End() token.Pos   // TODO(naman) implement using Iterate?
	// adds i to the position of any contained non-recursively contained pythonscanner.Words
	AddOffset(i int)    // TODO(naman) implement using Iterate
	walk(v EdgeVisitor) // TODO(naman) implement using Iterate

	// Iterate iterates over the receiver's contained Nodes and Words; it should not recurse
	Iterate(IterationHandler)
	// CopyIterable makes a shallow copy of the receiver such that each visited reference during iteration is at a new memory location
	// in particular it copies contained slices (but not contained Nodes or *Words, since those are visited by reference)
	CopyIterable() Node
}

// Subscript represents a node that can appear within slices in the syntax tree
type Subscript interface {
	Node
	subscriptNode()
}

// Expr represents expressions in the syntax tree
type Expr interface {
	Node
	exprNode()
}

// Stmt represents statements in the syntax tree
type Stmt interface {
	Node
	stmtNode()
}

// Scope represents nodes that define a lexical scope.
type Scope interface {
	Node
	scopeNode()
}

// Comprehension represents nodes that are list/set/dict comprehensions
type Comprehension interface {
	Scope
	comprehensionNode()
}

// IsNil determines whether a node is nil. There is really no other (safe, efficient) way to do this!
func IsNil(node Node) bool {
	if node == nil {
		return true
	}
	switch node := node.(type) {
	case *NameExpr:
		return node == nil
	case *TupleExpr:
		return node == nil
	case *IndexExpr:
		return node == nil
	case *AttributeExpr:
		return node == nil
	case *NumberExpr:
		return node == nil
	case *StringExpr:
		return node == nil
	case *ListExpr:
		return node == nil
	case *SetExpr:
		return node == nil
	case *DictExpr:
		return node == nil
	case *ComprehensionExpr:
		return node == nil
	case *ListComprehensionExpr:
		return node == nil
	case *DictComprehensionExpr:
		return node == nil
	case *SetComprehensionExpr:
		return node == nil
	case *UnaryExpr:
		return node == nil
	case *BinaryExpr:
		return node == nil
	case *CallExpr:
		return node == nil
	case *LambdaExpr:
		return node == nil
	case *ReprExpr:
		return node == nil
	case *IfExpr:
		return node == nil
	case *YieldExpr:
		return node == nil
	case *AwaitExpr:
		return node == nil
	case *BadExpr:
		return node == nil
	case *DottedExpr:
		return node == nil
	case *DottedAsName:
		return node == nil
	case *ImportAsName:
		return node == nil
	case *ImportNameStmt:
		return node == nil
	case *ImportFromStmt:
		return node == nil
	case *IndexSubscript:
		return node == nil
	case *SliceSubscript:
		return node == nil
	case *EllipsisExpr:
		return node == nil
	case *KeyValuePair:
		return node == nil
	case *Generator:
		return node == nil
	case *Argument:
		return node == nil
	case *BadStmt:
		return node == nil
	case *ExprStmt:
		return node == nil
	case *AnnotationStmt:
		return node == nil
	case *AssignStmt:
		return node == nil
	case *AugAssignStmt:
		return node == nil
	case *ClassDefStmt:
		return node == nil
	case *Parameter:
		return node == nil
	case *ArgsParameter:
		return node == nil
	case *FunctionDefStmt:
		return node == nil
	case *AssertStmt:
		return node == nil
	case *ContinueStmt:
		return node == nil
	case *BreakStmt:
		return node == nil
	case *DelStmt:
		return node == nil
	case *ExecStmt:
		return node == nil
	case *PassStmt:
		return node == nil
	case *PrintStmt:
		return node == nil
	case *RaiseStmt:
		return node == nil
	case *ReturnStmt:
		return node == nil
	case *YieldStmt:
		return node == nil
	case *GlobalStmt:
		return node == nil
	case *NonLocalStmt:
		return node == nil
	case *Branch:
		return node == nil
	case *IfStmt:
		return node == nil
	case *ForStmt:
		return node == nil
	case *WhileStmt:
		return node == nil
	case *ExceptClause:
		return node == nil
	case *TryStmt:
		return node == nil
	case *WithItem:
		return node == nil
	case *WithStmt:
		return node == nil
	case *Module:
		return node == nil
	}
	panic(fmt.Sprintf("unknown node type: %T", node))
}

// IsLiteral determines if the provided expression represents a python literal.
func IsLiteral(expr Expr) bool {
	switch expr.(type) {
	case *NumberExpr, *StringExpr,
		*TupleExpr, *ListExpr, *SetExpr, *DictExpr,
		*ComprehensionExpr, *ListComprehensionExpr, *DictComprehensionExpr,
		*SetComprehensionExpr, *LambdaExpr, *ReprExpr,
		*YieldExpr, *IfExpr:
		return true
	default:
		return false
	}
}

// IsTerminal node in the python grammar.
func IsTerminal(node Node) bool {
	switch node.(type) {
	case *NameExpr, *EllipsisExpr,
		*PassStmt, *ContinueStmt,
		*BreakStmt, *StringExpr,
		*NumberExpr:
		return true
	default:
		return false
	}
}

// ---
// Helpers for Begin() and End() positions

func switchBegin(xs ...Expr) token.Pos {
	for _, x := range xs {
		if !IsNil(x) {
			return x.Begin()
		}
	}
	panic("could not compute begin position because all children were nil")
}

func switchEnd(xs ...Expr) token.Pos {
	for i := len(xs) - 1; i >= 0; i-- {
		if !IsNil(xs[i]) {
			return xs[i].End()
		}
	}
	panic("could not compute end position because all children were nil")
}

func addWordOffset(w *pythonscanner.Word, n int) {
	if w != nil {
		w.Begin = token.Pos(int(w.Begin) + n)
		w.End = token.Pos(int(w.End) + n)
	}
}

// ---
// Imports

// DottedAsName (used in imports only)
type DottedAsName struct {
	External *DottedExpr
	Internal *NameExpr
}

// Begin gets the byte offset of the first character in this node
func (n *DottedAsName) Begin() token.Pos { return n.External.Begin() }

// End gets the byte offset one past the last character in this node
func (n *DottedAsName) End() token.Pos {
	if n.Internal != nil {
		return n.Internal.End()
	}
	return n.External.End()
}

func (n *DottedAsName) walk(v EdgeVisitor) {
	walkEdge(v, n, n.External, "External")
	if n.Internal != nil {
		walkEdge(v, n, n.Internal, "Internal")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *DottedAsName) AddOffset(i int) {}

// Iterate implements Node
func (n *DottedAsName) Iterate(h IterationHandler) {
	h.VisitNode(dottedExprRef{&n.External})
	h.VisitNode(nameExprRef{&n.Internal})
}

// CopyIterable implements Node
func (n *DottedAsName) CopyIterable() Node {
	new := *n
	return &new
}

// ImportAsName (used in imports only)
type ImportAsName struct {
	External *NameExpr
	Internal *NameExpr // can be nil
}

// Begin gets the byte offset of the first character in this node
func (n *ImportAsName) Begin() token.Pos { return n.External.Begin() }

// End gets the byte offset one past the last character in this node
func (n *ImportAsName) End() token.Pos { return switchEnd(n.External, n.Internal) }

func (n *ImportAsName) walk(v EdgeVisitor) {
	walkEdge(v, n, n.External, "External")
	if n.Internal != nil {
		walkEdge(v, n, n.Internal, "Internal")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ImportAsName) AddOffset(i int) {}

// Iterate implements Node
func (n *ImportAsName) Iterate(h IterationHandler) {
	h.VisitNode(nameExprRef{&n.External})
	h.VisitNode(nameExprRef{&n.Internal})
}

// CopyIterable implements Node
func (n *ImportAsName) CopyIterable() Node {
	new := *n
	return &new
}

// ImportNameStmt - e.g. "import foo.bar as baz, ham as spam"
type ImportNameStmt struct {
	Import *pythonscanner.Word
	Names  []*DottedAsName
	Commas []*pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *ImportNameStmt) Begin() token.Pos { return n.Import.Begin }

// End gets the byte offset one past the last character in this node
func (n *ImportNameStmt) End() token.Pos {
	switch {
	case len(n.Names) > 0 && len(n.Commas) > 0:
		name := n.Names[len(n.Names)-1]
		comma := n.Commas[len(n.Commas)-1]
		if name.End() > comma.End {
			return name.End()
		}
		return comma.End
	case len(n.Names) > 0:
		return n.Names[len(n.Names)-1].End()
	case len(n.Commas) > 0:
		return n.Commas[len(n.Commas)-1].End
	default:
		return n.Import.End
	}
}

func (n *ImportNameStmt) stmtNode() {}
func (n *ImportNameStmt) walk(v EdgeVisitor) {
	for _, name := range n.Names {
		walkEdge(v, n, name, "Names")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ImportNameStmt) AddOffset(i int) {
	addWordOffset(n.Import, i)
	for _, comma := range n.Commas {
		addWordOffset(comma, i)
	}
}

// Iterate implements Node
func (n *ImportNameStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Import)
	h.VisitSlice(dottedAsNameSlice{&n.Names})
	for i := range n.Commas {
		h.VisitWord(&n.Commas[i])
	}
}

// CopyIterable implements Node
func (n *ImportNameStmt) CopyIterable() Node {
	new := *n
	new.Names = append([]*DottedAsName{}, new.Names...)
	new.Commas = append([]*pythonscanner.Word{}, new.Commas...)
	return &new
}

// ImportFromStmt - e.g. "from foo import bar, ham as spam"
type ImportFromStmt struct {
	From       *pythonscanner.Word // From is the "from" token
	Package    *DottedExpr
	Dots       []*pythonscanner.Word // Leading dots before package name
	Import     *pythonscanner.Word   // Import is the "import" token
	LeftParen  *pythonscanner.Word   // LeftParen is the left parenthesis (python3)
	Wildcard   *pythonscanner.Word   // Wildcard will be nil except for wildcard imports
	Names      []*ImportAsName       // Names are the names being imported
	Commas     []*pythonscanner.Word // Commas are the commas between the imported names
	RightParen *pythonscanner.Word   // RightParen is the right parenthesis (python3)
}

// Begin gets the byte offset of the first character in this node
func (n *ImportFromStmt) Begin() token.Pos { return n.From.Begin }

// End gets the byte offset one past the last character in this node
func (n *ImportFromStmt) End() token.Pos {
	switch {
	case n.RightParen != nil:
		return n.RightParen.End
	case n.Wildcard != nil:
		return n.Wildcard.End
	case len(n.Names) > 0 && len(n.Commas) > 0:
		name := n.Names[len(n.Names)-1]
		comma := n.Commas[len(n.Commas)-1]
		if name.End() > comma.End {
			return name.End()
		}
		return comma.End
	case len(n.Names) > 0:
		return n.Names[len(n.Names)-1].End()
	case len(n.Commas) > 0:
		return n.Commas[len(n.Commas)-1].End
	case n.Import != nil:
		return n.Import.End
	case !IsNil(n.Package):
		return n.Package.End()
	case len(n.Dots) > 0:
		return n.Dots[len(n.Dots)-1].End
	default:
		return n.From.End
	}
}

func (n *ImportFromStmt) stmtNode() {}
func (n *ImportFromStmt) walk(v EdgeVisitor) {
	if n.Package != nil {
		walkEdge(v, n, n.Package, "Package")
	}
	for _, name := range n.Names {
		walkEdge(v, n, name, "Names")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ImportFromStmt) AddOffset(i int) {
	addWordOffset(n.From, i)
	addWordOffset(n.Import, i)
	addWordOffset(n.LeftParen, i)
	addWordOffset(n.Wildcard, i)
	addWordOffset(n.RightParen, i)
	for _, dot := range n.Dots {
		addWordOffset(dot, i)
	}
	for _, comma := range n.Commas {
		addWordOffset(comma, i)
	}
}

// Iterate implements Node
func (n *ImportFromStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.From)
	h.VisitNode(dottedExprRef{&n.Package})
	h.VisitWord(&n.Import)
	h.VisitWord(&n.LeftParen)

	h.VisitSlice(importAsNameSlice{&n.Names})
	for i := range n.Commas {
		h.VisitWord(&n.Commas[i])
	}
	h.VisitWord(&n.RightParen)

	h.VisitWord(&n.Wildcard)
}

// CopyIterable implements Node
func (n *ImportFromStmt) CopyIterable() Node {
	new := *n
	new.Dots = append([]*pythonscanner.Word{}, n.Dots...)
	new.Names = append([]*ImportAsName{}, n.Names...)
	new.Commas = append([]*pythonscanner.Word{}, n.Commas...)
	return &new
}

// ---
// Subscripts

// IndexSubscript represents an index, e.g. "123"
type IndexSubscript struct {
	Value Expr
}

// Begin gets the byte offset of the first character in this node
func (n *IndexSubscript) Begin() token.Pos { return n.Value.Begin() }

// End gets the byte offset one past the last character in this node
func (n *IndexSubscript) End() token.Pos { return n.Value.End() }

func (n *IndexSubscript) subscriptNode() {}
func (n *IndexSubscript) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *IndexSubscript) AddOffset(i int) {}

// Iterate implements Node
func (n *IndexSubscript) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *IndexSubscript) CopyIterable() Node {
	new := *n
	return &new
}

// SliceSubscript represents a slice, e.g. "4:5:6"
type SliceSubscript struct {
	Lower       Expr
	FirstColon  *pythonscanner.Word
	Upper       Expr
	SecondColon *pythonscanner.Word
	Step        Expr
}

// Begin gets the byte offset of the first character in this node
func (n *SliceSubscript) Begin() token.Pos {
	if !IsNil(n.Lower) {
		return n.Lower.Begin()
	}
	return n.FirstColon.Begin
}

// End gets the byte offset one past the last character in this node
func (n *SliceSubscript) End() token.Pos {
	if !IsNil(n.Step) {
		return n.Step.End()
	}
	if n.SecondColon != nil {
		return n.SecondColon.End
	}
	if !IsNil(n.Upper) {
		return n.Upper.End()
	}
	if n.FirstColon != nil {
		return n.FirstColon.End
	}
	return n.Lower.End()
}

func (n *SliceSubscript) subscriptNode() {}
func (n *SliceSubscript) walk(v EdgeVisitor) {
	if !IsNil(n.Lower) {
		walkEdge(v, n, n.Lower, "Lower")
	}
	if !IsNil(n.Upper) {
		walkEdge(v, n, n.Upper, "Upper")
	}
	if !IsNil(n.Step) {
		walkEdge(v, n, n.Step, "Step")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *SliceSubscript) AddOffset(i int) {
	addWordOffset(n.FirstColon, i)
	addWordOffset(n.SecondColon, i)
}

// Iterate implements Node
func (n *SliceSubscript) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Lower})
	h.VisitWord(&n.FirstColon)
	h.VisitNode(exprRef{&n.Upper})
	h.VisitWord(&n.SecondColon)
	h.VisitNode(exprRef{&n.Step})
}

// CopyIterable implements Node
func (n *SliceSubscript) CopyIterable() Node {
	new := *n
	return &new
}

// ---
// Expressions

// BadExpr respresents an expression that was not parseable.
type BadExpr struct {
	// TODO(naman) why do we need From/To in addition to Word?
	From token.Pos
	To   token.Pos
	Word *pythonscanner.Word

	// Approximation contains the Exprs that were (approximately) parsed
	// in the BadExpr region, these expressions may
	// not be syntactically correct and may contain BadTokens
	// or BadExprs.
	Approximation []Expr
}

// Begin gets the byte offset of the first character in this node
func (n *BadExpr) Begin() token.Pos { return n.From }

// End gets the byte offset one past the last character in this node
func (n *BadExpr) End() token.Pos { return n.To }

func (n *BadExpr) exprNode() {}
func (n *BadExpr) walk(v EdgeVisitor) {
	if n.Approximation != nil {
		walkExprList(v, n, n.Approximation, "Approximation")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *BadExpr) AddOffset(i int) {
	n.From = token.Pos(int(n.From) + i)
	n.To = token.Pos(int(n.To) + i)
	addWordOffset(n.Word, i)
}

// Iterate implements Node
func (n *BadExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Word) // TODO(naman) the ordering here is unclear
	h.VisitSlice(exprSlice{&n.Approximation})
}

// CopyIterable implements Node
func (n *BadExpr) CopyIterable() Node {
	new := *n
	new.Approximation = append([]Expr{}, n.Approximation...)
	return &new
}

// EllipsisExpr represents an ellipsis, i.e. "..."
type EllipsisExpr struct {
	Periods [3]*pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *EllipsisExpr) Begin() token.Pos { return n.Periods[0].Begin }

// End gets the byte offset one past the last character in this node
func (n *EllipsisExpr) End() token.Pos { return n.Periods[2].End }
func (n *EllipsisExpr) exprNode()      {}

func (n *EllipsisExpr) subscriptNode()     {}
func (n *EllipsisExpr) walk(v EdgeVisitor) {}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *EllipsisExpr) AddOffset(i int) {
	addWordOffset(n.Periods[0], i)
	addWordOffset(n.Periods[1], i)
	addWordOffset(n.Periods[2], i)
}

// Iterate implements Node
func (n *EllipsisExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Periods[0])
	h.VisitWord(&n.Periods[1])
	h.VisitWord(&n.Periods[2])
}

// CopyIterable implements Node
func (n *EllipsisExpr) CopyIterable() Node {
	new := *n
	return &new
}

// NameExpr represents an identifier, e.g. "foo"
type NameExpr struct {
	Ident *pythonscanner.Word // Ident is the identifier being evaluated, assigned, or deleted
	Usage Usage               // Usage indicates whether this expression was being evaluated, assigned, or deleted
}

// Begin gets the byte offset of the first character in this node
func (n *NameExpr) Begin() token.Pos { return n.Ident.Begin }

// End gets the byte offset one past the last character in this node
func (n *NameExpr) End() token.Pos { return n.Ident.End }

func (n *NameExpr) exprNode()          {}
func (n *NameExpr) walk(v EdgeVisitor) {}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *NameExpr) AddOffset(i int) {
	addWordOffset(n.Ident, i)
}

// Iterate implements Node
func (n *NameExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Ident)
}

// CopyIterable implements Node
func (n *NameExpr) CopyIterable() Node {
	new := *n
	return &new
}

// DottedExpr represents a dotted list of names, e.g. "foo.bar.baz"
type DottedExpr struct {
	Names []*NameExpr
	// Dots will always obey len(Dots) = len(Names) - 1.
	Dots []*pythonscanner.Word
}

// Join gets a dot-separated string representing the expression, e.g. "foo.bar.baz"
func (n *DottedExpr) Join() string {
	var s []string
	for _, name := range n.Names {
		s = append(s, name.Ident.Literal)
	}
	return strings.Join(s, ".")
}

// Begin gets the byte offset of the first character in this node
func (n *DottedExpr) Begin() token.Pos { return n.Names[0].Begin() }

// End gets the byte offset one past the last character in this node
func (n *DottedExpr) End() token.Pos { return n.Names[len(n.Names)-1].End() }

func (n *DottedExpr) exprNode() {}
func (n *DottedExpr) walk(v EdgeVisitor) {
	walkNameList(v, n, n.Names, "Names")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *DottedExpr) AddOffset(i int) {
	for _, dot := range n.Dots {
		addWordOffset(dot, i)
	}
}

// Iterate implements Node
func (n *DottedExpr) Iterate(h IterationHandler) {
	h.VisitSlice(nameSlice{&n.Names})
	for i := range n.Dots {
		h.VisitWord(&n.Dots[i])
	}
}

// CopyIterable implements Node
func (n *DottedExpr) CopyIterable() Node {
	new := *n
	new.Names = append([]*NameExpr{}, n.Names...)
	new.Dots = append([]*pythonscanner.Word{}, n.Dots...)
	return &new
}

// TupleExpr represents a tuple, e.g. "(1, 2, 3)"
type TupleExpr struct {
	LeftParen  *pythonscanner.Word // LeftParen may be nil, but equal to RightParen
	Elts       []Expr              // Elts may be empty only if LeftParen, RightParen are non-nil
	Commas     []*pythonscanner.Word
	RightParen *pythonscanner.Word // RightParen may be nil, but equal to LeftParen
	Usage      Usage               // Usage indicates whether this expression was being evaluated, assigned, or deleted
}

// Begin gets the byte offset of the first character in this node
func (n *TupleExpr) Begin() token.Pos {
	if n.LeftParen != nil {
		return n.LeftParen.Begin
	}
	return n.Elts[0].Begin()
}

// End gets the byte offset one past the last character in this node
func (n *TupleExpr) End() token.Pos {
	if n.RightParen != nil {
		return n.RightParen.End
	}
	if len(n.Commas) == 0 {
		// TODO: hack until https://github.com/kiteco/kiteco/pull/8107/files is merged
		return n.Elts[len(n.Elts)-1].End()
	}
	finalComma := n.Commas[len(n.Commas)-1].End
	finalElt := n.Elts[len(n.Elts)-1].End()
	if finalComma > finalElt {
		return finalComma
	}
	return finalElt
}

func (n *TupleExpr) exprNode() {}
func (n *TupleExpr) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Elts, "Elts")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *TupleExpr) AddOffset(i int) {
	addWordOffset(n.LeftParen, i)
	for _, comma := range n.Commas {
		addWordOffset(comma, i)
	}
	addWordOffset(n.RightParen, i)
}

// Iterate implements Node
func (n *TupleExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftParen)
	h.VisitSlice(exprSlice{&n.Elts})
	for i := range n.Commas {
		h.VisitWord(&n.Commas[i])
	}
	h.VisitWord(&n.RightParen)
}

// CopyIterable implements Node
func (n *TupleExpr) CopyIterable() Node {
	new := *n
	new.Elts = append([]Expr{}, n.Elts...)
	new.Commas = append([]*pythonscanner.Word{}, n.Commas...)
	return &new
}

// IndexExpr represents an index expression, e.g. "foo[1, 2:3]"
type IndexExpr struct {
	Value      Expr
	LeftBrack  *pythonscanner.Word
	Subscripts []Subscript
	RightBrack *pythonscanner.Word
	Usage      Usage // Usage indicates whether this expression was being evaluated, assigned, or deleted
}

// Begin gets the byte offset of the first character in this node
func (n *IndexExpr) Begin() token.Pos { return n.Value.Begin() }

// End gets the byte offset one past the last character in this node
func (n *IndexExpr) End() token.Pos { return n.RightBrack.End }

func (n *IndexExpr) exprNode() {}
func (n *IndexExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
	for _, subs := range n.Subscripts {
		walkEdge(v, n, subs, "Subscripts")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *IndexExpr) AddOffset(i int) {
	addWordOffset(n.LeftBrack, i)
	addWordOffset(n.RightBrack, i)
}

// Iterate implements Node
func (n *IndexExpr) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Value})
	h.VisitWord(&n.LeftBrack)
	h.VisitSlice(subscriptSlice{&n.Subscripts})
	h.VisitWord(&n.RightBrack)
}

// CopyIterable implements Node
func (n *IndexExpr) CopyIterable() Node {
	new := *n
	new.Subscripts = append([]Subscript{}, n.Subscripts...)
	return &new
}

// AttributeExpr represents an attribute expression, e.g. "foo.bar"
type AttributeExpr struct {
	Value     Expr
	Dot       *pythonscanner.Word
	Attribute *pythonscanner.Word
	Usage     Usage // Usage indicates whether this expression was being evaluated, assigned, or deleted
}

// Begin gets the byte offset of the first character in this node
func (n *AttributeExpr) Begin() token.Pos { return n.Value.Begin() }

// End gets the byte offset one past the last character in this node
func (n *AttributeExpr) End() token.Pos {
	if n.Attribute != nil {
		return n.Attribute.End
	}

	return n.Dot.End
}

func (n *AttributeExpr) exprNode() {}
func (n *AttributeExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *AttributeExpr) AddOffset(i int) {
	addWordOffset(n.Dot, i)
	addWordOffset(n.Attribute, i)
}

// Iterate implements Node
func (n *AttributeExpr) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Value})
	h.VisitWord(&n.Dot)
	h.VisitWord(&n.Attribute)
}

// CopyIterable implements Node
func (n *AttributeExpr) CopyIterable() Node {
	new := *n
	return &new
}

// NumberExpr represents a number literal, e.g. "123"
type NumberExpr struct {
	Number *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *NumberExpr) Begin() token.Pos { return n.Number.Begin }

// End gets the byte offset one past the last character in this node
func (n *NumberExpr) End() token.Pos { return n.Number.End }

func (n *NumberExpr) exprNode()          {}
func (n *NumberExpr) walk(v EdgeVisitor) {}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *NumberExpr) AddOffset(i int) {
	addWordOffset(n.Number, i)
}

// Iterate implements Node
func (n *NumberExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Number)
}

// CopyIterable implements Node
func (n *NumberExpr) CopyIterable() Node {
	new := *n
	return &new
}

// StringExpr represents a string literal, e.g. "'xyz'"
type StringExpr struct {
	Strings []*pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *StringExpr) Begin() token.Pos { return n.Strings[0].Begin }

// End gets the byte offset one past the last character in this node
func (n *StringExpr) End() token.Pos { return n.Strings[len(n.Strings)-1].End }

func (n *StringExpr) exprNode()          {}
func (n *StringExpr) walk(v EdgeVisitor) {}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *StringExpr) AddOffset(i int) {
	for _, s := range n.Strings {
		addWordOffset(s, i)
	}
}

// Iterate implements Node
func (n *StringExpr) Iterate(h IterationHandler) {
	for i := range n.Strings {
		h.VisitWord(&n.Strings[i])
	}
}

// CopyIterable implements Node
func (n *StringExpr) CopyIterable() Node {
	new := *n
	new.Strings = append([]*pythonscanner.Word{}, n.Strings...)
	return &new
}

// stringLiteral assumes that lexical analysis has validated the string already
func stringLiteral(str string) string {
	// parse prefix; TODO(naman) b, u, f?
	var raw bool
loop:
	for i, ch := range str {
		switch ch {
		case '"', '\'':
			str = str[i:]
			break loop
		case 'r', 'R':
			raw = true
		case 'b', 'B':
		case 'u', 'U':
		case 'f', 'F':
		}
	}

	quote := str[0]
	switch quote {
	case '"', '\'':
	default:
		rollbar.Error(errors.Errorf("invalid initial quote character for string literal"), quote)
	}

	// strconv.UnquoteChar uses Go escape semantics, which are mostly the same as Python 3,
	// except Python 3 support \N{name} for specifying a Unicode character by name.
	// In those cases, we copy the full escape sequence instead of failing outright.

	var buf bytes.Buffer
	buf.Grow(3 * len(str) / 2) // heuristic taken from strconv.Unquote
	for len(str) > 0 {
		// flexibly remove quotes from start/end, assuming the lexer did its job (so there are no unescaped quotes in the middle)
		// note that the string may be incomplete, so end quotes may not match start quotes
		if str[0] == quote {
			str = str[1:]
			continue
		}

		if raw {
			c := str[0]
			str = str[1:]
			buf.WriteByte(c)
			if c == '\\' && len(str) > 0 && str[0] == quote {
				// we don't want to skip this quote, so handle it now
				buf.WriteByte(str[0])
				str = str[1:]
			}
			continue
		}

		c, _, tail, err := strconv.UnquoteChar(str, quote)
		if c == utf8.RuneError || err != nil {
			// this will cause us to end up copying the full byte sequence until we resynchronize with UTF-8
			buf.WriteByte(str[0])
			str = str[1:]
			continue
		}
		str = tail
		buf.WriteRune(c)
	}
	return buf.String()
}

// Literal gets a string representation of the StringExpr
func (n *StringExpr) Literal() string {
	var parts []string
	for _, part := range n.Strings {
		parts = append(parts, stringLiteral(part.Literal))
	}
	return strings.Join(parts, "")
}

// ListExpr represents a list literal, e.g. "[1, 2, 3]"
type ListExpr struct {
	LeftBrack  *pythonscanner.Word
	Values     []Expr
	Usage      Usage
	RightBrack *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *ListExpr) Begin() token.Pos { return n.LeftBrack.Begin }

// End gets the byte offset one past the last character in this node
func (n *ListExpr) End() token.Pos { return n.RightBrack.End }

func (n *ListExpr) exprNode() {}
func (n *ListExpr) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Values, "Values")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ListExpr) AddOffset(i int) {
	addWordOffset(n.LeftBrack, i)
	addWordOffset(n.RightBrack, i)
}

// Iterate implements Node
func (n *ListExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftBrack)
	h.VisitSlice(exprSlice{&n.Values})
	h.VisitWord(&n.RightBrack)
}

// CopyIterable implements Node
func (n *ListExpr) CopyIterable() Node {
	new := *n
	new.Values = append([]Expr{}, n.Values...)
	return &new
}

// SetExpr represents a set literal, e.g. "{1, 2, 3}"
type SetExpr struct {
	LeftBrace  *pythonscanner.Word
	Values     []Expr
	RightBrace *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *SetExpr) Begin() token.Pos { return n.LeftBrace.Begin }

// End gets the byte offset one past the last character in this node
func (n *SetExpr) End() token.Pos { return n.RightBrace.End }

func (n *SetExpr) exprNode() {}
func (n *SetExpr) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Values, "Values")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *SetExpr) AddOffset(i int) {
	addWordOffset(n.LeftBrace, i)
	addWordOffset(n.RightBrace, i)
}

// Iterate implements Node
func (n *SetExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftBrace)
	h.VisitSlice(exprSlice{&n.Values})
	h.VisitWord(&n.RightBrace)
}

// CopyIterable implements Node
func (n *SetExpr) CopyIterable() Node {
	new := *n
	new.Values = append([]Expr{}, n.Values...)
	return &new
}

// KeyValuePair represents a key:value in a dict literal, e.g. foo:123
type KeyValuePair struct {
	Key   Expr
	Value Expr
}

// Begin gets the byte offset of the first character in this node
func (n *KeyValuePair) Begin() token.Pos { return n.Key.Begin() }

// End gets the byte offset one past the last character in this node
func (n *KeyValuePair) End() token.Pos { return n.Value.End() }

func (n *KeyValuePair) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Key, "Key")
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *KeyValuePair) AddOffset(i int) {}

// Iterate implements Node
func (n *KeyValuePair) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Key})
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *KeyValuePair) CopyIterable() Node {
	new := *n
	return &new
}

// DictExpr represents a dict literal, e.g. {"foo": 1, "bar": 2}
type DictExpr struct {
	LeftBrace  *pythonscanner.Word
	Items      []*KeyValuePair
	RightBrace *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *DictExpr) Begin() token.Pos { return n.LeftBrace.Begin }

// End gets the byte offset one past the last character in this node
func (n *DictExpr) End() token.Pos { return n.RightBrace.End }

func (n *DictExpr) exprNode() {}
func (n *DictExpr) walk(v EdgeVisitor) {
	for _, item := range n.Items {
		walkEdge(v, n, item, "Items")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *DictExpr) AddOffset(i int) {
	addWordOffset(n.LeftBrace, i)
	addWordOffset(n.RightBrace, i)
}

// Iterate implements Node
func (n *DictExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftBrace)
	h.VisitSlice(keyValuePairSlice{&n.Items})
	h.VisitWord(&n.RightBrace)
}

// CopyIterable implements Node
func (n *DictExpr) CopyIterable() Node {
	new := *n
	new.Items = append([]*KeyValuePair{}, n.Items...)
	return &new
}

// Generator represents part of a comprehension after the "for", e.g. "for x in y if z".
// It includes the "for" keyword and the optional "async" keyword.
type Generator struct {
	Async    *pythonscanner.Word
	For      *pythonscanner.Word
	Vars     []Expr // Vars always has len >= 1
	Iterable Expr
	Filters  []Expr
}

// Begin gets the byte offset of the first character in this node
func (n *Generator) Begin() token.Pos {
	if n.Async != nil {
		return n.Async.Begin
	}
	return n.For.Begin
}

// End gets the byte offset one past the last character in this node
func (n *Generator) End() token.Pos {
	if len(n.Filters) > 0 {
		return n.Filters[len(n.Filters)-1].End()
	}
	return n.Iterable.End()
}

func (n *Generator) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Vars, "Vars")
	walkEdge(v, n, n.Iterable, "Iterables")
	walkExprList(v, n, n.Filters, "Filters")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *Generator) AddOffset(i int) {
	addWordOffset(n.Async, i)
	addWordOffset(n.For, i)
}

// Iterate implements Node
func (n *Generator) Iterate(h IterationHandler) {
	h.VisitWord(&n.Async)
	h.VisitWord(&n.For)
	h.VisitSlice(exprSlice{&n.Vars})
	h.VisitNode(exprRef{&n.Iterable})
	h.VisitSlice(exprSlice{&n.Filters})
}

// CopyIterable implements Node
func (n *Generator) CopyIterable() Node {
	new := *n
	new.Vars = append([]Expr{}, n.Vars...)
	new.Filters = append([]Expr{}, n.Filters...)
	return &new
}

// BaseComprehension represents the key, value and the generators in
// a list/dict/set/basic comprehension expression, e.g., "x:123 for x in y if z"
// in "{x:123 for x in y if z}".
type BaseComprehension struct {
	Key        Expr
	Value      Expr
	Generators []*Generator
}

func (bc *BaseComprehension) iterate(h IterationHandler) {
	h.VisitNode(exprRef{&bc.Key})
	h.VisitNode(exprRef{&bc.Value})
	h.VisitSlice(generatorSlice{&bc.Generators})
}

func (bc *BaseComprehension) copy() *BaseComprehension {
	new := *bc
	new.Generators = append([]*Generator{}, bc.Generators...)
	return &new
}

// ComprehensionExpr represents generator comprehension, e.g. "(x+1 for x in y if z)"
type ComprehensionExpr struct {
	LeftParen *pythonscanner.Word
	*BaseComprehension
	RightParen *pythonscanner.Word
}

func (n *ComprehensionExpr) scopeNode()         {}
func (n *ComprehensionExpr) comprehensionNode() {}

// Begin gets the byte offset of the first character in this node
func (n *ComprehensionExpr) Begin() token.Pos {
	// Note: n.LeftParen will be nil for generators as function arguments, e.g.:
	//   foo(x for y in z)
	if n.LeftParen != nil {
		return n.LeftParen.Begin
	}
	return n.Value.Begin()
}

// End gets the byte offset one past the last character in this node
func (n *ComprehensionExpr) End() token.Pos {
	// Note: n.RightParen will be nil for generators as function arguments, e.g.:
	//   foo(x for y in z)
	if n.RightParen != nil {
		return n.RightParen.End
	}
	return n.Generators[len(n.Generators)-1].End()
}

func (n *ComprehensionExpr) exprNode() {}
func (n *ComprehensionExpr) walk(v EdgeVisitor) {
	if !IsNil(n.Key) {
		walkEdge(v, n, n.Key, "Key")
	}
	walkEdge(v, n, n.Value, "Value")
	for _, gen := range n.Generators {
		walkEdge(v, n, gen, "Generators")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ComprehensionExpr) AddOffset(i int) {
	addWordOffset(n.LeftParen, i)
	addWordOffset(n.RightParen, i)
}

// Iterate implements Node
func (n *ComprehensionExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftParen)
	n.BaseComprehension.iterate(h)
	h.VisitWord(&n.RightParen)
}

// CopyIterable implements Node
func (n *ComprehensionExpr) CopyIterable() Node {
	new := *n
	new.BaseComprehension = n.BaseComprehension.copy()
	return &new
}

// ListComprehensionExpr represents a list comprehension, e.g. "[x+1 for x in y if z]"
type ListComprehensionExpr struct {
	LeftBrack *pythonscanner.Word
	*BaseComprehension
	RightBrack *pythonscanner.Word
}

func (n *ListComprehensionExpr) scopeNode()         {}
func (n *ListComprehensionExpr) comprehensionNode() {}

// Begin gets the byte offset of the first character in this node
func (n *ListComprehensionExpr) Begin() token.Pos { return n.LeftBrack.Begin }

// End gets the byte offset one past the last character in this node
func (n *ListComprehensionExpr) End() token.Pos { return n.RightBrack.End }

func (n *ListComprehensionExpr) exprNode() {}
func (n *ListComprehensionExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
	for _, gen := range n.Generators {
		walkEdge(v, n, gen, "Generators")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ListComprehensionExpr) AddOffset(i int) {
	addWordOffset(n.LeftBrack, i)
	addWordOffset(n.RightBrack, i)
}

// Iterate implements Node
func (n *ListComprehensionExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftBrack)
	n.BaseComprehension.iterate(h)
	h.VisitWord(&n.RightBrack)
}

// CopyIterable implements Node
func (n *ListComprehensionExpr) CopyIterable() Node {
	new := *n
	new.BaseComprehension = n.BaseComprehension.copy()
	return &new
}

// DictComprehensionExpr represents a dict comprehension, e.g. "{x:123 for x in y if z}"
type DictComprehensionExpr struct {
	LeftBrace *pythonscanner.Word
	*BaseComprehension
	RightBrace *pythonscanner.Word
}

func (n *DictComprehensionExpr) scopeNode()         {}
func (n *DictComprehensionExpr) comprehensionNode() {}

// Begin gets the byte offset of the first character in this node
func (n *DictComprehensionExpr) Begin() token.Pos { return n.LeftBrace.Begin }

// End gets the byte offset one past the last character in this node
func (n *DictComprehensionExpr) End() token.Pos { return n.RightBrace.End }

func (n *DictComprehensionExpr) exprNode() {}
func (n *DictComprehensionExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Key, "Key")
	walkEdge(v, n, n.Value, "Value")
	for _, gen := range n.Generators {
		walkEdge(v, n, gen, "Generators")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *DictComprehensionExpr) AddOffset(i int) {
	addWordOffset(n.LeftBrace, i)
	addWordOffset(n.RightBrace, i)
}

// Iterate implements Node
func (n *DictComprehensionExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftBrace)
	n.BaseComprehension.iterate(h)
	h.VisitWord(&n.RightBrace)
}

// CopyIterable implements Node
func (n *DictComprehensionExpr) CopyIterable() Node {
	new := *n
	new.BaseComprehension = n.BaseComprehension.copy()
	return &new
}

// SetComprehensionExpr represents a set comprehension, e.g. "{x for x in y if z}"
type SetComprehensionExpr struct {
	LeftBrace *pythonscanner.Word
	*BaseComprehension
	RightBrace *pythonscanner.Word
}

func (n *SetComprehensionExpr) scopeNode()         {}
func (n *SetComprehensionExpr) comprehensionNode() {}

// Begin gets the byte offset of the first character in this node
func (n *SetComprehensionExpr) Begin() token.Pos { return n.LeftBrace.Begin }

// End gets the byte offset one past the last character in this node
func (n *SetComprehensionExpr) End() token.Pos { return n.RightBrace.End }

func (n *SetComprehensionExpr) exprNode() {}
func (n *SetComprehensionExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
	for _, gen := range n.Generators {
		walkEdge(v, n, gen, "Generators")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *SetComprehensionExpr) AddOffset(i int) {
	addWordOffset(n.LeftBrace, i)
	addWordOffset(n.RightBrace, i)
}

// Iterate implements Node
func (n *SetComprehensionExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftBrace)
	n.BaseComprehension.iterate(h)
	h.VisitWord(&n.RightBrace)
}

// CopyIterable implements Node
func (n *SetComprehensionExpr) CopyIterable() Node {
	new := *n
	new.BaseComprehension = n.BaseComprehension.copy()
	return &new
}

// UnaryExpr represents an operator with one operand, e.g. "~123"
type UnaryExpr struct {
	Op    *pythonscanner.Word // can be add, sub, bitnot, or not
	Value Expr
}

// Begin gets the byte offset of the first character in this node
func (n *UnaryExpr) Begin() token.Pos { return n.Op.Begin }

// End gets the byte offset one past the last character in this node
func (n *UnaryExpr) End() token.Pos { return n.Value.End() }

func (n *UnaryExpr) exprNode() {}
func (n *UnaryExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *UnaryExpr) AddOffset(i int) {
	addWordOffset(n.Op, i)
}

// Iterate implements Node
func (n *UnaryExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Op)
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *UnaryExpr) CopyIterable() Node {
	new := *n
	return &new
}

// BinaryExpr represents an operator with two operands, e.g. "1 + 2"
type BinaryExpr struct {
	Left  Expr
	Op    *pythonscanner.Word
	Right Expr
}

// Begin gets the byte offset of the first character in this node
func (n *BinaryExpr) Begin() token.Pos { return n.Left.Begin() }

// End gets the byte offset one past the last character in this node
func (n *BinaryExpr) End() token.Pos { return n.Right.End() }

func (n *BinaryExpr) exprNode() {}
func (n *BinaryExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Left, "Left")
	walkEdge(v, n, n.Right, "Right")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *BinaryExpr) AddOffset(i int) {
	addWordOffset(n.Op, i)
}

// Iterate implements Node
func (n *BinaryExpr) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Left})
	h.VisitWord(&n.Op)
	h.VisitNode(exprRef{&n.Right})
}

// CopyIterable implements Node
func (n *BinaryExpr) CopyIterable() Node {
	new := *n
	return &new
}

// Argument represents an argument passed to a function, possibly with a keyword, e.g. "foo=123"
type Argument struct {
	// Name will be nil if this argument was not passed as a keyword argument
	Name Expr
	// Equals will be nil if this argument was not passed as a keyword argument
	Equals *pythonscanner.Word
	Value  Expr
}

// Begin gets the byte offset of the first character in this node
func (n *Argument) Begin() token.Pos { return switchBegin(n.Name, n.Value) }

// End gets the byte offset one past the last character in this node
func (n *Argument) End() token.Pos { return n.Value.End() }

func (n *Argument) walk(v EdgeVisitor) {
	if !IsNil(n.Name) {
		walkEdge(v, n, n.Name, "Name")
	}
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *Argument) AddOffset(i int) {
	addWordOffset(n.Equals, i)
}

// Iterate implements Node
func (n *Argument) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Name})
	h.VisitWord(&n.Equals)
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *Argument) CopyIterable() Node {
	new := *n
	return &new
}

// CallExpr represents a function call, e.g. "foo(a, b=1, *x, **y)"
type CallExpr struct {
	Func       Expr
	LeftParen  *pythonscanner.Word
	Args       []*Argument
	Vararg     Expr
	Kwarg      Expr
	Commas     []*pythonscanner.Word
	RightParen *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *CallExpr) Begin() token.Pos { return n.Func.Begin() }

// End gets the byte offset one past the last character in this node
func (n *CallExpr) End() token.Pos {
	// Exact call expression
	if n.RightParen != nil {
		return n.RightParen.End
	}

	// Approximate call expression
	max := n.LeftParen.End
	if l := len(n.Args); l > 0 {
		max = n.Args[l-1].End()
	}

	if l := len(n.Commas); l > 0 {
		pos := n.Commas[l-1].End
		if pos > max {
			max = pos
		}
	}
	return max
}

func (n *CallExpr) exprNode() {}
func (n *CallExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Func, "Func")
	for _, arg := range n.Args {
		walkEdge(v, n, arg, "Args")
	}
	if !IsNil(n.Vararg) {
		walkEdge(v, n, n.Vararg, "Vararg")
	}
	if !IsNil(n.Kwarg) {
		walkEdge(v, n, n.Kwarg, "Kwarg")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *CallExpr) AddOffset(i int) {
	addWordOffset(n.LeftParen, i)
	addWordOffset(n.RightParen, i)
	for _, comma := range n.Commas {
		addWordOffset(comma, i)
	}
}

// Iterate implements Node
func (n *CallExpr) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Func})
	h.VisitWord(&n.LeftParen)
	h.VisitSlice(argumentSlice{&n.Args})
	for i := range n.Commas {
		h.VisitWord(&n.Commas[i])
	}
	h.VisitNode(exprRef{&n.Vararg})
	h.VisitNode(exprRef{&n.Kwarg})
	h.VisitWord(&n.RightParen)
}

// CopyIterable implements Node
func (n *CallExpr) CopyIterable() Node {
	new := *n
	new.Args = append([]*Argument{}, n.Args...)
	new.Commas = append([]*pythonscanner.Word{}, n.Commas...)
	return &new
}

// LambdaExpr represents a lambda expression, e.g. "lambda x: x+1"
type LambdaExpr struct {
	Lambda     *pythonscanner.Word
	Parameters []*Parameter   // Parameters can be empty
	Vararg     *ArgsParameter // Vararg can be nil
	Kwarg      *ArgsParameter // Kwarg can be nil
	Body       Expr
}

// Begin gets the byte offset of the first character in this node
func (n *LambdaExpr) Begin() token.Pos { return n.Lambda.Begin }

// End gets the byte offset one past the last character in this node
func (n *LambdaExpr) End() token.Pos { return n.Body.End() }

func (n *LambdaExpr) scopeNode() {}
func (n *LambdaExpr) exprNode()  {}
func (n *LambdaExpr) walk(v EdgeVisitor) {
	for _, param := range n.Parameters {
		walkEdge(v, n, param, "Parameters")
	}
	if n.Vararg != nil {
		walkEdge(v, n, n.Vararg, "Vararg")
	}
	if n.Kwarg != nil {
		walkEdge(v, n, n.Kwarg, "Kwarg")
	}
	walkEdge(v, n, n.Body, "Body")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *LambdaExpr) AddOffset(i int) {
	addWordOffset(n.Lambda, i)
}

// Iterate implements Node
func (n *LambdaExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Lambda)
	h.VisitSlice(parameterSlice{&n.Parameters})
	h.VisitNode(argsParamRef{&n.Vararg})
	h.VisitNode(argsParamRef{&n.Kwarg})
	h.VisitNode(exprRef{&n.Body})
}

// CopyIterable implements Node
func (n *LambdaExpr) CopyIterable() Node {
	new := *n
	new.Parameters = append([]*Parameter{}, n.Parameters...)
	return &new
}

// ReprExpr represents an old-style backticked expressions, e.g. "`foo`"
type ReprExpr struct {
	LeftBacktick  *pythonscanner.Word
	Value         Expr
	RightBacktick *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *ReprExpr) Begin() token.Pos { return n.LeftBacktick.Begin }

// End gets the byte offset one past the last character in this node
func (n *ReprExpr) End() token.Pos { return n.RightBacktick.End }

func (n *ReprExpr) exprNode() {}
func (n *ReprExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ReprExpr) AddOffset(i int) {
	addWordOffset(n.LeftBacktick, i)
	addWordOffset(n.RightBacktick, i)
}

// Iterate implements Node
func (n *ReprExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.LeftBacktick)
	h.VisitNode(exprRef{&n.Value})
	h.VisitWord(&n.RightBacktick)
}

// CopyIterable implements Node
func (n *ReprExpr) CopyIterable() Node {
	new := *n
	return &new
}

// IfExpr represents a conditional expression, e.g "foo if condition else bar"
type IfExpr struct {
	Body      Expr
	Condition Expr
	Else      Expr
}

// Begin gets the byte offset of the first character in this node
func (n *IfExpr) Begin() token.Pos { return n.Body.Begin() }

// End gets the byte offset one past the last character in this node
func (n *IfExpr) End() token.Pos { return n.Else.End() }

func (n *IfExpr) exprNode() {}
func (n *IfExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Body, "Body")
	walkEdge(v, n, n.Condition, "Condition")
	walkEdge(v, n, n.Else, "Else")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *IfExpr) AddOffset(i int) {}

// Iterate implements Node
func (n *IfExpr) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Body})
	h.VisitNode(exprRef{&n.Condition})
	h.VisitNode(exprRef{&n.Else})
}

// CopyIterable implements Node
func (n *IfExpr) CopyIterable() Node {
	new := *n
	return &new
}

// YieldExpr represents a yield expression, e.g. "yield 123"
type YieldExpr struct {
	Yield *pythonscanner.Word
	Value Expr // Value may be nil
}

// Begin gets the byte offset of the first character in this node
func (n *YieldExpr) Begin() token.Pos { return n.Yield.Begin }

// End gets the byte offset one past the last character in this node
func (n *YieldExpr) End() token.Pos {
	if !IsNil(n.Value) {
		return n.Value.End()
	}
	return n.Yield.End
}

func (n *YieldExpr) exprNode() {}
func (n *YieldExpr) walk(v EdgeVisitor) {
	if !IsNil(n.Value) {
		walkEdge(v, n, n.Value, "Value")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *YieldExpr) AddOffset(i int) {
	addWordOffset(n.Yield, i)
}

// Iterate implements Node
func (n *YieldExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Yield)
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *YieldExpr) CopyIterable() Node {
	new := *n
	return &new
}

// AwaitExpr represents an await expression, e.g. "await f()"
type AwaitExpr struct {
	Await *pythonscanner.Word
	Value Expr
}

// Begin gets the byte offset of the first character in this node
func (n *AwaitExpr) Begin() token.Pos { return n.Await.Begin }

// End gets the byte offset one past the last character in this node
func (n *AwaitExpr) End() token.Pos { return n.Value.End() }

func (n *AwaitExpr) exprNode() {}
func (n *AwaitExpr) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *AwaitExpr) AddOffset(i int) {
	addWordOffset(n.Await, i)
}

// Iterate implements Node
func (n *AwaitExpr) Iterate(h IterationHandler) {
	h.VisitWord(&n.Await)
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *AwaitExpr) CopyIterable() Node {
	new := *n
	return &new
}

// ---
// Statements

// BadStmt represents a statement that was not parsed correctly.
type BadStmt struct {
	// TODO(naman) do we need From/To in addition to Word?
	From token.Pos
	To   token.Pos
	Word *pythonscanner.Word

	// Approximation contains the Stmts that were (approximately) parsed
	// in the BadStmt region, these statements may
	// not be syntactically correct and may contain BadTokens
	// or BadStmts.
	Approximation []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *BadStmt) Begin() token.Pos { return n.From }

// End gets the byte offset one past the last character in this node
func (n *BadStmt) End() token.Pos { return n.To }

func (n *BadStmt) stmtNode() {}
func (n *BadStmt) walk(v EdgeVisitor) {
	walkStmtList(v, n, n.Approximation, "Approximation")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *BadStmt) AddOffset(i int) {
	n.From = token.Pos(int(n.From) + i)
	n.To = token.Pos(int(n.To) + i)
	addWordOffset(n.Word, i)
}

// Iterate implements Node
func (n *BadStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Word) // TODO(naman) ordering unclear
	h.VisitSlice(stmtSlice{&n.Approximation})
}

// CopyIterable implements Node
func (n *BadStmt) CopyIterable() Node {
	new := *n
	new.Approximation = append([]Stmt{}, n.Approximation...)
	return &new
}

// ExprStmt represents a statement that consists of just one expression, e.g. "foo.dosomething()"
type ExprStmt struct {
	Value Expr // Value != nil
}

// Begin gets the byte offset of the first character in this node
func (n *ExprStmt) Begin() token.Pos { return n.Value.Begin() }

// End gets the byte offset one past the last character in this node
func (n *ExprStmt) End() token.Pos { return n.Value.End() }

func (n *ExprStmt) stmtNode() {}
func (n *ExprStmt) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ExprStmt) AddOffset(i int) {}

// Iterate implements Node
func (n *ExprStmt) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *ExprStmt) CopyIterable() Node {
	new := *n
	return &new
}

// AnnotationStmt represents an annotation statement, e.g. `foo: bar`, `x.y: bar`, `foo['foo']: bar`
type AnnotationStmt struct {
	Target     Expr
	Annotation Expr // must be non-nil
}

// Begin gets the byte offset of the first character in this node
func (n *AnnotationStmt) Begin() token.Pos { return n.Target.Begin() }

// End gets the byte offset one past the last character in this node
func (n *AnnotationStmt) End() token.Pos { return n.Annotation.End() }

func (n *AnnotationStmt) stmtNode() {}
func (n *AnnotationStmt) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Target, "Target")
	walkEdge(v, n, n.Annotation, "Annotation")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *AnnotationStmt) AddOffset(i int) {}

// Iterate implements Node
func (n *AnnotationStmt) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Target})
	h.VisitNode(exprRef{&n.Annotation})
}

// CopyIterable implements Node
func (n *AnnotationStmt) CopyIterable() Node {
	new := *n
	return &new
}

// AssignStmt represents an assignment, e.g. "a,b = c,d = e,f = 1,2"
type AssignStmt struct {
	Targets    []Expr // Targets will be [Tuple[a, b], Tuple[c, d], Tuple[e, f]] in the example above
	Annotation Expr   // if there's an Annotation, then len(Targets) == 1 && Targets[0] is not a TupleExpr
	Value      Expr   // Value will be Tuple[1, 2] in the example above (it is never nil)
}

// Begin gets the byte offset of the first character in this node
func (n *AssignStmt) Begin() token.Pos { return n.Targets[0].Begin() }

// End gets the byte offset one past the last character in this node
func (n *AssignStmt) End() token.Pos { return n.Value.End() }

func (n *AssignStmt) stmtNode() {}
func (n *AssignStmt) walk(v EdgeVisitor) {
	for _, target := range n.Targets {
		walkEdge(v, n, target, "Targets")
	}
	if !IsNil(n.Annotation) {
		walkEdge(v, n, n.Annotation, "Annotation")
	}
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *AssignStmt) AddOffset(i int) {}

// Iterate implements Node
func (n *AssignStmt) Iterate(h IterationHandler) {
	h.VisitSlice(exprSlice{&n.Targets})
	h.VisitNode(exprRef{&n.Annotation})
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *AssignStmt) CopyIterable() Node {
	new := *n
	new.Targets = append([]Expr{}, n.Targets...)
	return &new
}

// AugAssignStmt represents an "aug" assignment, e.g. "a += 1"
type AugAssignStmt struct {
	Target Expr
	Op     *pythonscanner.Word
	Value  Expr // Value != nil
}

// Begin gets the byte offset of the first character in this node
func (n *AugAssignStmt) Begin() token.Pos { return n.Target.Begin() }

// End gets the byte offset one past the last character in this node
func (n *AugAssignStmt) End() token.Pos { return n.Value.End() }

func (n *AugAssignStmt) stmtNode() {}
func (n *AugAssignStmt) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Target, "Target")
	walkEdge(v, n, n.Value, "Value")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *AugAssignStmt) AddOffset(i int) {
	addWordOffset(n.Op, i)
}

// Iterate implements Node
func (n *AugAssignStmt) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Target})
	h.VisitWord(&n.Op)
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *AugAssignStmt) CopyIterable() Node {
	new := *n
	return &new
}

// ClassDefStmt represents a class definition, e.g. "class foo(object): ..."
type ClassDefStmt struct {
	Class      *pythonscanner.Word
	Decorators []Expr
	Name       *NameExpr
	Args       []*Argument // Args contains both the base classes and the keyword arguments
	Vararg     Expr        // Vararg is the *arg: python 3 permits "class Foo(*bases): pass"
	Kwarg      Expr        // Kwarg is the **kwarg: python 3 permits "class Foo(**kwarg): pass"
	Body       []Stmt
}

// Bases gets the expressions for all the arguments that do not have keywords
func (n *ClassDefStmt) Bases() []Expr {
	var bases []Expr
	for _, arg := range n.Args {
		if IsNil(arg.Name) {
			bases = append(bases, arg.Value)
		}
	}
	return bases
}

// Begin gets the byte offset of the first character in this node
func (n *ClassDefStmt) Begin() token.Pos {
	if len(n.Decorators) > 0 {
		return n.Decorators[0].Begin()
	}
	return n.Class.Begin
}

// End gets the byte offset one past the last character in this node
func (n *ClassDefStmt) End() token.Pos { return n.Body[len(n.Body)-1].End() }

func (n *ClassDefStmt) scopeNode() {}
func (n *ClassDefStmt) stmtNode()  {}
func (n *ClassDefStmt) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Decorators, "Decorators")
	walkEdge(v, n, n.Name, "Name")
	for _, arg := range n.Args {
		walkEdge(v, n, arg, "Args")
	}
	if !IsNil(n.Vararg) {
		walkEdge(v, n, n.Vararg, "Vararg")
	}
	if !IsNil(n.Kwarg) {
		walkEdge(v, n, n.Kwarg, "Kwarg")
	}
	walkStmtList(v, n, n.Body, "Body")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ClassDefStmt) AddOffset(i int) {
	addWordOffset(n.Class, i)
}

// Iterate implements Node
func (n *ClassDefStmt) Iterate(h IterationHandler) {
	h.VisitSlice(exprSlice{&n.Decorators})
	h.VisitWord(&n.Class)
	h.VisitNode(nameExprRef{&n.Name})
	h.VisitSlice(argumentSlice{&n.Args})
	h.VisitNode(exprRef{&n.Vararg})
	h.VisitNode(exprRef{&n.Kwarg})
	h.VisitSlice(stmtSlice{&n.Body})
}

// CopyIterable implements Node
func (n *ClassDefStmt) CopyIterable() Node {
	new := *n
	new.Decorators = append([]Expr{}, n.Decorators...)
	new.Args = append([]*Argument{}, n.Args...)
	new.Body = append([]Stmt{}, n.Body...)
	return &new
}

// Parameter represents a parameter defined in a function signature (not a function
// call), e.g. "foo" or "foo:int" or "foo=1" or "foo:int=1"
type Parameter struct {
	Name        Expr
	Annotation  Expr // Annotation can be nil
	Default     Expr // Default can be nil
	KeywordOnly bool // KeywordOnly is true if this parameter must be a keyword argument
}

// Begin gets the byte offset of the first character in this node
func (n *Parameter) Begin() token.Pos { return n.Name.Begin() }

// End gets the byte offset one past the last character in this node
func (n *Parameter) End() token.Pos { return switchEnd(n.Name, n.Annotation, n.Default) }

func (n *Parameter) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Name, "Name")
	if !IsNil(n.Annotation) {
		walkEdge(v, n, n.Annotation, "Annotation")
	}
	if !IsNil(n.Default) {
		walkEdge(v, n, n.Default, "Default")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *Parameter) AddOffset(i int) {}

// Iterate implements Node
func (n *Parameter) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Name})
	h.VisitNode(exprRef{&n.Annotation})
	h.VisitNode(exprRef{&n.Default})
}

// CopyIterable implements Node
func (n *Parameter) CopyIterable() Node {
	new := *n
	return &new
}

// ArgsParameter represents *args or **kwargs parameters, which have only a
// name and an optional annotation.
type ArgsParameter struct {
	Name       *NameExpr
	Annotation Expr
}

// Begin gets the byte offset of the first character in this node
func (n *ArgsParameter) Begin() token.Pos { return n.Name.Begin() }

// End gets the byte offset one past the last character in this node
func (n *ArgsParameter) End() token.Pos { return switchEnd(n.Name, n.Annotation) }

func (n *ArgsParameter) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Name, "Name")
	if !IsNil(n.Annotation) {
		walkEdge(v, n, n.Annotation, "Annotation")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ArgsParameter) AddOffset(i int) {}

// Iterate implements Node
func (n *ArgsParameter) Iterate(h IterationHandler) {
	h.VisitNode(nameExprRef{&n.Name})
	h.VisitNode(exprRef{&n.Annotation})
}

// CopyIterable implements Node
func (n *ArgsParameter) CopyIterable() Node {
	new := *n
	return &new
}

// FunctionDefStmt represents a function definition, e.g. "def foo(a, b=1, *x, **y): ..."
type FunctionDefStmt struct {
	Decorators []Expr
	Async      *pythonscanner.Word // Async can be nil
	Def        *pythonscanner.Word
	Name       *NameExpr
	LeftParen  *pythonscanner.Word
	Parameters []*Parameter   // Parameters can be empty
	Vararg     *ArgsParameter // Vararg can be nil
	Kwarg      *ArgsParameter // Kwarg can be nil
	Annotation Expr
	RightParen *pythonscanner.Word
	Body       []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *FunctionDefStmt) Begin() token.Pos {
	if len(n.Decorators) > 0 {
		return n.Decorators[0].Begin()
	}
	if n.Async != nil {
		return n.Async.Begin
	}
	return n.Def.Begin
}

// End gets the byte offset one past the last character in this node
func (n *FunctionDefStmt) End() token.Pos { return n.Body[len(n.Body)-1].End() }

func (n *FunctionDefStmt) scopeNode() {}
func (n *FunctionDefStmt) stmtNode()  {}
func (n *FunctionDefStmt) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Decorators, "Decorators")
	walkEdge(v, n, n.Name, "Name")
	for _, param := range n.Parameters {
		walkEdge(v, n, param, "Parameters")
	}
	if !IsNil(n.Vararg) {
		walkEdge(v, n, n.Vararg, "Vararg")
	}
	if !IsNil(n.Kwarg) {
		walkEdge(v, n, n.Kwarg, "Kwarg")
	}
	if !IsNil(n.Annotation) {
		walkEdge(v, n, n.Annotation, "Annotation")
	}
	walkStmtList(v, n, n.Body, "Body")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *FunctionDefStmt) AddOffset(i int) {
	addWordOffset(n.Async, i)
	addWordOffset(n.Def, i)
	addWordOffset(n.LeftParen, i)
	addWordOffset(n.RightParen, i)
}

// Iterate implements Node
func (n *FunctionDefStmt) Iterate(h IterationHandler) {
	h.VisitSlice(exprSlice{&n.Decorators})
	h.VisitWord(&n.Async)
	h.VisitWord(&n.Def)
	h.VisitNode(nameExprRef{&n.Name})
	h.VisitWord(&n.LeftParen)
	h.VisitSlice(parameterSlice{&n.Parameters})
	h.VisitNode(argsParamRef{&n.Vararg})
	h.VisitNode(argsParamRef{&n.Kwarg})
	h.VisitWord(&n.RightParen)
	h.VisitNode(exprRef{&n.Annotation})
	h.VisitSlice(stmtSlice{&n.Body})
}

// CopyIterable implements Node
func (n *FunctionDefStmt) CopyIterable() Node {
	new := *n
	new.Decorators = append([]Expr{}, n.Decorators...)
	new.Parameters = append([]*Parameter{}, n.Parameters...)
	new.Body = append([]Stmt{}, n.Body...)
	return &new
}

// AssertStmt represents an assertion, e.g. "assert x, 'expected x to be true'"
type AssertStmt struct {
	Assert    *pythonscanner.Word
	Condition Expr
	Message   Expr
}

// Begin gets the byte offset of the first character in this node
func (n *AssertStmt) Begin() token.Pos { return n.Assert.Begin }

// End gets the byte offset one past the last character in this node
func (n *AssertStmt) End() token.Pos { return switchEnd(n.Condition, n.Message) }

func (n *AssertStmt) stmtNode() {}
func (n *AssertStmt) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Condition, "Condition")
	if !IsNil(n.Message) {
		walkEdge(v, n, n.Message, "Message")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *AssertStmt) AddOffset(i int) {
	addWordOffset(n.Assert, i)
}

// Iterate implements Node
func (n *AssertStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Assert)
	h.VisitNode(exprRef{&n.Condition})
	h.VisitNode(exprRef{&n.Message})
}

// CopyIterable implements Node
func (n *AssertStmt) CopyIterable() Node {
	new := *n
	return &new
}

// ContinueStmt represents a continue statement
type ContinueStmt struct {
	Continue *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *ContinueStmt) Begin() token.Pos { return n.Continue.Begin }

// End gets the byte offset one past the last character in this node
func (n *ContinueStmt) End() token.Pos { return n.Continue.End }

func (n *ContinueStmt) stmtNode()          {}
func (n *ContinueStmt) walk(v EdgeVisitor) {}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ContinueStmt) AddOffset(i int) {
	addWordOffset(n.Continue, i)
}

// Iterate implements Node
func (n *ContinueStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Continue)
}

// CopyIterable implements Node
func (n *ContinueStmt) CopyIterable() Node {
	new := *n
	return &new
}

// BreakStmt represents a break statement
type BreakStmt struct {
	Break *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *BreakStmt) Begin() token.Pos { return n.Break.Begin }

// End gets the byte offset one past the last character in this node
func (n *BreakStmt) End() token.Pos { return n.Break.End }

func (n *BreakStmt) stmtNode() {}
func (n *BreakStmt) walk(v EdgeVisitor) {
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *BreakStmt) AddOffset(i int) {
	addWordOffset(n.Break, i)
}

// Iterate implements Node
func (n *BreakStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Break)
}

// CopyIterable implements Node
func (n *BreakStmt) CopyIterable() Node {
	new := *n
	return &new
}

// DelStmt represents a delete statment, e.g. "del foo[key]"
type DelStmt struct {
	Del     *pythonscanner.Word
	Targets []Expr // Targets always has len >= 1
}

// Begin gets the byte offset of the first character in this node
func (n *DelStmt) Begin() token.Pos { return n.Del.Begin }

// End gets the byte offset one past the last character in this node
func (n *DelStmt) End() token.Pos { return n.Targets[len(n.Targets)-1].End() }

func (n *DelStmt) stmtNode() {}
func (n *DelStmt) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Targets, "Targets")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *DelStmt) AddOffset(i int) {
	addWordOffset(n.Del, i)
}

// Iterate implements Node
func (n *DelStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Del)
	h.VisitSlice(exprSlice{&n.Targets})
}

// CopyIterable implements Node
func (n *DelStmt) CopyIterable() Node {
	new := *n
	new.Targets = append([]Expr{}, n.Targets...)
	return &new
}

// ExecStmt represents an exec statement, e.g. "exec mycode in mydict"
type ExecStmt struct {
	Exec    *pythonscanner.Word
	Body    Expr
	Globals Expr
	Locals  Expr
}

// Begin gets the byte offset of the first character in this node
func (n *ExecStmt) Begin() token.Pos { return n.Exec.Begin }

// End gets the byte offset one past the last character in this node
func (n *ExecStmt) End() token.Pos { return switchEnd(n.Body, n.Globals, n.Locals) }

func (n *ExecStmt) stmtNode() {}
func (n *ExecStmt) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Body, "Body")
	if !IsNil(n.Globals) {
		walkEdge(v, n, n.Globals, "Globals")
	}
	if !IsNil(n.Locals) {
		walkEdge(v, n, n.Locals, "Locals")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ExecStmt) AddOffset(i int) {
	addWordOffset(n.Exec, i)
}

// Iterate implements Node
func (n *ExecStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Exec)
	h.VisitNode(exprRef{&n.Body})
	h.VisitNode(exprRef{&n.Globals})
	h.VisitNode(exprRef{&n.Locals})
}

// CopyIterable implements Node
func (n *ExecStmt) CopyIterable() Node {
	new := *n
	return &new
}

// PassStmt represents a pass statement
type PassStmt struct {
	Pass *pythonscanner.Word
}

// Begin gets the byte offset of the first character in this node
func (n *PassStmt) Begin() token.Pos { return n.Pass.Begin }

// End gets the byte offset one past the last character in this node
func (n *PassStmt) End() token.Pos { return n.Pass.End }

func (n *PassStmt) stmtNode()          {}
func (n *PassStmt) walk(v EdgeVisitor) {}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *PassStmt) AddOffset(i int) {
	addWordOffset(n.Pass, i)
}

// Iterate implements Node
func (n *PassStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Pass)
}

// CopyIterable implements Node
func (n *PassStmt) CopyIterable() Node {
	new := *n
	return &new
}

// PrintStmt represents a print statement, e.g. "print xyz"
type PrintStmt struct {
	Print   *pythonscanner.Word // Print represents the "print" keyword
	Dest    Expr
	Values  []Expr
	NewLine bool
}

// Begin gets the byte offset of the first character in this node
func (n *PrintStmt) Begin() token.Pos { return n.Print.Begin }

// End gets the byte offset one past the last character in this node
func (n *PrintStmt) End() token.Pos {
	if len(n.Values) > 0 {
		return n.Values[len(n.Values)-1].End()
	}
	if !IsNil(n.Dest) {
		return n.Dest.End()
	}
	return n.Print.End
}

func (n *PrintStmt) stmtNode() {}
func (n *PrintStmt) walk(v EdgeVisitor) {
	if !IsNil(n.Dest) {
		walkEdge(v, n, n.Dest, "Dest")
	}
	walkExprList(v, n, n.Values, "Values")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *PrintStmt) AddOffset(i int) {
	addWordOffset(n.Print, i)
}

// Iterate implements Node
func (n *PrintStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Print)
	h.VisitNode(exprRef{&n.Dest})
	h.VisitSlice(exprSlice{&n.Values})
}

// CopyIterable implements Node
func (n *PrintStmt) CopyIterable() Node {
	new := *n
	new.Values = append([]Expr{}, n.Values...)
	return &new
}

// RaiseStmt represents a raise statement, e.g. "raise MyError(123)"
type RaiseStmt struct {
	Raise     *pythonscanner.Word
	Type      Expr
	Instance  Expr
	Traceback Expr
}

// Begin gets the byte offset of the first character in this node
func (n *RaiseStmt) Begin() token.Pos { return n.Raise.Begin }

// End gets the byte offset one past the last character in this node
func (n *RaiseStmt) End() token.Pos {
	if !IsNil(n.Traceback) {
		return n.Traceback.End()
	}
	if !IsNil(n.Instance) {
		return n.Instance.End()
	}
	if !IsNil(n.Type) {
		return n.Type.End()
	}
	return n.Raise.End
}

func (n *RaiseStmt) stmtNode() {}
func (n *RaiseStmt) walk(v EdgeVisitor) {
	if !IsNil(n.Type) {
		walkEdge(v, n, n.Type, "Type")
	}
	if !IsNil(n.Instance) {
		walkEdge(v, n, n.Instance, "Instance")
	}
	if !IsNil(n.Traceback) {
		walkEdge(v, n, n.Traceback, "Traceback")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *RaiseStmt) AddOffset(i int) {
	addWordOffset(n.Raise, i)
}

// Iterate implements Node
func (n *RaiseStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Raise)
	h.VisitNode(exprRef{&n.Type})
	h.VisitNode(exprRef{&n.Instance})
	h.VisitNode(exprRef{&n.Traceback})
}

// CopyIterable implements Node
func (n *RaiseStmt) CopyIterable() Node {
	new := *n
	return &new
}

// ReturnStmt represents a return statement, e.g. "return xyz"
type ReturnStmt struct {
	Return *pythonscanner.Word
	Value  Expr // Value may be nil
}

// Begin gets the byte offset of the first character in this node
func (n *ReturnStmt) Begin() token.Pos { return n.Return.Begin }

// End gets the byte offset one past the last character in this node
func (n *ReturnStmt) End() token.Pos {
	if !IsNil(n.Value) {
		return n.Value.End()
	}
	return n.Return.End
}

func (n *ReturnStmt) stmtNode() {}
func (n *ReturnStmt) walk(v EdgeVisitor) {
	if !IsNil(n.Value) {
		walkEdge(v, n, n.Value, "Value")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ReturnStmt) AddOffset(i int) {
	addWordOffset(n.Return, i)
}

// Iterate implements Node
func (n *ReturnStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Return)
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *ReturnStmt) CopyIterable() Node {
	new := *n
	return &new
}

// YieldStmt represents a "yield" statement, e.g. "yield foo"
type YieldStmt struct {
	Yield *pythonscanner.Word
	Value Expr // Value may be nil
}

// Begin gets the byte offset of the first character in this node
func (n *YieldStmt) Begin() token.Pos { return n.Yield.Begin }

// End gets the byte offset one past the last character in this node
func (n *YieldStmt) End() token.Pos {
	if !IsNil(n.Value) {
		return n.Value.End()
	}
	return n.Yield.End
}

func (n *YieldStmt) stmtNode() {}
func (n *YieldStmt) walk(v EdgeVisitor) {
	if !IsNil(n.Value) {
		walkEdge(v, n, n.Value, "Value")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *YieldStmt) AddOffset(i int) {
	addWordOffset(n.Yield, i)
}

// Iterate implements Node
func (n *YieldStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Yield)
	h.VisitNode(exprRef{&n.Value})
}

// CopyIterable implements Node
func (n *YieldStmt) CopyIterable() Node {
	new := *n
	return &new
}

// GlobalStmt represents a "global" statement, e.g. "global x, y, z"
type GlobalStmt struct {
	Global *pythonscanner.Word
	Names  []*NameExpr
}

// Begin gets the byte offset of the first character in this node
func (n *GlobalStmt) Begin() token.Pos { return n.Global.Begin }

// End gets the byte offset one past the last character in this node
func (n *GlobalStmt) End() token.Pos { return n.Names[len(n.Names)-1].End() }

func (n *GlobalStmt) stmtNode() {}
func (n *GlobalStmt) walk(v EdgeVisitor) {
	walkNameList(v, n, n.Names, "Names")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *GlobalStmt) AddOffset(i int) {
	addWordOffset(n.Global, i)
}

// Iterate implements Node
func (n *GlobalStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Global)
	h.VisitSlice(nameSlice{&n.Names})
}

// CopyIterable implements Node
func (n *GlobalStmt) CopyIterable() Node {
	new := *n
	new.Names = append([]*NameExpr{}, n.Names...)
	return &new
}

// NonLocalStmt represents a "nonlocal" statement, e.g. "nonlocal x, y, z"
type NonLocalStmt struct {
	NonLocal *pythonscanner.Word
	Names    []*NameExpr
}

// Begin gets the byte offset of the first character in this node
func (n *NonLocalStmt) Begin() token.Pos { return n.NonLocal.Begin }

// End gets the byte offset one past the last character in this node
func (n *NonLocalStmt) End() token.Pos { return n.Names[len(n.Names)-1].End() }

func (n *NonLocalStmt) stmtNode() {}
func (n *NonLocalStmt) walk(v EdgeVisitor) {
	walkNameList(v, n, n.Names, "Names")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *NonLocalStmt) AddOffset(i int) {
	addWordOffset(n.NonLocal, i)
}

// Iterate implements Node
func (n *NonLocalStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.NonLocal)
	h.VisitSlice(nameSlice{&n.Names})
}

// CopyIterable implements Node
func (n *NonLocalStmt) CopyIterable() Node {
	new := *n
	new.Names = append([]*NameExpr{}, n.Names...)
	return &new
}

// Branch represents a single condition and body within an if statement
type Branch struct {
	Condition Expr
	Body      []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *Branch) Begin() token.Pos { return n.Condition.Begin() }

// End gets the byte offset one past the last character in this node
func (n *Branch) End() token.Pos { return n.Body[len(n.Body)-1].End() }

func (n *Branch) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Condition, "Condition")
	walkStmtList(v, n, n.Body, "Body")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *Branch) AddOffset(i int) {}

// Iterate implements Node
func (n *Branch) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Condition})
	h.VisitSlice(stmtSlice{&n.Body})
}

// CopyIterable implements Node
func (n *Branch) CopyIterable() Node {
	new := *n
	new.Body = append([]Stmt{}, n.Body...)
	return &new
}

// IfStmt represents an if statement, e.g. "if foo: ... elif bar: ... else: ..."
type IfStmt struct {
	If       *pythonscanner.Word
	Branches []*Branch
	Else     []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *IfStmt) Begin() token.Pos { return n.If.Begin }

// End gets the byte offset one past the last character in this node
func (n *IfStmt) End() token.Pos {
	if len(n.Else) > 0 {
		return n.Else[len(n.Else)-1].End()
	}
	return n.Branches[len(n.Branches)-1].End()
}

func (n *IfStmt) stmtNode() {}
func (n *IfStmt) walk(v EdgeVisitor) {
	for _, branch := range n.Branches {
		walkEdge(v, n, branch, "Branches")
	}
	walkStmtList(v, n, n.Else, "Else")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *IfStmt) AddOffset(i int) {
	addWordOffset(n.If, i)
}

// Iterate implements Node
func (n *IfStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.If)
	h.VisitSlice(branchSlice{&n.Branches})
	h.VisitSlice(stmtSlice{&n.Else})
}

// CopyIterable implements Node
func (n *IfStmt) CopyIterable() Node {
	new := *n
	new.Branches = append([]*Branch{}, n.Branches...)
	new.Else = append([]Stmt{}, n.Else...)
	return &new
}

// ForStmt represents a for loop, e.g. "for x in y: ..."
type ForStmt struct {
	Async    *pythonscanner.Word
	For      *pythonscanner.Word
	Targets  []Expr
	Iterable Expr
	Body     []Stmt
	Else     []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *ForStmt) Begin() token.Pos {
	if n.Async != nil {
		return n.Async.Begin
	}
	return n.For.Begin
}

// End gets the byte offset one past the last character in this node
func (n *ForStmt) End() token.Pos {
	if len(n.Else) > 0 {
		return n.Else[len(n.Else)-1].End()
	}
	return n.Body[len(n.Body)-1].End()
}

func (n *ForStmt) stmtNode() {}
func (n *ForStmt) walk(v EdgeVisitor) {
	walkExprList(v, n, n.Targets, "Targets")
	walkEdge(v, n, n.Iterable, "Iterable")
	walkStmtList(v, n, n.Body, "Body")
	walkStmtList(v, n, n.Else, "Else")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ForStmt) AddOffset(i int) {
	addWordOffset(n.Async, i)
	addWordOffset(n.For, i)
}

// Iterate implements Node
func (n *ForStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Async)
	h.VisitWord(&n.For)
	h.VisitSlice(exprSlice{&n.Targets})
	h.VisitNode(exprRef{&n.Iterable})
	h.VisitSlice(stmtSlice{&n.Body})
	h.VisitSlice(stmtSlice{&n.Else})
}

// CopyIterable implements Node
func (n *ForStmt) CopyIterable() Node {
	new := *n
	new.Targets = append([]Expr{}, n.Targets...)
	new.Body = append([]Stmt{}, n.Body...)
	new.Else = append([]Stmt{}, n.Else...)
	return &new
}

// WhileStmt represents a while loop, e.g. "while foo: ..."
type WhileStmt struct {
	While     *pythonscanner.Word
	Condition Expr
	Body      []Stmt
	Else      []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *WhileStmt) Begin() token.Pos { return n.While.Begin }

// End gets the byte offset one past the last character in this node
func (n *WhileStmt) End() token.Pos {
	if len(n.Else) > 0 {
		return n.Else[len(n.Else)-1].End()
	}
	return n.Body[len(n.Body)-1].End()
}

func (n *WhileStmt) stmtNode() {}
func (n *WhileStmt) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Condition, "Condition")
	walkStmtList(v, n, n.Body, "Body")
	walkStmtList(v, n, n.Else, "Else")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *WhileStmt) AddOffset(i int) {
	addWordOffset(n.While, i)
}

// Iterate implements Node
func (n *WhileStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.While)
	h.VisitNode(exprRef{&n.Condition})
	h.VisitSlice(stmtSlice{&n.Body})
	h.VisitSlice(stmtSlice{&n.Else})
}

// CopyIterable implements Node
func (n *WhileStmt) CopyIterable() Node {
	new := *n
	new.Body = append([]Stmt{}, n.Body...)
	new.Else = append([]Stmt{}, n.Else...)
	return &new
}

// ExceptClause represents an "except" clause within a try statement, e.g. "except Exception as ex: ..."
type ExceptClause struct {
	Except *pythonscanner.Word
	Type   Expr
	Target Expr
	Body   []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *ExceptClause) Begin() token.Pos { return n.Except.Begin }

// End gets the byte offset one past the last character in this node
func (n *ExceptClause) End() token.Pos { return n.Body[len(n.Body)-1].End() }

func (n *ExceptClause) walk(v EdgeVisitor) {
	if !IsNil(n.Type) {
		walkEdge(v, n, n.Type, "Type")
	}
	if !IsNil(n.Target) {
		walkEdge(v, n, n.Target, "Target")
	}
	walkStmtList(v, n, n.Body, "Body")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *ExceptClause) AddOffset(i int) {
	addWordOffset(n.Except, i)
}

// Iterate implements Node
func (n *ExceptClause) Iterate(h IterationHandler) {
	h.VisitWord(&n.Except)
	h.VisitNode(exprRef{&n.Type})
	h.VisitNode(exprRef{&n.Target})
	h.VisitSlice(stmtSlice{&n.Body})
}

// CopyIterable implements Node
func (n *ExceptClause) CopyIterable() Node {
	new := *n
	new.Body = append([]Stmt{}, n.Body...)
	return &new
}

// TryStmt represents a try...except block, e.g. "try: ... except TypeError: ... else: ... finally: ..."
type TryStmt struct {
	Try      *pythonscanner.Word
	Body     []Stmt
	Handlers []*ExceptClause
	Else     []Stmt
	Finally  []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *TryStmt) Begin() token.Pos { return n.Try.Begin }

// End gets the byte offset one past the last character in this node
func (n *TryStmt) End() token.Pos {
	if len(n.Finally) > 0 {
		return n.Finally[len(n.Finally)-1].End()
	}
	if len(n.Else) > 0 {
		return n.Else[len(n.Else)-1].End()
	}
	if len(n.Handlers) > 0 {
		return n.Handlers[len(n.Handlers)-1].End()
	}
	return n.Body[len(n.Body)-1].End()
}

func (n *TryStmt) stmtNode() {}
func (n *TryStmt) walk(v EdgeVisitor) {
	walkStmtList(v, n, n.Body, "Body")
	for _, clause := range n.Handlers {
		walkEdge(v, n, clause, "Handlers")
	}
	walkStmtList(v, n, n.Else, "Else")
	walkStmtList(v, n, n.Finally, "Finally")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *TryStmt) AddOffset(i int) {
	addWordOffset(n.Try, i)
}

// Iterate implements Node
func (n *TryStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Try)
	h.VisitSlice(stmtSlice{&n.Body})
	h.VisitSlice(exceptClauseSlice{&n.Handlers})
	h.VisitSlice(stmtSlice{&n.Else})
	h.VisitSlice(stmtSlice{&n.Finally})
}

// CopyIterable implements Node
func (n *TryStmt) CopyIterable() Node {
	new := *n
	new.Body = append([]Stmt{}, n.Body...)
	new.Handlers = append([]*ExceptClause{}, n.Handlers...)
	new.Else = append([]Stmt{}, n.Else...)
	new.Finally = append([]Stmt{}, n.Finally...)
	return &new
}

// WithItem represents a value and variable name inside a "with" statement, e.g. "foo as bar"
type WithItem struct {
	Value  Expr
	Target Expr
}

// Begin gets the byte offset of the first character in this node
func (n *WithItem) Begin() token.Pos { return n.Value.Begin() }

// End gets the byte offset one past the last character in this node
func (n *WithItem) End() token.Pos { return switchEnd(n.Value, n.Target) }

func (n *WithItem) walk(v EdgeVisitor) {
	walkEdge(v, n, n.Value, "Value")
	if !IsNil(n.Target) {
		walkEdge(v, n, n.Target, "Target")
	}
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *WithItem) AddOffset(i int) {}

// Iterate implements Node
func (n *WithItem) Iterate(h IterationHandler) {
	h.VisitNode(exprRef{&n.Value})
	h.VisitNode(exprRef{&n.Target})
}

// CopyIterable implements Node
func (n *WithItem) CopyIterable() Node {
	new := *n
	return &new
}

// WithStmt represents a "with" statement, e.g. "with foo as bar: ..."
type WithStmt struct {
	Async *pythonscanner.Word
	With  *pythonscanner.Word
	Items []*WithItem
	Body  []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *WithStmt) Begin() token.Pos {
	if n.Async != nil {
		return n.Async.Begin
	}
	return n.With.Begin
}

// End gets the byte offset one past the last character in this node
func (n *WithStmt) End() token.Pos { return n.Body[len(n.Body)-1].End() }

func (n *WithStmt) stmtNode() {}
func (n *WithStmt) walk(v EdgeVisitor) {
	for _, item := range n.Items {
		walkEdge(v, n, item, "Items")
	}
	walkStmtList(v, n, n.Body, "Body")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *WithStmt) AddOffset(i int) {
	addWordOffset(n.Async, i)
	addWordOffset(n.With, i)
}

// Iterate implements Node
func (n *WithStmt) Iterate(h IterationHandler) {
	h.VisitWord(&n.Async)
	h.VisitWord(&n.With)
	h.VisitSlice(withItemSlice{&n.Items})
	h.VisitSlice(stmtSlice{&n.Body})
}

// CopyIterable implements Node
func (n *WithStmt) CopyIterable() Node {
	new := *n
	new.Items = append([]*WithItem{}, n.Items...)
	new.Body = append([]Stmt{}, n.Body...)
	return &new
}

// Module is the root AST node for a source file
type Module struct {
	Body []Stmt
}

// Begin gets the byte offset of the first character in this node
func (n *Module) Begin() token.Pos {
	if len(n.Body) == 0 {
		return token.Pos(0)
	}
	return n.Body[0].Begin()
}

// End gets the byte offset one past the last character in this node
func (n *Module) End() token.Pos {
	if len(n.Body) == 0 {
		return token.Pos(0)
	}
	return n.Body[len(n.Body)-1].End()
}

func (n *Module) scopeNode() {}

func (n *Module) walk(v EdgeVisitor) {
	walkStmtList(v, n, n.Body, "Body")
}

// AddOffset adds i to the position of any contained non-recursively contained pythonscanner.Words
func (n *Module) AddOffset(i int) {}

// Iterate implements Node
func (n *Module) Iterate(h IterationHandler) {
	h.VisitSlice(stmtSlice{&n.Body})
}

// CopyIterable implements Node
func (n *Module) CopyIterable() Node {
	new := *n
	new.Body = append([]Stmt{}, n.Body...)
	return &new
}
"""
