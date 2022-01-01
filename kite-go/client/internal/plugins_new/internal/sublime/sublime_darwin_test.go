package sublime

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			if id == bundleID {
				return true
			}
			return false
		},
	}
	mgr := newTestManager(mockProcessManager)
	require.Equal(t,
		&editor.InstallConfig{
			RequiresRestart:       true,
			Running:               true,
			InstallWhileRunning:   true,
			UpdateWhileRunning:    true,
			UninstallWhileRunning: true,
		}, mgr.InstallConfig(context.Background()))
}

func TestInstallation(t *testing.T) {
	dir, cleanup := setupSublimeBundle(t)
	defer cleanup()

	processMgr := &process.MockManager{
		BundleLocationsData: func(id string) ([]string, error) {
			if id == bundleID {
				return []string{dir}, nil
			}
			return nil, fmt.Errorf("unexpected use of bundle")
		},
		VersionData: func(appPath string) (string, error) {
			if appPath == dir {
				return "1.2.3", nil
			}
			return "", fmt.Errorf("unexpected use of version")
		},
	}

	mgr, err := NewManager(processMgr)
	require.NoError(t, err)

	editors, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 1, len(editors))
}

func TestRunning(t *testing.T) {
	dir, cleanup := setupSublimeBundle(t)
	defer cleanup()

	isRunning := true
	processMgr := &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return isRunning && id == bundleID
		},
	}

	mgr := &macSublime{
		process:             processMgr,
		sublimeUserDataPath: dir,
	}

	testBasics(t, mgr)
	testInstallUninstallUpdate(t, mgr)
}

func TestDetectRunningEditors(t *testing.T) {
	// the command package doesn't include the kite package
	processMgr := &process.MockManager{
		RunningApplicationsData: func() (process.List, error) {
			return []process.Process{
				{Pid: 1, BundleID: bundleID, BundleLocation: "/Applications/Sublime3"},
			}, nil
		},
	}

	mgr := &macSublime{
		process: processMgr,
	}

	editors, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, 1)
}

func TestOpenFile(t *testing.T) {
	sublimePath, cleanup := setupSublimeBundle(t)
	defer cleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			expectedPath := filepath.Join(sublimePath, "Contents", "SharedSupport", "bin", "subl")
			targ := "/foo/bar:1"
			if name == expectedPath && arg[0] == targ {
				return []byte(""), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(processMgr)
	editorPath := filepath.Join(sublimePath, "Contents/MacOS/Sublime Text")
	_, err := mgr.OpenFile(context.Background(), id, editorPath, "/foo/bar", 1)
	require.NoError(t, err)
}

func setupSublimeBundle(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir("", "kite-sublimetext")
	require.NoError(t, err)
	return dir, func() {
		os.RemoveAll(dir)
	}
}
