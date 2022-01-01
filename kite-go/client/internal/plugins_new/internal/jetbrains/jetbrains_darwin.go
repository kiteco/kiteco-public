package jetbrains

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	errorsui "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/errors"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

// NewJetBrainsManagers returns a slice of plugin managers for all supported JetBrains IDEs.
func NewJetBrainsManagers(process process.MacProcessManager, betaChannel bool) ([]editor.Plugin, error) {
	home, err := process.Home()
	if err != nil {
		return nil, err
	}

	return []editor.Plugin{
		// IntelliJ
		newManager(intellijID, intellijName,
			[]string{"com.jetbrains.intellij-EAP", "com.jetbrains.intellij", "com.jetbrains.intellij.ce"},
			intellijProductIDs,
			intellijToolboxName, process, home, betaChannel),
		// PyCharm
		newManager(pycharmID, pycharmName,
			[]string{"com.jetbrains.pycharm-EAP", "com.jetbrains.pycharm", "com.jetbrains.pycharm.ce"},
			pycharmProductIDs,
			pycharmToolboxName, process, home, betaChannel),
		// GoLand
		newManager(golandID, golandName,
			[]string{"com.jetbrains.goland-EAP", "com.jetbrains.goland"},
			golandProductIDs,
			golandToolboxName, process, home, betaChannel),
		// WebStorm
		newManager(webstormID, webstormName,
			[]string{"com.jetbrains.WebStorm-EAP", "com.jetbrains.WebStorm"},
			webstormProductIDs,
			webstormToolboxName, process, home, betaChannel),
		// PhpStorm
		newManager(phpstormID, phpstormName,
			[]string{"com.jetbrains.PhpStorm-EAP", "com.jetbrains.PhpStorm"},
			phpstormProductIDs,
			phpstormToolboxName, process, home, betaChannel),
		// Rider
		newManager(riderID, riderName,
			[]string{"com.jetbrains.rider-EAP", "com.jetbrains.rider"},
			riderProductIDs,
			riderToolboxName, process, home, betaChannel),
		// CLion
		newManager(clionID, clionName,
			[]string{"com.jetbrains.CLion-EAP", "com.jetbrains.CLion"},
			clionProductIDs,
			clionToolboxName, process, home, betaChannel),
		// RubyMine
		newManager(rubymineID, rubymineName,
			[]string{"com.jetbrains.rubymine-EAP", "com.jetbrains.rubymine"},
			rubymineProductIDs,
			rubymineToolboxName, process, home, betaChannel),
		// AppCode
		newManager(appcodeID, "AppCode",
			[]string{"com.jetbrains.AppCode-EAP", "com.jetbrains.AppCode"},
			[]string{appcodeID},
			"AppCode", process, home, betaChannel),
		// Android Studio
		newManager(androidStudioID, androidStudioName,
			[]string{"com.google.android.studio-EAP", "com.google.android.studio"},
			androidStudioProductIDs,
			androidStudioToolboxName, process, home, betaChannel),
	}, nil
}

func newManager(id string, name string, bundleIDs []string, productIDs []string, toolboxProductName string,
	process process.MacProcessManager, home string, betaChannel bool) *macJetBrains {
	return &macJetBrains{
		id:                 id,
		name:               name,
		process:            process,
		bundleIDs:          bundleIDs,
		productDs:          productIDs,
		toolboxDir:         filepath.Join(home, "Library", "Application Support", "JetBrains", "Toolbox"),
		toolboxProductName: toolboxProductName,
		pluginName:         pluginDirName,
		userHome:           home,
		betaChannel:        betaChannel,
	}
}

type macJetBrains struct {
	id      string
	name    string
	icon    string
	process process.MacProcessManager

	bundleIDs          []string
	productDs          []string
	toolboxDir         string
	toolboxProductName string
	pluginName         string
	userHome           string
	betaChannel        bool
}

// ID implements editor.Plugin
func (m *macJetBrains) ID() string {
	return m.id
}

// AdditionalIDs implements helper interface AdditionalIDs
func (m *macJetBrains) AdditionalIDs() []string {
	return m.productDs
}

// Name implements editor.Plugin
func (m *macJetBrains) Name() string {
	return m.name
}

// InstallConfig implements editor.Plugin
func (m *macJetBrains) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: true,
		Running:                  m.isIDERunning(),
		InstallWhileRunning:      true,
		UpdateWhileRunning:       false,
		UninstallWhileRunning:    false,
	}
}

// DetectEditors implements editor.Plugin
func (m *macJetBrains) DetectEditors(ctx context.Context) ([]string, error) {
	homes := checkToolboxFolder(m.toolboxDir, m.toolboxProductName)

	for _, ID := range m.bundleIDs {
		bundleLocations, err := m.process.BundleLocations(ctx, ID)
		if err != nil {
			continue
		}
		for _, ideHome := range bundleLocations {
			homes = append(homes, ideHome)
		}
	}

	return homes, nil
}

// InstalledProductIDs implements helper interface InstalledProductIDs
func (m *macJetBrains) InstalledProductIDs(ctx context.Context) []string {
	ids := make([]string, 0)
	homes, err := m.DetectEditors(ctx)
	if err != nil {
		return ids
	}
	for _, home := range homes {
		v, err := findProductVersion(buildFileLocation(home))
		if err == nil {
			ids = append(ids, v.ProductID)
		}
	}
	return ids
}

// DetectEditors implements editor.Plugin
func (m *macJetBrains) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := m.process.RunningApplications()
	if err != nil {
		return nil, err
	}
	return list.Matching(func(process process.Process) string {
		if shared.StringsContain(m.bundleIDs, process.BundleID) && process.BundleLocation != "" {
			return process.BundleLocation
		}
		return ""
	}), nil
}

// DetectEditors implements editor.Plugin
func (m *macJetBrains) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	config, _, err := editorConfig(editorPath)
	return config, err
}

// IsInstalled implements editor.Plugin
func (m *macJetBrains) IsInstalled(ctx context.Context, editorPath string) bool {
	parent, _, err := m.pluginsDirectory(editorPath)
	return err == nil && fs.DirExists(filepath.Join(parent, m.pluginName))
}

// Install implements editor.Plugin
func (m *macJetBrains) Install(ctx context.Context, editorPath string) error {
	parent, _, err := m.pluginsDirectory(editorPath)
	if err != nil {
		return err
	}
	return installOrUpdatePlugin(editorPath, parent, m.pluginName, m.betaChannel)
}

// Uninstall implements editor.Plugin
func (m *macJetBrains) Uninstall(ctx context.Context, editorPath string) error {
	parent, _, err := m.pluginsDirectory(editorPath)
	if err != nil {
		return err
	}
	return shared.UninstallPlugin(filepath.Join(parent, m.pluginName))
}

// Update implements editor.Plugin
func (m *macJetBrains) Update(ctx context.Context, editorPath string) error {
	// Install calls installOrUpdatePlugin
	return m.Install(ctx, editorPath)
}

func (m *macJetBrains) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	if editorPath == "" {
		var err error
		if editorPath, err = findInstalledProductEditor(ctx, id, m); err != nil {
			return nil, err
		}
	}

	// The editorPath sent by the editor is suffixed with /Contents
	editorPath = strings.TrimSuffix(editorPath, "/Contents")

	var args []string
	if line >= 1 {
		args = []string{"--line", strconv.Itoa(line)}
	}
	args = append(args, filePath) // filepath has to be the last arg to make --line work
	cmd := exec.Command("open", append([]string{"-na", editorPath, "--args"}, args...)...)
	_, err := cmd.CombinedOutput()
	return nil, err
}

// pluginDirectories returns the directory which contains all plugins and the directory to contain the kite-pycharm plugin and an optional error
// This most often is parent, parent/kite-pycharm
// TODO support custom configured paths: https://intellij-support.jetbrains.com/hc/en-us/articles/207240985
func (m *macJetBrains) pluginsDirectory(ideHome string) (string, string, error) {
	v, err := findProductVersion(buildFileLocation(ideHome))
	if err != nil {
		return "", "", err
	}

	var configDir, pluginsDir string
	if v.Branch < 201 {
		// <= 2019.3, https://www.jetbrains.com/help/idea/2019.3/tuning-the-ide.html#config-directory
		configDir = filepath.Join(m.userHome, "Library", "Preferences", v.ProductVersion())
		pluginsDir = filepath.Join(m.userHome, "Library", "Application Support", v.ProductVersion())
	} else {
		// >= 2020.1, https://www.jetbrains.com/help/idea/2020.1/tuning-the-ide.html#
		// Android Studio >= 4.1 are located at .../Google/ instead of .../JetBrains/
		var manufacturer string
		if v.IsAndroidStudio() {
			manufacturer = "Google"
		} else {
			manufacturer = "JetBrains"
		}
		configDir = filepath.Join(m.userHome, "Library", "Application Support", manufacturer, v.ProductVersion())
		pluginsDir = filepath.Join(m.userHome, "Library", "Application Support", manufacturer, v.ProductVersion(), "plugins")
	}

	if !fs.DirExists(configDir) {
		configDir = ""
	}

	if toolboxPlugins := filepath.Join(filepath.Dir(ideHome), filepath.Base(ideHome)+".plugins"); fs.DirExists(toolboxPlugins) {
		return toolboxPlugins, configDir, nil
	}

	if !fs.DirExists(pluginsDir) {
		msg := fmt.Sprintf("Plugin configuration directory %s doesn't exist.\nPlease try going through the %s first-time configuration flow and restarting Kite", pluginsDir, m.Name())
		return "", "", errorsui.NewUI(msg, fmt.Sprintf("Plugin configuration directory %s doesn't exist", pluginsDir))
	}

	return pluginsDir, configDir, nil
}

// isIDERunning return true if any of the given bundles is running
func (m *macJetBrains) isIDERunning() bool {
	for _, id := range m.bundleIDs {
		if m.process.IsBundleRunning(id) {
			return true
		}
	}
	return false
}
