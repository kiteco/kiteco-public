//+build linux

package watch

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/stretchr/testify/require"
)

// Test_WatcherLinux asserts that node_modules is filtered by default on Linux
func Test_WatcherLinux(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// we can't use the default temp here, because the default filters
	// always filter data in /tmp
	tmpDir, err := ioutil.TempDir(os.ExpandEnv("$HOME"), "kite-fswatcher-linux")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// create our files before watching is started
	// not passing content because that would trigger two modification events per file
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	nodeModulesSubDir := filepath.Join(nodeModulesDir, "subdir")
	err = os.MkdirAll(nodeModulesSubDir, 0700)
	require.NoError(t, err)

	readyChan := make(chan bool, 1)

	// unbuffered to simplify testing
	changesChan := make(chan []Event, 100)

	_, _ = NewFilesystem(ctx, []string{filepath.Join(tmpDir)}, changesChan, readyChan, Options{})

	// wait for ready
	select {
	case <-ctx.Done():
		require.Fail(t, "timed out")
	case <-readyChan:
	}

	err = ioutil.WriteFile(filepath.Join(nodeModulesDir, "file.js"), nil, 0700)
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(nodeModulesSubDir, "file_in_subdir.js"), nil, 0700)
	require.NoError(t, err)

	collectCtx, collectCancel := context.WithTimeout(ctx, 5*time.Second)
	defer collectCancel()
	events := collectAllChanges(collectCtx, changesChan)
	require.Empty(t, events, "no changes must be recorded for node_modules directories")
}

func Test_WatchAndUnwatch(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	fs := inotifyFilesystem{
		w:           watcher,
		watchCounts: make(map[string]uint16),
	}

	require.Empty(t, fs.watchCounts)

	tempDir, err := ioutil.TempDir("", "kite-watcher")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = fs.Watch(tempDir)
	require.NoError(t, err)

	require.EqualValues(t, 1, fs.watchCounts[tempDir])

	err = fs.Watch(tempDir)
	require.NoError(t, err)
	require.EqualValues(t, 2, fs.watchCounts[tempDir])

	// now call Unwatch two times
	err = fs.Unwatch(tempDir)
	require.NoError(t, err)
	require.EqualValues(t, 1, fs.watchCounts[tempDir])

	err = fs.Unwatch(tempDir)
	require.NoError(t, err)
	require.Empty(t, fs.watchCounts)

	// check with a missing path
	err = fs.Watch(filepath.Join(tempDir, "not-there.py"))
	require.Error(t, err)
	require.Empty(t, fs.watchCounts, "Watch for a missing file must not register a watch or increase the count")
}

// collect all watcher changes until the context expires
func collectAllChanges(ctx context.Context, ch <-chan []Event) map[string]localfiles.EventType {
	result := make(map[string]localfiles.EventType)

	for {
		select {
		case <-ctx.Done():
			return result

		case evt := <-ch:
			log.Printf("change: %v\n", evt)
			for _, e := range evt {
				result[e.Path] = e.Type
			}
		}
	}
}
