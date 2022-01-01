package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMoveOrCopy(t *testing.T) {
	// create two temp dirs, remove some files from one of them and restore it
	// at the end, both have to contain the same data

	source, err := ioutil.TempDir("", "kite-copy-source")
	require.NoError(t, err)
	defer os.RemoveAll(source)

	targetParent, err := ioutil.TempDir("", "kite-copy-target")
	require.NoError(t, err)
	defer os.RemoveAll(targetParent)

	target := filepath.Join(targetParent, "target")

	// create files and subdirs in source
	filenames := []string{"a", "b", "c", "d", ".e"}
	err = createTestFiles(source, filenames)
	require.NoError(t, err)
	err = createTestFiles(filepath.Join(source, "sub1"), filenames)
	require.NoError(t, err)
	err = createTestFiles(filepath.Join(source, "sub2"), filenames)
	require.NoError(t, err)

	err = MoveOrCopyDir(source, target)
	require.NoError(t, err)

	// files in subdirs have to exist now in target
	require.True(t, FileExists(filepath.Join(target, "a")))
	require.True(t, FileExists(filepath.Join(target, "b")))
	require.True(t, FileExists(filepath.Join(target, "c")))
	require.True(t, FileExists(filepath.Join(target, "d")))
	require.True(t, FileExists(filepath.Join(target, ".e")))

	require.True(t, FileExists(filepath.Join(target, "sub1", "a")))
	require.True(t, FileExists(filepath.Join(target, "sub1", "b")))
	require.True(t, FileExists(filepath.Join(target, "sub1", "c")))
	require.True(t, FileExists(filepath.Join(target, "sub1", "d")))
	require.True(t, FileExists(filepath.Join(target, "sub1", ".e")))

	require.True(t, FileExists(filepath.Join(target, "sub2", "a")))
	require.True(t, FileExists(filepath.Join(target, "sub2", "b")))
	require.True(t, FileExists(filepath.Join(target, "sub2", "c")))
	require.True(t, FileExists(filepath.Join(target, "sub2", "d")))
	require.True(t, FileExists(filepath.Join(target, "sub2", ".e")))
}

func TestCopyDir(t *testing.T) {
	source, err := ioutil.TempDir("", "kite-copy-source")
	require.NoError(t, err)
	defer os.RemoveAll(source)

	err = createTestFiles(source, []string{"a.txt", ".hidden.txt"})
	require.NoError(t, err)

	targetParent, err := ioutil.TempDir("", "kite-copy-target")
	require.NoError(t, err)
	defer os.RemoveAll(targetParent)

	target := filepath.Join(targetParent, "target")
	err = CopyDir(source, target)
	require.NoError(t, err)
}

func TestCopySubDir(t *testing.T) {
	source, err := ioutil.TempDir("", "kite-copy-source")
	require.NoError(t, err)
	defer os.RemoveAll(source)

	subdirPath := filepath.Join(source, "subdir1", "subdir1.1")
	err = os.MkdirAll(subdirPath, 0700)
	require.NoError(t, err)

	err = createTestFiles(subdirPath, []string{"a.txt", "b.txt"})
	require.NoError(t, err)

	targetParent, err := ioutil.TempDir("", "kite-copy-target")
	require.NoError(t, err)
	defer os.RemoveAll(targetParent)

	target := filepath.Join(targetParent, "target")
	err = CopyDir(source, target)
	require.NoError(t, err)

	require.True(t, FileExists(filepath.Join(target, "subdir1", "subdir1.1", "a.txt")))
	require.True(t, FileExists(filepath.Join(target, "subdir1", "subdir1.1", "b.txt")))
}

func createTestFiles(parentDir string, filenames []string) error {
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return err
	}

	for _, f := range filenames {
		if err := ioutil.WriteFile(filepath.Join(parentDir, f), []byte(f), 0600); err != nil {
			return err
		}
	}
	return nil
}
