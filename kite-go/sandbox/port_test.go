package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnusedPort(t *testing.T) {
	p, err := UnusedPort()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, p)
}
