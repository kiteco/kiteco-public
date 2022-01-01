package pythonast

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

// NodeSliceRef is abstractly isomorphic to *[]Node, but can flexibly encapsulate e.g []*Argument, []Stmt, etc
type NodeSliceRef interface {
	Get(int) NodeRef
	Assign([]Node) bool
	Len() int
	Equal(NodeSliceRef) bool
}

type nameSlice struct{ ns *[]*NameExpr }
type dottedAsNameSlice struct{ ns *[]*DottedAsName }
type importAsNameSlice struct{ ns *[]*ImportAsName }
type exprSlice struct{ ns *[]Expr }
type subscriptSlice struct{ ns *[]Subscript }
type keyValuePairSlice struct{ ns *[]*KeyValuePair }
type generatorSlice struct{ ns *[]*Generator }
type argumentSlice struct{ ns *[]*Argument }
type parameterSlice struct{ ns *[]*Parameter }
type stmtSlice struct{ ns *[]Stmt }
type branchSlice struct{ ns *[]*Branch }
type exceptClauseSlice struct{ ns *[]*ExceptClause }
type withItemSlice struct{ ns *[]*WithItem }

func (s nameSlice) Get(i int) NodeRef {
	return nameExprRef{&(*s.ns)[i]}
}

func (s nameSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*NameExpr, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*NameExpr)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s nameSlice) Len() int {
	return len(*s.ns)
}

func (s nameSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(nameSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s dottedAsNameSlice) Get(i int) NodeRef {
	return dottedAsNameRef{&(*s.ns)[i]}
}

func (s dottedAsNameSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*DottedAsName, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*DottedAsName)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s dottedAsNameSlice) Len() int {
	return len(*s.ns)
}

func (s dottedAsNameSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(dottedAsNameSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s importAsNameSlice) Get(i int) NodeRef {
	return importAsNameRef{&(*s.ns)[i]}
}

func (s importAsNameSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*ImportAsName, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*ImportAsName)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s importAsNameSlice) Len() int {
	return len(*s.ns)
}

func (s importAsNameSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(importAsNameSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s exprSlice) Get(i int) NodeRef {
	return exprRef{&(*s.ns)[i]}
}

func (s exprSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]Expr, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(Expr)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s exprSlice) Len() int {
	return len(*s.ns)
}

func (s exprSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(exprSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s subscriptSlice) Get(i int) NodeRef {
	return subscriptRef{&(*s.ns)[i]}
}

func (s subscriptSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]Subscript, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(Subscript)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s subscriptSlice) Len() int {
	return len(*s.ns)
}

func (s subscriptSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(subscriptSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s keyValuePairSlice) Get(i int) NodeRef {
	return keyValueRef{&(*s.ns)[i]}
}

func (s keyValuePairSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*KeyValuePair, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*KeyValuePair)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s keyValuePairSlice) Len() int {
	return len(*s.ns)
}

func (s keyValuePairSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(keyValuePairSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s generatorSlice) Get(i int) NodeRef {
	return generatorRef{&(*s.ns)[i]}
}

func (s generatorSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*Generator, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*Generator)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s generatorSlice) Len() int {
	return len(*s.ns)
}

func (s generatorSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(generatorSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s argumentSlice) Get(i int) NodeRef {
	return argumentRef{&(*s.ns)[i]}
}

func (s argumentSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*Argument, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*Argument)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s argumentSlice) Len() int {
	return len(*s.ns)
}

func (s argumentSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(argumentSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s parameterSlice) Get(i int) NodeRef {
	return paramRef{&(*s.ns)[i]}
}

func (s parameterSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*Parameter, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*Parameter)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s parameterSlice) Len() int {
	return len(*s.ns)
}

func (s parameterSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(parameterSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s stmtSlice) Get(i int) NodeRef {
	return stmtRef{&(*s.ns)[i]}
}

func (s stmtSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]Stmt, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(Stmt)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s stmtSlice) Len() int {
	return len(*s.ns)
}

func (s stmtSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(stmtSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s branchSlice) Get(i int) NodeRef {
	return branchRef{&(*s.ns)[i]}
}

func (s branchSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*Branch, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*Branch)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s branchSlice) Len() int {
	return len(*s.ns)
}

func (s branchSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(branchSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s exceptClauseSlice) Get(i int) NodeRef {
	return exceptClauseRef{&(*s.ns)[i]}
}

func (s exceptClauseSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*ExceptClause, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*ExceptClause)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s exceptClauseSlice) Len() int {
	return len(*s.ns)
}

func (s exceptClauseSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(exceptClauseSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

func (s withItemSlice) Get(i int) NodeRef {
	return withItemRef{&(*s.ns)[i]}
}

func (s withItemSlice) Assign(ns []Node) bool {
	if ns == nil {
		*s.ns = nil
		return true
	}

	nsc := make([]*WithItem, 0, len(ns))
	for _, n := range ns {
		nc, ok := n.(*WithItem)
		if !ok {
			return false
		}
		nsc = append(nsc, nc)
	}
	*s.ns = nsc
	return true
}

func (s withItemSlice) Len() int {
	return len(*s.ns)
}

func (s withItemSlice) Equal(other NodeSliceRef) bool {
	if ss, ok := other.(withItemSlice); ok {
		return s.ns == ss.ns
	}
	return false
}

// NodeRef is abstractly isomorphic to *Node, but can flexibly encapsulate e.g. *Expr, **NameExpr, etc
type NodeRef interface {
	Lookup() Node
	// Assign returns false if the new Node is of an incompatible underlying type
	Assign(new Node) bool
}
type exprRef struct{ n *Expr }
type stmtRef struct{ n *Stmt }
type subscriptRef struct{ n *Subscript }
type nameExprRef struct{ n **NameExpr }
type dottedExprRef struct{ n **DottedExpr }
type dottedAsNameRef struct{ n **DottedAsName }
type importAsNameRef struct{ n **ImportAsName }
type keyValueRef struct{ n **KeyValuePair }
type generatorRef struct{ n **Generator }
type argumentRef struct{ n **Argument }
type paramRef struct{ n **Parameter }
type argsParamRef struct{ n **ArgsParameter }
type branchRef struct{ n **Branch }
type exceptClauseRef struct{ n **ExceptClause }
type withItemRef struct{ n **WithItem }

func (r exprRef) Lookup() Node { return *r.n }
func (r exprRef) Assign(new Node) bool {
	if n, ok := new.(Expr); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r stmtRef) Lookup() Node { return *r.n }
func (r stmtRef) Assign(new Node) bool {
	if n, ok := new.(Stmt); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r subscriptRef) Lookup() Node { return *r.n }
func (r subscriptRef) Assign(new Node) bool {
	if n, ok := new.(Subscript); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r nameExprRef) Lookup() Node { return *r.n }
func (r nameExprRef) Assign(new Node) bool {
	if n, ok := new.(*NameExpr); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r dottedExprRef) Lookup() Node { return *r.n }
func (r dottedExprRef) Assign(new Node) bool {
	if n, ok := new.(*DottedExpr); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r dottedAsNameRef) Lookup() Node { return *r.n }
func (r dottedAsNameRef) Assign(new Node) bool {
	if n, ok := new.(*DottedAsName); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r importAsNameRef) Lookup() Node { return *r.n }
func (r importAsNameRef) Assign(new Node) bool {
	if n, ok := new.(*ImportAsName); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r keyValueRef) Lookup() Node { return *r.n }
func (r keyValueRef) Assign(new Node) bool {
	if n, ok := new.(*KeyValuePair); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r generatorRef) Lookup() Node { return *r.n }
func (r generatorRef) Assign(new Node) bool {
	if n, ok := new.(*Generator); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r argumentRef) Lookup() Node { return *r.n }
func (r argumentRef) Assign(new Node) bool {
	if n, ok := new.(*Argument); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r paramRef) Lookup() Node { return *r.n }
func (r paramRef) Assign(new Node) bool {
	if n, ok := new.(*Parameter); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r argsParamRef) Lookup() Node { return *r.n }
func (r argsParamRef) Assign(new Node) bool {
	if n, ok := new.(*ArgsParameter); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r branchRef) Lookup() Node { return *r.n }
func (r branchRef) Assign(new Node) bool {
	if n, ok := new.(*Branch); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r exceptClauseRef) Lookup() Node { return *r.n }
func (r exceptClauseRef) Assign(new Node) bool {
	if n, ok := new.(*ExceptClause); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}
func (r withItemRef) Lookup() Node { return *r.n }
func (r withItemRef) Assign(new Node) bool {
	if n, ok := new.(*WithItem); ok || IsNil(new) {
		*r.n = n
		return true
	}
	return false
}

// IterationHandler handles AST iteration callbacks; it is responsible for recursively iterating over children if necessary
// NOTE: the order in which VisitNode/VisitSlice/VisitWord are interleaved is not neccesarily
//       meaningful since we can no longer order everything correctly; however the order IS determinstic
//       so for the same node type the fields will always be visited in the same order.
// TODO(naman) ideally we'd be able to handle slices of words here (e.g. appending to slices)
type IterationHandler interface {
	VisitNode(NodeRef)
	VisitSlice(NodeSliceRef)
	VisitWord(**pythonscanner.Word)
}

// Iterate iterates over the (possibly nil) AST rooted at n, using the provided IterationHandler
func Iterate(h IterationHandler, n Node) {
	if IsNil(n) {
		return
	}
	n.Iterate(h)
}

// VisitNodeSlice calls `c.VisitNode` on each node ref in `s`
func VisitNodeSlice(c IterationHandler, s NodeSliceRef) {
	for i := 0; i < s.Len(); i++ {
		c.VisitNode(s.Get(i))
	}
}

// - deep copy

type deepCopier map[Node]Node

func (c deepCopier) VisitSlice(s NodeSliceRef) {
	VisitNodeSlice(c, s)
}

func (c deepCopier) VisitNode(r NodeRef) {
	n := r.Lookup()
	if IsNil(n) {
		return
	}

	// r's parent should already be a fresh copy (see implementation of DeepCopy below), so we can assign to it
	if !r.Assign(c.deepCopy(n)) {
		panic("incompatible type produced during deep copy")
	}
}

func (c deepCopier) VisitWord(w **pythonscanner.Word) {
	if *w == nil {
		return
	}
	// don't be tempted to do &(**w); that doesn't make a copy
	tmp := **w
	*w = &tmp
}
func (c deepCopier) deepCopy(n Node) Node {
	copy := n.CopyIterable()
	copy.Iterate(c)
	c[n] = copy
	return copy
}

// DeepCopy recursively copies an AST Node, including all contained Nodes & Words
// returns a map from old node to new node
func DeepCopy(root Node) map[Node]Node {
	c := deepCopier(make(map[Node]Node, CountNodes(root)))
	c.deepCopy(root)
	return c
}
