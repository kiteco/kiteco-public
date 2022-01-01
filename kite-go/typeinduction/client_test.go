package typeinduction

import (
	"math"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateType(t *testing.T) {
	manager := pythonresource.MockManager(t, nil, "numpy.ndarray", "builtins.list", "numpy.zeros")
	types := []*Type{
		&Type{Name: "numpy.ndarray", Attributes: []Element{
			Element{"sum", math.Log(.4)},
			Element{"transpose", math.Log(.3)},
			Element{"append", math.Log(.3)},
		}},
		&Type{Name: "builtins.list", Attributes: []Element{
			Element{"append", math.Log(.6)},
			Element{"extend", math.Log(.4)},
		}},
	}

	functions := []*Function{
		&Function{Name: "numpy.zeros", ReturnType: []Element{
			Element{"numpy.ndarray", math.Log(.9)},
			Element{"builtins.list", math.Log(.1)},
		}},
	}

	client := ModelFromData(types, functions, manager, nil)
	zerosFunc, err := manager.PathSymbol(pythonimports.NewDottedPath("numpy.zeros"))
	require.Nil(t, err)

	estimate := client.EstimateType(Observation{
		ReturnedFrom: zerosFunc,
	})
	require.NotNil(t, estimate)
	assert.Equal(t, "numpy.ndarray", estimate.MostProbableType.Canonical().Path().String())

	estimate = client.EstimateType(Observation{
		ReturnedFrom: zerosFunc,
		Attributes:   []string{"extend"},
	})
	require.NotNil(t, estimate)
	assert.Equal(t, "builtins.list", estimate.MostProbableType.Canonical().Path().String())

	estimate = client.EstimateType(Observation{
		ReturnedFrom: zerosFunc,
		Attributes:   []string{"transpose", "append"},
	})
	require.NotNil(t, estimate)
	assert.Equal(t, "numpy.ndarray", estimate.MostProbableType.Canonical().Path().String())
}
