// +build windows

package permissions

import (
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SanitizePath(t *testing.T) {
	//make sure that it's lowercased
	path, err := sanitizePath(`:windows:c:USERS:userName:test.PY`)
	assert.NoError(t, err)
	assert.Equal(t, `c:\users\username\test.py`, path)

	//tilde expansion is not done on Windows. Paths with a tilde are not parsed as absolute paths and result in an error
	path, err = sanitizePath("~/TEST.py")
	assert.EqualError(t, err, "non absolute path ~/TEST.py")
}

func Test_SanitizePaths(t *testing.T) {
	u, err := user.Current()
	require.NoError(t, err)
	home := u.HomeDir

	paths, err := sanitizePaths(filepath.Join(home, "test.py"), filepath.Join(home, "test2.py"), filepath.Join(home, "test/"))
	require.NoError(t, err)
	require.Len(t, paths, 3)

	homeLowercased := strings.ToLower(u.HomeDir)
	assert.Equal(t, filepath.Join(homeLowercased, "test.py"), paths[0])
	assert.Equal(t, filepath.Join(homeLowercased, "test2.py"), paths[1])
	assert.Equal(t, filepath.Join(homeLowercased, "test/"), paths[2])

	paths, err = sanitizePaths("~/test.py")
	require.EqualError(t, err, "error sanitizing path ~/test.py: non absolute path ~/test.py")
}
