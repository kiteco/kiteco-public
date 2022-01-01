package webutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseByteOrRuneOffset(t *testing.T) {
	content := []byte("世界 = hello")

	pos, err := ParseByteOrRuneOffset(content, "6", "")
	require.NoError(t, err)
	assert.Equal(t, 6, pos)

	pos, err = ParseByteOrRuneOffset(content, "", "2")
	require.NoError(t, err)
	assert.Equal(t, 6, pos)

	pos, err = ParseByteOrRuneOffset(content, "xyz", "")
	assert.Error(t, err)

	pos, err = ParseByteOrRuneOffset(content, "", "xyz")
	assert.Error(t, err)

	pos, err = ParseByteOrRuneOffset(content, "", "")
	assert.Error(t, err)

	pos, err = ParseByteOrRuneOffset(content, "6", "2")
	assert.Error(t, err)
}
