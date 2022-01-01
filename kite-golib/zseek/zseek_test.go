package zseek

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ZSeek(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_diskmap_builderandmap")
	require.Nil(t, err)
	tmpFile := filepath.Join(tmpDir, "test.diskmap")
	defer os.RemoveAll(tmpDir)
	f, err := os.Create(tmpFile)
	require.Nil(t, err)

	allbuff := bytes.Buffer{}
	zwriter := NewWriterSize(f, 128)

	// Use MultiWriter to copy full contents to allbuff for a
	// full buffer read check later.
	w := io.MultiWriter(zwriter, &allbuff)

	n := 100
	rep := 100

	var off int64
	offsetMap := make(map[int64][]byte)

	// Write a bunch of data, keeping track of what we expect at each offset
	for i := 0; i < n; i++ {
		data := bytes.Repeat([]byte(fmt.Sprintf("hello world %d", i)), rep)
		n, err := w.Write(data)
		require.Nil(t, err)

		offsetMap[off] = data
		off += int64(n)
	}

	// Close zseek writer
	err = zwriter.Close()
	require.Nil(t, err)

	// Close file
	err = f.Close()
	require.Nil(t, err)

	// Open file
	in, err := os.Open(tmpFile)
	require.Nil(t, err)
	defer in.Close()

	r, err := NewReader(in)
	require.Nil(t, err)

	// Try seeking releative to start
	for offset, data := range offsetMap {
		_, err := r.Seek(offset, io.SeekStart)
		require.Nil(t, err)

		var test bytes.Buffer
		_, err = io.CopyN(&test, r, int64(len(data)))
		require.Nil(t, err)

		require.Equal(t, data, test.Bytes())
	}

	// Try seeking relative to end
	for offset, data := range offsetMap {
		_, err := r.Seek(-(off - offset), io.SeekEnd)
		require.Nil(t, err)

		var test bytes.Buffer
		_, err = io.CopyN(&test, r, int64(len(data)))
		require.Nil(t, err)

		require.Equal(t, data, test.Bytes())
	}

	_, err = r.Seek(0, io.SeekStart)
	require.Nil(t, err)

	// Try seeking relative to current
	var lastOffset int64
	for offset, data := range offsetMap {
		off := offset - lastOffset

		var err error
		lastOffset, err = r.Seek(off, io.SeekCurrent)
		require.Nil(t, err)

		var test bytes.Buffer
		n, err := io.CopyN(&test, r, int64(len(data)))
		require.Nil(t, err)
		lastOffset += int64(n)

		require.Equal(t, data, test.Bytes())
	}

	_, err = r.Seek(0, io.SeekStart)
	require.Nil(t, err)

	// Read the whole buffer
	all, err := ioutil.ReadAll(r)
	require.Nil(t, err)
	require.Equal(t, allbuff.Bytes(), all)
}
