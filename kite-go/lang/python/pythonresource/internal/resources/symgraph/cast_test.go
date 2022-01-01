package symgraph

import (
	"reflect"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/reflection"
)

func TestCast_DottedPath(t *testing.T) {
	if !reflection.StructurallyEqual(reflect.TypeOf(pythonimports.DottedPath{}), reflect.TypeOf(DottedPath{})) {
		t.Logf("DottedPath type not structurally equal to pythonimports.DottedPath")
		t.Fail()
	}
}
