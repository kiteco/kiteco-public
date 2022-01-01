package jetbrains

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	errorsui "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/errors"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	snapDir       = "/snap"
	standaloneDir = "/opt"
)

// NewJetBrainsManagers returns a slice of plugin managers for all supported JetBrains IDEs.
func NewJetBrainsManagers(process process.LinuxProcessManager, betaChannel bool) ([]editor.Plugin, error) {
	home, err := process.Home()
	if err != nil {
		return nil, err
	}

	return []editor.Plugin{
		// IntelliJ Community, Education and Ultimate
		newManager(intellijID, intellijName, "idea.sh", intellijToolboxName, "intellij-idea-*", "idea*",
			intellijProductIDs, process, home, betaChannel),
		// PyCharm Community, Professional, Education, Anaconda Community, Anaconda Professional
		newManager(pycharmID, pycharmName, "pycharm.sh", pycharmToolboxName, "pycharm-*", "pycharm*",
			pycharmProductIDs, process, home, betaChannel),
		// GoLand
		newManager(golandID, golandName, "goland.sh", golandToolboxName, "goland*", "goland*",
			golandProductIDs, process, home, betaChannel),
		// WebStorm
		newManager(webstormID, webstormName, "webstorm.sh", webstormToolboxName, "webstorm*", "webstorm*",
			webstormProductIDs, process, home, betaChannel),
		// PhpStorm
		newManager(phpstormID, phpstormName, "phpstorm.sh", phpstormToolboxName, "phpstorm*", "phpstorm*",
			phpstormProductIDs, process, home, betaChannel),
		// Rider
		newManager(riderID, riderName, "rider.sh", riderToolboxName, "rider*", "rider*",
			riderProductIDs, process, home, betaChannel),
		// CLion
		newManager(clionID, clionName, "clion.sh", clionToolboxName, "clion*", "clion*",
			clionProductIDs, process, home, betaChannel),
		// RubyMine
		newManager(rubymineID, rubymineName, "rubymine.sh", rubymineToolboxName, "rubymine*", "rubymine*",
			rubymineProductIDs, process, home, betaChannel),
		// AppCode isn't supported on Linux
		// Android Studio
		newManager(androidStudioID, androidStudioName, "studio.sh", androidStudioToolboxName, "android-studio*", "android-studio*",
			androidStudioProductIDs, process, home, betaChannel),
	}, nil
}

func newManager(id string, name string, shellScriptName string, toolboxProductName string,
	snapDirPrefix string, standaloneDirPrefix string,
	supportedProductIDs []string,
	process process.LinuxProcessManager, home string, betaChannel bool) *linuxJetBrains {

	toolboxPaths := []string{"/opt/JetBrains/", "/opt/JetBrains/Toolbox", filepath.Join(home, ".local/share/JetBrains/Toolbox")}
	desktopFileLocations := []string{filepath.Join(home, ".local", "share", "applications"), "/usr/share/applications"}
	return &linuxJetBrains{
		id:                   id,
		name:                 name,
		process:              process,
		shellScriptName:      shellScriptName,
		toolboxDirs:          toolboxPaths,
		toolboxProductName:   toolboxProductName,
		snapDirPrefix:        snapDirPrefix,
		standaloneDirPrefix:  standaloneDirPrefix,
		desktopFileLocations: desktopFileLocations,
		supportedProductIDs:  supportedProductIDs,
		pluginName:           pluginDirName,
		userHome:             home,
		betaChannel:          betaChannel,
	}
}

type linuxJetBrains struct {
	id      string
	name    string
	process process.LinuxProcessManager

	toolboxDirs          []string
	toolboxProductName   string
	shellScriptName      string
	snapDirPrefix        string
	standaloneDirPrefix  string
	desktopFileLocations []string
	supportedProductIDs  []string
	pluginName           string
	userHome             string
	betaChannel          bool
}

// AdditionalIDs implements helper interface AdditionalIDs
func (i *linuxJetBrains) AdditionalIDs() []string {
	return i.supportedProductIDs
}

// ID implements editor.Plugin
func (i *linuxJetBrains) ID() string {
	return i.id
}

// Name implements editor.Plugin
func (i *linuxJetBrains) Name() string {
	return i.name
}

// InstallConfig implements editor.Plugin
func (i *linuxJetBrains) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: true,
		Running:                  i.isIDERunning(ctx, ""),
		InstallWhileRunning:      true,
		UpdateWhileRunning:       false,
		UninstallWhileRunning:    false,
	}
}

// DetectEditors implements editor.Plugin
func (i *linuxJetBrains) DetectEditors(ctx context.Context) ([]string, error) {
	var idePaths []string
	for _, toolboxDir := range i.toolboxDirs {
		idePaths = append(idePaths, checkToolboxFolder(toolboxDir, i.toolboxProductName)...)
	}
	idePaths = append(idePaths, i.checkShellScript(i.shellScriptName))
	idePaths = append(idePaths, i.checkSnapsFolder(snapDir)...)
	idePaths = append(idePaths, i.checkStandaloneFolders(standaloneDir)...)
	idePaths = append(idePaths, i.checkDesktopFiles()...)
	// empty and duplicate values are handled by findEditors
	return idePaths, nil
}

// DetectRunningEditors implements plugin.Plugin
func (i *linuxJetBrains) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := i.process.List(ctx)
	if err != nil {
		return nil, err
	}

	return list.Matching(ctx, func(ctx context.Context, process process.Process) string {
		cmdline, err := process.CmdlineSliceWithContext(ctx)
		if err != nil {
			return ""
		}

		// detect via fsnotifier or fsnotifier64 process
		if location := findInstallByFsnotifier(cmdline, i.supportedProductIDs); location != "" {
			return location
		}
		// detect via classpath bootstrap.jar entry
		if location := findInstallByClasspath(cmdline, i.supportedProductIDs); location != "" {
			return location
		}
		return ""
	})
}

// InstalledProductIDs implements helper interface InstalledProductIDs
func (i *linuxJetBrains) InstalledProductIDs(ctx context.Context) []string {
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

// EditorConfig implements plugin.Plugin
func (i *linuxJetBrains) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	config, _, err := editorConfig(editorPath)
	return config, err
}

// IsInstalled implements editor.Plugin
func (i *linuxJetBrains) IsInstalled(ctx context.Context, editorPath string) bool {
	parent, err := i.pluginsDirectory(editorPath)
	return err == nil && fs.DirExists(filepath.Join(parent, i.pluginName))
}

// Install implements editor.Plugin
func (i *linuxJetBrains) Install(ctx context.Context, editorPath string) error {
	parent, err := i.pluginsDirectory(editorPath)
	if err != nil {
		return err
	}
	return installOrUpdatePlugin(editorPath, parent, i.pluginName, i.betaChannel)
}

// Uninstall implements editor.Plugin
func (i *linuxJetBrains) Uninstall(ctx context.Context, editorPath string) error {
	parent, err := i.pluginsDirectory(editorPath)
	if err != nil {
		return err
	}
	return shared.UninstallPlugin(filepath.Join(parent, i.pluginName))
}

// Update implements editor.Plugin
func (i *linuxJetBrains) Update(ctx context.Context, editorPath string) error {
	// Install calls installOrUpdatePlugin
	return i.Install(ctx, editorPath)
}

func (i *linuxJetBrains) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	if editorPath == "" {
		var err error
		if editorPath, err = findInstalledProductEditor(ctx, id, i); err != nil {
			return nil, err
		}
	}

	if editorPath == "" && !fs.DirExists(editorPath) {
		return nil, errors.Errorf("Editor directory '%s' does not exist")
	}

	shellScriptPath := filepath.Join(editorPath, "bin", i.shellScriptName)
	if !fs.FileExists(shellScriptPath) {
		return nil, errors.Errorf("no executable found at %s", shellScriptPath)
	}

	var args []string
	if line >= 1 {
		// values on the cmdline seems to take 1-based values, but the SDK API is 0-based
		args = []string{"--line", strconv.Itoa(line), filePath}
	} else {
		args = []string{filePath}
	}

	errorChan := make(chan error, 1)
	go func() {
		defer close(errorChan)
		if output, err := i.process.Run(shellScriptPath, args...); err != nil {
			log.Printf("error launching JetBrains editor %s, file %s, line %d: %s", shellScriptPath, filePath, line, output)
			errorChan <- err
		}
	}()
	return errorChan, nil
}

// pluginDirectories returns the directory to store the kite-pycharm plugin and an optional error
func (i *linuxJetBrains) pluginsDirectory(ideHome string) (string, error) {
	v, err := findProductVersion(buildFileLocation(ideHome))
	if err != nil {
		return "", err
	}

	if toolboxPlugins := filepath.Join(filepath.Dir(ideHome), filepath.Base(ideHome)+".plugins"); fs.DirExists(toolboxPlugins) {
		return toolboxPlugins, nil
	}

	var configDir, pluginsDir string
	if v.Branch < 201 {
		// <= 2019.3, https://www.jetbrains.com/help/idea/2019.3/tuning-the-ide.html#plugins-directory
		// at first startup $HOME/.IntelliJIDEA2019.3/{config,system} is created
		// plugins are stored at $HOME/.IntelliJIDEA2019.3/config/plugins/, this dir is NOT created at first startup
		// this also covers Android Studio 4.0, which is based on 193.x
		configDir = filepath.Join(i.userHome, fmt.Sprintf(".%s", v.ProductVersion()), "config")
		pluginsDir = filepath.Join(configDir, "plugins")
	} else {
		// >= 2020.1, https://www.jetbrains.com/help/idea/2020.1/tuning-the-ide.html#plugins-directory
		// 2020.1 and later stores plugins at $XDG_DATA_HOME/JetBrains/IntelliJIdea2020.1/
		// We're using the fallback logic defined at
		// https://github.com/JetBrains/intellij-community/blob/40e5ae6b084eb7e444e1f1fd30c4a4e8696db977/platform/platform-impl/src/com/intellij/ide/actions/RevealFileAction.java#L264
		// The config is stored at $HOME/.config/IntelliJIdea2020.1/, but we don't need to check this dir.
		// At first startup $XDG_DATA_HOME/JetBrains/IntelliJIdea2020.1/ is created,
		// This also covers Android Studion >= 4.1, which is based on 2020.1
		xdgConfigHome, _ := os.LookupEnv("XDG_DATA_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(i.userHome, ".local", "share")
		}

		// Android Studio >= 4.1 is located at .../Google/ instead of .../JetBrains/
		var manufacturer string
		if v.IsAndroidStudio() {
			manufacturer = "Google"
		} else {
			manufacturer = "JetBrains"
		}
		pluginsDir = filepath.Join(xdgConfigHome, manufacturer, v.ProductVersion())
		configDir = pluginsDir
	}

	if !fs.DirExists(configDir) {
		msg := fmt.Sprintf("Configuration directory %s doesn't exist.\nPlease try going through the %s first-time configuration flow and restarting Kite", configDir, i.Name())
		return "", errorsui.NewUI(msg, fmt.Sprintf("Configuration directory %s doesn't exist", configDir))
	}

	return pluginsDir, nil
}

// checkShellScript checks the path for a shell script and returns the IDE home.
func (i *linuxJetBrains) checkShellScript(scriptName string) string {
	p, err := exec.LookPath(scriptName)
	if err != nil {
		return ""
	}

	if target, err := filepath.EvalSymlinks(p); err == nil {
		p = target
	}

	// the script is usually at $ideHome/bin/intellij.sh, we need to return $ideHome
	resolved, err := filepath.Abs(filepath.Dir(filepath.Dir(p)))
	if err != nil {
		return ""
	}
	return resolved
}

// checkSnapsFolder checks /snap/, the default location for Snapcraft.
func (i *linuxJetBrains) checkSnapsFolder(snapDir string) []string {
	results := checkDirPattern("", filepath.Join(snapDir, i.snapDirPrefix))
	// Need to append /current to every path in results
	var installs []string
	for _, result := range results {
		installs = append(installs, filepath.Join(result, "/current"))
	}
	return shared.DedupePaths(installs)
}

// checkStandaloneFolders checks /opt/ for editors installed manually.
func (i *linuxJetBrains) checkStandaloneFolders(dir string) []string {
	return checkDirPattern("", filepath.Join(dir, i.standaloneDirPrefix))
}

// checkDesktopFiles scans all desktop files in $HOME/.local and /usr/share/applications for JetBrains installations
func (i *linuxJetBrains) checkDesktopFiles() []string {
	var files []*shared.DesktopFile
	for _, path := range i.desktopFileLocations {
		if foundFiles, err := shared.CollectDesktopFiles(path); err == nil {
			files = append(files, foundFiles...)
		}
	}

	// map our supported products
	supportedProducts := make(map[string]bool)
	for _, id := range i.supportedProductIDs {
		supportedProducts[id] = true
	}

	var dirs []string
	for _, f := range files {
		if execPath, err := f.ExecPath(); err == nil {
			// the desktop files reference a script at $ideHome/bin/script.sh, so we have to keep $ideHome
			if filepath.IsAbs(execPath) {
				ideDir := filepath.Dir(filepath.Dir(execPath))

				// only keep entries which belong to a supported product
				// for example, we must not return a PyCharm entry when IntelliJ is queried
				v, err := findProductVersion(buildFileLocation(ideDir))
				if err == nil && supportedProducts[v.ProductID] {
					dirs = append(dirs, ideDir)
				}
			}
		}
	}
	return dirs
}

// isIDERunning returns whether a process of the IDE is currently running
// if editorPath is an empty string, then all IDEs matching shellscript name are detected as running
// if editorPath is a non-empty string, then only the running state of the IDE at editorPath is reported
func (i *linuxJetBrains) isIDERunning(ctx context.Context, editorPath string) bool {
	name := i.shellScriptName
	if editorPath != "" {
		name = filepath.Join(editorPath, "bin", i.shellScriptName)
	}

	running, err := i.process.IsProcessRunning(ctx, name)
	return running && err == nil
}
