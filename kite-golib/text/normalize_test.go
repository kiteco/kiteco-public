package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	text := "numpy.array Construct an array [of `float`s]    "
	text = Normalize(text)

	assert.Equal(t, "numpy array Construct an array of float", text)
}
