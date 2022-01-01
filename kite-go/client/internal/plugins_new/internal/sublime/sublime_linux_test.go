package sublime

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fmt"

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
			return shared.StringsContain(runningExecutableNames, name), nil
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
	tmpDir, cleanup := setupSublimeBundle(t)
	defer cleanup()
	// update path to include our temp dir as first element
	path := os.Getenv("PATH")
	defer os.Setenv("PATH", path)
	err := os.Setenv("PATH", strings.Join([]string{tmpDir, path}, string(os.PathListSeparator)))
	require.NoError(t, err)

	// add a dummy executable subl in our temp dir
	err = ioutil.WriteFile(filepath.Join(tmpDir, "subl"), []byte("#!/bin/bash"), 0700)
	require.NoError(t, err)

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			if filepath.Base(name) == "subl" && len(arg) == 1 && arg[0] == "--version" {
				return []byte("Sublime Text Build 3176"), nil
			}
			return nil, fmt.Errorf("unable to execute %s", name)
		},
		IsProcessRunningData: func(name string) (bool, error) {
			return false, nil
		},
	}

	userData, cleanup := setupSublimeBundle(t)
	defer cleanup()

	mgr := &linuxSublime{
		process:             processMgr,
		sublimeUserDataPath: userData,
	}
	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	assert.EqualValues(t, 1, len(editors))
	assert.EqualValues(t, filepath.Join(tmpDir, "subl"), editors[0].Path)

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
	dir, cleanup := setupSublimeBundle(t)
	defer cleanup()

	isRunning := true
	processMgr := &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			return isRunning && name == "sublime_text", nil
		},
	}

	mgr := &linuxSublime{
		process:             processMgr,
		sublimeUserDataPath: dir,
	}

	testBasics(t, mgr)
	testInstallUninstallUpdate(t, mgr)
}

func TestDetectRunningEditors(t *testing.T) {
	dir, cleanup := setupSublimeBundle(t)
	defer cleanup()

	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("sublime_text", "/snap/sublime-text/58/opt/sublime_text/sublime_text", []string{"sublime_text"}),
				process.NewMockProcess("subl3", "/usr/bin/subl3", []string{"subl3"}),
			}, nil
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			name = filepath.Base(name)
			if (name == "subl3" || name == "sublime_text") && len(arg) == 1 && arg[0] == "--version" {
				return []byte("Sublime Text Build 3176"), nil
			}
			return nil, fmt.Errorf("unable to execute %s", name)
		},
	}

	mgr := &linuxSublime{
		process:             processMgr,
		sublimeUserDataPath: dir,
	}

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 2, "both sublime_text and subl3 must be detected")

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 2, "both sublime_text and subl3 must be detected as valid editors")
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
