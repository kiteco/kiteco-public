package jetbrains

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-errors/errors"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	kiteerrors "github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	intellijToolboxProcess = []string{
		"/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/jre64/bin/java",
		"-classpath",
		"/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/lib/bootstrap.jar:/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/lib/extensions.jar:/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/lib/util.jar:/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/lib/jdom.jar:/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/lib/log4j.jar:/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/lib/trove4j.jar:/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/lib/jna.jar:/opt/JetBrains/apps/IDEA-U/ch-1/191.7479.19/jre64/lib/tools.jar",
		"-Didea.paths.selector=IntelliJIdea2019.1",
		"com.intellij.idea.Main"}

	pycharmCustomProcess = []string{
		"/home/user/bin/my-python-ide/jre64/bin/java",
		"-classpath",
		"/home/user/bin/my-python-ide/lib/bootstrap.jar:/home/user/bin/my-python-ide/lib/extensions.jar:/home/user/bin/my-python-ide/lib/util.jar:/home/user/bin/my-python-ide/lib/jdom.jar:/home/user/bin/my-python-ide/lib/log4j.jar:/home/user/bin/my-python-ide/lib/trove4j.jar:/home/user/bin/my-python-ide/lib/jna.jar",
		"-Didea.paths.selector=PyCharmCE2019.1",
		"-Didea.platform.prefix=PyCharmCore",
		"com.intellij.idea.Main"}

	pycharmAnacondaCustomProcess = []string{
		"/home/user/bin/my-python-anaconda-ide/jbr/bin/java",
		"-classpath",
		"/home/user/bin/my-python-anaconda-ide/lib/bootstrap.jar:/home/user/bin/my-python-anaconda-ide/lib/extensions.jar:/home/user/bin/my-python-anaconda-ide/lib/util.jar:/home/user/bin/my-python-anaconda-ide/lib/jdom.jar:/home/user/bin/my-python-anaconda-ide/lib/log4j.jar:/home/user/bin/my-python-anaconda-ide/lib/trove4j.jar:/home/user/bin/my-python-anaconda-ide/lib/jna.jar",
		"-Didea.paths.selector=PyCharm2019.3",
		"-Didea.platform.prefix=Python",
		"com.intellij.idea.Main"}

	golandCustomProcess = []string{
		"/home/user/bin/my-goland-ide/jre64/bin/java",
		"-classpath",
		"/home/user/bin/my-goland-ide/lib/bootstrap.jar:/home/user/bin/my-goland-ide/lib/extensions.jar:/home/user/bin/my-goland-ide/lib/util.jar:/home/user/bin/my-goland-ide/lib/jdom.jar:/home/user/bin/my-goland-ide/lib/log4j.jar:/home/user/bin/my-goland-ide/lib/trove4j.jar:/home/user/bin/my-goland-ide/lib/jna.jar",
		"-Didea.paths.selector=GoLand2019.3",
		"-Didea.platform.prefix=GoLand",
		"com.intellij.idea.Main"}

	webstormCustomProcess = []string{
		"/home/user/bin/my-webstorm-ide/jre64/bin/java",
		"-classpath",
		"/home/user/bin/my-webstorm-ide/lib/bootstrap.jar:/home/user/bin/my-webstorm-ide/lib/extensions.jar:/home/user/bin/my-webstorm-ide/lib/util.jar:/home/user/bin/my-webstorm-ide/lib/jdom.jar:/home/user/bin/my-webstorm-ide/lib/log4j.jar:/home/user/bin/my-webstorm-ide/lib/trove4j.jar:/home/user/bin/my-webstorm-ide/lib/jna.jar",
		"-Didea.paths.selector=WebStorm2019.3",
		"-Didea.platform.prefix=WebStorm",
		"com.intellij.idea.Main"}
)

// findManagerByIDTest locates a manager by ID, only for tests
func findManagerByIDTest(id string, process process.LinuxProcessManager) (editor.Plugin, error) {
	managers, err := NewJetBrainsManagers(process, true)
	if err != nil {
		return nil, err
	}

	for _, mgr := range managers {
		if mgr.ID() == id {
			return mgr, nil
		}
	}

	return nil, kiteerrors.New("unable for find JetBrains plugin manager with ID %s", id)
}

func replaceAll(data []string, search, replacement string) []string {
	var result []string
	for _, v := range data {
		result = append(result, strings.Replace(v, search, replacement, -1))
	}
	return result
}

func TestInstallConfig(t *testing.T) {
	mockProcessManager := &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			if name == "idea.sh" || name == "pycharm.sh" {
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
			if name == "idea.sh" || name == "pycharm.sh" {
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
			if name == "idea.sh" || name == "goland.sh" {
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
			if name == "webstorm.sh" {
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

	ideDir := setupIDEInstallation(t, "IU-193.4574.5", baseDir, "idea", "IU-193.4574.5")
	scriptPath := createIdeaScript(t, ideDir, "idea.sh")
	// add parents of idea.sh and pycharm.sh into $PATH
	defer updatePathEnv(t, filepath.Dir(scriptPath))()

	mgr, err := newIntelliJTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	// tests the basic install flow with an installation detected from $PATH
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "IntelliJIdea2019.3")
}

func TestPycharmFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "PC-193.4574.5", baseDir, "idea", "PC-193.4574.5")
	scriptPath := createIdeaScript(t, ideDir, "pycharm.sh")
	// add parents of idea.sh and pycharm.sh into $PATH
	defer updatePathEnv(t, filepath.Dir(scriptPath))()

	mgr, err := newPyCharmTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	// tests the basic install flow with an installation detected from $PATH
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "PyCharmCE2019.3")
}

func TestGoLandFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-goland-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "GO-193.4574.5", baseDir, "idea", "GO-193.4574.5")
	scriptPath := createIdeaScript(t, ideDir, "goland.sh")
	// add parents of idea.sh and goland.sh into $PATH
	defer updatePathEnv(t, filepath.Dir(scriptPath))()

	mgr, err := newGoLandTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	// tests the basic install flow with an installation detected from $PATH
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "GoLand2019.3")
}

func TestWebStormFlow(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-webstorm-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "WS-193.4574.5", baseDir, "idea", "WS-193.4574.5")
	scriptPath := createIdeaScript(t, ideDir, "webstorm.sh")
	// add parents of idea.sh and goland.sh into $PATH
	defer updatePathEnv(t, filepath.Dir(scriptPath))()

	mgr, err := newWebStormTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	// tests the basic install flow with an installation detected from $PATH
	testBasicInstallFlow(t, mgr, baseDir, ideDir, "WebStorm2019.3")
}

func Test2019_3(t *testing.T) {
	skipMarketplaceDownloads(t)

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
	require.Len(t, pluginJars, 1, "plugin must be installed")
}

func Test2020_1(t *testing.T) {
	skipMarketplaceDownloads(t)

	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "PC-201.1234.5", baseDir, "pycharm", "PC-201.1234.5")
	mgr, err := newPyCharmTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	// mgr uses userHome, which is equal to basedir in this test
	pluginDir := filepath.Join(baseDir, ".local", "share", "JetBrains", "PyCharmCE2020.1", pluginDirName)
	err = os.MkdirAll(pluginDir, 0700)
	require.NoError(t, err)

	// we've not yet set XDG_CONFIG_LOCATION, we're using userhome/.config as fallback
	err = mgr.Install(context.Background(), ideDir)
	require.NoError(t, err)

	// make sure that the plugin for was installed at the correct location
	pluginJars, err := filepath.Glob(fmt.Sprintf("%s/*/kite-pycharm-*.jar", pluginDir))
	require.NoError(t, err)
	require.Len(t, pluginJars, 1, "plugin must be installed")

	// now, set XDG_DATA_HOME and try again
	prevXDGValue, _ := os.LookupEnv("XDG_DATA_HOME")
	xdgDir := filepath.Join(baseDir, "my-xdg-config", "data")
	configDir := filepath.Join(xdgDir, "JetBrains", "PyCharmCE2020.1")
	err = os.MkdirAll(configDir, 0700)
	require.NoError(t, err)
	defer os.Setenv("XDG_DATA_HOME", prevXDGValue)
	os.Setenv("XDG_DATA_HOME", xdgDir)

	err = mgr.Install(context.Background(), ideDir)
	require.NoError(t, err)

	// make sure that the plugin for was installed at the correct location
	pluginJars, err = filepath.Glob(fmt.Sprintf("%s/%s/*/kite-pycharm-*.jar", configDir, pluginDirName))
	require.NoError(t, err)
	require.Len(t, pluginJars, 1, "plugin must be installed")
}

func TestPyCharmAnaconda2019_3(t *testing.T) {
	skipMarketplaceDownloads(t)

	baseDir, cleanup := shared.SetupTempDir(t, "kite-pycharm-anaconda-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "PYA-193.4574.5", baseDir, "idea", "PYA-193.4574.5")
	mgr, err := newPyCharmTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	pluginDir := setupPluginsDir(t, baseDir, "PyCharm2019.3")

	err = mgr.Install(context.Background(), ideDir)
	require.NoError(t, err)

	// make sure that the plugin for 2019.3+ was installed
	pluginJars, err := filepath.Glob(fmt.Sprintf("%s/*/kite-pycharm-*.jar", pluginDir))
	require.NoError(t, err)
	require.Len(t, pluginJars, 1, "plugin must be installed")
}

func TestIntelliJInstallWhileRunning(t *testing.T) {
	skipMarketplaceDownloads(t)

	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")

	// running IDE is added to $PATH
	runningIDE := setupIDEInstallation(t, "IU-193.4574.5", baseDir, "idea", "IU-193.4574.5")
	scriptPathRunning := createIdeaScript(t, runningIDE, "idea.sh")
	// add parents of idea.sh and pycharm.sh into $PATH
	defer updatePathEnv(t, filepath.Dir(scriptPathRunning))()

	// stopped IDE is setup as a toolbox installation
	stoppedIDE := setupIDEInstallation(t, "IC-201.4574.5", toolboxDir, "apps", "IDEA-C", "ch0", "201.4574.5")

	procMgr := &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			return name == scriptPathRunning, nil
		},
	}
	mgr, err := newIntelliJTestManager(baseDir, toolboxDir, procMgr, true)
	require.NoError(t, err)

	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.EqualValues(t, 2, len(editors), "expected that both running and stopped IDEs are detected")

	err = mgr.Install(context.Background(), runningIDE)
	require.Error(t, err, "install must fail when the particular IntelliJ IDE is running")

	// plugin dir is required for a successful installation
	setupNewPluginsDir(t, mgr.(*linuxJetBrains).userHome, "IdeaIC2020.1")
	err = mgr.Install(context.Background(), stoppedIDE)
	require.NoError(t, err, "install must succeed when the particular IntelliJ IDE isn't running")
}

func TestCheckSnapsFolder(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	mgr, err := newIntelliJTestManager(baseDir, "", &process.MockManager{}, true)
	mgr.(*linuxJetBrains).snapDirPrefix = "intellij-idea-*"
	require.NoError(t, err)

	// Create $tempDir/snap/intelliJ-idea-community
	err = os.MkdirAll(filepath.Join(baseDir, snapDir, "intellij-idea-community"), 0700)
	require.NoError(t, err)

	require.Equal(t,
		[]string{filepath.Join(baseDir, snapDir, "intellij-idea-community", "current")},
		mgr.(*linuxJetBrains).checkSnapsFolder(filepath.Join(baseDir, snapDir)))
}

func TestCheckStandaloneFolder(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	mgr, err := newPyCharmTestManager(baseDir, "", &process.MockManager{}, true)
	mgr.(*linuxJetBrains).standaloneDirPrefix = "pycharm*"
	require.NoError(t, err)

	// Create $tempDir/opt/pycharm-community-2019.1.1
	err = os.MkdirAll(filepath.Join(baseDir, standaloneDir, "pycharm-community-2019.1.1"), 0700)
	require.NoError(t, err)

	require.Equal(t,
		[]string{filepath.Join(baseDir, standaloneDir, "pycharm-community-2019.1.1")},
		mgr.(*linuxJetBrains).checkStandaloneFolders(filepath.Join(baseDir, standaloneDir)))
}

func TestDesktopFiles(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	mgr, err := newPyCharmTestManager(baseDir, "", &process.MockManager{}, true)
	require.NoError(t, err)

	// setup a dummy PyCharm install
	ideDir := setupIDEInstallation(t, "PC-181.4574.5", baseDir, "pycharm", "PC-181.4574.5")
	pycharmScriptPath := createIdeaScript(t, ideDir, "pycharm")

	// setup a dummy IntelliJ install, this must not be detected by the pycharm manager
	ideDirIntelliJ := setupIDEInstallation(t, "IU-181.4574.5", baseDir, "intellij", "IU-181.4574.5")
	intellijScriptPath := createIdeaScript(t, ideDirIntelliJ, "intellij")

	// valid desktop file: create $userHome/.local/share/applications/idea-pycharm.desktop
	desktopFileDir := filepath.Join(baseDir, ".local", "share", "applications")
	err = os.MkdirAll(desktopFileDir, 0700)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(desktopFileDir, "idea-pycharm.desktop"), []byte(fmt.Sprintf("[Desktop Entry]\nExec=%s", pycharmScriptPath)), 0600)
	require.NoError(t, err)

	// valid desktop file referencing the intellij install (must not be in the result)
	err = ioutil.WriteFile(filepath.Join(desktopFileDir, "idea-intellij.desktop"), []byte(fmt.Sprintf("[Desktop Entry]\nExec=%s", intellijScriptPath)), 0600)
	require.NoError(t, err)

	// invalid desktop file: create $userHome/.local/share/applications/idea-pycharm.desktop which points to /opt/dir/
	err = ioutil.WriteFile(filepath.Join(desktopFileDir, "invalid.desktop"), []byte("[Desktop Entry]\nExec=/opt/dir/script.sh"), 0600)
	require.NoError(t, err)

	// only supported products must be retained, checkDesktopFiles() is checking the product ID
	require.Equal(t, []string{ideDir}, mgr.(*linuxJetBrains).checkDesktopFiles())

	// only the valid PyCharm install must be kept in the detected locations
	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 1)
	require.EqualValues(t, ideDir, editors[0].Path)
}

func TestDetectRunningPyCharm(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	intellij := replaceAll(intellijToolboxProcess, "/opt/", baseDir+"/")
	pycharm := replaceAll(pycharmCustomProcess, "/home/user", baseDir+"/")
	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("pycharm", pycharm[0], pycharm),
				process.NewMockProcess("intellij", intellij[0], intellij),
			}, nil
		},
	}

	setupIDEInstallation(t, "PC-2019.1", baseDir, "bin/my-python-ide")

	mgr, err := findManagerByIDTest(pycharmID, processMgr)
	require.NoError(t, err)
	list, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.EqualValues(t, baseDir+"/bin/my-python-ide", list[0])
}

func TestDetectRunningPyCharmAnaconda(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	intellij := replaceAll(intellijToolboxProcess, "/opt/", baseDir+"/")
	pycharm := replaceAll(pycharmAnacondaCustomProcess, "/home/user", baseDir+"/")
	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("pycharm", pycharm[0], pycharm),
				process.NewMockProcess("intellij", intellij[0], intellij),
			}, nil
		},
	}

	setupIDEInstallation(t, "PYA-2019.3", baseDir, "bin/my-python-anaconda-ide")

	mgr, err := findManagerByIDTest(pycharmID, processMgr)
	require.NoError(t, err)
	list, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.EqualValues(t, baseDir+"/bin/my-python-anaconda-ide", list[0])
}

func TestDetectRunningGoLand(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	intellij := replaceAll(intellijToolboxProcess, "/opt/", baseDir+"/")
	goland := replaceAll(golandCustomProcess, "/home/user", baseDir+"/")
	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("goland", goland[0], goland),
				process.NewMockProcess("intellij", intellij[0], intellij),
			}, nil
		},
	}

	setupIDEInstallation(t, "GO-2019.1", baseDir, "bin/my-goland-ide")

	mgr, err := findManagerByIDTest(golandID, processMgr)
	require.NoError(t, err)
	list, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.EqualValues(t, baseDir+"/bin/my-goland-ide", list[0])
}

func TestDetectRunningWebStorm(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	intellij := replaceAll(intellijToolboxProcess, "/opt/", baseDir+"/")
	webstorm := replaceAll(webstormCustomProcess, "/home/user", baseDir+"/")
	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("webstorm", webstorm[0], webstorm),
				process.NewMockProcess("intellij", intellij[0], intellij),
			}, nil
		},
	}

	setupIDEInstallation(t, "WS-2019.1", baseDir, "bin/my-webstorm-ide")

	mgr, err := findManagerByIDTest(webstormID, processMgr)
	require.NoError(t, err)
	list, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.EqualValues(t, baseDir+"/bin/my-webstorm-ide", list[0])
}

func TestDetectRunningIntelliJ(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	intellij := replaceAll(intellijToolboxProcess, "/opt/", baseDir+"/")
	pycharm := replaceAll(pycharmCustomProcess, "/home/user", baseDir+"/")
	processMgr := &process.MockManager{
		ListData: func() (process.List, error) {
			return []process.Process{
				process.NewMockProcess("pycharm", pycharm[0], pycharm),
				process.NewMockProcess("intellij", intellij[0], intellij),
			}, nil
		},
	}

	setupIDEInstallation(t, "IU-191.7479.19", baseDir, "/JetBrains/apps/IDEA-U/ch-1/191.7479.19")

	mgr, err := findManagerByIDTest(intellijID, processMgr)
	require.NoError(t, err)
	list, err := mgr.DetectRunningEditors(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.EqualValues(t, baseDir+"/JetBrains/apps/IDEA-U/ch-1/191.7479.19", list[0])
}

// https://github.com/kiteco/kiteco/issues/8868
func TestSymlinkedSnapsFolder(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "")
	defer cleanup()

	mgr, err := newIntelliJTestManager(baseDir, "", &process.MockManager{
		IsProcessRunningData: func(name string) (bool, error) {
			log.Printf("running: %s", name)
			return false, nil
		},
	}, true)
	mgr.(*linuxJetBrains).snapDirPrefix = "intellij-idea-*"
	require.NoError(t, err)

	// Create $tempDir/snap/intelliJ-idea-community/169
	installDir := filepath.Join(baseDir, snapDir, "intellij-idea-community", "169")
	err = os.MkdirAll(installDir, 0700)
	require.NoError(t, err)

	// $tempDir/snap/intelliJ-idea-community/current -> $tempDir/snap/intelliJ-idea-community/169
	currentDir := filepath.Join(baseDir, snapDir, "intellij-idea-community", "current")
	err = os.Symlink(installDir, currentDir)
	require.NoError(t, err)

	require.Equal(t,
		[]string{installDir},
		mgr.(*linuxJetBrains).checkSnapsFolder(filepath.Join(baseDir, snapDir)))
}

func TestOpenFile(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "IU-193.4574.5", baseDir, "idea", "IU-193.4574.5")
	scriptPath := createIdeaScript(t, ideDir, "idea.sh")
	// add parents of idea.sh and pycharm.sh into $PATH, this allows detection
	defer updatePathEnv(t, filepath.Dir(scriptPath))()

	mgr, err := newIntelliJTestManager(baseDir, filepath.Join(baseDir, "toolbox"), &process.MockManager{
		RunResult: func(name string, arg ...string) ([]byte, error) {
			if name == scriptPath && shared.StringsContain(arg, filepath.Join(baseDir, "a.txt")) {
				return []byte(""), nil
			}
			return nil, errors.Errorf("not found")
		},
		CustomDir: func() (string, error) {
			return baseDir, nil
		},
	}, true)
	require.NoError(t, err)

	_, err = mgr.OpenFile(context.Background(), "IU", "", filepath.Join(baseDir, "a.txt"), 2)
	require.NoError(t, err)
}

func TestInstalledProductIDsGoland(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-goland-ide")
	defer cleanup()

	ideDir := setupIDEInstallation(t, "GO-193.4574.5", baseDir, "idea", "GO-193.4574.5")
	scriptPath := createIdeaScript(t, ideDir, "goland.sh")
	// add parents of idea.sh and goland.sh into $PATH
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
	err = ioutil.WriteFile(scriptPath, []byte("#dummy script"), 0700)
	require.NoError(t, err)
	return scriptPath
}
