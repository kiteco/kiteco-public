package tarball

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Construct a map representation of a filesystem, e.g:
//    foo/bar.txt => "contents of this file"
//    foo/subdir/xx.txt => "other contents of this file"
func mapFromPath(path string) (map[string]string, error) {
	m := make(map[string]string)
	err := filepath.Walk(path, func(subpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relpath, err := filepath.Rel(path, subpath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			m[relpath] = ""
			return nil
		}

		f, err := os.Open(subpath)
		if err != nil {
			return err
		}

		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		m[relpath] = string(contents)
		return nil
	})

	return m, err
}

// Helper to construct a file and all parent directories
func createFile(path string, contents string) error {
	// Create tha parent dir if necessary
	err := os.MkdirAll(filepath.Dir(path), 0777)

	// Only return the error if it's not an "X already exists" error
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Write the data to the file
	return ioutil.WriteFile(path, []byte(contents), 0777)
}

// Check that two paths are either files with the same contents or are directories that
// contain identical files (and subdirectories).
func assertPathsIdentical(t *testing.T, expected, actual string) {
	expectedfs, err := mapFromPath(expected)
	assert.Nil(t, err)

	actualfs, err := mapFromPath(actual)
	assert.Nil(t, err)

	assert.Equal(t, expectedfs, actualfs, "")
}

func TestPlainFile(t *testing.T) {
	// Create a source and target directory
	f, err := ioutil.TempFile("", "")
	assert.Nil(t, err)
	defer f.Close()
	defer os.Remove(f.Name())

	scratchdir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(scratchdir)

	// Write to the file
	_, err = f.WriteString("blah blah blah")
	assert.Nil(t, err)

	// Read into a tarball
	tb, err := PackGzipBytes(f.Name())
	assert.Nil(t, err)

	// Write to an output directory *inside* scratchdir
	dest := filepath.Join(scratchdir, "out")
	err = UnpackGzipBytes(dest, tb)
	assert.Nil(t, err)

	assertPathsIdentical(t, f.Name(), dest)
}

func TestEmptyDir(t *testing.T) {
	// Create a source and target directory
	src, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(src)

	scratchdir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(scratchdir)

	// Read into a tarball
	tb, err := PackGzipBytes(src)
	assert.Nil(t, err)

	// Write to an output directory *inside* scratchdir
	dest := filepath.Join(scratchdir, "out")
	err = UnpackGzipBytes(dest, tb)
	assert.Nil(t, err)

	assertPathsIdentical(t, src, dest)
}

func TestFlatDir(t *testing.T) {
	// Create a source and target directory
	src, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(src)

	scratchdir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(scratchdir)

	// Add some files to the source directory
	err = createFile(filepath.Join(src, "foo"), "foo")
	assert.Nil(t, err)

	err = createFile(filepath.Join(src, "bar"), "ham")
	assert.Nil(t, err)

	err = createFile(filepath.Join(src, "baz"), "spam")
	assert.Nil(t, err)

	// Read into a tarball
	tb, err := PackGzipBytes(src)
	assert.Nil(t, err)

	// Write to an output directory *inside* scratchdir
	dest := filepath.Join(scratchdir, "out")
	err = UnpackGzipBytes(dest, tb)
	assert.Nil(t, err)

	assertPathsIdentical(t, src, dest)
}

func TestMultipleSubdirs(t *testing.T) {
	// Create a source and target directory
	src, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(src)

	scratchdir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(scratchdir)

	// Add some files to the source directory
	err = createFile(filepath.Join(src, "foo"), "foo")
	assert.Nil(t, err)

	err = createFile(filepath.Join(src, "bar/baz"), "ham")
	assert.Nil(t, err)

	err = createFile(filepath.Join(src, "bar/bazz"), "spam")
	assert.Nil(t, err)

	// Read into a tarball
	tb, err := PackGzipBytes(src)
	assert.Nil(t, err)

	// Write to an output directory *inside* scratchdir
	dest := filepath.Join(scratchdir, "out")
	err = UnpackGzipBytes(dest, tb)
	assert.Nil(t, err)

	assertPathsIdentical(t, src, dest)
}

func TestSymlink(t *testing.T) {
	// Create a temporary directory
	src, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(src)

	// Create a symlink within that dir
	os.Symlink("foo", filepath.Join(src, "a_symbolic_link"))

	// Attempt to pack into tarball
	_, err = PackGzipBytes(src)
	assert.Error(t, err, "Expected PackGzipBytes to fail on symlinks.")
}

// TestWalk tests Walk and Visitor functionality
func TestWalk(t *testing.T) {
	src, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	defer os.RemoveAll(src)

	// Add some files to the source directory
	files := map[string]string{
		"foo":      "foo",
		"bar/baz":  "ham",
		"bar/bazz": "spam",
	}

	dirs := make(map[string]struct{})
	for fn, contents := range files {
		dirs[filepath.Dir(fn)] = struct{}{}
		err := createFile(filepath.Join(src, fn), contents)
		assert.Nil(t, err)
	}

	buf, err := PackGzipBytes(src)
	assert.Nil(t, err)

	seen := make(map[string]struct{})

	visitorFn := func(header *tar.Header, r io.Reader) error {
		switch header.Typeflag {
		case tar.TypeDir:
			_, exists := dirs[header.Name]
			if !exists && header.Name != "." && header.Name != ".." {
				return fmt.Errorf("got unexpected directory: %s", header.Name)
			}
		case tar.TypeReg:
			contents, exists := files[header.Name]
			if !exists {
				return fmt.Errorf("got unexpected filename: %s", header.Name)
			}

			buf, err := ioutil.ReadAll(r)
			if err != nil {
				return err
			}

			if contents != string(buf) {
				return fmt.Errorf("content mismatch: expected %s, got %s", contents, string(buf))
			}

			seen[header.Name] = struct{}{}
		}
		return nil
	}

	gzipReader, err := gzip.NewReader(bytes.NewBuffer(buf))
	assert.Nil(t, err)

	err = Walk(gzipReader, visitorFn)
	assert.Nil(t, err)

	for fn := range files {
		_, exists := seen[fn]
		assert.True(t, exists, "expected to find filename: "+fn)
	}
}
