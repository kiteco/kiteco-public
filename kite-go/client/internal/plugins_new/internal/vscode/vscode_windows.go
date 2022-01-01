package vscode

import (
	"context"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

const (
	vsCodeName        = "Microsoft VS Code"
	insidersName      = "Microsoft VS Code Insiders"
	vsCodeCmd         = "code.cmd"
	insidersCmd       = "code-insiders.cmd"
	vsCodePluginDir   = ".vscode"
	insidersPluginDir = ".vscode-insiders"
)

var (
	exeNames   = []string{"Code.exe", "Code - Insiders.exe"}
	exeNameMap = map[string]string{
		vsCodeName:   "Code.exe",
		insidersName: "Code - Insiders.exe",
	}
	commonPaths = []string{
		filepath.Join(fs.ProgramFilesX86(), vsCodeName, exeNameMap[vsCodeName]),
		filepath.Join(fs.ProgramFilesX86(), insidersName, exeNameMap[insidersName]),
		filepath.Join(fs.LocalAppData(), "Programs", vsCodeName, exeNameMap[vsCodeName]),
		filepath.Join(fs.LocalAppData(), "Programs", insidersName, exeNameMap[insidersName]),
	}
)

// NewManager returns a new vscode plugin manager suitable for Windows
func NewManager(process process.WindowsProcessManager) (editor.Plugin, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	return &windowsVSCode{
		process:     process,
		userHome:    usr.HomeDir,
		commonPaths: commonPaths,
	}, nil
}

func newTestManager(baseDir string, process *process.MockManager) *windowsVSCode {
	return &windowsVSCode{
		process:     process,
		userHome:    baseDir,
		commonPaths: []string{},
	}
}

type windowsVSCode struct {
	process     process.WindowsProcessManager
	userHome    string
	commonPaths []string
}

// ID implements editor.Plugin
func (m *windowsVSCode) ID() string {
	return vscodeID
}

// Name implements editor.Plugin
func (m *windowsVSCode) Name() string {
	return vscodeName
}

// InstallConfig implements editor.Plugin
func (m *windowsVSCode) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: true,
		Running:                  m.isRunning(ctx),
		InstallWhileRunning:      true,
		UpdateWhileRunning:       true,
		UninstallWhileRunning:    true,
	}
}

// DetectEditors implements editor.Plugin. It returns only those editors which have a cli binary alongside the main binary
func (m *windowsVSCode) DetectEditors(ctx context.Context) ([]string, error) {
	startMenu := m.process.FilterStartMenuTargets(func(entry string) bool {
		if fs.FileExists(entry) && fs.FileExists(m.cliPath(entry)) {
			for _, exeName := range exeNameMap {
				if filepath.Base(entry) == exeName {
					return true
				}
			}
		}
		return false
	})

	common := shared.MapStrings(m.commonPaths, func(path string) string {
		if fs.FileExists(path) && fs.FileExists(m.cliPath(path)) {
			return path
		}
		return ""
	})

	return system.Union(startMenu, common), nil
}

// DetectRunningEditors implements editor.Plugin
func (m *windowsVSCode) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := m.process.List(ctx)
	if err != nil {
		return nil, err
	}
	// only return
	return list.Matching(ctx, func(ctx context.Context, process process.Process) string {
		exe, err := process.ExeWithContext(ctx)
		if err == nil {
			// only keep the locations where a cli is present
			name := filepath.Base(exe)
			if shared.StringsContain(exeNames, name) && fs.FileExists(m.cliPath(exe)) {
				return exe
			}
		}
		return ""
	})
}

// EditorConfig implements editor.Plugin
func (m *windowsVSCode) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	v := m.process.ExeProductVersion(editorPath)
	return system.Editor{
		Path:    editorPath,
		Version: v,
	}, nil
}

// IsInstalled implements editor.Plugin
func (m *windowsVSCode) IsInstalled(ctx context.Context, editorPath string) bool {
	return isInstalled(m, editorPath)
}

// Install implements editor.Plugin
func (m *windowsVSCode) Install(ctx context.Context, editorPath string) error {
	return install(m, editorPath)
}

// Uninstall implements editor.Plugin
func (m *windowsVSCode) Uninstall(ctx context.Context, editorPath string) error {
	// fixme remove this? calling the cli is slow
	if !m.IsInstalled(ctx, editorPath) {
		return nil
	}
	return uninstall(m, editorPath)
}

// Update implements editor.Plugin
func (m *windowsVSCode) Update(ctx context.Context, editorPath string) error {
	return update(m, editorPath)
}

func (m *windowsVSCode) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	// If editorPath is sent by the plugin, it needs to be sanitized first
	if strings.HasSuffix(editorPath, "\\resources\\app") {
		editorPath = strings.TrimSuffix(editorPath, "\\resources\\app")
		build := filepath.Base(editorPath)
		editorPath = filepath.Join(editorPath, exeNameMap[build])
	}
	return openFile(m, editorPath, filePath, line)
}

func (m *windowsVSCode) runVSCode(editorPath string, args ...string) ([]byte, error) {
	// kiteco/issue-tracker#239, refer to this page for details:
	// SO link: /questions/37878185/what-does-compat-layer-actually-do
	return m.process.RunWithEnv(m.cliPath(editorPath), []string{"__COMPAT_LAYER=RUNASINVOKER"}, args...)
}

func (m *windowsVSCode) cliPath(exePath string) string {
	// Check if path ends with Code - Insiders.exe
	if filepath.Base(exePath) == exeNameMap[insidersName] {
		return filepath.Join(filepath.Dir(exePath), "bin", insidersCmd)
	}
	return filepath.Join(filepath.Dir(exePath), "bin", vsCodeCmd)
}

func (m *windowsVSCode) userExtensionsDir(editorPath string) string {
	if filepath.Base(editorPath) == exeNameMap[insidersName] {
		return filepath.Join(m.userHome, insidersPluginDir, "extensions")
	}
	return filepath.Join(m.userHome, vsCodePluginDir, "extensions")
}

func (m *windowsVSCode) isRunning(ctx context.Context) bool {
	for _, exeName := range exeNameMap {
		result, err := m.process.IsProcessRunning(ctx, exeName)
		if err == nil && result {
			return true
		}
	}
	return false
}
