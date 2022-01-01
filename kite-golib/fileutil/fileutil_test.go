package fileutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReader(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "foo")
	err = ioutil.WriteFile(path, nil, 0777)
	require.NoError(t, err)

	f, err := NewReader(path)
	require.NoError(t, err)
	defer f.Close()
	assert.IsType(t, &os.File{}, f)

	g, err := NewReader(filepath.Join(dir, "bar"))
	assert.Error(t, err)
	assert.Nil(t, g)
}

func TestDownloadedFile(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	asserts := assert.New(t)
	path, err := DownloadedFile(tmpFile.Name())
	asserts.NoError(err)
	asserts.Equal(tmpFile.Name(), path, "a local path should return local path")
}
