package neovim

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/stretchr/testify/require"
)

func setupTestManager(baseDir string) *windowsNeovim {
	return newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.HasSuffix(name, "nvim.exe")) && args == "--version" {
				versionString := "NVIM v0.3.4"
				return []byte(versionString), nil
			}
			return nil, fmt.Errorf("Invalid Executable Name %s and/or args %s", name, args)
		},
		BinaryLocation: func(id string) []string {
			return []string{filepath.Join(baseDir, id)}
		},
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
		VersionData: func(exePath string) string {
			if filepath.Base(exePath) == "nvim-qt.exe" {
				return "0.2.13.0"
			}
			return ""
		},
	})
}

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(id string) (bool, error) {
			for _, exeName := range executableNames {
				if id == exeName {
					return true, nil
				}
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
	baseDir, cleanup := shared.SetupTempDir(t, "kite-neovim")
	defer cleanup()
	mgr := setupTestManager(baseDir)
	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, len(paths))
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Equal(t, 2, len(editors))

	sort.Slice(editors, func(i, j int) bool {
		return strings.Compare(editors[i].Path, editors[j].Path) > 0
	})
	require.Equal(t, filepath.Join(baseDir, "nvim.exe"), editors[0].Path)
	require.Equal(t, filepath.Join(baseDir, "nvim-qt.exe"), editors[1].Path)
}

func TestDetectEditorsError(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-neovim")
	defer cleanup()
	mgr := newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.HasSuffix(name, "nvim.exe")) && args == "--version" {
				versionString := ""
				return []byte(versionString), nil
			}
			return nil, fmt.Errorf("Invalid Executable Name %s and/or args %s", name, args)
		},
		BinaryLocation: func(id string) []string {
			return []string{filepath.Join(baseDir, id)}
		},
	})
	paths, _ := mgr.DetectEditors(context.Background())
	require.Len(t, paths, 2, "an existing file must be detected")
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 0, "an editor file which fails to execute must not return an editor config")
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

func TestInstallUninstallFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-neovim")
	defer cleanup()

	mgr := setupTestManager(baseDir)
	require.False(t, mgr.IsInstalled(context.Background(), ""))
	mgr.Install(context.Background(), "")
	require.True(t, mgr.IsInstalled(context.Background(), ""))
	mgr.Uninstall(context.Background(), "")
	require.False(t, mgr.IsInstalled(context.Background(), ""))
}

func TestDetectRunningEditors(t *testing.T) {
	mgr := newTestManager(&process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				// regular vim, must not be detected
				process.NewMockProcess("vim", `C:\Program Files\vim\vim.exe`, []string{"vim", "test.py"}),
				// neovim, 1 and 2 are aliases of the same installation
				process.NewMockProcess("nvim.exe", `C:\Program Files\nvim\nvim.exe`, []string{"nvim", "test.py"}),
				process.NewMockProcess("nvim-qt.exe", `C:\Program Files\nvim\nvim-qt.exe`, []string{"nvim-qt", "test.py"}),
				process.NewMockProcess("nvim.exe", `C:\Users\username\nvim\nvim.exe`, []string{"nvim"}),
				process.NewMockProcess("nvim-qt.exe", `C:\Users\username\Download\nvim\nvim-qt.exe`, []string{"nvim-qt", "test.py"}),
			}, nil
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			base := filepath.Base(name)
			if (base == "nvim.exe") && args == "--version" {
				return []byte("NVIM v0.2.2"), nil
			}
			return nil, fmt.Errorf("invalid Executable Name %s and/or args %s", name, args)
		},
		VersionData: func(exePath string) string {
			base := filepath.Base(exePath)
			if base == "nvim-qt.exe" {
				return "0.2.13.0"
			}
			return ""
		},
	})

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 4)

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 4)
}
