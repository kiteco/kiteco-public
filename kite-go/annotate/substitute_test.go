package annotate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstitute(t *testing.T) {
	x := "abc $def$ ghi $jkl$"
	y := substitute(x, map[string]string{
		"def": "xxx",
		"jkl": "yyy",
	})
	assert.Equal(t, "abc xxx ghi yyy", y)
}
