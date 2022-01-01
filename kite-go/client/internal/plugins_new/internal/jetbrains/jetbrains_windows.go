package jetbrains

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	errorsui "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/errors"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

var (
	commonJetbrainsPaths = []string{
		filepath.Join(fs.ProgramFiles(), "JetBrains"),
		filepath.Join(fs.ProgramFilesX86(), "JetBrains"),
		filepath.Join(fs.RoamingAppData(), "JetBrains"),
	}

	fsnotifierNames = []string{"fsnotifier.exe", "fsnotifier64.exe"}
)

// NewJetBrainsManagers returns a slice of plugin managers for all supported JetBrains IDEs.
func NewJetBrainsManagers(process process.WindowsProcessManager, betaChannel bool) ([]editor.Plugin, error) {
	return newJetBrainsManagers(process, commonJetbrainsPaths, betaChannel)
}

// newJetBrainsManagers returns a slice of plugin managers for all supported JetBrains IDEs.
func newJetBrainsManagers(process process.WindowsProcessManager, commonPaths []string, betaChannel bool) ([]editor.Plugin, error) {
	home, err := process.Home()
	if err != nil {
		return nil, err
	}

	return []editor.Plugin{
		// IntelliJ
		newManager(intellijID, intellijName, "IntelliJ", intellijToolboxName,
			[]string{"idea.exe", "idea64.exe"}, intellijProductIDs, commonPaths, process, home, betaChannel),
		// PyCharm
		newManager(pycharmID, pycharmName, "PyCharm", pycharmToolboxName,
			[]string{"pycharm.exe", "pycharm64.exe"}, pycharmProductIDs, commonPaths, process, home, betaChannel),
		// GoLand
		newManager(golandID, golandName, "GoLand", golandToolboxName,
			[]string{"goland.exe", "goland64.exe"}, golandProductIDs, commonPaths, process, home, betaChannel),
		// WebStorm
		newManager(webstormID, webstormName, "WebStorm", webstormToolboxName,
			[]string{"webstorm.exe", "webstorm64.exe"}, webstormProductIDs, commonPaths, process, home, betaChannel),
		// PhpStorm
		newManager(phpstormID, phpstormName, "PhpStorm", phpstormToolboxName,
			[]string{"phpstorm.exe", "phpstorm64.exe"}, phpstormProductIDs, commonPaths, process, home, betaChannel),
		// Rider
		newManager(riderID, riderName, "Rider", riderToolboxName,
			[]string{"rider.exe", "rider64.exe"}, riderProductIDs, commonPaths, process, home, betaChannel),
		// CLion
		newManager(clionID, clionName, "CLion", clionToolboxName,
			[]string{"clion.exe", "clion64.exe"}, clionProductIDs, commonPaths, process, home, betaChannel),
		// RubyMine
		newManager(rubymineID, rubymineName, "RubyMine", rubymineToolboxName,
			[]string{"rubymine.exe", "rubymine64.exe"}, rubymineProductIDs, commonPaths, process, home, betaChannel),
		// AppCode isn't supported on Windows
		// Android Studio
		newManager(androidStudioID, androidStudioName, "AndroidStudio", androidStudioToolboxName,
			[]string{"studio.exe", "studio64.exe"}, androidStudioProductIDs, []string{
				filepath.Join(fs.ProgramFiles(), "Google"),
				filepath.Join(fs.ProgramFilesX86(), "Google"),
				filepath.Join(fs.RoamingAppData(), "Google"),
			}, process, home, betaChannel),
	}, nil
}

func newManager(id string, name, installFolderName, toolboxFolderName string, exeNames []string,
	productIDs []string, commonPaths []string, process process.WindowsProcessManager, homeDir string,
	betaChannel bool) *windowsJetBrains {

	return &windowsJetBrains{
		id:                 id,
		name:               name,
		process:            process,
		toolboxDir:         filepath.Join(fs.LocalAppData(), "JetBrains", "Toolbox"),
		exeNames:           exeNames,
		productIDs:         productIDs,
		installFolderName:  installFolderName,
		toolboxProductName: toolboxFolderName,
		pluginName:         pluginDirName,
		userHome:           homeDir,
		installPaths:       commonPaths,
		betaChannel:        betaChannel,
	}
}

type windowsJetBrains struct {
	id         string
	name       string
	process    process.WindowsProcessManager
	toolboxDir string

	exeNames           []string
	productIDs         []string
	pluginName         string
	installFolderName  string
	toolboxProductName string

	userHome     string
	installPaths []string
	betaChannel  bool
}

// ID implements editor.Plugin
func (i *windowsJetBrains) ID() string {
	return i.id
}

// AdditionalIDs implements helper interface AdditionalIDs
func (i *windowsJetBrains) AdditionalIDs() []string {
	return i.productIDs
}

// Name implements editor.Plugin
func (i *windowsJetBrains) Name() string {
	return i.name
}

// InstallConfig implements editor.Plugin
func (i *windowsJetBrains) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: true,
		Running:                  i.isIDERunning(ctx),
		InstallWhileRunning:      true,
		UpdateWhileRunning:       false,
		UninstallWhileRunning:    false,
	}
}

// DetectEditors implements editor.Plugin
func (i *windowsJetBrains) DetectEditors(ctx context.Context) ([]string, error) {
	regular := i.checkJetbrainsFolder(i.installFolderName)
	toolbox := checkToolboxFolder(i.toolboxDir, i.toolboxProductName)

	targets := i.process.FilterStartMenuTargets(func(target string) bool {
		return shared.StringsContain(i.exeNames, filepath.Base(target))
	})
	// we found IDEHOME/bin/idea.exe, we want IDEHOME
	targets = shared.MapStrings(targets, func(target string) string {
		return filepath.Clean(filepath.Dir(filepath.Dir(target)))
	})

	return system.Union(regular, toolbox, targets), nil
}

// DetectRunningEditors implements editor.Plugin
func (i *windowsJetBrains) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := i.process.List(ctx)
	if err != nil {
		return nil, err
	}

	return list.Matching(ctx, func(ctx context.Context, process process.Process) string {
		name, err := process.NameWithContext(ctx)
		if err != nil {
			return ""
		}

		if shared.StringsContain(i.exeNames, name) || shared.StringsContain(fsnotifierNames, name) {
			exe, err := process.ExeWithContext(ctx)
			if err != nil {
				return ""
			}

			// walk tree upwards until we find a build.txt file
			// check the content to make sure we've detected a compatible editor
			baseDir, err := locateInstallRoot(exe, i.productIDs)
			if err == nil {
				return baseDir
			}
		}

		return ""
	})
}

// InstalledProductIDs implements helper interface InstalledProductIDs
func (i *windowsJetBrains) InstalledProductIDs(ctx context.Context) []string {
	ids := make([]string, 0)
	idepaths, err := i.DetectEditors(ctx)
	if err != nil {
		return ids
	}
	for _, idePath := range idepaths {
		v, err := findProductVersion(buildFileLocation(idePath))
		if err == nil {
			ids = append(ids, v.ProductID)
		}
	}
	return ids
}

// DetectEditors implements editor.Plugin
// On Windows it expects the base directory of an IDE, e.g. c:\Programs\JetBrains\IU-192.1234.5
func (i *windowsJetBrains) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	config, _, err := editorConfig(editorPath)
	return config, err
}

// IsInstalled implements editor.Plugin
func (i *windowsJetBrains) IsInstalled(ctx context.Context, editorPath string) bool {
	parent, err := i.pluginsDirectory(editorPath)
	return err == nil && fs.DirExists(filepath.Join(parent, i.pluginName))
}

// Install implements editor.Plugin
func (i *windowsJetBrains) Install(ctx context.Context, editorPath string) error {
	parent, err := i.pluginsDirectory(editorPath)
	if err != nil {
		return err
	}
	return installOrUpdatePlugin(editorPath, parent, i.pluginName, i.betaChannel)
}

// Uninstall implements editor.Plugin
func (i *windowsJetBrains) Uninstall(ctx context.Context, editorPath string) error {
	parent, err := i.pluginsDirectory(editorPath)
	if err != nil {
		return err
	}
	return shared.UninstallPlugin(filepath.Join(parent, i.pluginName))
}

// Update implements editor.Plugin
func (i *windowsJetBrains) Update(ctx context.Context, editorPath string) error {
	// Install calls installOrUpdatePlugin
	return i.Install(ctx, editorPath)
}

func (i *windowsJetBrains) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	if editorPath == "" {
		var err error
		if editorPath, err = findInstalledProductEditor(ctx, id, i); err != nil {
			return nil, err
		}
	}

	if editorPath == "" && !fs.DirExists(editorPath) {
		return nil, errors.Errorf("Editor directory '%s' does not exist")
	}

	for _, exeName := range i.exeNames {
		batName := strings.Replace(exeName, ".exe", ".bat", -1)
		scriptPath := filepath.Join(editorPath, "bin", batName)
		if fs.FileExists(scriptPath) {
			var args []string
			if line >= 1 {
				args = []string{"--line", strconv.Itoa(line - 1), filePath}
			} else {
				args = []string{filePath}
			}

			errorChan := make(chan error, 1)
			go func() {
				defer close(errorChan)
				if output, err := i.process.Run(scriptPath, args...); err != nil {
					log.Printf("error launching JetBrains editor %s, file %s, line %d: %s", scriptPath, filePath, line, output)
					errorChan <- err
				}
			}()
			return errorChan, nil
		}
	}

	return nil, errors.Errorf("no executable found for editor %s", i.id)
}

func (i *windowsJetBrains) pluginsDirectory(ideHome string) (string, error) {
	v, err := findProductVersion(buildFileLocation(ideHome))
	if err != nil {
		return "", err
	}

	// fixme this doesn't seem to be in use on Windows, but it doesn't hurt to keep it
	if toolboxPlugins := filepath.Join(filepath.Dir(ideHome), filepath.Base(ideHome)+".plugins"); fs.DirExists(toolboxPlugins) {
		return toolboxPlugins, nil
	}

	var pluginsDir string
	if v.Branch < 201 {
		// <= 2019.3, https://www.jetbrains.com/help/idea/2019.3/tuning-the-ide.html#plugins-directory
		// at first startup $HOME/.IntelliJIDEA2019.3/{config,system} is created
		// plugins are stored at $HOME/.IntelliJIDEA2019.3/config/plugins/, this dir is NOT created at first startup
		pluginsDir = filepath.Join(i.userHome, fmt.Sprintf(".%s", v.ProductVersion()), "config", "plugins")
	} else {
		// >= 2020.1, https://www.jetbrains.com/help/idea/2020.1/tuning-the-ide.html#plugins-directory
		// 2020.1 and later stores plugins at C:\Users\jansorg\AppData\Roaming\JetBrains\GoLand2020.1\plugins
		// At first startup C:\Users\jansorg\AppData\Roaming\JetBrains\GoLand2020.1\plugins is created
		appData, ok := os.LookupEnv("APPDATA")
		if !ok {
			appData = filepath.Join(i.userHome, "AppData", "Roaming")
		}
		// Android Studio >= 4.1 is located at ...\Google\ instead of ...\JetBrains\
		var manufacturer string
		if v.IsAndroidStudio() {
			manufacturer = "Google"
		} else {
			manufacturer = "JetBrains"
		}
		pluginsDir = filepath.Join(appData, manufacturer, v.ProductVersion(), "plugins")
	}

	// e.g. maps version string PyCharm2018.2 to target location $HOME/.PyCharm2018.2/config/plugins
	if !fs.DirExists(pluginsDir) {
		msg := fmt.Sprintf("Plugin configuration directory %s doesn't exist.\nPlease try going through the %s first-time configuration flow and restarting Kite", pluginsDir, i.Name())
		return "", errorsui.NewUI(msg, fmt.Sprintf("Plugin configuration directory %s doesn't exist", pluginsDir))
	}

	return pluginsDir, nil
}

func (i *windowsJetBrains) isIDERunning(ctx context.Context) bool {
	for _, exe := range i.exeNames {
		if ok, err := i.process.IsProcessRunning(ctx, exe); ok && err == nil {
			return true
		}
	}
	return false
}

// checkJetbrainsFolder checks the default Jetbrains folder for an install of a given product.
func (i *windowsJetBrains) checkJetbrainsFolder(product string) []string {
	var homes []string
	for _, dir := range i.installPaths {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if strings.Contains(f.Name(), product) {
				homes = append(homes, filepath.Join(dir, f.Name()))
			}
		}
	}
	return homes
}
