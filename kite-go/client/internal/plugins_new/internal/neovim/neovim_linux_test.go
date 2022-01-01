package neovim

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
	"github.com/stretchr/testify/require"
)

func setupTestManager(baseDir string) *linuxNeovim {
	return newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.Contains(name, "nvim")) && args == "--version" {
				versionString := "NVIM v0.2.2"
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
	})
}

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(id string) (bool, error) {
			if id == executableName {
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
	baseDir, cleanup := shared.SetupTempDir(t, "kite-neovim")
	defer cleanup()
	mgr := setupTestManager(baseDir)
	paths, _ := mgr.DetectEditors(context.Background())
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Equal(t, 1, len(editors))
	require.Equal(t, filepath.Join(baseDir, executableName), editors[0].Path)
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
				// regular vim, must not be detected
				process.NewMockProcess("vim", "/usr/bin/vim", []string{"vim", "test.py"}),
				// neovim, 1 and 2 are aliases of the same installation
				process.NewMockProcess("nvim", "/usr/bin/nvim", []string{"nvim", "test.py"}),
				process.NewMockProcess("nvim-qt", "/usr/bin/nvim-qt", []string{"nvim-qt", "test.py"}),
				process.NewMockProcess("nvim", "/home/user/bin/nvim", []string{"nvim", "test.py"}),
			}, nil
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			base := filepath.Base(name)
			if (base == "nvim" || base == "nvim-qt") && args == "--version" {
				return []byte("NVIM v0.2.2"), nil
			}
			return nil, fmt.Errorf("invalid Executable Name %s and/or args %s", name, args)
		},
	})

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 3)

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 3)
}
