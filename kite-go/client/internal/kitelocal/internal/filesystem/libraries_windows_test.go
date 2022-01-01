// +build windows

package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FindMatches(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	matching := filepath.Join(tempDir, "venv", "Lib", "site-packages")
	os.MkdirAll(matching, os.ModePerm)
	other := filepath.Join(tempDir, "venv", "lib", "pyython3.4", "other")
	os.MkdirAll(other, os.ModePerm)

	patterns := virtualEnvLibraryPatterns[runtime.GOOS]
	paths := findMatchingPaths([]string{tempDir}, patterns)
	assert.Len(t, paths, 1, "expected one matching path")
	assert.Equal(t, paths[0], matching)

	mgr := NewLibraryManager("", "", nil)
	mgr.AddProject([]string{tempDir})
	canonPath, err := canonicalizePath(matching)
	require.NoError(t, err)
	obj, ok := mgr.dirs.Load(canonPath)
	assert.True(t, ok)
	assert.Equal(t, obj.(LibraryType), VirtualEnvMatch)
	canonPath, err = canonicalizePath(other)
	require.NoError(t, err)
	_, ok = mgr.dirs.Load(canonPath)
	assert.False(t, ok)
}
