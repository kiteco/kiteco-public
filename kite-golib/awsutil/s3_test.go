package awsutil

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func putfile(t *testing.T, path, contents string) {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0777)
	require.NoError(t, err)
	err = ioutil.WriteFile(path, []byte(contents), 0777)
	require.NoError(t, err)
}

func TestCachePath(t *testing.T) {
	asserts := assert.New(t)
	y := url.URL{Scheme: "s3", Host: "kite.com", Path: "foo.txt"}
	asserts.Equal("/var/kite/s3cache/kite.com/foo.txt", CachePath(&y))
}

func TestNewCachedS3Reader_CacheHit(t *testing.T) {
	if !awsTests {
		t.Skip(`Use "go test -aws" to run tests that rely on AWS connectivity`)
	}

	// Point the cache system to a tempdir that for testing purposes
	var err error
	cacheroot, err = ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(cacheroot)

	uri := "s3://kite-data/experiments/testdata/abc.txt"
	cachepath := filepath.Join(cacheroot, "kite-data/experiments/testdata/abc.txt")

	// Place the file in the cache
	putfile(t, cachepath, "abc")
	r, err := NewCachedS3Reader(uri)
	require.NoError(t, err)
	assert.IsType(t, &os.File{}, r)
}

func TestNewCachedS3Reader_CacheMiss(t *testing.T) {
	if !awsTests {
		t.Skip(`Use "go test -aws" to run tests that rely on AWS connectivity`)
	}

	// Point the cache system to a tempdir that for testing purposes
	var err error
	cacheroot, err = ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(cacheroot)

	uri := "s3://kite-data/experiments/testdata/abc.txt"
	cachepath := filepath.Join(cacheroot, "kite-data/experiments/testdata/abc.txt")

	// Place the file in the cache
	putfile(t, cachepath, "xxx")
	r, err := NewCachedS3Reader(uri)
	require.NoError(t, err)
	assert.IsType(t, &lateCopyReader{}, r)
}

func TestNewCachedS3Reader_EndToEnd(t *testing.T) {
	if !awsTests {
		t.Skip(`Use "go test -aws" to run tests that rely on AWS connectivity`)
	}

	uri := "s3://kite-data/experiments/testdata/abc.txt"

	// Point the cache system to a tempdir that for testing purposes
	var err error
	cacheroot, err = ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(cacheroot)

	// Open the file
	r, err := NewCachedS3Reader(uri)
	require.NoError(t, err)
	assert.IsType(t, &lateCopyReader{}, r)

	_, err = ioutil.ReadAll(r)
	require.NoError(t, err)
	err = r.Close()
	require.NoError(t, err)

	// Open the file again - should be cached this time
	r, err = NewCachedS3Reader(uri)
	require.NoError(t, err)
	assert.IsType(t, &os.File{}, r)
}
