package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnabled(t *testing.T) {
	require.True(t, isVscodeEnabled())
}
