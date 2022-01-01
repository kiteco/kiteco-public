package awsutil

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error {
	return nil
}

type errorReadCloser struct {
}

func (errorReadCloser) Read(buf []byte) (int, error) {
	return 0, errors.New("mock error")
}

func (errorReadCloser) Close() error {
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestCopyingReader_Normal(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(dir)

	require.NoError(t, err)
	path := filepath.Join(dir, "test")

	r := nopCloser{bytes.NewBufferString("test")}
	copyr, err := newLateCopyReader(r, path, dir, nil)
	require.NoError(t, err)
	temp := copyr.temp.Name()

	assert.False(t, fileExists(path))

	data, err := ioutil.ReadAll(copyr)
	require.NoError(t, err)
	assert.Equal(t, "test", string(data))

	assert.True(t, fileExists(path))

	err = copyr.Close()
	require.NoError(t, err)

	assert.True(t, fileExists(path))

	cachedata, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "test", string(cachedata))
	assert.False(t, fileExists(temp))
}

func TestCopyingReader_NothingRead(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(dir)

	require.NoError(t, err)
	path := filepath.Join(dir, "test")

	r := nopCloser{bytes.NewBufferString("test")}
	copyr, err := newLateCopyReader(r, path, dir, nil)
	require.NoError(t, err)
	temp := copyr.temp.Name()

	assert.False(t, fileExists(path))

	err = copyr.Close()
	require.NoError(t, err)

	assert.False(t, fileExists(path))
	assert.False(t, fileExists(temp))
}

func TestCopyingReader_IncompleteRead(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(dir)

	require.NoError(t, err)
	path := filepath.Join(dir, "test")

	r := nopCloser{bytes.NewBufferString("test")}
	copyr, err := newLateCopyReader(r, path, dir, nil)
	require.NoError(t, err)
	temp := copyr.temp.Name()

	buf := make([]byte, 2) // not enough space to hold full data
	n, err := r.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, n, 2)

	assert.False(t, fileExists(path))

	err = copyr.Close()
	require.NoError(t, err)

	assert.False(t, fileExists(path))
	assert.False(t, fileExists(temp))
}

func TestCopyingReader_ErrorOnRead(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(dir)

	require.NoError(t, err)
	path := filepath.Join(dir, "test")

	var r errorReadCloser
	copyr, err := newLateCopyReader(r, path, dir, nil)
	require.NoError(t, err)
	temp := copyr.temp.Name()

	_, err = ioutil.ReadAll(copyr)
	require.Error(t, err)

	assert.False(t, fileExists(path))

	err = copyr.Close()
	require.NoError(t, err)

	assert.False(t, fileExists(path))
	assert.False(t, fileExists(temp))
}

func TestCopyingReader_DestinationAlreadyExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(dir)

	require.NoError(t, err)
	path := filepath.Join(dir, "test")

	// Create destination path
	f, err := os.Create(path)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	// Create the lateCopyReader
	r := nopCloser{bytes.NewBufferString("test")}
	copyr, err := newLateCopyReader(r, path, dir, nil)
	require.NoError(t, err)
	temp := copyr.temp.Name()

	// Read the stream all the way to EOF
	data, err := ioutil.ReadAll(copyr)
	require.NoError(t, err)
	assert.Equal(t, "test", string(data))

	err = copyr.Close()
	require.NoError(t, err)

	// Check that the file has the expected contents
	assert.True(t, fileExists(path))
	cachedata, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "test", string(cachedata))
	assert.False(t, fileExists(temp))
}

func TestTryCache_Hit(t *testing.T) {
	if !awsTests {
		t.Skip(`Use "go test -aws" to run tests that rely on AWS connectivity`)
	}

	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = f.WriteString("abc")
	require.NoError(t, err)
	f.Close()

	localpath := f.Name()
	s3url, err := ValidateURI("s3://kite-data/experiments/testdata/abc.txt")
	require.NoError(t, err)

	etag, err := checksumS3URL(s3url)
	require.NoError(t, err)

	r, err := tryCache(etag, localpath)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestTryCache_Miss(t *testing.T) {
	if !awsTests {
		t.Skip(`Use "go test -aws" to run tests that rely on AWS connectivity`)
	}

	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = f.WriteString("xxx")
	require.NoError(t, err)
	f.Close()

	localpath := f.Name()
	s3url, err := ValidateURI("s3://kite-data/experiments/testdata/abc.txt")
	require.NoError(t, err)

	etag, err := checksumS3URL(s3url)
	require.NoError(t, err)

	r, _ := tryCache(etag, localpath)
	assert.Nil(t, r)
}
