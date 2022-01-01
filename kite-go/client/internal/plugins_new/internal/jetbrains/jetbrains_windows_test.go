package jetbrains

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
func findManagerByIDTest(id string, process process.WindowsProcessManager, commonPaths []string, betaChannel bool) (*windowsJetBrains, error) {
	managers, err := newJetBrainsManagers(process, commonPaths, betaChannel)
	if err != nil {
		return nil, err
	}

	for _, mgr := range managers {
		if mgr.ID() == id {
			return mgr.(*windowsJetBrains), nil
		}
	}

	return nil, kiteerrors.New("unable for find JetBrains plugin manager with ID %s", id)
}

func TestInstallConfig(t *testing.T) {
	// IntelliJ
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			if strings.HasPrefix(name, "idea") {
				return true, nil
			}
			return false, nil
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
		IsProcessRunningData: func(name string) (bool, error) {
			if strings.HasPrefix(name, "pycharm") {
				return true, nil
			}
			return false, nil
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
		IsProcessRunningData: func(name string) (bool, error) {
			if strings.HasPrefix(name, "goland") {
				return true, nil
			}
			return false, nil
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

	// WebStorm
	mockProcessManager = &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			if strings.HasPrefix(name, "webstorm") {
				return true, nil
			}
			return false, nil
		},
	}
	mgr, _ = newWebStormTestManager("", "", mockProcessManager, true)
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

func TestIntelliJFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "IU-193.4574.5", toolboxDir, "apps", "IDEA-U", "ch-0", "193.4574.5")

	mgr, err := newIntelliJTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "IntelliJIdea2019.3")
}

func TestIntelliJCEFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "IC-193.4574.5", toolboxDir, "apps", "IDEA-C", "ch-0", "193.4574.5")

	mgr, err := newIntelliJTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "IdeaIC2019.3")
}

func TestPycharmFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "PY-193.4574.5", toolboxDir, "apps", "PyCharm-P", "ch-0", "193.4574.5")

	mgr, err := newPyCharmTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "PyCharm2019.3")
}

func TestPycharmCEFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "PC-193.4574.5", toolboxDir, "apps", "PyCharm-C", "ch-0", "193.4574.5")

	mgr, err := newPyCharmTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "PyCharmCE2019.3")
}

func TestPycharmCEAnacondaFlow(t *testing.T) {
	// PyCharm for Anaconda isn't available via Toolbox,
	// therefore we're setting this up in the common paths (aka C:\Program Files\JetBrains\...)

	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	programFilesDir := filepath.Join(baseDir, "Program Files")
	ideDir := setupIDEInstallation(t, "PCA-193.4574.5", programFilesDir, "PyCharm Community Edition with Anaconda Plugin 2019.3.1")

	toolboxDir := filepath.Join(baseDir, "toolbox")

	mgr, err := newPyCharmTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	// manually update common paths
	mgr.(*windowsJetBrains).installPaths = []string{programFilesDir}

	testBasicInstallFlow(t, mgr, baseDir, ideDir, "PyCharmCE2019.3")
}

func TestPycharmAnacondaFlow(t *testing.T) {
	// PyCharm for Anaconda isn't available via Toolbox,
	// therefore we're setting this up in the common paths (aka C:\Program Files\JetBrains\...)

	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	programFilesDir := filepath.Join(baseDir, "Program Files")
	ideDir := setupIDEInstallation(t, "PYA-193.4574.5", programFilesDir, "PyCharm Professional Edition with Anaconda Plugin 2019.3.1")

	toolboxDir := filepath.Join(baseDir, "toolbox")

	mgr, err := newPyCharmTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	// manually update common paths
	mgr.(*windowsJetBrains).installPaths = []string{programFilesDir}

	testBasicInstallFlow(t, mgr, baseDir, ideDir, "PyCharm2019.3")
}

func TestGoLandFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-goland-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "GO-193.5233.112", toolboxDir, "apps", "Goland", "ch-0", "193.5233.112")

	mgr, err := newGoLandTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "GoLand2019.3")
}

func TestWebStormFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-webstorm-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "WS-193.5233.112", toolboxDir, "apps", "WebStorm", "ch-0", "193.5233.112")

	mgr, err := newWebStormTestManager(baseDir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "WebStorm2019.3")
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

	// using a custom value for %APPDATALOCAL
	appDataLocalDir := filepath.Join(baseDir, "CustomAppData", "Local")
	err = os.MkdirAll(appDataLocalDir, 0700)
	require.NoError(t, err)
	oldAppDataLocalDir, _ := os.LookupEnv("APPDATA")
	os.Setenv("APPDATA", appDataLocalDir)
	defer os.Setenv("APPDATA", oldAppDataLocalDir)

	pluginDir := filepath.Join(appDataLocalDir, "JetBrains", "PyCharmCE2020.1", "plugins")
	err = os.MkdirAll(pluginDir, 0700)
	err = mgr.Install(context.Background(), ideDir)
	require.NoError(t, err)

	pluginJars, err := filepath.Glob(fmt.Sprintf("%s\\%s\\*\\kite-pycharm-*.jar", pluginDir, pluginDirName))
	require.NoError(t, err)
	require.Len(t, pluginJars, 1, "plugin with version suffix 193 must be installed")
}

func TestCommonLocations(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	// on windows IntelliJ is installed at
	// C:\Program Files\JetBrains\IntelliJ IDEA Community Edition 2018.3.5, for example
	// PyCharm is installed at
	// C:\Program Files\JetBrains\PyCharm Community Edition 2018.3.5, for example
	// GoLand is installed at
	// C:\Program Files\JetBrains\GoLand 2019.3, for example

	commonDir := filepath.Join(baseDir, "programs", "JetBrains")
	intellijDir := setupIDEInstallation(t, "IC-2018.3.5", commonDir, "IntelliJ IDEA Community Edition 2018.3.5")
	pycharmDir := setupIDEInstallation(t, "PC-2018.3.5", commonDir, "PyCharm Community Edition 2018.3.5")
	golandDir := setupIDEInstallation(t, "GO-2018.3.5", commonDir, "GoLand 2018.3.5")

	// IntelliJ
	intelliJMgr, err := findManagerByIDTest(intellijID, &process.MockManager{}, []string{commonDir}, true)
	require.NoError(t, err)
	intelliJMgr.userHome = baseDir
	intelliJMgr.toolboxDir = filepath.Join(baseDir, "toolbox")

	paths, err := intelliJMgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, intelliJMgr)
	require.Len(t, editors, 1, "intellij must be found at the common location")
	require.EqualValues(t, intellijDir, editors[0].Path, "intellij must be found at the common location but must not include the PyCharm location")

	// PyCharm
	pycharmMgr, err := findManagerByIDTest(pycharmID, &process.MockManager{}, []string{commonDir}, true)
	require.NoError(t, err)
	pycharmMgr.userHome = baseDir
	pycharmMgr.toolboxDir = filepath.Join(baseDir, "toolbox")

	paths, err = pycharmMgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors = shared.MapEditors(context.Background(), paths, pycharmMgr)
	require.Len(t, editors, 1, "pycharm must be found at the common location")
	require.EqualValues(t, pycharmDir, editors[0].Path, "pycharm must be found at the common location but must not include the IntelliJ installation")

	// GoLand
	golandMgr, err := findManagerByIDTest(golandID, &process.MockManager{}, []string{commonDir}, true)
	require.NoError(t, err)
	golandMgr.userHome = baseDir
	golandMgr.toolboxDir = filepath.Join(baseDir, "toolbox")

	paths, err = golandMgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors = shared.MapEditors(context.Background(), paths, golandMgr)
	require.Len(t, editors, 1, "goland must be found at the common location")
	require.EqualValues(t, golandDir, editors[0].Path, "goland must be found at the common location but must not include the IntelliJ installation")
}

func TestDetectRunningIntelliJ(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	commonDir := filepath.Join(baseDir, "programs", "JetBrains")
	intellijDir := setupIDEInstallation(t, "IC-2018.3.5", commonDir, "IntelliJ IDEA Community Edition 2018.3.5")

	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("idea64.exe", filepath.Join(intellijDir, "bin", "idea64.exe"), []string{"idea64.exe"}),
			}, nil
		},
	}

	// IntelliJ
	intelliJMgr, err := findManagerByIDTest(intellijID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	intelliJMgr.userHome = baseDir

	paths, err := intelliJMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 1)
	editors := shared.MapEditors(context.Background(), paths, intelliJMgr)
	require.Len(t, editors, 1, "intellij must be found at the common location")
	require.EqualValues(t, intellijDir, editors[0].Path, "intellij must be found at the common location but must not include the PyCharm location")

	// PyCharm
	pycharmMgr, err := findManagerByIDTest(pycharmID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	pycharmMgr.userHome = baseDir

	paths, err = pycharmMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Empty(t, paths, "PyCharm must not be detected when there's only an IntelliJ process")
}

func TestDetectRunningPyCharm(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	commonDir := filepath.Join(baseDir, "programs", "JetBrains")
	pycharmDir := setupIDEInstallation(t, "PC-2018.3.5", commonDir, "PyCharm Community Edition 2018.3.5")

	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("pycharm64.exe", filepath.Join(pycharmDir, "bin", "pycharm64.exe"), []string{"pycharm64.exe"}),
			}, nil
		},
	}

	// IntelliJ
	intelliJMgr, err := findManagerByIDTest(intellijID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	intelliJMgr.userHome = baseDir

	paths, err := intelliJMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 0, "IntelliJ must not be detected when there's only a PyCharm process")

	// PyCharm
	pycharmMgr, err := findManagerByIDTest(pycharmID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	pycharmMgr.userHome = baseDir

	paths, err = pycharmMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 1)
}

func TestDetectRunningGoLand(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-goland-ide")
	defer cleanup()

	commonDir := filepath.Join(baseDir, "programs", "JetBrains")
	pycharmDir := setupIDEInstallation(t, "GO-2018.3.5", commonDir, "GoLand 2018.3.5")

	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("goland64.exe", filepath.Join(pycharmDir, "bin", "goland64.exe"), []string{"goland64.exe"}),
			}, nil
		},
	}

	// IntelliJ
	intelliJMgr, err := findManagerByIDTest(intellijID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	intelliJMgr.userHome = baseDir

	paths, err := intelliJMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 0, "IntelliJ must not be detected when there's only a GoLand process")

	// GoLand
	golandMgr, err := findManagerByIDTest(golandID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	golandMgr.userHome = baseDir

	paths, err = golandMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 1)
}

func TestDetectRunningWebStorm(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-webstorm-ide")
	defer cleanup()

	commonDir := filepath.Join(baseDir, "programs", "JetBrains")
	pycharmDir := setupIDEInstallation(t, "WS-2018.3.5", commonDir, "WebStorm 2018.3.5")

	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("webstorm64.exe", filepath.Join(pycharmDir, "bin", "webstorm64.exe"), []string{"webstorm64.exe"}),
			}, nil
		},
	}

	// IntelliJ
	intelliJMgr, err := findManagerByIDTest(intellijID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	intelliJMgr.userHome = baseDir

	paths, err := intelliJMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 0, "IntelliJ must not be detected when there's only a GoLand process")

	// WebStorm
	webstormMgr, err := findManagerByIDTest(webstormID, processMgr, []string{commonDir}, true)
	require.NoError(t, err)
	webstormMgr.userHome = baseDir

	paths, err = webstormMgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, paths, 1)
}

func TestOpenFile(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "IU-193.4574.5", toolboxDir, "apps", "IDEA-U", "ch-0", "193.4574.5")

	scriptPath := createIdeaScript(t, ideDir, "idea.bat")
	mgr, err := newIntelliJTestManager(baseDir, toolboxDir, &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			if name == scriptPath && shared.StringsContain(arg, filepath.Join(baseDir, "a.txt")) {
				return []byte(""), nil
			}
			return nil, fmt.Errorf("not found")
		},
	}, true)
	require.NoError(t, err)

	// add parents of idea.sh and pycharm.sh into $PATH, this allows detection
	defer updatePathEnv(t, filepath.Dir(scriptPath))()

	_, err = mgr.OpenFile(context.Background(), "IU", "", filepath.Join(baseDir, "a.txt"), 2)
	require.NoError(t, err)
}

func TestInstalledProductIDsGoland(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-goland-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideDir := setupIDEInstallation(t, "GO-193.5233.112", toolboxDir, "apps", "Goland", "ch-0", "193.5233.112")
	scriptPath := createIdeaScript(t, ideDir, "goland.bat")
	// add parents of idea.sh and pycharm.sh into $PATH, this allows detection
	defer updatePathEnv(t, filepath.Dir(scriptPath))()

	mgr, err := newGoLandTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	ipid, ok := mgr.(internal.InstalledProductIDs)
	require.True(t, ok, "goland manager must implement InstalledProductIDs")
	assert.Contains(t, ipid.InstalledProductIDs(context.Background()), "GO")
}

func createIdeaScript(t *testing.T, ideHome string, scriptName string) string {
	dir := filepath.Join(ideHome, "bin")
	err := os.MkdirAll(dir, 0700)
	require.NoError(t, err)

	scriptPath := filepath.Join(dir, scriptName)
	err = ioutil.WriteFile(scriptPath, []byte("REM dummy script"), 0700)
	require.NoError(t, err)
	return scriptPath
}
