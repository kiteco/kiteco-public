// +build !windows

package localpath

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertColonToNative(t *testing.T, expected, input string) {
	actual, err := ColonToNative(input)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestColonToNative(t *testing.T) {
	assertColonToNative(t, "/Users/alex/test.py", ":Users:alex:test.py")
	assertColonToNative(t, "/Users/alex/", ":Users:alex:")
}

func TestIsRootDir(t *testing.T) {
	assert.Equal(t, true, IsRootDir("/"))
	assert.Equal(t, false, IsRootDir("/path"))
}

func TestExpandTilde(t *testing.T) {
	p := ExpandTilde("~/file.py")
	assert.Equal(t, fmt.Sprintf("%s/file.py", os.ExpandEnv("$HOME")), p)

	p = ExpandTilde("/home/user/file.py")
	assert.Equal(t, "/home/user/file.py", p)
}

func TestFromUnix(t *testing.T) {
	p, err := FromUnix("/home/user/file.py")
	assert.NoError(t, err)
	assert.Equal(t, "/home/user/file.py", p)
}

func TestRelativePath(t *testing.T) {
	_, err := ToUnix("Relative/Path")
	assert.Error(t, err)
}

func TestAbsolutePath(t *testing.T) {
	path, err := ToUnix("/Absolute/Path")
	assert.NoError(t, err)
	assert.Equal(t, "/Absolute/Path", path)
}
