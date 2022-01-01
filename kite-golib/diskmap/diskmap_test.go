package diskmap

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DiskMap_BuilderAndMap(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_diskmap_builderandmap")
	require.Nil(t, err)
	tmpFile := filepath.Join(tmpDir, "test.diskmap")
	defer os.RemoveAll(tmpDir)

	n := 10
	rep := 100

	builder := NewBuilder()
	for i := 0; i < n; i++ {
		builder.Add(fmt.Sprintf("%d", i), bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep))
	}

	err = builder.WriteToFile(tmpFile)
	require.Nil(t, err)

	dmap, err := NewMap(tmpFile)
	require.Nil(t, err)

	for i := 0; i < n; i++ {
		data, err := dmap.Get(fmt.Sprintf("%d", i))
		require.Nil(t, err, "error getting key")
		require.Equal(t, bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep), data)
	}
}
func Test_DiskMap_Keys(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_diskmap_builderandmap")
	require.Nil(t, err)
	tmpFile := filepath.Join(tmpDir, "test.diskmap")
	defer os.RemoveAll(tmpDir)

	n := 10
	rep := 100

	var keysAdded []string
	builder := NewBuilder()
	for i := 0; i < n; i++ {
		builder.Add(fmt.Sprintf("%d", i), bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep))
		keysAdded = append(keysAdded, fmt.Sprintf("%d", i))
	}

	err = builder.WriteToFile(tmpFile)
	require.Nil(t, err)

	dmap, err := NewMap(tmpFile)
	require.Nil(t, err)

	keys, err := dmap.Keys()
	require.Nil(t, err)
	require.Equal(t, keysAdded, keys)
}

func Test_DiskMap_NotFound(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_diskmap_notfound")
	require.Nil(t, err)
	tmpFile := filepath.Join(tmpDir, "test.diskmap")
	defer os.RemoveAll(tmpDir)

	n := 10
	rep := 10

	builder := NewBuilder()
	for i := 0; i < n; i++ {
		builder.Add(fmt.Sprintf("%d", i), bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep))
	}

	err = builder.WriteToFile(tmpFile)
	require.Nil(t, err)

	dmap, err := NewMap(tmpFile)
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
