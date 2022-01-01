package vim

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testExePath = "C:\\Program Files (x86)\\Vim\\vim81\\vim.exe"
)

func setupTestManager(baseDir string) *windowsVim {
	return newTestManager(&process.MockManager{
		StartMenuData: func() []string {
			return []string{testExePath}
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if strings.Contains(name, "vim") && args == "--version" {
				versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
				patchString := "Included patches: 1-503, 505-680, 682-1283"
				return []byte(versionString + "\n" + patchString), nil
			}
			return nil, fmt.Errorf("Invalid Executable Name %s and/or args %s", name, args)
		},
		BinaryLocation: func(id string) []string {
			return []string{filepath.Join(baseDir, id)}
		},
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	})
}

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(id string) (bool, error) {
			if id == exeName {
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

func TestDetectEditors(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-vim")
	defer cleanup()
	mgr := setupTestManager(baseDir)
	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	// # of editors should be equal to length of pathExeNames + start menu entry
	require.Equal(t, len(pathExeNames)+1, len(editors))
	require.Equal(t, testExePath, editors[0].Path)
	for i, p := range pathExeNames {
		require.Equal(t, filepath.Join(baseDir, p), editors[i+1].Path)
	}
}

func TestDetectEditorsError(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-vim")
	defer cleanup()
	mgr := newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if strings.Contains(name, "vim") && args == "--version" {
				versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
				patchString := "Included patches: 1-26, 505-680, 682-1283"
				return []byte(versionString + "\n" + patchString), nil
			}
			return nil, fmt.Errorf("Invalid Executable Name %s and/or args %s", name, args)
		},
		BinaryLocation: func(id string) []string {
			return []string{filepath.Join(baseDir, "vim")}
		},
	})
	// Should complain about Patch Version 26
	paths, _ := mgr.DetectEditors(context.Background())
	require.Equal(t, 1, len(paths))
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Equal(t, 0, len(editors))

	mgr = newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if strings.Contains(name, "vim") && args == "--version" {
				versionString := ""
				patchString := ""
				return []byte(versionString + "\n" + patchString), nil
			}
			return nil, fmt.Errorf("Invalid Executable Name %s and/or args %s", name, args)
		},
		BinaryLocation: func(id string) []string {
			return []string{filepath.Join(baseDir, "vim")}
		},
	})
	// Should complain about not being able to find the version.
	paths, _ = mgr.DetectEditors(context.Background())
	editors = shared.MapEditors(context.Background(), paths, mgr)
	require.Equal(t, 0, len(editors))
}

func TestInstallUninstallFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-vim")
	defer cleanup()

	mgr := setupTestManager(baseDir)
	require.False(t, mgr.IsInstalled(context.Background(), ""))
	mgr.Install(context.Background(), "")
	require.True(t, mgr.IsInstalled(context.Background(), ""))
	mgr.Uninstall(context.Background(), "")
	require.False(t, mgr.IsInstalled(context.Background(), ""))
}

func TestInstallUninstallFlowWithErrors(t *testing.T) {
	mgr := newTestManager(&process.MockManager{
		CustomDir: func() (string, error) {
			return "", errors.New("Test Expecting Error")
		},
	})
	require.False(t, mgr.IsInstalled(context.Background(), ""))
	require.Error(t, mgr.Install(context.Background(), ""))
	require.Error(t, mgr.Uninstall(context.Background(), ""))
}

func TestDetectRunningEditors(t *testing.T) {
	mgr := newTestManager(&process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("vim.exe", `C:\Program Files\vim\vim.exe`, []string{"vim.exe", "test.py"}),
				process.NewMockProcess("gvim.exe", `C:\Program Files\vim\gvim.exe`, []string{"gvim.exe", "test.py"}),
				process.NewMockProcess("vim.exe", `C:\Users\username\bin\vim\vim.exe`, []string{"vim.exe"}),
			}, nil
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if filepath.Base(name) == "vim.exe" && args == "--version" {
				versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
				patchString := "Included patches: 1-503, 505-680, 682-1283"
				return []byte(versionString + "\n" + patchString), nil
			}
			return nil, fmt.Errorf("invalid Executable Name %s and/or args %s", name, args)
		},
	})

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 2, "vim, but not gvim, must be detected")
	assert.EqualValues(t, `C:\Program Files\vim\vim.exe`, paths[0])
	assert.EqualValues(t, `C:\Users\username\bin\vim\vim.exe`, paths[1])

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 2)
}
