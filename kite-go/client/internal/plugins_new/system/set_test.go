package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnion(t *testing.T) {
	x := []string{"a", "b", "c"}
	y := []string{"b", "c", "d"}
	z := []string{"a", "b", "c"}

	union := Union(x, y, z)
	assert.Len(t, union, 4)
	assert.Equal(t, []string{"a", "b", "c", "d"}, union)
}
