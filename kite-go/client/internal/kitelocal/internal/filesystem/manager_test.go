package filesystem

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/client/readdir"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component(t *testing.T) {
	m := &Manager{}
	component.TestImplements(t, m, component.Implements{
		Initializer: true,
		Terminater:  true,
	})
}

func Test_WatcherEmptyRootDir(t *testing.T) {
	mgr := NewManager(Options{
		RootDir: "",
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
	mgr.Initialize(component.InitializerOptions{Platform: &platform.Platform{IsNewInstall: false}})
	defer mgr.Terminate()

	// on linux, we only watch directories that we explicitly add to the watcher
	var expectedWatchCount int
	if runtime.GOOS != "linux" {
		expectedWatchCount = 1
	}
	assert.EqualValues(t, expectedWatchCount, mgr.watcher.watcherFS.WatchCount())
}

func Test_WalkDir(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

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

	// Create library directories, 11 in total

	// 10 temp files at toplevel
	err = createTempFiles(tempDir, 10, "py")
	require.NoError(t, err)

	// 10 files in library dir
	distPkg := filepath.Join(tempDir, "dist-packages")
	err = os.Mkdir(distPkg, 0700)
	require.NoError(t, err)
	err = createTempFiles(distPkg, 10, "py")
	require.NoError(t, err)

	// 10 files each in 10 library subdirs, 10 files each in 10 non-library subdirs
	for i := 0; i < 10; i++ {
		subdir := filepath.Join(tempDir, fmt.Sprintf("subdir_%d", i))
		err := os.Mkdir(subdir, 0700)
		require.NoError(t, err)
		libdir := filepath.Join(subdir, "site-packages")
		err = os.Mkdir(libdir, 0700)
		require.NoError(t, err)
		err = createTempFiles(libdir, 10, "py")
		require.NoError(t, err)

		nonlibdir := filepath.Join(subdir, "non-site-packages")
		err = os.Mkdir(nonlibdir, 0700)
		require.NoError(t, err)
		err = createTempFiles(nonlibdir, 10, "py")
		require.NoError(t, err)

		err = createTempFiles(subdir, 5, "txt")
		require.NoError(t, err)
	}

	// init: start watcher and library walker
	mgr.Initialize(component.InitializerOptions{Platform: &platform.Platform{IsNewInstall: false}})
	defer mgr.Terminate()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	select {
	case <-ctx.Done():
		require.FailNow(t, "timeout while waiting for fs walk to finish")
	case <-mgr.ReadyChan():
		// files contains the library directories found
		files := mgr.Files()
		require.EqualValues(t, 11, len(files), "expected 11 supported library directories to be found during walk of the root dir")
	}

	select {
	case <-mgr.ReadyChan():
		break
	default:
		require.FailNow(t, "reading from a closed ReadyChan must succeed")
	}

	// number of library directories should be unchanged
	files := mgr.Files()
	assert.EqualValues(t, 11, len(files))
}

func Test_WalkExcludedDirs(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		IsDirAccepted: func(dir string) bool {
			return !strings.Contains(dir, "Excluded_")
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// 10 temp files at toplevel, 3 subdirs with 10 files each = 40 valid files in total
	err = createTempFiles(tempDir, 10, "py")
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		subdir := filepath.Join(tempDir, fmt.Sprintf("subdir_%d", i))
		err := os.Mkdir(subdir, 0700)
		require.NoError(t, err)
		libdir := filepath.Join(subdir, "site-packages")
		err = os.Mkdir(libdir, 0700)
		require.NoError(t, err)

		err = createTempFiles(libdir, 10, "py")
		require.NoError(t, err)
	}

	// 3 excluded dirs with 10 files each
	for i := 0; i < 3; i++ {
		subdir := filepath.Join(tempDir, fmt.Sprintf("Excluded_%d", i))
		err := os.Mkdir(subdir, 0700)
		require.NoError(t, err)
		libdir := filepath.Join(subdir, "site-packages")
		err = os.Mkdir(libdir, 0700)
		require.NoError(t, err)

		err = createTempFiles(libdir, 10, "py")
		require.NoError(t, err)
	}

	// init: start watcher and background fs walker
	mgr.Initialize(component.InitializerOptions{Platform: &platform.Platform{IsNewInstall: false}})
	defer mgr.Terminate()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	select {
	case <-ctx.Done():
		require.FailNow(t, "timeout while waiting for fs walk to finish")
	case <-mgr.ReadyChan():
		// files contains the library directories found
		files := mgr.Files()
		require.EqualValues(t, 3, len(files), "expected 3 supported library directories to be found during walk of the root dir")
	}

	// allow earlier fs events to be processed
	collectChangeEvents(mgr, 10*time.Second)

	// add a file in an excluded dir, this will trigger a change on darwin and windows
	excludedFile1 := filepath.Join(tempDir, "Excluded_1", "newFile_3.py")
	assert.NoError(t, createFile(excludedFile1))

	changes := collectChangeEvents(mgr, 10*time.Second)
	var expected int
	if runtime.GOOS != "linux" {
		// linux filters files at the watcher level
		expected = 1
	}
	require.EqualValues(t, expected, len(changes), "changes are sent for excluded files (excluding linux)")
}

func Test_FileSystemWalk(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		IsDirAccepted: func(dir string) bool {
			// Default filters remove AppData, which is where temp directories are created
			return true
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// Create accepted files: 11 temp files at toplevel
	file1 := filepath.Join(tempDir, "test.py")
	assert.NoError(t, createFile(file1))
	err = createTempFiles(tempDir, 10, "py")
	require.NoError(t, err)

	// Create accepted files: 11 files in first subdir, 10 in second
	// Create unaccepted file: 5 in each subdir
	var file2 string
	for i := 0; i < 2; i++ {
		subdir := filepath.Join(tempDir, fmt.Sprintf("subdir_%d", i))
		err := os.Mkdir(subdir, 0700)
		require.NoError(t, err)
		if i == 0 {
			file2 = filepath.Join(subdir, "test.py")
			assert.NoError(t, createFile(file2))
		}
		err = createTempFiles(subdir, 10, "py")
		require.NoError(t, err)

		err = createTempFiles(subdir, 5, "txt")
		require.NoError(t, err)
	}

	mgr.LocalFS.initialize(component.InitializerOptions{})
	defer mgr.Terminate()

	var files []string
	err = mgr.LocalFS.Walk(kitectx.Background(), filepath.Dir(file1), func(path string, fi localcode.FileInfo, err error) error {
		if err != nil {
			if fi.IsDir {
				assert.Equal(t, localcode.ErrLibDir, err)
				return localcode.ErrSkipDir
			}
			assert.Equal(t, localcode.ErrUnacceptedFile, err)
			return nil
		}
		if !fi.IsDir {
			files = append(files, path)
		}
		return nil
	})
	// tempDir should have 32 children
	assert.Len(t, files, 32, "should have 32 files")
}

func Test_SkipLibraryDir(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// Create unaccepted files: 10 library dir files
	libraryDir := filepath.Join(tempDir, "site-packages")
	err = os.Mkdir(libraryDir, os.ModePerm)
	require.NoError(t, err)
	err = createTempFiles(libraryDir, 10, "py")
	require.NoError(t, err)
	vals := readdir.List(libraryDir)
	assert.EqualValues(t, 10, len(vals))

	mgr.LocalFS.initialize(component.InitializerOptions{})
	defer mgr.Terminate()

	// confirm Walk skips library directories
	var files []string
	err = mgr.LocalFS.Walk(kitectx.Background(), tempDir, func(path string, fi localcode.FileInfo, err error) error {
		if err != nil {
			if fi.IsDir {
				assert.Equal(t, localcode.ErrLibDir, err)
				return localcode.ErrSkipDir
			}
			assert.Equal(t, localcode.ErrUnacceptedFile, err)
			return nil
		}
		if !fi.IsDir {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, len(files))
}

func Test_DirNotAccepted(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs-bad")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		IsDirAccepted: func(file string) bool {
			return !strings.Contains(file, "bad")
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// Create 10 accepted files
	err = createTempFiles(tempDir, 10, "py")
	require.NoError(t, err)

	// Create more accepted files: 10 files in first subdir, 10 in second
	for i := 0; i < 2; i++ {
		subdir := filepath.Join(tempDir, fmt.Sprintf("subdir_%d", i))
		err := os.Mkdir(subdir, 0700)
		require.NoError(t, err)
		err = createTempFiles(subdir, 10, "py")
		require.NoError(t, err)
	}

	mgr.LocalFS.initialize(component.InitializerOptions{})
	defer mgr.Terminate()

	// confirm Walk returns only top-level files for filtered directory
	var files []string
	err = mgr.LocalFS.Walk(kitectx.Background(), tempDir, func(path string, fi localcode.FileInfo, err error) error {
		if err != nil {
			if fi.IsDir {
				assert.Equal(t, localcode.ErrLibDir, err)
				return localcode.ErrSkipDir
			}
			assert.Equal(t, localcode.ErrUnacceptedFile, err)
			return nil
		}
		if !fi.IsDir {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 10, len(files), "walk should not recurse when directory is not accepted")
}

func Test_Stat(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		IsDirAccepted: func(dir string) bool {
			// Default filters remove AppData, which is where temp directories are created
			return !strings.Contains(dir, "bad")
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// Create accepted files
	file1 := filepath.Join(tempDir, "test.py")
	assert.NoError(t, createFile(file1))
	badDir := filepath.Join(tempDir, "bad")
	err = os.Mkdir(badDir, 0700)
	require.NoError(t, err)
	file2 := filepath.Join(badDir, "test.py")
	assert.NoError(t, createFile(file2))

	mgr.LocalFS.initialize(component.InitializerOptions{})
	defer mgr.Terminate()

	// calling Stat on a file in a non-filtered directory should succeed
	_, err = mgr.LocalFS.Stat(file1)
	assert.NoError(t, err)

	// calling Stat on a file in a filtered directory should succeed
	_, err = mgr.LocalFS.Stat(file2)
	assert.NoError(t, err)

	// calling Stat on a non-absolute path should fail
	dir, err := os.Getwd()
	assert.NoError(t, err)
	relFile := filepath.Join(dir, "test.py")
	assert.NoError(t, createFile(relFile))
	defer os.Remove(relFile)
	_, err = mgr.LocalFS.Stat("test.py")
	assert.Error(t, err, localcode.ErrNonAbsolutePath)
}

func Test_Glob(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		IsDirAccepted: func(dir string) bool {
			// Default filters remove AppData, which is where temp directories are created
			return !strings.Contains(dir, "bad")
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// Create egg files
	eggFile1 := filepath.Join(tempDir, "test1.egg")
	assert.NoError(t, createFile(eggFile1))
	eggFile2 := filepath.Join(tempDir, "test2.egg")
	assert.NoError(t, createFile(eggFile2))
	badFile := filepath.Join(tempDir, "bad")
	assert.NoError(t, createFile(badFile))

	mgr.LocalFS.initialize(component.InitializerOptions{})
	defer mgr.Terminate()

	// calling Glob should return two matches
	matches, err := mgr.LocalFS.Glob(tempDir, "*.egg")
	assert.NoError(t, err)
	assert.Len(t, matches, 2)

	// confirm that files are canonicalized (converting to native path should succeed)
	for _, f := range matches {
		_, err := localpath.FromUnix(f)
		assert.NoError(t, err)
	}
}

func dropEvents(mgr *Manager) {
	log.Println("------ dropping events -----")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mgr.Changes():
		default:
		}
	}
}

func collectChangeEvents(mgr *Manager, duration time.Duration) []string {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var changes []string
	for {
		select {
		case <-ctx.Done():
			return changes
		case ch := <-mgr.Changes():
			for _, p := range ch.Paths {
				changes = append(changes, p)
			}
		}
	}
}

func awaitChangeEvents(mgr *Manager, expected ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	changes := make(map[string]bool)
loop:
	for {
		select {
		case <-ctx.Done():
			break loop

		case ch := <-mgr.Changes():
			for _, p := range ch.Paths {
				changes[p] = true
			}
			if len(changes) == len(expected) {
				break loop
			}
		}
	}

	for _, file := range expected {
		file = prepareFilepath(file)
		if ok := changes[file]; !ok {
			return fmt.Errorf("no event found for file %s. All: %v", file, changes)
		}
	}

	return nil
}

func createTempFiles(root string, count int, fileExt string) error {
	for i := 0; i < count; i++ {
		if err := ioutil.WriteFile(filepath.Join(root, fmt.Sprintf("file_%d.%s", i, fileExt)), []byte{}, 0600); err != nil {
			return err
		}
	}
	return nil
}

func createFile(filePath string) error {
	return ioutil.WriteFile(filePath, []byte{}, 0600)
}

func updateFile(filePath string, content string) error {
	return ioutil.WriteFile(filePath, []byte(content), 0600)
}

func deleteFile(filePath string) error {
	return os.Remove(filePath)
}

func prepareFilepath(path string) string {
	path, _ = localpath.ToUnix(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func awaitFileRemovals(max time.Duration, mgr *Manager, expectedCount int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), max)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout while waiting for removal of %d files", expectedCount)
		case evt := <-mgr.Changes():
			return fmt.Errorf("unexpected change event %v", evt)
		default:
			if atomic.LoadInt64(&deleteCount.Value) == expectedCount {
				return nil
			}
		}
	}
}

func waitFor(t *testing.T, max time.Duration, ch <-chan bool) {
	ctx, cancel := context.WithTimeout(context.Background(), max)
	defer cancel()

	select {
	case <-ctx.Done():
		require.Fail(t, "timeout while waiting for channel")
	case <-ch:
	}
}
