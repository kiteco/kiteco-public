package pythonast

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/stretchr/testify/assert"
)

type refValidator map[uintptr]struct{}
type valValidator map[uintptr]struct{}

func (h refValidator) VisitSlice(s NodeSliceRef) {
	VisitNodeSlice(h, s)
}

func (h refValidator) VisitNode(r NodeRef) {
	h[reflect.ValueOf(r).Field(0).Pointer()] = struct{}{}
	n := r.Lookup()
	if !IsNil(n) {
		n.Iterate(h)
	}
}
func (h refValidator) VisitWord(r **pythonscanner.Word) {
	h[uintptr(unsafe.Pointer(r))] = struct{}{}
}

func (h valValidator) VisitSlice(s NodeSliceRef) {
	VisitNodeSlice(h, s)
}

func (h valValidator) VisitNode(r NodeRef) {
	v := reflect.ValueOf(r).Field(0).Elem()
	// if r contains a *Node, *Expr, etc, then v is an interface value, so peek inside the interface to get the pointer
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	h[v.Pointer()] = struct{}{}

	n := r.Lookup()
	if !IsNil(n) {
		n.Iterate(h)
	}
}
func (h valValidator) VisitWord(r **pythonscanner.Word) {
	h[uintptr(unsafe.Pointer(*r))] = struct{}{}
}

func validateIterate(t testing.TB, n Node) {
	nodeInterfaceType := reflect.TypeOf((*Node)(nil)).Elem()
	wordType := reflect.TypeOf((*pythonscanner.Word)(nil))
	baseComprehensionType := reflect.TypeOf((*BaseComprehension)(nil))

	// n is a Node value containing a pointer to a struct
	reference := make(refValidator)

	var collect func(reflect.Value)
	collect = func(v reflect.Value) {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			ty := f.Type()
			switch ty.Kind() {
			case reflect.Slice, reflect.Array:
				if elemTy := ty.Elem(); elemTy == wordType || elemTy.Implements(nodeInterfaceType) {
					for i := 0; i < f.Len(); i++ {
						reference[f.Index(i).UnsafeAddr()] = struct{}{}
					}
				}
			default:
				if ty == wordType || ty.Implements(nodeInterfaceType) {
					reference[f.UnsafeAddr()] = struct{}{}
				} else if ty == baseComprehensionType {
					collect(f.Elem())
				}
			}
		}
	}

	collect(reflect.ValueOf(n).Elem())

	actual := make(refValidator)
	n.Iterate(actual)
	assert.Equal(t, reference, actual)
}

// TestIterate tests iterate for all Nodes, Words that are contained directly in Nodes (i.e. not in a slice/array)
// Once we add functionality for allowing the iteration handler to append to slices, we should be able to test that as well.
func TestIterate(t *testing.T) {
	validateIterate(t, &CallExpr{})
	validateIterate(t, &NameExpr{})
	validateIterate(t, &TupleExpr{})
	validateIterate(t, &IndexExpr{})
	validateIterate(t, &AttributeExpr{})
	validateIterate(t, &NumberExpr{})
	validateIterate(t, &StringExpr{})
	validateIterate(t, &ListExpr{})
	validateIterate(t, &SetExpr{})
	validateIterate(t, &DictExpr{})
	validateIterate(t, &ComprehensionExpr{BaseComprehension: &BaseComprehension{}})
	validateIterate(t, &ListComprehensionExpr{BaseComprehension: &BaseComprehension{}})
	validateIterate(t, &DictComprehensionExpr{BaseComprehension: &BaseComprehension{}})
	validateIterate(t, &SetComprehensionExpr{BaseComprehension: &BaseComprehension{}})
	validateIterate(t, &UnaryExpr{})
	validateIterate(t, &BinaryExpr{})
	validateIterate(t, &CallExpr{})
	validateIterate(t, &LambdaExpr{})
	validateIterate(t, &ReprExpr{})
	validateIterate(t, &IfExpr{})
	validateIterate(t, &YieldExpr{})
	validateIterate(t, &AwaitExpr{})
	validateIterate(t, &BadExpr{})
	validateIterate(t, &DottedExpr{})
	validateIterate(t, &DottedAsName{})
	validateIterate(t, &ImportAsName{})
	validateIterate(t, &ImportNameStmt{})
	validateIterate(t, &ImportFromStmt{})
	validateIterate(t, &IndexSubscript{})
	validateIterate(t, &SliceSubscript{})
	validateIterate(t, &EllipsisExpr{})
	validateIterate(t, &KeyValuePair{})
	validateIterate(t, &Generator{})
	validateIterate(t, &Argument{})
	validateIterate(t, &BadStmt{})
	validateIterate(t, &ExprStmt{})
	validateIterate(t, &AnnotationStmt{})
	validateIterate(t, &AssignStmt{})
	validateIterate(t, &AugAssignStmt{})
	validateIterate(t, &ClassDefStmt{})
	validateIterate(t, &Parameter{})
	validateIterate(t, &ArgsParameter{})
	validateIterate(t, &FunctionDefStmt{})
	validateIterate(t, &AssertStmt{})
	validateIterate(t, &ContinueStmt{})
	validateIterate(t, &BreakStmt{})
	validateIterate(t, &DelStmt{})
	validateIterate(t, &ExecStmt{})
	validateIterate(t, &PassStmt{})
	validateIterate(t, &PrintStmt{})
	validateIterate(t, &RaiseStmt{})
	validateIterate(t, &ReturnStmt{})
	validateIterate(t, &YieldStmt{})
	validateIterate(t, &GlobalStmt{})
	validateIterate(t, &NonLocalStmt{})
	validateIterate(t, &Branch{})
	validateIterate(t, &IfStmt{})
	validateIterate(t, &ForStmt{})
	validateIterate(t, &WhileStmt{})
	validateIterate(t, &ExceptClause{})
	validateIterate(t, &TryStmt{})
	validateIterate(t, &WithItem{})
	validateIterate(t, &WithStmt{})
	validateIterate(t, &Module{})
}

func TestDeepCopy(t *testing.T) {
	// TODO(naman) use a more complicated AST here
	originalPtrs, afterCopyPtrs, copyPtrs := make(valValidator), make(valValidator), make(valValidator)

	outer.Iterate(originalPtrs)

	copy := DeepCopy(outer)[outer]

	outer.Iterate(afterCopyPtrs)
	copy.Iterate(copyPtrs)

	// check that DeepCopy didn't modify the original
	assert.Equal(t, originalPtrs, afterCopyPtrs)
	// check that the copy and original are disjoint but the same size
	assert.Equal(t, len(originalPtrs), len(copyPtrs))
	for p := range copyPtrs {
		if p == uintptr(0) {
			continue
		}
		_, ok := originalPtrs[p]
		assert.False(t, ok)
	}
}
