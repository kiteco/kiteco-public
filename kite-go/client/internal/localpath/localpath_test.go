package localpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyPath(t *testing.T) {
	path, err := ToUnix("")
	assert.NoError(t, err)
	assert.Equal(t, "", path)
}
