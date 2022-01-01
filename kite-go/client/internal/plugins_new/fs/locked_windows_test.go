package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// this tests MoveOrCopy with a locked file in the source folder
// this has to fail and has to trigger a restore of the original source folder
func Test_MoveOrCopyWithLocked(t *testing.T) {
	source, err := ioutil.TempDir("", "kite-copy-source")
	require.NoError(t, err)
	defer os.RemoveAll(source)

	targetParent, err := ioutil.TempDir("", "kite-copy-target")
	require.NoError(t, err)
	defer os.RemoveAll(targetParent)
	target := filepath.Join(targetParent, "target")

	err = createTestFiles(source, []string{"a.txt", "b.txt"})
	require.NoError(t, err)

	err = createTestFiles(source, []string{"locked.txt", "unlocked.txt"})
	require.NoError(t, err)

	lockedFile, err := os.Open(filepath.Join(source, "locked.txt"))
	require.NoError(t, err)
	defer lockedFile.Close()

	err = lockFileEx(lockedFile, lockFileExclusiveLockFlag)
	require.NoError(t, err)
	defer unlockFileEx(lockedFile)

	err = MoveOrCopyDir(source, source)
	require.Error(t, err, "renaming a directory with a locked file inside must fail")
	require.FileExists(t, filepath.Join(source, "locked.txt"))
	require.FileExists(t, filepath.Join(source, "unlocked.txt"))

	err = unlockFileEx(lockedFile)
	require.NoError(t, err)

	err = lockedFile.Close()
	require.NoError(t, err)

	err = MoveOrCopyDir(source, target)
	require.NoError(t, err, "renaming a directory after the file was unlocked has to succeed")
	require.False(t, DirExists(source))
	require.True(t, DirExists(target))
}
