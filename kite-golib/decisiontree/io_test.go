//go:generate go-bindata -o bindata.go -pkg decisiontree testdata/...

package decisiontree

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadModel(t *testing.T) {
	buf := bytes.NewBuffer(MustAsset("testdata/model.json"))
	model, err := Load(buf)
	assert.NoError(t, err, "")
	assert.Len(t, model.Trees, 5, "")
	assert.Equal(t, 5, model.Trees[0].Depth, "")
	assert.Equal(t, 2, model.Trees[0].FeatureSize, "")

	inputs := [][]float64{{0., 0.}, {0., 1.}, {1., 0.}, {10., 20}}
	expected := []float64{-22.0931034483, -20.4486770413, 0.627811584404, -6.37127659574}
	for i, input := range inputs {
		assert.InEpsilon(t, expected[i], model.Evaluate(input), 1e-8, "at i=%d", i)
	}
}
