package atom

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
	"github.com/stretchr/testify/require"
)

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return id == bundleID
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

func TestInstallFlow(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-atom")
	defer cleanup()

	atomBundlePath, apmPath, atomCleanup := setupAtomBundle(t, dir)
	defer atomCleanup()

	// state use by the mock manager
	isInstalled := false

	// this process manager tracks the install and uninstall commands to mock a typical apm workflow
	processMgr := &process.MockManager{
		BundleLocationsData: func(id string) ([]string, error) {
			if id == bundleID {
				return []string{atomBundlePath}, nil
			}
			return nil, fmt.Errorf("unknown bundle id %s", id)
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")
			if name == apmPath && args == "list --installed --bare --packages" {
				if isInstalled {
					return []byte("image-view@0.64.0\nkite@0.106.0\nwelcome@0.36.8"), nil
				}
				return []byte("image-view@0.64.0\nwelcome@0.36.8"), nil
			}

			if name == apmPath && args == "install kite" {
				isInstalled = true
				return []byte("Installing kite to ..."), nil
			}

			if name == apmPath && args == "uninstall kite" {
				if !isInstalled {
					return nil, fmt.Errorf("Uninstalling kite ✗\nFailed to delete kite: Does not exist")
				}

				isInstalled = false
				return []byte("Installing kite to ..."), nil
			}
			if name == apmPath && args == updateArg+" "+apmPluginID+" "+noConfirmArg {
				return []byte{}, nil
			}
			return nil, fmt.Errorf("unknown command %s %s", name, args)
		},
	}

	mgr := newTestManager(processMgr)
	testBasicInstallFlow(t, mgr, atomBundlePath)
}

func TestIsInstalled(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-atom")
	defer cleanup()

	apmBundlePath, apmPath, atomCleanup := setupAtomBundle(t, dir)
	defer atomCleanup()

	// command output includes kite package
	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			if name == apmPath && strings.Join(arg, " ") == "list --installed --bare --packages" {
				return []byte("image-view@0.64.0\nkite@0.106.0\nwelcome@0.36.8"), nil
			}
			return nil, fmt.Errorf("unknown command")
		},
	}

	mgr := newTestManager(processMgr)
	require.True(t, mgr.IsInstalled(context.Background(), apmBundlePath))
}

func TestIsNotInstalled(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-atom")
	defer cleanup()

	// the command package doesn't include the kite package
	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			if name == "apm" && strings.Join(arg, " ") == "list --installed --bare --packages" {
				return []byte("image-view@0.64.0\nwelcome@0.36.8"), nil
			}
			return nil, fmt.Errorf("unknown command")
		},
	}

	mgr := newTestManager(processMgr)
	require.False(t, mgr.IsInstalled(context.Background(), dir))
}

func TestIsInstallApmError(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-atom")
	defer cleanup()

	// the command package doesn't include the kite package
	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			if name == "apm" && strings.Join(arg, " ") == "list --installed --bare --packages" {
				return nil, fmt.Errorf("error executing apm command")
			}
			return nil, fmt.Errorf("unknown command")
		},
	}

	mgr := newTestManager(processMgr)
	require.False(t, mgr.IsInstalled(context.Background(), dir))
}

func TestDetectEditors(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-atom")
	defer cleanup()

	bundlePath, _, atomCleanup := setupAtomBundle(t, dir)
	defer atomCleanup()

	// the command package doesn't include the kite package
	processMgr := &process.MockManager{
		BundleLocationsData: func(id string) ([]string, error) {
			if id == bundleID {
				if fs.DirExists(bundlePath) {
					return []string{bundlePath}, nil
				}
				return []string{}, nil
			}
			return nil, fmt.Errorf("unknown bundle ID %s", id)
		},
	}

	mgr := newTestManager(processMgr)

	editors, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, 1)

	// remove the atom file and make sure that it's not detected
	err = os.RemoveAll(bundlePath)
	require.NoError(t, err)
	editors, err = mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	require.Empty(t, editors)
}

func TestDetectRunningEditors(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-atom")
	defer cleanup()

	bundlePath, _, atomCleanup := setupAtomBundle(t, dir)
	defer atomCleanup()

	bundlePath2, _, atomCleanup2 := setupAtomBundle(t, dir)
	defer atomCleanup2()

	// the command package doesn't include the kite package
	processMgr := &process.MockManager{
		RunningApplicationsData: func() (process.List, error) {
			return []process.Process{
				{Pid: 1, BundleID: bundleID, BundleLocation: bundlePath},
				{Pid: 2, BundleID: bundleID, BundleLocation: bundlePath2},
			}, nil
		},
	}

	mgr := newTestManager(processMgr)

	editors, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, 2)
}

func TestOpenFile(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// setup mock binary
	bundlePath, _, atomCleanup := setupAtomBundle(t, dir)
	defer atomCleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			targ := "/foo/bar:1"
			expected := filepath.Join(bundlePath, "Contents", "Resources", "app", "atom.sh")
			if name == expected && arg[0] == targ {
				return []byte(""), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(processMgr)
	editorPath := filepath.Join(bundlePath, "Contents/Resources/app/apm/bin/apm")
	_, err := mgr.OpenFile(context.Background(), "atom", editorPath, "/foo/bar", 1)
	require.NoError(t, err)
}

// setups a dummy atom app bundle at the given path and returns the path to it and a cleanup function
func setupAtomBundle(t *testing.T, baseDir string) (string, string, func()) {
	bundlePath := filepath.Join(baseDir, "atom", "Atom.app")
	err := os.MkdirAll(bundlePath, 0700)
	require.NoError(t, err)

	apmPath := filepath.Join(bundlePath, "Contents", "Resources", "app", "apm", "bin", "apm")
	err = os.MkdirAll(filepath.Dir(apmPath), 0700)
	require.NoError(t, err)

	err = ioutil.WriteFile(apmPath, []byte(""), 0700)
	require.NoError(t, err)

	return bundlePath, apmPath, func() {
		os.RemoveAll(bundlePath)
	}
}
