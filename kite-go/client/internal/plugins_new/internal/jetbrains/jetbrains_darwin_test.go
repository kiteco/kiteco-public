package jetbrains

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	kiteerrors "github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findManagerByIDTest locates a manager by ID, only for tests
func findManagerByIDTest(id string, process process.MacProcessManager, betaChannel bool) (*macJetBrains, error) {
	managers, err := NewJetBrainsManagers(process, betaChannel)
	if err != nil {
		return nil, err
	}

	for _, mgr := range managers {
		if mgr.ID() == id {
			return mgr.(*macJetBrains), nil
		}
	}

	return nil, kiteerrors.New("unable for find JetBrains plugin manager with ID %s", id)
}

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			for _, bundleID := range []string{"com.jetbrains.intellij-EAP", "com.jetbrains.intellij", "com.jetbrains.intellij.ce"} {
				if id == bundleID {
					return true
				}
			}
			return false
		},
	}
	mgr, _ := newIntelliJTestManager("", "", mockProcessManager, true)
	require.Equal(t,
		&editor.InstallConfig{
			RequiresRestart:          true,
			MultipleInstallLocations: true,
			Running:                  true,
			InstallWhileRunning:      true,
			UpdateWhileRunning:       false,
			UninstallWhileRunning:    false,
		}, mgr.InstallConfig(context.Background()))

	// PyCharm
	mockProcessManager = &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			for _, bundleID := range []string{"com.jetbrains.pycharm-EAP", "com.jetbrains.pycharm", "com.jetbrains.pycharm.ce"} {
				if id == bundleID {
					return true
				}
			}
			return false
		},
	}
	mgr, _ = newPyCharmTestManager("", "", mockProcessManager, true)
	require.Equal(t,
		&editor.InstallConfig{
			RequiresRestart:          true,
			MultipleInstallLocations: true,
			Running:                  true,
			InstallWhileRunning:      true,
			UpdateWhileRunning:       false,
			UninstallWhileRunning:    false,
		}, mgr.InstallConfig(context.Background()))

	// GoLand
	mockProcessManager = &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			for _, bundleID := range []string{"com.jetbrains.goland-EAP", "com.jetbrains.goland"} {
				if id == bundleID {
					return true
				}
			}
			return false
		},
	}
	mgr, _ = newGoLandTestManager("", "", mockProcessManager, true)
	require.Equal(t,
		&editor.InstallConfig{
			RequiresRestart:          true,
			MultipleInstallLocations: true,
			Running:                  true,
			InstallWhileRunning:      true,
			UpdateWhileRunning:       false,
			UninstallWhileRunning:    false,
		}, mgr.InstallConfig(context.Background()))
}

func TestIsPycharmRunning(t *testing.T) {
	processMgr := &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return shared.StringsContain([]string{"com.jetbrains.pycharm-EAP", "com.jetbrains.pycharm", "com.jetbrains.pycharm.ce"}, id)
		},
	}

	pycharm, err := findManagerByIDTest(pycharmID, processMgr, true)
	require.NoError(t, err)

	intellij, err := findManagerByIDTest(intellijID, processMgr, true)
	require.NoError(t, err)

	assert.True(t, pycharm.isIDERunning())
	assert.False(t, intellij.isIDERunning())
}

func TestIsGoLandRunning(t *testing.T) {
	processMgr := &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return shared.StringsContain([]string{"com.jetbrains.goland-EAP", "com.jetbrains.goland"}, id)
		},
	}

	goland, err := findManagerByIDTest(golandID, processMgr, true)
	require.NoError(t, err)

	intellij, err := findManagerByIDTest(intellijID, processMgr, true)
	require.NoError(t, err)

	assert.True(t, goland.isIDERunning())
	assert.False(t, intellij.isIDERunning())
}

func TestIsIntellijRunning(t *testing.T) {
	processMgr := &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return shared.StringsContain([]string{"com.jetbrains.intellij-EAP", "com.jetbrains.intellij", "com.jetbrains.intellij.ce"}, id)
		},
	}

	pycharm, err := findManagerByIDTest(pycharmID, processMgr, true)
	require.NoError(t, err)

	intellij, err := findManagerByIDTest(intellijID, processMgr, true)
	require.NoError(t, err)

	assert.False(t, pycharm.isIDERunning())
	assert.True(t, intellij.isIDERunning())
}

func TestPycharmFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	bundleDir := setupIDEInstallation(t, "PC-193.4574.5", filepath.Join(baseDir, "pycharm-2019.3"))

	mgr, err := newPyCharmTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			// IntelliJ is running, but no PyCharm
			return id == "com.jetbrains.intellij.ce"
		},
		BundleLocationsData: func(id string) ([]string, error) {
			if id == "com.jetbrains.pycharm.ce" {
				return []string{bundleDir}, nil
			}
			return []string{}, nil
		},
	}, true)
	require.NoError(t, err)

	testBasicInstallFlow(t, mgr, baseDir, bundleDir, "PyCharmCE2019.3")
}

func TestPycharmAnacondaFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	bundleDir := setupIDEInstallation(t, "PYA-193.4574.5", filepath.Join(baseDir, "PyCharm with Anaconda plugin"))

	mgr, err := newPyCharmTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			// no running IDE
			return false
		},
		BundleLocationsData: func(id string) ([]string, error) {
			if id == "com.jetbrains.pycharm" {
				return []string{bundleDir}, nil
			}
			return []string{}, nil
		},
	}, true)
	require.NoError(t, err)

	testBasicInstallFlow(t, mgr, baseDir, bundleDir, "PyCharm2019.3")
}

func TestGoLandFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-goland-ide")
	defer cleanup()

	bundleDir := setupIDEInstallation(t, "GO-193.4574.5", filepath.Join(baseDir, "goland-2019.3"))

	mgr, err := newGoLandTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return id == "com.jetbrains.intellij.ce"
		},
		BundleLocationsData: func(id string) ([]string, error) {
			if id == "com.jetbrains.goland" { //fixme validate
				return []string{bundleDir}, nil
			}
			return []string{}, nil
		},
	}, true)
	require.NoError(t, err)

	testBasicInstallFlow(t, mgr, baseDir, bundleDir, "GoLand2019.3")
}

func TestWebStormFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-webstorm-ide")
	defer cleanup()

	bundleDir := setupIDEInstallation(t, "WS-193.4574.5", filepath.Join(baseDir, "webstorm-2019.3"))

	mgr, err := newWebStormTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return id == "com.jetbrains.intellij.ce"
		},
		BundleLocationsData: func(id string) ([]string, error) {
			if id == "com.jetbrains.WebStorm" { //fixme validate
				return []string{bundleDir}, nil
			}
			return []string{}, nil
		},
	}, true)
	require.NoError(t, err)

	testBasicInstallFlow(t, mgr, baseDir, bundleDir, "WebStorm2019.3")
}

func TestIntellijFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	bundleDir := setupIDEInstallation(t, "IU-193.4574.5", filepath.Join(baseDir, "ides-2019.3"))

	mgr, err := newIntelliJTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		IsBundleRunningData: func(id string) bool {
			return id == "com.jetbrains.pycharm.ce"
		},
		BundleLocationsData: func(id string) ([]string, error) {
			if id == "com.jetbrains.intellij.ce" {
				return []string{bundleDir}, nil
			}
			return []string{}, nil
		},
	}, true)
	require.NoError(t, err)

	testBasicInstallFlow(t, mgr, baseDir, bundleDir, "IntelliJIdea2019.3")
}

func Test2019_3(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "PC-193.4574.5", baseDir, "idea", "PC-193.4574.5")
	mgr, err := newPyCharmTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	pluginDir := setupPluginsDir(t, baseDir, "PyCharmCE2019.3")

	err = mgr.Install(context.Background(), ideDir)
	require.NoError(t, err)

	// make sure that the plugin for 2019.3+ was installed
	pluginJars, err := filepath.Glob(fmt.Sprintf("%s/*/kite-pycharm-*.jar", pluginDir))
	require.NoError(t, err)
	require.Len(t, pluginJars, 1, "plugin with version suffix 193 must be installed")
}

func Test2020_1(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "PC-201.1234.5", baseDir, "idea", "PC-201.1234.5")
	mgr, err := newPyCharmTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	pluginDir := filepath.Join(baseDir, "Library", "Application Support", "JetBrains", "PyCharmCE2020.1", "plugins", pluginDirName)
	err = os.MkdirAll(pluginDir, 0700)
	require.NoError(t, err)

	err = mgr.Install(context.Background(), ideDir)
	require.NoError(t, err)

	// make sure that the plugin for 2019.3+ was installed
	pluginJars, err := filepath.Glob(fmt.Sprintf("%s/*/kite-pycharm-*.jar", pluginDir))
	require.NoError(t, err)
	require.Len(t, pluginJars, 1, "plugin with version suffix 193 must be installed")
}

func TestDetectRunningEditors(t *testing.T) {
	mgr := process.MockManager{
		RunningApplicationsData: func() (process.List, error) {
			var list []process.Process

			all := append(append([]string{"com.jetbrains.pycharm-EAP", "com.jetbrains.pycharm", "com.jetbrains.pycharm.ce"}, "com.jetbrains.intellij-EAP", "com.jetbrains.intellij", "com.jetbrains.intellij.ce"), "com.jetbrains.goland-EAP", "com.jetbrains.goland")
			for i, id := range all {
				list = append(list, process.Process{
					Pid:            i,
					BundleID:       id,
					BundleLocation: fmt.Sprintf("/Applications/%s", id),
				})
			}

			return list, nil
		},
	}

	pycharm, err := findManagerByIDTest(pycharmID, &mgr, true)
	require.NoError(t, err)

	goland, err := findManagerByIDTest(golandID, &mgr, true)
	require.NoError(t, err)

	intellij, err := findManagerByIDTest(intellijID, &mgr, true)
	require.NoError(t, err)

	pycharmList, err := pycharm.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, pycharmList, 3, "pycharm must not detect intellij bundles")
	for i, id := range []string{"com.jetbrains.pycharm-EAP", "com.jetbrains.pycharm", "com.jetbrains.pycharm.ce"} {
		assert.EqualValues(t, fmt.Sprintf("/Applications/%s", id), pycharmList[i])
	}

	golandList, err := goland.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, golandList, 2, "goland must not detect intellij bundles")
	for i, id := range []string{"com.jetbrains.goland-EAP", "com.jetbrains.goland"} {
		assert.EqualValues(t, fmt.Sprintf("/Applications/%s", id), golandList[i])
	}

	intellijList, err := intellij.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, intellijList, 3, "intellij mgr must not detect pycharm bundles")
	for i, id := range []string{"com.jetbrains.intellij-EAP", "com.jetbrains.intellij", "com.jetbrains.intellij.ce"} {
		assert.EqualValues(t, fmt.Sprintf("/Applications/%s", id), intellijList[i])
	}
}

func Test_InstalledProductIDsGoland(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-goland-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "GO-193.4574.5", filepath.Join(baseDir, "goland-2019.3"))
	mgr, err := newGoLandTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
		BundleLocationsData: func(id string) ([]string, error) {
			if id == "com.jetbrains.goland" {
				return []string{ideDir}, nil
			}
			return []string{}, nil
		},
	}, true)
	require.NoError(t, err)

	ipid, ok := mgr.(internal.InstalledProductIDs)
	require.True(t, ok, "goland manager must implement InstalledProductIDs")
	assert.Contains(t, ipid.InstalledProductIDs(context.Background()), "GO")
}
