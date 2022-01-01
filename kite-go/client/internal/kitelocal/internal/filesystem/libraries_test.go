package filesystem

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseAnacondaEnvs(t *testing.T) {
	output := `
	# conda environments:
#
base                     //anaconda3
myenv                 *  //anaconda3/envs/myenv
                         /Users/hrysoula/Documents/venv
                      `
	actual := []string{
		"//anaconda3",
		"//anaconda3/envs/myenv",
		"/Users/hrysoula/Documents/venv",
	}
	found := parseAnacondaEnvs(output)
	assert.Equal(t, len(actual), len(found))
	for i, val := range actual {
		assert.Equal(t, val, found[i])
	}
}

func Test_KiteLibraries(t *testing.T) {
	usr, err := user.Current()
	require.NoError(t, err)

	kiteDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)
	defer os.RemoveAll(kiteDir)

	// add symlinks to libraries directory
	tmpDir, err := ioutil.TempDir("", "test-libraries")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	lib := filepath.Join(tmpDir, "foo")
	err = os.Mkdir(lib, os.ModePerm)
	require.NoError(t, err)
	kiteLibs := filepath.Join(kiteDir, "libraries")
	err = os.Mkdir(kiteLibs, os.ModePerm)
	require.NoError(t, err)
	linkPath := filepath.Join(kiteLibs, "foo")
	err = os.Symlink(lib, linkPath)
	require.NoError(t, err)

	m := NewLibraryManager(usr.HomeDir, kiteDir, nil)

	// check that the symlinked library is found
	// note: temp directories can be symlinks,
	// so evaluate symlinks prior to comparison
	linkPath, err = filepath.EvalSymlinks(lib)
	require.NoError(t, err)
	absPath, err := filepath.Abs(linkPath)
	require.NoError(t, err)
	canonDir, err := canonicalizePath(absPath)
	require.NoError(t, err)
	var found bool
	for _, dir := range m.Dirs() {
		// parent directory of symlink is added to the known libraries
		if dir == canonDir {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func Test_SysPathLibraries(t *testing.T) {
	usr, err := user.Current()
	require.NoError(t, err)

	kiteDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)
	defer os.RemoveAll(kiteDir)

	m := NewLibraryManager(usr.HomeDir, kiteDir, nil)
	m.sysPathLibs()

	dirs := m.Dirs()
	found := false
	for _, dir := range dirs {
		if dir == "" {
			found = true
			break
		}
	}
	assert.False(t, found, "syspath libraries should not include empty string")
}
