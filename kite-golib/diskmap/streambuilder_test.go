package diskmap

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DiskMap_BuilderAndMap_StreamBuilder(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_diskmap_builderandmap")
	require.Nil(t, err)
	tmpFile, err := ioutil.TempFile(tmpDir, "test.diskmap")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	n := 10
	rep := 100

	builder := NewStreamBuilder(tmpFile)
	for i := 0; i < n; i++ {
		builder.Add(fmt.Sprintf("%d", i), bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep))
	}

	err = builder.Close()
	require.Nil(t, err)
	err = tmpFile.Close()
	require.Nil(t, err)

	dmap, err := NewMap(tmpFile.Name())
	require.Nil(t, err)

	for i := 0; i < n; i++ {
		data, err := dmap.Get(fmt.Sprintf("%d", i))
		require.Nil(t, err, "error getting key")
		require.Equal(t, bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep), data)
	}
}

func Test_DiskMap_NotFound_StreamBuilder(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_diskmap_builderandmap")
	require.Nil(t, err)
	tmpFile, err := ioutil.TempFile(tmpDir, "test.diskmap")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	n := 10
	rep := 10

	builder := NewStreamBuilder(tmpFile)
	for i := 0; i < n; i++ {
		builder.Add(fmt.Sprintf("%d", i), bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep))
	}

	err = builder.Close()
	require.Nil(t, err)
	err = tmpFile.Close()
	require.Nil(t, err)

	dmap, err := NewMap(tmpFile.Name())
	require.Nil(t, err)

	// Check not found case near every key
	for i := 0; i < n+1; i++ {
		// NOTE: there is a space at the end of these keys
		buf, err := dmap.Get(fmt.Sprintf("%d ", i))
		require.Equal(t, ErrNotFound, err)
		require.Nil(t, buf)
	}

	buf, err := dmap.Get("")
	require.Equal(t, ErrNotFound, err)
	require.Nil(t, buf)
}
