package vscode

import (
	"context"
	"log"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

var (
	bundleIDs = []string{"com.microsoft.VSCode", "com.microsoft.VSCodeInsiders"}
)

// NewManager returns a new vscode plugin manager suitable for Linux
func NewManager(process process.MacProcessManager) (editor.Plugin, error) {
	home, err := process.Home()
	if err != nil {
		return nil, err
	}

	return &macVSCode{
		process:  process,
		userHome: home,
	}, nil
}

func newTestManager(baseDir string, process *process.MockManager) *macVSCode {
	return &macVSCode{
		process:  process,
		userHome: baseDir,
	}
}

type macVSCode struct {
	process  process.MacProcessManager
	userHome string
}

// ID implements editor.Plugin
func (m *macVSCode) ID() string {
	return vscodeID
}

// Name implements editor.Plugin
func (m *macVSCode) Name() string {
	return vscodeName
}

// InstallConfig implements editor.Plugin
func (m *macVSCode) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: true,
		Running:                  m.isRunning(),
		InstallWhileRunning:      true,
		UpdateWhileRunning:       true,
		UninstallWhileRunning:    true,
	}
}

// DetectEditors implements editor.Plugin. It returns only those editors which have a cli binary alongside the main binary
func (m *macVSCode) DetectEditors(ctx context.Context) ([]string, error) {
	var locations []string
	for _, bundleID := range bundleIDs {
		bundleLocations, err := m.process.BundleLocations(ctx, bundleID)
		if err != nil {
			log.Println(err)
			continue
		}
		locations = append(locations, bundleLocations...)
	}
	return locations, nil
}

func (m *macVSCode) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := m.process.RunningApplications()
	if err != nil {
		return nil, err
	}
	return list.Matching(func(process process.Process) string {
		if shared.StringsContain(bundleIDs, process.BundleID) && process.BundleLocation != "" {
			return process.BundleLocation
		}
		return ""
	}), nil
}

func (m *macVSCode) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	v, _ := m.process.AppVersion(editorPath)
	return system.Editor{
		Path:    editorPath,
		Version: v,
	}, nil
}

// IsInstalled implements editor.Plugin
func (m *macVSCode) IsInstalled(ctx context.Context, editorPath string) bool {
	return isInstalled(m, editorPath)
}

// Install implements editor.Plugin
func (m *macVSCode) Install(ctx context.Context, editorPath string) error {
	return install(m, editorPath)
}

// Uninstall implements editor.Plugin
func (m *macVSCode) Uninstall(ctx context.Context, editorPath string) error {
	// fixme remove this? calling the cli is slow
	if !m.IsInstalled(ctx, editorPath) {
		return nil
	}
	return uninstall(m, editorPath)
}

// Update implements editor.Plugin
func (m *macVSCode) Update(ctx context.Context, editorPath string) error {
	return update(m, editorPath)
}

func (m *macVSCode) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	// If editorPath is sent by the plugin, it's suffixed with /Contents/Resources/app already
	editorPath = strings.TrimSuffix(editorPath, "/Contents/Resources/app")
	return openFile(m, editorPath, filePath, line)
}

// isRunning checks if an instance of VSCode is running.
func (m *macVSCode) isRunning() bool {
	for _, bundleID := range bundleIDs {
		if m.process.IsBundleRunning(bundleID) {
			return true
		}
	}
	return false
}

func (m *macVSCode) runVSCode(editorPath string, args ...string) ([]byte, error) {
	return m.process.Run(m.cliPath(editorPath), args...)
}

func (m *macVSCode) cliPath(editorPath string) string {
	return filepath.Join(editorPath, "Contents", "Resources", "app", "bin", "code")
}

func (m *macVSCode) userExtensionsDir(editorPath string) string {
	extensionsDir := ".vscode"
	if filepath.Base(editorPath) == "Visual Studio Code - Insiders.app" {
		extensionsDir = ".vscode-insiders"
	}
	return filepath.Join(m.userHome, extensionsDir, "extensions")
}
