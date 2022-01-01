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
	"github.com/stretchr/testify/require"
)

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(id string) (bool, error) {
			for _, name := range binaryNames {
				if name == id {
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

	// 1st entry is the preferred binary (code-oss), the others are the alternative names
	binaries, vscodeCleanup := setupVSCodeBinary(t, dir)
	defer vscodeCleanup()
	require.NotEmpty(t, binaries)

	kiteExtensionDir := filepath.Join(dir, ".vscode-oss", "extensions", "kiteco.kite")

	// this process manager tracks the install and uninstall commands to mock a typical cli workflow
	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")

			if name == binaries[0] && args == "--version" {
				return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
			}

			if name == binaries[0] && args == "--list-extensions" {
				if fs.DirExists(kiteExtensionDir) {
					return []byte("some.extension\nkiteco.kite\nanother.extension"), nil
				}
				return nil, nil
			}

			if name == binaries[0] && args == "--install-extension kiteco.kite" {
				if fs.DirExists(kiteExtensionDir) {
					return nil, fmt.Errorf("extension 'kiteco.kite' is already installed")
				}
				// this is needed for the uninstall which removes this directory
				os.MkdirAll(kiteExtensionDir, 0700)
				return []byte("Installing\nExtension 'kiteco.kite' v0.74.0 was successfully installed!"), nil
			}

			if name == binaries[0] &&
				args == installExtensionArg+" "+vscodeMarketplaceID+" "+forceArg {
				// this is needed for the uninstall which removes this directory
				os.MkdirAll(kiteExtensionDir, 0700)
				return []byte("Updating\nExtension 'kiteco.kite' v0.74.0 was successfully installed!"), nil
			}

			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)
	testBasicInstallFlow(t, mgr, binaries[0])
}

func TestIsInstalled(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	binaries, vscodeCleanup := setupVSCodeBinary(t, dir)
	defer vscodeCleanup()

	// command output includes kite package
	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")
			if name == binaries[0] && args == "--list-extensions" {
				return []byte("some.extension\nkiteco.kite\nother.extension"), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)
	require.True(t, mgr.IsInstalled(context.Background(), binaries[0]))
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

	binaries, vscodeCleanup := setupVSCodeBinary(t, dir)
	defer vscodeCleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, " ")
			if shared.StringsContain(binaries, name) && args == "--version" {
				return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}
	mgr := newTestManager(dir, processMgr)

	editors, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, len(binaryNames))

	for i := range binaries {
		err = os.Remove(binaries[i])
		require.NoError(t, err)
		editors, err = mgr.DetectEditors(context.Background())
		require.NoError(t, err)
		require.Len(t, editors, len(binaryNames)-(i+1))
	}
}

func TestDetectRunningEditors(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	codePath := writeFileAt(t, filepath.Join(dir, "code", "code"), true)
	codeBinPath := writeFileAt(t, filepath.Join(dir, "code", "bin", "code"), true)
	insidersPath := writeFileAt(t, filepath.Join(dir, "code-insiders", "code-insiders"), true)
	insidersBinPath := writeFileAt(t, filepath.Join(dir, "code-insiders", "bin", "code-insiders"), true)

	mgr := newTestManager("", &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("code", codePath, []string{"code"}),
				process.NewMockProcess("code-insiders", insidersPath, []string{"code-insiders"}),
			}, nil
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (name == codeBinPath || name == insidersBinPath) && args == "--version" {
				return []byte("1.31.1\n1b8e8302e405050205e69b59abb3559592bb9e60\nx64"), nil
			}
			return nil, fmt.Errorf("unknown binary")
		},
	})

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 2)

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 2)
}

func TestOpenFile(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	binaries, vscodeCleanup := setupVSCodeBinary(t, dir)
	defer vscodeCleanup()

	processMgr := &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			targ := []string{gotoArg, "/foo/bar:1"}
			targs := strings.Join(targ, " ")
			args := strings.Join(arg, " ")
			if name == binaries[0] && args == targs {
				return []byte(""), nil
			}
			return nil, fmt.Errorf("unknown command %s", name)
		},
	}

	mgr := newTestManager(dir, processMgr)
	_, err := mgr.OpenFile(context.Background(), "vscode", binaries[0], "/foo/bar", 1)
	require.NoError(t, err)
}

// setups a dummy vsoce binary at the given path and returns the path to the binaries and a cleanup function
func setupVSCodeBinary(t *testing.T, baseDir string) ([]string, func()) {
	binDir := filepath.Join(baseDir, "vscode")

	// we setup dummy binaries as seen on Linux, e.g. Arch.
	// DetectEditors must only detect the base dir once as both binaries come with the same package
	var binaries []string
	for _, name := range binaryNames {
		codePath := filepath.Join(binDir, name)
		writeFileAt(t, codePath, true)
		binaries = append(binaries, codePath)
		// create the matching vscode extension dir in the base dir
		err := os.MkdirAll(filepath.Join(baseDir, ".vs"+name, "extensions"), 0700)
		require.NoError(t, err)
	}

	// update path to contain our temp dir
	oldPath := os.Getenv("PATH")
	err := os.Setenv("PATH", strings.Join([]string{baseDir, binDir}, string(os.PathListSeparator)))
	require.NoError(t, err)

	return binaries, func() {
		os.Setenv("PATH", oldPath)
	}
}

func writeFileAt(t *testing.T, filePath string, executable bool) string {
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0700)
	require.NoError(t, err)

	var perms os.FileMode = 0600
	if executable {
		perms = 0700
	}

	err = ioutil.WriteFile(filePath, []byte(""), perms)
	require.NoError(t, err)
	return filePath
}
