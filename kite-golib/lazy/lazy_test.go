package lazy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_LoadError(t *testing.T) {
	var loadCount int
	loadErr := fmt.Errorf("some load error")

	load := func() error {
		loadCount++
		return loadErr
	}
	unload := func() {}

	loader := NewLoader(load, unload)

	err := loader.LoadAndLock()
	require.Error(t, err)
	require.Equal(t, loadErr, err)
	require.Equal(t, 1, loadCount)

	err = loader.LoadAndLock()
	require.Error(t, err)
	require.Equal(t, loadErr, err)
	require.Equal(t, 1, loadCount)

	loader.Unload()

	err = loader.LoadAndLock()
	require.Error(t, err)
	require.Equal(t, loadErr, err)
	require.Equal(t, 2, loadCount)
}
