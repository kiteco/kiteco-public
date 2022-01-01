package jetbrains

import (
	"archive/zip"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/jetbrains/buildnumber"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/version"
)

const (
	pluginDirName = "kite-pycharm"

	intellijID          = "intellij"
	intellijName        = "IntelliJ IDEA"
	intellijToolboxName = "IDEA"

	pycharmID          = "pycharm"
	pycharmName        = "PyCharm"
	pycharmToolboxName = "PyCharm"

	golandID          = "goland"
	golandName        = "GoLand"
	golandToolboxName = "Goland"

	webstormID          = "webstorm"
	webstormName        = "WebStorm"
	webstormToolboxName = "WebStorm"

	phpstormID          = "phpstorm"
	phpstormName        = "PhpStorm"
	phpstormToolboxName = "PhpStorm"

	riderID          = "rider"
	riderName        = "Rider"
	riderToolboxName = "Rider"

	clionID          = "clion"
	clionName        = "CLion"
	clionToolboxName = "CLion"

	rubymineID          = "rubymine"
	rubymineName        = "RubyMine"
	rubymineToolboxName = "RubyMine"

	androidStudioID          = "android-studio"
	androidStudioName        = "Android Studio"
	androidStudioToolboxName = "AndroidStudio"

	// only on macOS
	appcodeID = "appcode"
)

var (
	errBuildConfigMissing   = fmt.Errorf("build.txt file doesn't exist")
	intellijProductIDs      = []string{"IU", "IC", "IE"}
	pycharmProductIDs       = []string{"PY", "PC", "PE", "PYA", "PCA"}
	golandProductIDs        = []string{"GO"}
	webstormProductIDs      = []string{"WS"}
	phpstormProductIDs      = []string{"PS"}
	riderProductIDs         = []string{"RD"}
	clionProductIDs         = []string{"CL"}
	rubymineProductIDs      = []string{"RM"}
	androidStudioProductIDs = []string{"AI"}
)

func buildFileLocation(ideHome string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(ideHome, "Contents", "Resources", "build.txt")
	}
	return filepath.Join(ideHome, "build.txt")
}

// checkToolboxFolder returns all folders below the toolbox root path which match the given product ID (e.g. IDEA)
func checkToolboxFolder(toolboxDir string, productID string) []string {
	var pattern string
	if runtime.GOOS == "darwin" {
		// e.g. /Users/username/Library/Application\ Support/JetBrains/Toolbox/apps/IDEA-U/ch-0/163.15529.8/IntelliJ IDEA.app
		pattern = filepath.Join(toolboxDir, "apps", "*", "*", "*", "*.app")
	} else {
		// e.g. /Users/username/Library/Application\ Support/JetBrains/Toolbox/apps/IDEA-U/ch-0/163.15529.8
		pattern = filepath.Join(toolboxDir, "apps", "*", "*", "*")
	}
	return checkDirPattern(productID, pattern)
}

// checkDirPatter returns all folder below the pattern path which match the given product ID.
func checkDirPattern(productID string, pattern string) []string {
	dirs, err := filepath.Glob(pattern)
	if err != nil { // only error possible is ErrBadPattern
		log.Printf("Error pattern matching JetBrains installs: %s", err.Error())
		return nil
	}

	var filtered []string
	for _, dir := range dirs {
		if fs.DirExists(dir) && strings.Contains(dir, productID) {
			filtered = append(filtered, dir)
		}
	}
	return filtered
}

// findProductVersion gets the version of a particular install.
func findProductVersion(buildFilePath string) (buildnumber.BuildNumber, error) {
	bytes, err := ioutil.ReadFile(buildFilePath)
	if os.IsNotExist(err) {
		return buildnumber.BuildNumber{}, errBuildConfigMissing
	}
	if err != nil {
		return buildnumber.BuildNumber{}, err
	}
	return buildnumber.FromString(string(bytes))
}

func locateInstallRoot(path string, supportedProductIDs []string) (string, error) {
	dir := filepath.Dir(path)
	for {
		buildFilePath := buildFileLocation(dir)
		if fs.FileExists(buildFilePath) {
			content, err := ioutil.ReadFile(buildFilePath)
			if err != nil {
				return "", err
			}

			for _, id := range supportedProductIDs {
				if strings.HasPrefix(string(content), id+"-") {
					return dir, nil
				}
			}
		}

		parent := filepath.Dir(dir)
		// on Windows, filepath.Dir("C:|") returns "C:\"
		if dir == parent {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no installation foundFiles")
}

func findInstallByFsnotifier(cmdline, supportedProductIDs []string) string {
	if len(cmdline) > 0 {
		if strings.Contains(cmdline[0], "fsnotifier") {
			root, err := locateInstallRoot(cmdline[0], supportedProductIDs)
			if err == nil {
				return root
			}
		}
	}
	return ""
}

// findInstallByClasspath returns the installation root if the given cmdline contains a valid IntelliJ/PyCharm classpath
// argument. The root is calculated based on this value.
// If no valid cmdline was found, then an empty string is returned.
func findInstallByClasspath(cmdline, supportedProductIDs []string) string {
	// locate classpath, split, filter for bootstrap.jar and return first parent dir with build.txt in it
	for i, arg := range cmdline {
		if arg == "-classpath" && len(cmdline) > i+1 {
			// arg following -classpath
			classpath := cmdline[i+1]
			for _, entry := range strings.Split(classpath, ":") {
				if strings.Contains(entry, "bootstrap.jar") {
					if root, err := locateInstallRoot(entry, supportedProductIDs); err == nil {
						return root
					}
				}
			}
		}
	}
	return ""
}

func editorConfig(editorPath string) (system.Editor, buildnumber.BuildNumber, error) {
	v, err := findProductVersion(buildFileLocation(editorPath))
	if err != nil {
		return system.Editor{}, v, err
	}

	return system.Editor{
		Path:            editorPath,
		Version:         v.String(),
		Compatibility:   v.CompatibilityMessage(),
		RequiredVersion: v.RequiredVersion(),
	}, v, nil
}

// installOrUpdatePlugin tries to atomically update the plugin if it's already installed
// if it's not yet there, it installs the plugin
func installOrUpdatePlugin(editorPath, pluginParentDir, pluginDirName string, betaChannel bool) error {
	_, version, err := editorConfig(editorPath)
	if err != nil {
		return err
	}

	return shared.InstallOrUpdatePluginAssets(pluginParentDir, pluginDirName, func(targetParentDir string) error {
		// extraction target "targetParentDir" and the current install location "pluginParentDir"
		// aren't necessarily the same. targetParentDir could be temporary directory for an atomic update.
		return extractPluginData(version, targetParentDir, pluginParentDir, betaChannel)
	})
}

func extractPluginData(ideBuild buildnumber.BuildNumber, targetParentDir, installedPluginParentDir string, betaChannel bool) error {
	// 2019.2 and earlier isn't supported anymore
	if ideBuild.Branch <= 192 {
		return errors.Errorf("Version 2019.2 and earlier isn't supported")
	}

	// make sure that we don't install an older version over an installed, newer version
	// pluginVersion.txt was added for version 1.7.0 when the plugin was published on the marketplace
	// a missing pluginVersion.txt file indicates that the version is 1.6.x or older
	if !betaChannel && !shouldInstallNewRemoteVersion(ideBuild, filepath.Join(installedPluginParentDir, pluginDirName, "pluginVersion.txt"), requestLatestAvailableVersion) {
		return shared.AlreadyInstalledError
	}

	tempFile, err := ioutil.TempFile("", "kite-pycharm-*.zip")
	if err != nil {
		return err
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	if betaChannel {
		// staged beta plugin
		if err := downloadBetaChannelPlugin(tempFile); err != nil {
			return err
		}
	} else {
		// production build, download zip, closes tempFile
		if err := downloadKitePluginZip(tempFile, ideBuild); err != nil {
			rollbar.Error(err)
			return err
		}
	}

	// unzip the zip into pluginParentDir
	zipReader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		rollbar.Error(errors.Errorf("error opening zip file: %s", err.Error()))
		return err
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		if !strings.HasPrefix(f.Name, pluginDirName) {
			err := errors.Errorf("unexpected entry in plugin zip", f.Name)
			rollbar.Error(err)
			return err
		}

		if err := extractZipEntry(f, targetParentDir); err != nil {
			rollbar.Error(errors.Errorf("failed to extract zip file entry: %s", err.Error()))
			return err
		}
	}
	return nil
}

// shouldInstallNewRemoteVersion returns if an update of the Kite plugin is required.
// Please note, that this executes an HTTP request.
func shouldInstallNewRemoteVersion(ideBuild buildnumber.BuildNumber, installedVersionFile string, latestRemoteVersion func(number buildnumber.BuildNumber) (string, error)) bool {
	fileData, err := ioutil.ReadFile(installedVersionFile)
	if err != nil {
		// an old version without the version file is installed
		return true
	}

	installed, err := version.Parse(strings.Trim(string(fileData), "\n"))
	if err != nil {
		// assuming that installing is better than skipping it in this case
		return true
	}

	// get latest available version on the JetBrains marketplace
	remoteVersionData, err := latestRemoteVersion(ideBuild)
	if err != nil {
		// this shouldn't happen, but continue with the installation in this case
		return true
	}

	remoteVersion, err := version.Parse(remoteVersionData)
	if err != nil {
		// this shouldn't happen, but continue with the installation in this case
		return true
	}

	return remoteVersion.LargerThan(installed)
}

func findInstalledProductEditor(ctx context.Context, productID string, mgr editor.Plugin) (string, error) {
	// try to find an existing editor
	editorPaths, err := mgr.DetectEditors(ctx)
	if err != nil {
		return "", err
	}

	if len(editorPaths) == 0 {
		return "", errors.Errorf("no editor found for %s", mgr.ID())
	}

	for _, path := range editorPaths {
		config, build, err := editorConfig(path)
		if err == nil && build.ProductID == productID && config.Compatibility == "" {
			return path, nil
		}
	}

	//fixme check running?
	return "", errors.Errorf("no installed editor found for %s", productID)
}
