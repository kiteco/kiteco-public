package ranking

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogSumExp(t *testing.T) {
	a := math.Log(3)
	b := math.Log(10)
	c := math.Log(21)

	act := logSumExp([]float64{a, b, c})
	exp := math.Log(34)
	assert.Equal(t, exp, act)

	d := 1000000.0
	act = logSumExp([]float64{a, b, c, d})
	exp = math.Log(34) + d

	assert.Equal(t, d, act)
}

func TestSum(t *testing.T) {
	act := sum([]float64{1, 2, 3, 4, 5})
	exp := 15

	assert.EqualValues(t, exp, act)
}
