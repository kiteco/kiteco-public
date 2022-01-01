package jetbrains

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/stretchr/testify/require"
)

func newIntelliJTestManager(baseDir string, toolboxDir string, process *process.MockManager, betaChannel bool) (editor.Plugin, error) {
	mgr, err := findManagerByIDTest(intellijID, process)
	if err != nil {
		return nil, err
	}

	manager := mgr.(*linuxJetBrains)
	manager.betaChannel = betaChannel
	manager.userHome = baseDir
	manager.toolboxDirs = []string{toolboxDir}
	manager.snapDirPrefix = ""
	manager.standaloneDirPrefix = ""
	manager.desktopFileLocations = []string{filepath.Join(baseDir, ".local", "share", "applications")}
	return manager, nil
}

func newPyCharmTestManager(baseDir string, toolboxDir string, process *process.MockManager, betaChannel bool) (editor.Plugin, error) {
	mgr, err := findManagerByIDTest(pycharmID, process)
	if err != nil {
		return nil, err
	}

	manager := mgr.(*linuxJetBrains)
	manager.betaChannel = betaChannel
	manager.userHome = baseDir
	manager.toolboxDirs = []string{toolboxDir}
	manager.snapDirPrefix = ""
	manager.standaloneDirPrefix = ""
	manager.desktopFileLocations = []string{filepath.Join(baseDir, ".local", "share", "applications")}
	return manager, nil
}

func newGoLandTestManager(baseDir string, toolboxDir string, process *process.MockManager, betaChannel bool) (editor.Plugin, error) {
	mgr, err := findManagerByIDTest(golandID, process)
	if err != nil {
		return nil, err
	}

	manager := mgr.(*linuxJetBrains)
	manager.betaChannel = betaChannel
	manager.userHome = baseDir
	manager.toolboxDirs = []string{toolboxDir}
	manager.snapDirPrefix = ""
	manager.standaloneDirPrefix = ""
	manager.desktopFileLocations = []string{filepath.Join(baseDir, ".local", "share", "applications")}
	return manager, nil
}

func newWebStormTestManager(baseDir string, toolboxDir string, process *process.MockManager, betaChannel bool) (editor.Plugin, error) {
	mgr, err := findManagerByIDTest(webstormID, process)
	if err != nil {
		return nil, err
	}
	manager := mgr.(*linuxJetBrains)
	manager.betaChannel = betaChannel
	manager.userHome = baseDir
	manager.toolboxDirs = []string{toolboxDir}
	manager.snapDirPrefix = ""
	manager.standaloneDirPrefix = ""
	manager.desktopFileLocations = []string{filepath.Join(baseDir, ".local", "share", "applications")}
	return manager, nil
}

// setupIDEInstallation writes build.txt at the common location of the current OS and returns the ide home path
func setupIDEInstallation(t *testing.T, buildID string, path ...string) string {
	ideDir := filepath.Join(path...)
	err := os.MkdirAll(ideDir, 0700)
	require.NoError(t, err)

	buildFilePath := buildFileLocation(ideDir)
	os.MkdirAll(filepath.Dir(buildFilePath), 0700)

	err = ioutil.WriteFile(buildFilePath, []byte(buildID), 0600)
	require.NoError(t, err)
	return ideDir
}
