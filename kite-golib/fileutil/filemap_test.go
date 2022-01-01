package fileutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFileMap(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	w := NewFileMapWriter()
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("%d.txt", i)
		content := bytes.Repeat([]byte(fmt.Sprintf("content%d", i)), 100)
		buf := bytes.NewReader(content)
		err := w.AddFile(path, buf)
		require.NoError(t, err)
	}

	dataFile, err := ioutil.TempFile(dir, "data")
	require.NoError(t, err)
	dataPath := dataFile.Name()
	err = w.WriteData(dataFile)
	require.NoError(t, err)
	err = dataFile.Close()
	require.NoError(t, err)

	offsetFile, err := ioutil.TempFile(dir, "offsets")
	require.NoError(t, err)
	offsetPath := offsetFile.Name()
	err = w.WriteOffsets(offsetFile)
	require.NoError(t, err)
	err = offsetFile.Close()
	require.NoError(t, err)

	fm, err := NewTestFileMap(dataPath, offsetPath)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		path := fmt.Sprintf("%d.txt", i)
		expectedContent := bytes.Repeat([]byte(fmt.Sprintf("content%d", i)), 100)

		r, err := NewFileMapReader(path, fm)
		require.NoError(t, err)
		buf, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, expectedContent, buf)
	}
}
