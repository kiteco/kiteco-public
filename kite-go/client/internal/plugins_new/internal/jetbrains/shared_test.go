package jetbrains

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/jetbrains/buildnumber"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindProductVersion(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	buildFile := filepath.Join(baseDir, "build.txt")
	version, err := findProductVersion(buildFile)
	require.Error(t, err, "a missing build.txt file must result in an error")

	err = ioutil.WriteFile(buildFile, []byte("IU-182.1234.42"), 0600)
	require.NoError(t, err)
	version, err = findProductVersion(buildFile)
	require.NoError(t, err)
	require.EqualValues(t, "IU-182.1234.42", version.String())
	require.EqualValues(t, "IU", version.ProductID)
	require.EqualValues(t, 182, version.Branch)
	require.EqualValues(t, 1234, version.Build)
	require.EqualValues(t, ".42", version.Remainder)

	err = ioutil.WriteFile(buildFile, []byte("invalid-build-id"), 0600)
	require.NoError(t, err)
	_, err = findProductVersion(buildFile)
	require.Error(t, err, "incorrect content of build.txt must result in an error")
}

func TestStringsContain(t *testing.T) {
	assert.True(t, shared.StringsContain([]string{"a", "b"}, "a"))
	assert.True(t, shared.StringsContain([]string{"a", "b"}, "b"))

	assert.False(t, shared.StringsContain([]string{}, "a"))
	assert.False(t, shared.StringsContain([]string{"b"}, "a"))
	assert.False(t, shared.StringsContain([]string{"b"}, ""))
}

func TestMapsStrings(t *testing.T) {
	assert.EqualValues(t, []string{"a123", "b123"}, shared.MapStrings([]string{"a", "b"}, func(e string) string {
		return e + "123"
	}))
}

func TestFindEditors(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij")
	defer cleanup()

	dir1 := setupIDEInstallation(t, "IU-181.1234.5", baseDir, "IU-181.1234")
	dir2 := setupIDEInstallation(t, "IU-171.1234.5", baseDir, "IU-171.1234")
	dir3 := setupIDEInstallation(t, "IU-141.1234.5", baseDir, "IU-141.1234")

	// create dummy dirs without build.txt files
	dir4 := filepath.Join(baseDir, "IC-191.1234")
	err := os.MkdirAll(dir4, 0700)
	require.NoError(t, err)

	mgr, err := newIntelliJTestManager(baseDir, "", &process.MockManager{}, true)
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), []string{dir1, dir2, dir3, dir4}, mgr)

	require.EqualValues(t, 3, len(editors))
	require.EqualValues(t, dir3, editors[0].Path, "expecting editors to be sorted by path")
	require.EqualValues(t, dir2, editors[1].Path, "expecting editors to be sorted by path")
	require.EqualValues(t, dir1, editors[2].Path, "expecting editors to be sorted by path")

	require.EqualValues(t, "IU-141.1234.5", editors[0].Version, "expecting editors to be sorted by path")
	require.EqualValues(t, "IU-171.1234.5", editors[1].Version, "expecting editors to be sorted by path")
	require.EqualValues(t, "IU-181.1234.5", editors[2].Version, "expecting editors to be sorted by path")

	require.NotEmpty(t, editors[0].Compatibility, "expecting a message for the incompatible 141 version")
	require.NotEmpty(t, editors[1].Compatibility, "expecting a message for 171.x")
	require.NotEmpty(t, editors[2].Compatibility, "expecting a message for 181.x")
}

func TestIdeBasics(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	// test our test setup method
	procMgr := &process.MockManager{}
	intellij, err := newIntelliJTestManager(baseDir, "", procMgr, true)
	require.NoError(t, err)
	assert.EqualValues(t, intellijID, intellij.ID())
	assert.EqualValues(t, intellijName, intellij.Name())

	// test the exported method
	intellij, err = newIntelliJTestManager("", "", procMgr, true)
	require.NoError(t, err)
	assert.EqualValues(t, intellijID, intellij.ID())
	assert.EqualValues(t, intellijName, intellij.Name())

	// test our test setup method
	pycharm, err := newPyCharmTestManager(baseDir, "", procMgr, true)
	require.NoError(t, err)
	assert.EqualValues(t, pycharmID, pycharm.ID())
	assert.EqualValues(t, pycharmName, pycharm.Name())

	// test the exported method
	pycharm, err = newPyCharmTestManager("", "", procMgr, true)
	require.NoError(t, err)
	assert.EqualValues(t, pycharmID, pycharm.ID())
	assert.EqualValues(t, pycharmName, pycharm.Name())
}

func TestIsIntelliJInstalled(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	ideaUltimateDir := setupIDEInstallation(t, "IU-172.4574.5", toolboxDir, "IDEA", "ch0", "IU-172.4574.5")
	ideaCommunityDir := setupIDEInstallation(t, "IC-172.4574.5", toolboxDir, "IDEA", "ch1", "IC-172.4574.5")

	kitePluginDirUltimate := setupPluginsDir(t, baseDir, "IntelliJIdea2017.2")
	kitePluginDirCommunity := setupPluginsDir(t, baseDir, "IdeaIC2017.2")

	procMgr := &process.MockManager{}
	mgr, err := newIntelliJTestManager(baseDir, toolboxDir, procMgr, true)
	require.NoError(t, err)

	assert.False(t, mgr.IsInstalled(context.Background(), ideaUltimateDir), "without an existing plugin dir isInstalled must return false for %s", ideaUltimateDir)
	assert.False(t, mgr.IsInstalled(context.Background(), ideaCommunityDir), "without an existing plugin dir isInstalled must return false for %s", ideaCommunityDir)

	err = os.MkdirAll(kitePluginDirUltimate, 0700)
	require.NoError(t, err)
	assert.True(t, mgr.IsInstalled(context.Background(), ideaUltimateDir), "with an existing plugin dir isInstalled must return true for %s and plugin dir %s", ideaUltimateDir, kitePluginDirUltimate)
	assert.False(t, mgr.IsInstalled(context.Background(), ideaCommunityDir), "without an existing community plugin dir isInstalled must return false for %s", ideaCommunityDir)

	err = os.MkdirAll(kitePluginDirCommunity, 0700)
	require.NoError(t, err)
	assert.True(t, mgr.IsInstalled(context.Background(), ideaCommunityDir), "with an existing plugin dir isInstalled must return true for %s", ideaCommunityDir)
}

func TestIsPyCharmInstalled(t *testing.T) {
	baseDir, cleanup := shared.SetupTempDir(t, "kite-intellij-ide")
	defer cleanup()

	toolboxDir := filepath.Join(baseDir, "toolbox")
	pycharmProDir := setupIDEInstallation(t, "PY-172.4574.5", toolboxDir, "PyCharm", "ch0", "PY-172.4574.5")
	pycharmCommunityDir := setupIDEInstallation(t, "PC-172.4574.5", toolboxDir, "PyCharm", "ch1", "PC-172.4574.5")

	kitePluginDirPro := setupPluginsDir(t, baseDir, "PyCharm2017.2")
	kitePluginDirCommunity := setupPluginsDir(t, baseDir, "PyCharmCE2017.2")

	procMgr := &process.MockManager{}
	mgr, err := newPyCharmTestManager(baseDir, toolboxDir, procMgr, true)
	require.NoError(t, err)

	assert.False(t, mgr.IsInstalled(context.Background(), pycharmProDir), "without an existing plugin dir isInstalled must return false for %s", pycharmProDir)
	assert.False(t, mgr.IsInstalled(context.Background(), pycharmCommunityDir), "without an existing plugin dir isInstalled must return false for %s", pycharmCommunityDir)

	err = os.MkdirAll(kitePluginDirPro, 0700)
	require.NoError(t, err)
	assert.True(t, mgr.IsInstalled(context.Background(), pycharmProDir), "with an existing plugin dir isInstalled must return true for %s", pycharmProDir)
	assert.False(t, mgr.IsInstalled(context.Background(), pycharmCommunityDir), "without an existing community plugin dir isInstalled must return false for %s", pycharmCommunityDir)

	err = os.MkdirAll(kitePluginDirCommunity, 0700)
	require.NoError(t, err)
	assert.True(t, mgr.IsInstalled(context.Background(), pycharmCommunityDir), "with an existing plugin dir isInstalled must return true for %s", pycharmCommunityDir)
}

func TestDetectToolboxFolders(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-jetbrains-ide")
	defer cleanup()

	toolboxDir := filepath.Join(dir, "toolbox")
	ideaDir1 := setupIDEInstallation(t, "IC-172.4574.5", toolboxDir, "apps", "IDEA-C", "ch-0", "172.4574.5")
	ideaDir2 := setupIDEInstallation(t, "IU-183.5429.31", toolboxDir, "apps", "IDEA-U", "ch-0", "183.5429.31")

	pycharmDir1 := setupIDEInstallation(t, "PC-183.5429.31", toolboxDir, "apps", "PyCharm-C", "ch-0", "183.5429.31")
	pycharmDir2 := setupIDEInstallation(t, "PY-183.5429.31", toolboxDir, "apps", "PyCharm-P", "ch-0", "183.5429.31")

	pycharmMgr, err := newPyCharmTestManager(dir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)

	intellijMgr, err := newIntelliJTestManager(dir, toolboxDir, &process.MockManager{}, true)
	require.NoError(t, err)

	// pycharm detection, must detect pycharm folders only
	pycharmPaths, err := pycharmMgr.DetectEditors(context.Background())
	pycharmEditors := shared.MapEditors(context.Background(), pycharmPaths, pycharmMgr)
	require.NoError(t, err)
	sortEditorsByVersion(pycharmEditors)

	require.EqualValues(t, 2, len(pycharmEditors))
	require.EqualValues(t, pycharmDir1, pycharmEditors[0].Path)
	require.EqualValues(t, "PC-183.5429.31", pycharmEditors[0].Version)
	require.EqualValues(t, pycharmDir2, pycharmEditors[1].Path)
	require.EqualValues(t, "PY-183.5429.31", pycharmEditors[1].Version)

	// intellij detection, must detect intellij folders only
	intellijPaths, err := intellijMgr.DetectEditors(context.Background())
	require.NoError(t, err)
	intellijEditors := shared.MapEditors(context.Background(), intellijPaths, intellijMgr)
	sortEditorsByVersion(intellijEditors)

	require.EqualValues(t, 2, len(intellijEditors))
	require.EqualValues(t, ideaDir1, intellijEditors[0].Path)
	require.EqualValues(t, "IC-172.4574.5", intellijEditors[0].Version)
	require.EqualValues(t, ideaDir2, intellijEditors[1].Path)
	require.EqualValues(t, "IU-183.5429.31", intellijEditors[1].Version)
}

func TestInstalledPluginVersion(t *testing.T) {
	err := extractWithVersion(t, "")
	require.NoError(t, err, "the update must be successful if the installed version is too old to have a version file")

	err = extractWithVersion(t, "1.6.0")
	require.NoError(t, err, "the update must be successful if the bundled version is newer than the installed version")

	err = extractWithVersion(t, "100.0.0")
	require.Error(t, err, "update must be skipped if the installed version is newer than the bundled version")
	require.EqualValues(t, shared.AlreadyInstalledError, err)
}

func extractWithVersion(t *testing.T, installedVersion string) error {
	skipMarketplaceDownloads(t)

	parentDir, cleanup := shared.SetupTempDir(t, "kite-jetbrains-ide")
	defer cleanup()

	pluginDir := filepath.Join(parentDir, pluginDirName)
	_ = os.Mkdir(pluginDir, 0700)

	if installedVersion != "" {
		err := ioutil.WriteFile(filepath.Join(pluginDir, "pluginVersion.txt"), []byte(installedVersion), 0600)
		require.NoError(t, err)
	}

	build, err := buildnumber.FromString("IU-201.1234.5")
	require.NoError(t, err)
	return extractPluginData(build, parentDir, parentDir, false)
}

func testBasicInstallFlow(t *testing.T, mgr editor.Plugin, baseDir, ideDir, userSettingsDir string) {
	skipMarketplaceDownloads(t)

	setupPluginsDir(t, baseDir, userSettingsDir)

	require.False(t, mgr.IsInstalled(context.Background(), ideDir), "expected that the plugin isn't installed initially")

	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.EqualValues(t, 1, len(editors), "expected that %s is detected as valid ide installation", ideDir)
	require.EqualValues(t, ideDir, editors[0].Path, "expected that %s is detected as valid ide installation", ideDir)

	err = mgr.Install(context.Background(), ideDir)
	require.NoError(t, err)
	require.True(t, mgr.IsInstalled(context.Background(), ideDir))

	err = mgr.Update(context.Background(), ideDir)
	require.NoError(t, err)
	require.True(t, mgr.IsInstalled(context.Background(), ideDir))

	err = mgr.Uninstall(context.Background(), ideDir)
	require.NoError(t, err)
	require.False(t, mgr.IsInstalled(context.Background(), ideDir))
}

func Test_ShouldInstall(t *testing.T) {
	build, err := buildnumber.FromString("IC-201.1234.5")
	require.NoError(t, err)

	tempDir, err := ioutil.TempDir("", "kite")
	require.NoError(t, err)
	versionFilePath := filepath.Join(tempDir, "version.txt")
	err = ioutil.WriteFile(versionFilePath, []byte("1.8.15"), 0600)
	require.NoError(t, err)

	remoteVersion := func(version string) func(buildnumber.BuildNumber) (string, error) {
		return func(number buildnumber.BuildNumber) (string, error) {
			return version, nil
		}
	}

	shouldInstall := shouldInstallNewRemoteVersion(build, versionFilePath, remoteVersion("1.7.0"))
	require.False(t, shouldInstall, "an older version should not be installed")
	shouldInstall = shouldInstallNewRemoteVersion(build, versionFilePath, remoteVersion("1.8.14"))
	require.False(t, shouldInstall, "an older version should not be installed")

	shouldInstall = shouldInstallNewRemoteVersion(build, versionFilePath, remoteVersion("1.8.15"))
	require.False(t, shouldInstall, "the same version shouldn't be installed again")

	shouldInstall = shouldInstallNewRemoteVersion(build, versionFilePath, remoteVersion("1.8.16"))
	require.True(t, shouldInstall, "a newer patch version should be installed")

	shouldInstall = shouldInstallNewRemoteVersion(build, versionFilePath, remoteVersion("1.9.0"))
	require.True(t, shouldInstall, "a new minor version should be installed")

	shouldInstall = shouldInstallNewRemoteVersion(build, versionFilePath, remoteVersion("2.1.0"))
	require.True(t, shouldInstall, "a new major version should be installed")
}

// creates the parent dir which will contain the kite-pycharm directory
// returns the path to the plugin directory, but does not create it
func setupPluginsDir(t *testing.T, parentDir string, fullIdeName string) string {
	var dir string
	if runtime.GOOS == "darwin" {
		dir = filepath.Join(parentDir, "Library", "Application Support", fullIdeName)
	} else {
		dir = filepath.Join(parentDir, fmt.Sprintf(".%s", fullIdeName), "config", "plugins")
	}

	err := os.MkdirAll(dir, 0700)
	require.NoError(t, err)
	return filepath.Join(dir, pluginDirName)
}

// creates the parent dir which will contain the kite-pycharm directory, using the new locations of 2020.1+
// returns the path to the plugin directory, but does not create it
func setupNewPluginsDir(t *testing.T, parentDir string, fullIdeName string) string {
	var dir string
	if runtime.GOOS == "darwin" {
		dir = filepath.Join(parentDir, "Library", "Application Support", fullIdeName)
	} else {
		dir = filepath.Join(parentDir, ".local", "share", "JetBrains", fullIdeName)
	}

	err := os.MkdirAll(dir, 0700)
	require.NoError(t, err)
	return filepath.Join(dir, pluginDirName)
}

// updates the PATH environment and returns a function which reverts the change (useful as deferred call)
func updatePathEnv(t *testing.T, prefixEntry ...string) func() {
	oldValue := os.Getenv("PATH")

	err := os.Setenv("PATH", strings.Join(append(prefixEntry, oldValue), string(os.PathListSeparator)))
	require.NoError(t, err)

	return func() {
		os.Setenv("PATH", oldValue)
	}
}

func sortEditorsByVersion(editors []system.Editor) {
	sort.Slice(editors, func(i, j int) bool {
		return strings.Compare(editors[i].Version, editors[j].Version) < 0
	})
}

// modify this to run tests locally, which download from the JetBrains marketplace
// automated tests should never download from the public marketplace
func skipMarketplaceDownloads(t *testing.T) {
	t.Skip("skipping test with JetBrains plugin marketplace download")
}
