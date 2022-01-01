package popularsignatures

import (
	"reflect"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-golib/reflection"
)

func TestCast_Entity(t *testing.T) {
	if !reflection.StructurallyEqual(reflect.TypeOf([]*editorapi.Signature{}), reflect.TypeOf(Entity{})) {
		t.Logf("Entity type not structurally equal to []*editorapi.Signature")
		t.Fail()
	}
}
