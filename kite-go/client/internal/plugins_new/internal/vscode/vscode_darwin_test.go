package vscode

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
		IsBundleRunningData: func(id string) bool {
			for _, bundleID := range bundleIDs {
				if id == bundleID {
					return true
				}
			}
			return false
		},
	}
	mgr := newTestManager("", mockProcessManager)
	require.Equal(t,
		&editor.InstallConfig{
			RequiresRestart:          true,
			MultipleInstallLocations: true,
			Running:                  true,
			InstallWhileRunning:      true,
			UpdateWhileRunning:       true,
			UninstallWhileRunning:    true,
		}, mgr.InstallConfig(context.Background()))
}

func TestInstallFlow(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	bundlePath, vscodeCleanup := setupVSCodeBundle(t, dir, true)
	defer vscodeCleanup()

	kiteExtensionDir := filepath.Join(dir, ".vscode", "extensions", "kiteco.kite")

	// this process manager tracks the install and uninstall commands to mock a typical cli workflow
	processMgr := &process.MockManager{
		BundleLocationsData: func(id string) ([]string, error) {
			for _, bundleID := range bundleIDs {
				if id == bundleID {
					return []string{bundlePath}, nil
				}
			}
			return nil, fmt.Errorf("unknown bundle ID %s", id)
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")

			if strings.HasSuffix(name, "code") && args == "--version" {
				return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
			}

			if strings.HasSuffix(name, "code") && args == "--list-extensions" {
				if fs.DirExists(kiteExtensionDir) {
					return []byte("some.extension\nkiteco.kite\nanother.extension"), nil
				}
				return nil, nil
			}

			if strings.HasSuffix(name, "code") && args == "--install-extension kiteco.kite" {
				if fs.DirExists(kiteExtensionDir) {
					return nil, fmt.Errorf("extension 'kiteco.kite' is already installed")
				}
				// this is needed for the uninstall which removes this directory
				os.MkdirAll(kiteExtensionDir, 0700)
				return []byte("Installing\nExtension 'kiteco.kite' v0.74.0 was successfully installed!"), nil
			}

			if strings.HasSuffix(name, "code") &&
				args == installExtensionArg+" "+vscodeMarketplaceID+" "+forceArg {
				// this is needed for the uninstall which removes this directory
				os.MkdirAll(kiteExtensionDir, 0700)
				return []byte("Updating\nExtension 'kiteco.kite' v0.74.0 was successfully installed!"), nil
			}

			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)
	testBasicInstallFlow(t, mgr, bundlePath)
}

func TestIsInstalled(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// setup mock binaries with and without cli binaries
	bundlePath, vscodeCleanup := setupVSCodeBundle(t, dir, true)
	defer vscodeCleanup()

	// command output includes kite package
	processMgr := &process.MockManager{
		BundleLocationsData: func(id string) ([]string, error) {
			for _, bundleID := range bundleIDs {
				if id == bundleID {
					return []string{bundlePath}, nil
				}
			}
			return nil, fmt.Errorf("unknown bundle ID %s", id)
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")
			if strings.HasSuffix(name, "code") && args == "--list-extensions" {
				return []byte("some.extension\nkiteco.kite\nother.extension"), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)

	require.True(t, mgr.IsInstalled(context.Background(), bundlePath))
}

func TestIsNotInstalled(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// the command package doesn't include the kite package
	processMgr := &process.MockManager{}

	mgr := newTestManager(dir, processMgr)
	require.False(t, mgr.IsInstalled(context.Background(), dir))
}

func TestIsInstallCliError(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// the command package doesn't include the kite package
	processMgr := &process.MockManager{}

	mgr := newTestManager(dir, processMgr)
	require.False(t, mgr.IsInstalled(context.Background(), dir))
}

func TestDetectEditors(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// setup mock binaries with and without cli binaries
	exePath, vscodeCleanup := setupVSCodeBundle(t, dir, true)
	defer vscodeCleanup()

	exePathNoCli, vscodeCleanup := setupVSCodeBundle(t, dir, false)
	defer vscodeCleanup()

	processMgr := &process.MockManager{
		BundleLocationsData: func(id string) ([]string, error) {
			for _, bundleID := range bundleIDs {
				if id == bundleID {
					return []string{id}, nil
				}
			}
			return nil, fmt.Errorf("unknown bundle ID %s", id)
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")
			if (name == exePath || name == exePathNoCli) && args == "--version" {
				return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)

	editors, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, 2, "editor detection must not return installations without cli and has to handle duplicate entries")

	// remove the cli file and make sure that it's not detected
	err = os.RemoveAll(exePath)
	require.NoError(t, err)
	editors, err = mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, 2)
}

func TestExtensionsDirInsiders(t *testing.T) {
	mgr := newTestManager("test", &process.MockManager{})
	dir := mgr.userExtensionsDir("/Applications/Visual Studio Code - Insiders.app")
	require.Equal(t, "test/.vscode-insiders/extensions", dir)
}

func TestDetectRunningEditors(t *testing.T) {
	// the command package doesn't include the kite package
	processMgr := &process.MockManager{
		RunningApplicationsData: func() (process.List, error) {
			var list []process.Process
			for i, id := range bundleIDs {
				list = append(list, process.Process{
					Pid:            i,
					BundleID:       id,
					BundleLocation: fmt.Sprintf("/Applications/%s", id),
				})
			}
			return list, nil
		},
	}

	mgr := newTestManager("", processMgr)

	editors, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, len(bundleIDs))
	for i, id := range bundleIDs {
		assert.EqualValues(t, fmt.Sprintf("/Applications/%s", id), editors[i])
	}
}

// setups a dummy vscode binary at the given path and returns the path to the binaries and a cleanup function
func setupVSCodeBundle(t *testing.T, baseDir string, createCLI bool) (string, func()) {
	binDir, err := ioutil.TempDir(baseDir, "vscode-binary")
	require.NoError(t, err)

	bundleDir := filepath.Join(binDir, "com.test.VSCode")
	err = os.MkdirAll(bundleDir, 0700)
	require.NoError(t, err)

	if createCLI {
		cliPath := filepath.Join(bundleDir, "Contents", "Resources", "app", "bin", "code")
		err = os.MkdirAll(filepath.Dir(cliPath), 0700)
		require.NoError(t, err)
		err = ioutil.WriteFile(cliPath, []byte(""), 0700)
		require.NoError(t, err)
	}

	return bundleDir, func() {}
}

func TestOpenFile(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// setup mock binaries
	bundlePath, vscodeCleanup := setupVSCodeBundle(t, dir, true)
	defer vscodeCleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			targ := []string{gotoArg, "/foo/bar:1"}
			targs := strings.Join(targ, " ")
			args := strings.Join(arg, " ")
			if name == bundlePath+"/Contents/Resources/app/bin/code" && args == targs {
				return []byte(""), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)
	_, err := mgr.OpenFile(context.Background(), "vscode", filepath.Join(bundlePath, "/Contents/Resources/app"), "/foo/bar", 1)
	require.NoError(t, err)
}
