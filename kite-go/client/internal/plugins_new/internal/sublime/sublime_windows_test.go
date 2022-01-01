package sublime

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			if name == exeName {
				return true, nil
			}
			return false, nil
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

func TestFlow(t *testing.T) {
	installDir, cleanup := setupSublimeBundle(t)
	defer cleanup()
	// add a dummy exe file in our temp dir
	err := ioutil.WriteFile(filepath.Join(installDir, exeName), []byte{}, 0700)
	require.NoError(t, err)

	processMgr := &process.MockManager{
		VersionData: func(exePath string) string {
			return "3042"
		},
		IsProcessRunningData: func(name string) (bool, error) {
			return false, nil
		},
	}

	mgr := &winSublime{
		process:                processMgr,
		commonInstallLocations: []string{installDir},
	}

	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.EqualValues(t, 1, len(editors))
	assert.EqualValues(t, filepath.Join(installDir, exeName), editors[0].Path)

	// now install the plugin
	err = mgr.Install(context.Background(), editors[0].Path)
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(mgr.packagesDirectory(), pluginDirName), "expected that the plugin is installed inside of the packages directory")

	// update
	err = mgr.Update(context.Background(), editors[0].Path)
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(mgr.packagesDirectory(), pluginDirName), "expected the plugin data after a successful update")

	// uninstall
	err = mgr.Uninstall(context.Background(), editors[0].Path)
	require.NoError(t, err)
	assert.False(t, fs.DirExists(filepath.Join(mgr.packagesDirectory(), pluginDirName)), "expected the plugin data after a successful uninstall")
}

func TestRunning(t *testing.T) {
	isRunning := true
	processMgr := &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			return isRunning && name == exeName, nil
		},
	}

	mgr := &winSublime{
		process: processMgr,
	}

	testBasics(t, mgr)
	testInstallUninstallUpdate(t, mgr)
}

func TestDetectRunningEditors(t *testing.T) {
	dir, cleanup := setupSublimeBundle(t)
	defer cleanup()

	// add a dummy exe file in our temp dir
	installedExePath := filepath.Join(dir, exeName)
	err := ioutil.WriteFile(installedExePath, []byte{}, 0700)
	require.NoError(t, err)

	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("sublime_text", installedExePath, []string{"sublime_text"}),
			}, nil
		},
		VersionData: func(exePath string) string {
			if exePath == installedExePath {
				return "3042"
			}
			return ""
		},
	}

	mgr := &winSublime{
		process: processMgr,
	}

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 1)

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 1)
}

func TestOpenFile(t *testing.T) {
	sublimePath, cleanup := setupSublimeBundle(t)
	defer cleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			targ := "/foo/bar:1"
			if name == sublimePath && arg[0] == targ {
				return []byte(""), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(processMgr)
	_, err := mgr.OpenFile(context.Background(), id, sublimePath, "/foo/bar", 1)
	require.NoError(t, err)
}

func setupSublimeBundle(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir("", "kite-sublimetext")
	require.NoError(t, err)
	return dir, func() {
		os.RemoveAll(dir)
	}
}
