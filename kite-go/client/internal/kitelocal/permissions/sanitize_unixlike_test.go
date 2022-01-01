// +build !windows

package permissions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SanitizePath(t *testing.T) {
	path, err := sanitizePath("/home/user/test.py")
	assert.NoError(t, err)
	assert.Equal(t, "/home/user/test.py", path)

	path, err = sanitizePath("~/test.py")
	assert.NoError(t, err)
	//the tilde must have been expanded, this is not OS specific
	assert.NotEqual(t, "~/test.py", path)
}

func Test_SanitizePaths(t *testing.T) {
	paths, err := sanitizePaths("/home/user/test.py", "/home/user/test2.py", "/home/user/test/")
	assert.NoError(t, err)
	assert.Len(t, paths, 3)
	assert.Equal(t, "/home/user/test.py", paths[0])
	assert.Equal(t, "/home/user/test2.py", paths[1])
	assert.Equal(t, "/home/user/test/", paths[2])

	paths, err = sanitizePaths("~/test.py")
	assert.NoError(t, err)
	//the tilde must have been expanded, this is not OS specific
	assert.NotEqual(t, "~/test.py", paths[0])
}
