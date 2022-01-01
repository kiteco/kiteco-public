package localpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertToUnix(t *testing.T, expected, input string) {
	actual, err := ToUnix(input)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func assertFromUnix(t *testing.T, expected, input string) {
	actual, err := FromUnix(input)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func assertFromUnixError(t *testing.T, input string) {
	_, err := FromUnix(input)
	assert.Error(t, err)
}

func assertColonToNative(t *testing.T, expected, input string) {
	actual, err := ColonToNative(input)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestToUnix(t *testing.T) {
	assertToUnix(t, `/windows/C/dir/file`, `C:\dir\file`)
	assertToUnix(t, `/windows/x/a/b/c/d.py`, `x:\a\b\c\d.py`)
	assertToUnix(t, `/windows/a/`, `a:/`)
	assertToUnix(t, `/windows/unc/foo/bar/baz`, `\\foo\bar\baz`)
}

func TestFromUnix(t *testing.T) {
	assertFromUnix(t, `C:\dir\file`, `/windows/C/dir/file`)
	assertFromUnix(t, `x:\a\b\c\d.py`, `/windows/x/a/b/c/d.py`)
	assertFromUnix(t, `a:\`, `/windows/a/`)
	assertFromUnix(t, `\\foo\bar\baz`, `/windows/unc/foo/bar/baz`)

	assertFromUnix(t, `abc:\xyz`, `/windows/abc/xyz`)
	assertFromUnixError(t, ``)
	assertFromUnixError(t, `/`)
	assertFromUnixError(t, `/windows`)
	assertFromUnixError(t, `/windowsblah`)
	assertFromUnixError(t, `/windows//xxx`)
}

func TestColonToNative(t *testing.T) {
	assertColonToNative(t, `C:\Users\account1\Documents\foo.py`, ":windows:C:Users:account1:Documents:foo.py")
}

func TestIsRootDir(t *testing.T) {
	assert.Equal(t, true, IsRootDir(`C:\`))
	assert.Equal(t, true, IsRootDir(`\\host\share`))
	assert.Equal(t, true, IsRootDir(`\\host\share\`))
	assert.Equal(t, false, IsRootDir(`C:\path`))
	assert.Equal(t, false, IsRootDir(`\\host\share\path`))
}

func TestRelativePath(t *testing.T) {
	_, err := ToUnix(`.\Relative\Path`)
	assert.Error(t, err)
}

func TestAbsolutePath(t *testing.T) {
	path, err := ToUnix(`c:\Absolute\Path`)
	assert.NoError(t, err)
	assert.Equal(t, `/windows/c/Absolute/Path`, path)
}
