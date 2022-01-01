package pythonast

import (
	"go/token"
	"reflect"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/stretchr/testify/assert"
)

func validateAddOffset(t testing.TB, n Node) {
	wordType := reflect.TypeOf(pythonscanner.Word{})
	wordSliceType := reflect.TypeOf(([]pythonscanner.Word)(nil))
	wordPtrType := reflect.TypeOf((*pythonscanner.Word)(nil))
	wordPtrSliceType := reflect.TypeOf(([]*pythonscanner.Word)(nil))

	// n is a Node value containing a pointer to a struct
	s := reflect.ValueOf(n).Elem()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		switch f.Type() {
		case wordType:
			f.Set(reflect.ValueOf(pythonscanner.Word{}))
		case wordSliceType:
			f.Set(reflect.ValueOf([]pythonscanner.Word{pythonscanner.Word{}}))
		case wordPtrType:
			f.Set(reflect.ValueOf(&pythonscanner.Word{}))
		case wordPtrSliceType:
			f.Set(reflect.ValueOf([]*pythonscanner.Word{&pythonscanner.Word{}}))
		}
	}

	n.AddOffset(5)

	posType := reflect.TypeOf(token.Pos(0))
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		switch f.Type() {
		case wordType:
			w := f.Interface().(pythonscanner.Word)
			assert.Equal(t, token.Pos(5), w.Begin, s.Type().String())
			assert.Equal(t, token.Pos(5), w.End, s.Type().String())
		case wordSliceType:
			w := f.Interface().([]pythonscanner.Word)[0]
			assert.Equal(t, token.Pos(5), w.Begin, s.Type().String())
			assert.Equal(t, token.Pos(5), w.End, s.Type().String())
		case wordPtrType:
			w := f.Interface().(*pythonscanner.Word)
			assert.Equal(t, token.Pos(5), w.Begin, s.Type().String())
			assert.Equal(t, token.Pos(5), w.End, s.Type().String())
		case wordPtrSliceType:
			w := f.Interface().([]*pythonscanner.Word)[0]
			assert.Equal(t, token.Pos(5), w.Begin, s.Type().String())
			assert.Equal(t, token.Pos(5), w.End, s.Type().String())
		case posType:
			p := f.Interface().(token.Pos)
			assert.Equal(t, token.Pos(5), p, s.Type().String())
		}
	}
}

func TestAddOffset(t *testing.T) {
	validateAddOffset(t, &CallExpr{})
	validateAddOffset(t, &NameExpr{})
	validateAddOffset(t, &TupleExpr{})
	validateAddOffset(t, &IndexExpr{})
	validateAddOffset(t, &AttributeExpr{})
	validateAddOffset(t, &NumberExpr{})
	validateAddOffset(t, &StringExpr{})
	validateAddOffset(t, &ListExpr{})
	validateAddOffset(t, &SetExpr{})
	validateAddOffset(t, &DictExpr{})
	validateAddOffset(t, &ComprehensionExpr{})
	validateAddOffset(t, &ListComprehensionExpr{})
	validateAddOffset(t, &DictComprehensionExpr{})
	validateAddOffset(t, &SetComprehensionExpr{})
	validateAddOffset(t, &UnaryExpr{})
	validateAddOffset(t, &BinaryExpr{})
	validateAddOffset(t, &CallExpr{})
	validateAddOffset(t, &LambdaExpr{})
	validateAddOffset(t, &ReprExpr{})
	validateAddOffset(t, &IfExpr{})
	validateAddOffset(t, &YieldExpr{})
	validateAddOffset(t, &AwaitExpr{})
	validateAddOffset(t, &BadExpr{})
	validateAddOffset(t, &DottedExpr{})
	validateAddOffset(t, &DottedAsName{})
	validateAddOffset(t, &ImportAsName{})
	validateAddOffset(t, &ImportNameStmt{})
	validateAddOffset(t, &ImportFromStmt{})
	validateAddOffset(t, &IndexSubscript{})
	validateAddOffset(t, &SliceSubscript{})
	validateAddOffset(t, &EllipsisExpr{})
	validateAddOffset(t, &KeyValuePair{})
	validateAddOffset(t, &Generator{})
	validateAddOffset(t, &Argument{})
	validateAddOffset(t, &BadStmt{})
	validateAddOffset(t, &ExprStmt{})
	validateAddOffset(t, &AnnotationStmt{})
	validateAddOffset(t, &AssignStmt{})
	validateAddOffset(t, &AugAssignStmt{})
	validateAddOffset(t, &ClassDefStmt{})
	validateAddOffset(t, &Parameter{})
	validateAddOffset(t, &ArgsParameter{})
	validateAddOffset(t, &FunctionDefStmt{})
	validateAddOffset(t, &AssertStmt{})
	validateAddOffset(t, &ContinueStmt{})
	validateAddOffset(t, &BreakStmt{})
	validateAddOffset(t, &DelStmt{})
	validateAddOffset(t, &ExecStmt{})
	validateAddOffset(t, &PassStmt{})
	validateAddOffset(t, &PrintStmt{})
	validateAddOffset(t, &RaiseStmt{})
	validateAddOffset(t, &ReturnStmt{})
	validateAddOffset(t, &YieldStmt{})
	validateAddOffset(t, &GlobalStmt{})
	validateAddOffset(t, &NonLocalStmt{})
	validateAddOffset(t, &Branch{})
	validateAddOffset(t, &IfStmt{})
	validateAddOffset(t, &ForStmt{})
	validateAddOffset(t, &WhileStmt{})
	validateAddOffset(t, &ExceptClause{})
	validateAddOffset(t, &TryStmt{})
	validateAddOffset(t, &WithItem{})
	validateAddOffset(t, &WithStmt{})
	validateAddOffset(t, &Module{})
}
