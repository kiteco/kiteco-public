package watch

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/stretchr/testify/require"
)

func Test_Watcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmpDir, err := ioutil.TempDir("", "kite-fswatcher")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	readyChan := make(chan bool, 1)

	// unbuffered to simplify testing
	changesChan := make(chan []Event, 100)

	_, _ = NewFilesystem(ctx, []string{tmpDir}, changesChan, readyChan, Options{})

	// wait for ready
	select {
	case <-ctx.Done():
		require.Fail(t, "timed out")
	case <-readyChan:
	}

	fileCreated := filepath.Join(tmpDir, "file_created.py")
	fileModified := filepath.Join(tmpDir, "file_updated.py")
	fileRemoved := filepath.Join(tmpDir, "file_removed.py")

	// create our files, not passing content because that would trigger two modification events per file
	err = ioutil.WriteFile(fileCreated, nil, 0700)
	err = ioutil.WriteFile(fileModified, nil, 0700)
	err = ioutil.WriteFile(fileRemoved, nil, 0700)

	created, err := collectChanges(ctx, changesChan, fileCreated, fileModified, fileRemoved)
	require.NoError(t, err)
	require.EqualValues(t, localfiles.ModifiedEvent, created[fileCreated])
	require.EqualValues(t, localfiles.ModifiedEvent, created[fileModified])
	require.EqualValues(t, localfiles.ModifiedEvent, created[fileRemoved])

	// modify a file
	err = ioutil.WriteFile(fileModified, []byte("new content"), 0700)
	require.NoError(t, err)
	created, err = collectChanges(ctx, changesChan, fileModified)
	require.NoError(t, err)
	require.EqualValues(t, localfiles.ModifiedEvent, created[fileModified])

	// remove a file
	err = os.Remove(fileRemoved)
	require.NoError(t, err)
	created, err = collectChanges(ctx, changesChan, fileRemoved)
	require.NoError(t, err)
	require.EqualValues(t, localfiles.RemovedEvent, created[fileRemoved])
}

// collect changes takes a list of files which are expected to change
// it returns as soon as change events for all of these file have been delivered
// or when the context was cancelled
// it returns the mapping of changed file to the type of the last delivered change event
// it takes the list of files because a watcher may send more than one event per file and possibly out-of-order
func collectChanges(ctx context.Context, ch <-chan []Event, changedFiles ...string) (map[string]localfiles.EventType, error) {
	result := make(map[string]localfiles.EventType)

	expected := make(map[string]bool)
	for _, p := range changedFiles {
		expected[p] = true
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case evt := <-ch:
			log.Printf("change: %v\n", evt)

			for _, e := range evt {
				result[e.Path] = e.Type
				delete(expected, e.Path)
			}

			if len(expected) == 0 {
				return result, nil
			}
		}
	}
}
