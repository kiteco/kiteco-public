package pythonindex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyString(t *testing.T) {
	sa := suffixArray{}
	response := sa.prefixedBy("")

	assert.Empty(t, response)
}
