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
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/stretchr/testify/require"
)

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(id string) (bool, error) {
			for _, exeName := range exeNameMap {
				if id == exeName {
					return true, nil
				}
			}
			return false, nil
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

	exePaths, vscodeCleanup := setupVSCodeBinary(t, dir, true)
	defer vscodeCleanup()

	for _, exePath := range exePaths {
		cliPath := filepath.Join(filepath.Dir(exePath), "bin", vsCodeCmd)
		if strings.Contains(exePath, "Insiders") {
			cliPath = filepath.Join(filepath.Dir(exePath), "bin", insidersCmd)
		}
		kiteExtensionDir := filepath.Join(dir, vsCodePluginDir, "extensions", "kiteco.kite")
		if strings.Contains(exePath, "Insiders") {
			kiteExtensionDir = filepath.Join(dir, insidersPluginDir, "extensions", "kiteco.kite")
		}
		// this process manager tracks the install and uninstall commands to mock a typical cli workflow
		processMgr := &process.MockManager{
			StartMenuData: func() []string {
				return []string{exePath}
			},
			RunResultWithEnv: func(name string, additionalEnv []string, arg ...string) ([]byte, error) {
				if len(additionalEnv) != 1 || additionalEnv[0] != "__COMPAT_LAYER=RUNASINVOKER" {
					return nil, errors.Errorf("missing required __COMPAT_LAYER environment variable for vscode commands")
				}

				args := strings.Join(arg, " ")

				if name == cliPath && args == "--version" {
					return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
				}

				if name == cliPath && args == "--list-extensions" {
					if fs.DirExists(kiteExtensionDir) {
						return []byte("some.extension\nkiteco.kite\nanother.extension"), nil
					}
					return nil, nil
				}

				if name == cliPath && args == "--install-extension kiteco.kite" {
					if fs.DirExists(kiteExtensionDir) {
						return nil, errors.Errorf("Extension 'kiteco.kite' is already installed")
					}
					// this is needed for the uninstall which removes this directory
					os.MkdirAll(kiteExtensionDir, 0700)
					return []byte("Installing\nExtension 'kiteco.kite' v0.74.0 was successfully installed!"), nil
				}

				if name == cliPath &&
					args == installExtensionArg+" "+vscodeMarketplaceID+" "+forceArg {
					// this is needed for the uninstall which removes this directory
					os.MkdirAll(kiteExtensionDir, 0700)
					return []byte("Updating\nExtension 'kiteco.kite' v0.74.0 was successfully installed!"), nil
				}

				return nil, errors.Errorf("unknown command %s", name)
			},
		}

		mgr := newTestManager(dir, processMgr)
		testBasicInstallFlow(t, mgr, exePath)
	}
}

func TestIsInstalled(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// setup mock binaries with and without cli binaries
	exePaths, vscodeCleanup := setupVSCodeBinary(t, dir, true)
	defer vscodeCleanup()

	for _, exePath := range exePaths {
		cliPath := filepath.Join(filepath.Dir(exePath), "bin", vsCodeCmd)
		if strings.Contains(exePath, "Insiders") {
			cliPath = filepath.Join(filepath.Dir(exePath), "bin", insidersCmd)
		}
		// command output includes kite package
		processMgr := &process.MockManager{
			RunResultWithEnv: func(name string, additionalEnv []string, arg ...string) ([]byte, error) {
				args := strings.Join(arg, " ")
				if name == cliPath && args == "--list-extensions" {
					return []byte("some.extension\nkiteco.kite\nother.extension"), nil
				}
				return nil, errors.Errorf("unknown command %s", name)
			},
		}

		mgr := newTestManager(dir, processMgr)
		require.True(t, mgr.IsInstalled(context.Background(), exePath))
	}

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
	exePaths, vscodeCleanup := setupVSCodeBinary(t, dir, true)
	defer vscodeCleanup()

	exePathsNoCli, vscodeCleanup := setupVSCodeBinary(t, dir, false)
	defer vscodeCleanup()

	exeCommonPaths, commonPathCleanup := setupVSCodeBinary(t, dir, true)
	defer commonPathCleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")
			if (shared.StringsContain(exePaths, name) || shared.StringsContain(exePathsNoCli, name) || shared.StringsContain(exeCommonPaths, name)) && args == "--version" {
				return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
			}
			return nil, errors.Errorf("unknown command %s", name)
		},
		StartMenuData: func() []string {
			return append(exePaths, exePathsNoCli...)
		},
	}

	mgr := newTestManager(dir, processMgr)
	// common\tempDir-nnn\vscode.exe -> common\tempDir-nnn\
	mgr.commonPaths = exeCommonPaths

	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	// exePaths should return len(exeNameMap) results, and exeCommonPaths should return another len(exeNameMap) results.
	require.Len(t, editors, len(exeNameMap)*2, "editor detection must return exePaths and exeCommonPaths, but must not return installations without cli and has to handle duplicate entries")

	// remove the common paths
	for _, path := range exeCommonPaths {
		err = os.Remove(path)
		require.NoError(t, err)
	}
	paths, err = mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors = shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, len(exeNameMap))

	// remove the cli file and make sure that it's not detected
	for _, path := range exePaths {
		err = os.Remove(path)
		require.NoError(t, err)
	}
	require.NoError(t, err)
	paths, err = mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors = shared.MapEditors(context.Background(), paths, mgr)
	require.Empty(t, editors)
}

func TestDetectRunningEditors(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// setup mock binaries with and without cli binaries
	exePaths, vscodeCleanup := setupVSCodeBinary(t, dir, true)
	defer vscodeCleanup()

	exePathsNoCli, vscodeCleanup := setupVSCodeBinary(t, dir, false)
	defer vscodeCleanup()

	exeCommonPaths, commonPathCleanup := setupVSCodeBinary(t, dir, true)
	defer commonPathCleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")
			if (shared.StringsContain(exePaths, name) || shared.StringsContain(exePathsNoCli, name) || shared.StringsContain(exeCommonPaths, name)) && args == "--version" {
				return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
			}
			return nil, errors.Errorf("unknown command %s", name)
		},
		ListData: func() (process.List, error) {
			var result []process.Process
			for _, path := range system.Union(exePaths, exePathsNoCli, exeCommonPaths) {
				result = append(result, process.NewMockProcess(filepath.Base(path), path, []string{path}))
			}
			return result, nil
		},
	}

	mgr := newTestManager(dir, processMgr)
	mgr.commonPaths = exeCommonPaths

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, len(exeNameMap)*2, "every running editor with a cli must be detected (Code.exe and Code - Insiders.exe for all with cli present)")

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, len(exeNameMap)*2, "editor detection must return exePaths and exeCommonPaths, but must not return installations without cli and has to handle duplicate entries")
}

func TestOpenFile(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	// setup mock binaries with and without cli binaries
	exePaths, vscodeCleanup := setupVSCodeBinary(t, dir, true)
	defer vscodeCleanup()

	processMgr := &process.MockManager{
		RunResultWithEnv: func(name string, additionalEnv []string, arg ...string) ([]byte, error) {
			targ := []string{gotoArg, "/foo/bar:1"}
			targs := strings.Join(targ, " ")
			args := strings.Join(arg, " ")
			cmd := vsCodeCmd
			if strings.Contains(exePaths[0], "Insiders") {
				cmd = insidersCmd
			}
			if name == filepath.Join(filepath.Dir(exePaths[0]), "bin", cmd) && args == targs {
				return []byte(""), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)
	_, err := mgr.OpenFile(context.Background(), "vscode", filepath.Join(exePaths[0], "\\resources\\app"), "/foo/bar", 1)
	require.NoError(t, err)
}

// setups a dummy vscode binary at the given path and returns the path to the binaries and a cleanup function
func setupVSCodeBinary(t *testing.T, baseDir string, createCLI bool) ([]string, func()) {
	binDir, err := ioutil.TempDir(baseDir, "vscode-binary")
	require.NoError(t, err)

	var codePaths []string
	for _, exeName := range exeNameMap {
		codePath := filepath.Join(binDir, exeName)
		err = os.MkdirAll(filepath.Dir(codePath), 0700)
		require.NoError(t, err)
		err = ioutil.WriteFile(codePath, []byte(""), 0700)
		require.NoError(t, err)
		codePaths = append(codePaths, codePath)

		if createCLI {
			cliPath := filepath.Join(binDir, "bin", vsCodeCmd)
			if strings.Contains(exeName, "Insiders") {
				cliPath = filepath.Join(binDir, "bin", insidersCmd)
			}
			err = os.MkdirAll(filepath.Dir(cliPath), 0700)
			require.NoError(t, err)

			err = ioutil.WriteFile(cliPath, []byte(""), 0700)
			require.NoError(t, err)
		}
		// create the matching vscode extension dir in the base dir
		var err error
		if strings.Contains(exeName, "Insiders") {
			err = os.MkdirAll(filepath.Join(baseDir, insidersPluginDir, "extensions"), 0700)
		} else {
			err = os.MkdirAll(filepath.Join(baseDir, vsCodePluginDir, "extensions"), 0700)
		}
		require.NoError(t, err)
	}

	return codePaths, func() {}
}
