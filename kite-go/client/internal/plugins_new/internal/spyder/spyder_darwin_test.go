package spyder

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Manager(t *testing.T) {
	var tempDir string
	tempDir, err := ioutil.TempDir("", "kite-spyder")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tempDir, err = filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)

	iniData, err := ioutil.ReadFile(filepath.Join(prefix, "spyder.ini"))
	require.NoError(t, err)

	iniFilePath := filepath.Join(tempDir, ".spyder-py3", "config", "spyder.ini")
	err = os.MkdirAll(filepath.Dir(iniFilePath), 0700)
	require.NoError(t, err)

	err = ioutil.WriteFile(iniFilePath, iniData, 0600)
	require.NoError(t, err)

	// setup mocked conda dir
	condaDir := filepath.Join(tempDir, "conda")
	condaBinDir := filepath.Join(condaDir, "bin")
	err = os.MkdirAll(condaBinDir, 0700)
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(condaBinDir, "conda"), []byte{}, 0700)
	require.NoError(t, err)

	p := process.MockManager{
		BinaryLocation: func(id string) []string {
			if id == "conda" {
				return []string{filepath.Join(condaBinDir, "conda")}
			}
			return nil
		},
		RunResult: func(name string, arg ...string) ([]byte, error) {
			if name == filepath.Join(condaBinDir, "conda") {
				return []byte(` 
						[{
						"base_url": "https://conda.anaconda.org/spyder-ide",
						"build_number": 0,
						"build_string": "py37_0",
						"channel": "spyder-ide",
						"dist_name": "spyder-4.0.1-py37_0",
						"name": "spyder",
						"platform": "darwin-64",
						"version": "4.0.1"
					  	}]`), nil
			}
			return nil, errors.Errorf("unexpected command " + name)
		},
		CustomDir: func() (string, error) {
			return tempDir, nil
		},
	}

	mgr, err := NewManager(&p)
	require.NoError(t, err)
	mgr.(*macSpyder).commonCondaDirs = []string{condaDir}

	editors, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, editors, 1)

	config, err := mgr.EditorConfig(context.Background(), editors[0])
	require.NoError(t, err)
	require.Empty(t, config.Compatibility)
	require.EqualValues(t, "4.0.0", config.RequiredVersion)
	require.EqualValues(t, "4.0.1", config.Version)
	require.EqualValues(t, iniFilePath, config.Path)

	require.EqualValues(t, ID, mgr.ID())
	require.EqualValues(t, name, mgr.Name())

	testInstallUninstallUpdate(t, mgr, iniFilePath)

	// activate again, apply suboptimal settings and test the HTTP requests
	err = setKiteEnabled(iniFilePath, true)
	require.NoError(t, err)
	err = setSpyderConfigValue(iniFilePath, "editor", "automatic_completions_after_chars", "3")
	require.NoError(t, err)

	optimalSettings, running, err := SettingsStatus(context.Background(), mgr)
	require.NoError(t, err)
	require.False(t, optimalSettings)
	require.False(t, running)

	err = ApplyOptimalSettings(context.Background(), mgr)
	require.NoError(t, err)

	optimalSettings, running, err = SettingsStatus(context.Background(), mgr)
	require.NoError(t, err)
	require.True(t, optimalSettings, "after applying new settings, settings have to be reported as optimal")
	require.False(t, running)
}

func Test_IsRunning(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-spyder")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// setup tempDir as conda command, DetectRunningEditors expect a conda directory layout
	condaCmd := filepath.Join(tempDir, "bin", "conda")
	err = os.MkdirAll(filepath.Dir(condaCmd), 0700)
	require.NoError(t, err)
	err = ioutil.WriteFile(condaCmd, []byte{}, 0700)
	require.NoError(t, err)

	p := process.MockManager{
		AllRunningApplicationsData: func() (process.SimpleList, error) {
			return process.SimpleList{
				{
					Executable: filepath.Join(tempDir, "python.app/Contents/MacOS/python"),
					Arguments:  []string{filepath.Join(tempDir, "bin", "spyder")},
				},
			}, nil
		},
	}

	mgr, err := NewManager(&p)
	require.NoError(t, err)

	running, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, running, 1)

	config := mgr.InstallConfig(context.Background())
	require.True(t, config.Running)
}

func Test_IsNotRunning(t *testing.T) {
	p := process.MockManager{
		AllRunningApplicationsData: func() (process.SimpleList, error) {
			return process.SimpleList{}, nil
		},
	}

	mgr, err := NewManager(&p)
	require.NoError(t, err)
	running, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Empty(t, running)

	config := mgr.InstallConfig(context.Background())
	require.False(t, config.Running)
}

// test that install, update and uninstall succeed
func testInstallUninstallUpdate(t *testing.T, mgr editor.Plugin, configFilePath string) {
	err := mgr.Install(context.Background(), configFilePath)
	require.NoError(t, err, "installing must succeed")
	assert.True(t, mgr.IsInstalled(context.Background(), configFilePath), "plugin must be installed after a successful call of Install")

	err = mgr.Update(context.Background(), configFilePath)
	require.NoErrorf(t, err, "updating must succeed")
	assert.True(t, mgr.IsInstalled(context.Background(), configFilePath), "plugin must still be installed after Update")

	err = mgr.Uninstall(context.Background(), configFilePath)
	require.NoErrorf(t, err, "uninstalling must succeed")
	assert.False(t, mgr.IsInstalled(context.Background(), configFilePath), "plugin must be uninstalled after Uninstall")
}
