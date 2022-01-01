package pythonimports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCachedBuiltins(t *testing.T) {
	for _, v := range mockBuiltins {
		assert.NotEqual(t, None, v)
	}
}
