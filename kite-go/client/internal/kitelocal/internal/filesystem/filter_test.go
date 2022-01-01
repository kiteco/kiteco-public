package filesystem

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-golib/filters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func transformPath(t *testing.T, path string) string {
	unixPath, err := localpath.ToUnix(path)
	require.NoError(t, err)
	localPath, err := localpath.FromUnix(unixPath)
	require.NoError(t, err)
	return localPath
}

func testIsFilteredDir(t *testing.T, operatingSystem, path string, expected bool) {
	filtered := filters.IsFilteredDir(operatingSystem, path)
	assert.Equal(t, filtered, expected)
	filtered = filters.IsFilteredDir(operatingSystem, transformPath(t, path))
	assert.Equal(t, filtered, expected)
}
