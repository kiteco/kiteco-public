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

func setupTestManager(baseDir string) *macNeovim {
	return newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.HasSuffix(name, executableName)) && args == "--version" {
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
	})
}

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(id string) (bool, error) {
			return id == executableName, nil
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
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Equal(t, 1, len(editors))
	require.Equal(t, filepath.Join(baseDir, executableName), editors[0].Path)
}

func TestDetectEditorsError(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-neovim")
	defer cleanup()
	mgr := newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.HasSuffix(name, executableName)) && args == "--version" {
				versionString := ""
				return []byte(versionString), nil
			}
			return nil, fmt.Errorf("Invalid Executable Name %s and/or args %s", name, args)
		},
		BinaryLocation: func(id string) []string {
			return []string{filepath.Join(baseDir, "vim")}
		},
	})
	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Equal(t, 0, len(editors))
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
