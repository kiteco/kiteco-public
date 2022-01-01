package shared

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SnapPath(t *testing.T) {
	// If non-snap dir, should return original path
	path, err := SnapPath("/foo/bar", "foo", "bar")
	require.NoError(t, err)
	assert.EqualValues(t, path, "/foo/bar")

	dir, cleanup := SetupTempDir(t, "")
	defer cleanup()

	snapDir = dir
	expectedPath := filepath.Join(dir, "testSnap", "current", "testLocation")

	err = os.MkdirAll(filepath.Dir(expectedPath), 0700)
	require.NoError(t, err)
	err = ioutil.WriteFile(expectedPath, []byte(""), 0700)
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Dir(expectedPath))

	testPath := filepath.Join(dir, "bin", "testSnap") // ${snapDir}/bin/testSnap
	path, err = SnapPath(testPath, "testSnap", "testLocation")
	require.NoError(t, err)
	assert.EqualValues(t, expectedPath, path)
}
