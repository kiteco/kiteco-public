// +build linux darwin

package filesystem

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/stretchr/testify/require"
)

// make sure that symlink cycles are not walked by the fs walker
func Test_SymlinkCycleWalking(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-local-fs")
	require.NoError(t, err)

	// resolve symlinks in case that the path to the temp dir contains one (as seen on macOS)
	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// create a loop root/a/b --> root/a
	a := filepath.Join(tempDir, "a")
	b := filepath.Join(tempDir, "a", "b")

	os.Mkdir(a, 0700)

	err = os.Symlink(a, b)
	require.NoError(t, err)

	mgr := NewManager(Options{
		RootDir: tempDir,
		IsFileAccepted: func(file string) bool {
			return filepath.Ext(file) == ".py"
		},
		IsDirAccepted: func(dir string) bool {
			return true
		},
		DutyCycle: 0.5, // duty in 50% of the time
	})

	// init: start watcher and background fs walker
	mgr.Initialize(component.InitializerOptions{Platform: &platform.Platform{IsNewInstall: false}})
	defer mgr.Terminate()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		require.FailNow(t, "timeout while waiting for fs walk to finish")
	case <-mgr.ReadyChan():
		files := mgr.Files()
		require.EqualValues(t, 0, len(files), "expected no supported files in a root with only sub directories. Symlinks must not be contained. Files: %v", files)
	}
}
