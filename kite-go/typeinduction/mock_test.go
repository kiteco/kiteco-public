package typeinduction

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/stretchr/testify/assert"
)

func TestMockClient(t *testing.T) {
	manager := pythonresource.MockManager(t, nil, "func1", "func2", "type1", "type2")

	client := MockClient(manager, map[string]string{
		"func1": "type1",
		"func2": "type2",
	})

	func1, _ := manager.PathSymbol(pythonimports.NewDottedPath("func1"))
	func2, _ := manager.PathSymbol(pythonimports.NewDottedPath("func2"))

	estimate := client.EstimateType(Observation{
		ReturnedFrom: func1,
	})
	assert.Equal(t, "type1", estimate.MostProbableType.Canonical().Path().String())

	estimate = client.EstimateType(Observation{
		ReturnedFrom: func2,
	})
	assert.Equal(t, "type2", estimate.MostProbableType.Canonical().Path().String())
}
