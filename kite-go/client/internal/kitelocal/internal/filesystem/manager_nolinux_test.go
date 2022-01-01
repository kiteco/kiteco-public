// +build !linux

package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RecursiveWatching(t *testing.T) {
	// reset counts to make this test work with "go test -count ..."
	syncDirCount.Set(0)
	filesCount.Set(0)
	deleteCount.Set(0)
	storeCount.Set(0)
	walkStatCount.Set(0)
	eventsPerGroup.Set(0)

	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)

	// create subdirs
	subdir1 := filepath.Join(tempDir, "subdir_1")
	err = os.Mkdir(subdir1, 0700)
	require.NoError(t, err)
	subdir2 := filepath.Join(tempDir, "subdir_2")
	err = os.Mkdir(subdir2, 0700)
	require.NoError(t, err)

	// create initial files
	file1 := filepath.Join(tempDir, "file_1.py")
	assert.NoError(t, createFile(file1))
	file2 := filepath.Join(subdir1, "file_2.py")
	assert.NoError(t, createFile(file2))

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		IsDirAccepted: func(path string) bool {
			// Default filters remove AppData, which is where temp directories are created
			return true
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// init: start watcher and library walker
	mgr.Initialize(component.InitializerOptions{Platform: &platform.Platform{IsNewInstall: true}})
	mgr.StartWalk()
	defer mgr.Terminate()

	// wait for the walker to finish, no files should be found in the library walk
	waitFor(t, time.Minute, mgr.ReadyChan())
	require.EqualValues(t, 0, len(mgr.Files()), "expected 0 supported library directories to be found during walk of the root dir")

	// for for watcher to init, no files should be found in the library walk
	waitFor(t, time.Minute, mgr.watcher.readyChan)
	require.EqualValues(t, 0, len(mgr.Files()), "expected 0 supported library directories to be found during walk of the root dir")

	// now trigger changes and make sure that the watcher notifies about those
	updatedFile1 := filepath.Join(tempDir, "file_1.py")
	require.NoError(t, updateFile(updatedFile1, "my new content, file1"))
	updatedFile2 := filepath.Join(subdir1, "file_2.py")
	require.NoError(t, updateFile(updatedFile2, "my new content, file2"))

	err = awaitChangeEvents(mgr, updatedFile1, updatedFile2)
	require.NoError(t, err)

	// create unsupported files, these should not trigger changes
	err = createTempFiles(tempDir, 5, "txt")
	require.NoError(t, err)
	err = createTempFiles(tempDir, 5, "js")
	require.NoError(t, err)
	// add support files
	addedFile1 := filepath.Join(tempDir, "newFile_3.py")
	assert.NoError(t, createFile(addedFile1))
	addedFile2 := filepath.Join(subdir2, "newFile_4.py")
	assert.NoError(t, createFile(addedFile2))

	err = awaitChangeEvents(mgr, addedFile1, addedFile2)
	require.NoError(t, err)

	// delete files
	if atomic.LoadInt64(&deleteCount.Value) != 0 {
		require.Failf(t, "expected no removed files", "expected no removed files: %d", atomic.LoadInt64(&deleteCount.Value))
	}

	assert.NoError(t, deleteFile(filepath.Join(tempDir, "file_1.py")))
	assert.NoError(t, deleteFile(addedFile2))

	// wait until we received the events about deletion of the two files above
	err = awaitFileRemovals(10*time.Second, mgr, 2)
	require.NoError(t, err)

	// test final set of files, should be empty
	// number of library directories should be unchanged
	assert.EqualValues(t, 0, len(mgr.Files()))
}
