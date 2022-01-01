package vim

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

func setupTestManager(baseDir string) *linuxVim {
	return newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.Contains(name, "vim")) && args == "-v--version" {
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
			for _, name := range executableNames {
				if name == id {
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
	baseDir, cleanup := shared.SetupTempDir(t, "kite-vim")
	defer cleanup()
	mgr := setupTestManager(baseDir)
	paths, _ := mgr.DetectEditors(context.Background())
	editors := shared.MapEditors(context.Background(), paths, mgr)
	// # of editors should be equal to length of executableNames
	require.Equal(t, len(executableNames), len(editors))
	sort.Strings(executableNames)
	for i, p := range executableNames {
		require.Equal(t, filepath.Join(baseDir, p), editors[i].Path)
	}
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
	// /usr/bin/vim might be present on the current system
	// ListData() will be invoked with resolved symbolic links
	resolvedVimPath := shared.DedupePaths([]string{"/kite/bin/vim"})[0]

	mgr := newTestManager(&process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("vim", "/kite/bin/vim", []string{"vim", "test.py"}),
				process.NewMockProcess("gvim", "/kite/bin/vim", []string{"gvim", "test.py"}),
				process.NewMockProcess("vim", "/home/user/bin/vim", []string{"vim", "test.py"}),
			}, nil
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if ((name == "/kite/bin/vim" || name == resolvedVimPath) || name == "/home/user/bin/vim") && args == "-v--version" {
				versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
				patchString := "Included patches: 1-503, 505-680, 682-1283"
				return []byte(versionString + "\n" + patchString), nil
			}
			return nil, fmt.Errorf("invalid Executable Name %s and/or args %s", name, args)
		},
	})

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 3, "vim and gvim should both be detected")

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 2, "duplicates of vim and gvim must be removed")
}
