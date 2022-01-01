package templateset

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Implements http.File
type mockFile struct {
	r io.Reader
}

func (f mockFile) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f mockFile) Close() error {
	return nil
}

func (f mockFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f mockFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (f mockFile) Stat() (os.FileInfo, error) {
	return nil, nil
}

// Implements http.FileSystem
type mockFileSystem map[string]string

func (fs mockFileSystem) Open(name string) (http.File, error) {
	if data, ok := fs[name]; ok {
		return mockFile{bytes.NewBufferString(data)}, nil
	}
	return nil, fmt.Errorf("no file named %s", name)
}

func TestRender(t *testing.T) {
	mockfs := mockFileSystem{
		"templates/abc": "xyz",
	}

	tset := NewSet(mockfs, "templates", nil)

	var w bytes.Buffer
	err := tset.Render(&w, "abc", nil)
	require.NoError(t, err)
	assert.Equal(t, w.String(), "xyz")
}

func TestRenderWithPayload(t *testing.T) {
	mockfs := mockFileSystem{
		"templates/abc": "hello {{.name}}",
	}

	tset := NewSet(mockfs, "templates", nil)

	payload := map[string]string{
		"name": "bob",
	}

	var w bytes.Buffer
	err := tset.Render(&w, "abc", payload)
	require.NoError(t, err)
	assert.Equal(t, w.String(), "hello bob")
}

func TestRenderNoTemplateDir(t *testing.T) {
	mockfs := mockFileSystem{
		"abc": "xyz",
	}

	tset := NewSet(mockfs, "", nil)

	var w bytes.Buffer
	err := tset.Render(&w, "abc", nil)
	require.NoError(t, err)
	assert.Equal(t, w.String(), "xyz")
}

func TestNonexistentTemplate(t *testing.T) {
	mockfs := mockFileSystem{}
	tset := NewSet(mockfs, "", nil)

	var w bytes.Buffer
	err := tset.Render(&w, "abc", nil)
	assert.Error(t, err)
}

func TestMalformedTemplate(t *testing.T) {
	mockfs := mockFileSystem{
		"templates/abc": "{{malformed",
	}

	tset := NewSet(mockfs, "templates", nil)

	var w bytes.Buffer
	err := tset.Render(&w, "abc", nil)
	require.Error(t, err)
}
