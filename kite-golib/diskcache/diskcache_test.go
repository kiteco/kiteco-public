package diskcache

import (
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var opts = Options{
	MaxSize:         100,
	BytesUntilFlush: 10,
}

func assertCacheContents(t *testing.T, dir string, filenames ...string) {
	entries, err := ioutil.ReadDir(dir)
	require.NoError(t, err)

	var expected []string
	for _, f := range filenames {
		expected = append(expected, hash([]byte(f)))
	}

	var actual []string
	for _, entry := range entries {
		actual = append(actual, entry.Name())
	}

	sort.Strings(expected)
	sort.Strings(actual)
	assert.EqualValues(t, expected, actual)
}

func TestPutGetExists(t *testing.T) {
	c, err := OpenTemp(opts)
	require.NoError(t, err)
	defer os.RemoveAll(c.Path)

	assert.False(t, c.Exists([]byte("foo")))

	err = c.Put([]byte("foo"), []byte("bar"))
	require.NoError(t, err)

	assert.True(t, c.Exists([]byte("foo")))

	val, err := c.Get([]byte("foo"))
	require.NoError(t, err)
	assert.Equal(t, "bar", string(val))
}

func TestGetReader(t *testing.T) {
	c, err := OpenTemp(opts)
	require.NoError(t, err)
	defer os.RemoveAll(c.Path)

	err = c.Put([]byte("foo"), []byte("bar"))
	require.NoError(t, err)

	r, err := c.GetReader([]byte("foo"))
	require.NoError(t, err)
	val, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "bar", string(val))
}

func TestLRU(t *testing.T) {
	c, err := OpenTemp(Options{
		MaxSize:         10,
		BytesUntilFlush: 10,
	})
	require.NoError(t, err)
	defer os.RemoveAll(c.Path)

	err = c.Put([]byte("foo"), []byte("1234"))
	err = c.Put([]byte("bar"), []byte("1234"))
	err = c.Put([]byte("baz"), []byte("1234"))
	require.NoError(t, err)

	assertCacheContents(t, c.Path, "bar", "baz")
}
