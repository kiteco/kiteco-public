package vim

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
)

func setupTestManager(baseDir string) *macVim {
	return newTestManager(&process.MockManager{
		BundleLocationsData: func(id string) ([]string, error) {
			if id == bundleID {
				return []string{baseDir}, nil
			}
			return nil, fmt.Errorf("Invalid Bundle ID %s passed into BundleLocations", id)
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.HasSuffix(name, executableName) || strings.HasSuffix(name, macVimPath)) &&
				args == "--version" {
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
		IsBundleRunningData: func(id string) bool {
			if id == bundleID {
				return true
			}
			return false
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
	require.Equal(t, 2, len(editors))
	assert.Equal(t, filepath.Join(baseDir, macVimPath), editors[0].Path)
	assert.Equal(t, filepath.Join(baseDir, executableName), editors[1].Path)
}

func TestDetectEditorsError(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-vim")
	defer cleanup()
	mgr := newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.HasSuffix(name, executableName) || strings.HasSuffix(name, macVimPath)) &&
				args == "--version" {
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
	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Equal(t, 0, len(editors))

	mgr = newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if (strings.HasSuffix(name, executableName) || strings.HasSuffix(name, macVimPath)) &&
				args == "--version" {
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
	paths, err = mgr.DetectEditors(context.Background())
	require.NoError(t, err)
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

func TestDetectRunningEditors(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-vim")
	defer cleanup()
	mgr := newTestManager(&process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			args := strings.Join(arg, "")
			if strings.HasSuffix(name, macVimPath) && args == "--version" {
				versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
				patchString := "Included patches: 1-26, 505-680, 682-1283"
				return []byte(versionString + "\n" + patchString), nil
			}
			return nil, fmt.Errorf("invalid executable name %s and/or args %s", name, args)
		},
		RunningApplicationsData: func() (process.List, error) {
			return []process.Process{
				{
					Pid:            1,
					BundleID:       bundleID,
					BundleLocation: baseDir,
				},
			}, nil
		},
	})

	paths, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 1)
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
