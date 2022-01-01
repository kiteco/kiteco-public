package typeinduction

import (
	"math"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// MockClient returns a `*Client` that uses the provided functions
// and return types for inference.
func MockClient(manager pythonresource.Manager, funToReturn map[string]string) *Client {
	var types []*Type
	var functions []*Function

	for fun, ret := range funToReturn {
		types = append(types, &Type{
			Name: ret,
		})
		functions = append(functions, &Function{
			Name: fun,
			ReturnType: []Element{
				Element{ret, math.Log(.99)},
			},
		})
	}

	return ModelFromData(types, functions, manager, nil)
}
