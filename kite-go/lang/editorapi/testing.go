package editorapi

import (
	"math/rand"
	"reflect"
	"testing/quick"

	"github.com/kiteco/kiteco/kite-go/lang"
)

// Generate implements quick.Generator (from testing/quick)
func (ID) Generate(rand *rand.Rand, size int) reflect.Value {
	id, ok := quick.Value(reflect.TypeOf((*string)(nil)).Elem(), rand)
	if !ok {
		panic("failed to generate a random string")
	}
	return reflect.ValueOf(NewID(lang.Python, id.Interface().(string)))
}
