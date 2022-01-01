package atom

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system/nsbundle"
)

const (
	bundleID = "com.github.atom"
)

// NewManager returns a new atom plugin manager suitable for Linux
func NewManager(process process.MacProcessManager) (editor.Plugin, error) {
	return &macAtom{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *macAtom {
	return &macAtom{
		process: process,
	}
}

type macAtom struct {
	process process.MacProcessManager
}

// ID implements editor.Plugin
func (m *macAtom) ID() string {
	return atomID
}

// Name implements editor.Plugin
func (m *macAtom) Name() string {
	return atomName
}

// InstallConfig implements editor.Plugin
func (m *macAtom) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               m.process.IsBundleRunning(bundleID),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

// DetectEditors implements editor.Plugin
func (m *macAtom) DetectEditors(ctx context.Context) ([]string, error) {
	return m.process.BundleLocations(ctx, bundleID)
}

// DetectRunningEditors implements editor.Plugin
func (m *macAtom) DetectRunningEditors(ctx context.Context) ([]string, error) {
	processes, err := m.process.RunningApplications()
	if err != nil {
		return nil, err
	}

	return processes.Matching(func(process process.Process) string {
		if process.BundleID == bundleID && process.BundleLocation != "" {
			return process.BundleLocation
		}
		return ""
	}), nil
}

// EditorConfig implements editor.Plugin
func (m *macAtom) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	v, _ := nsbundle.AppVersion(editorPath)
	return system.Editor{
		Path:    editorPath,
		Version: v,
	}, nil
}

// IsInstalled implements editor.Plugin
func (m *macAtom) IsInstalled(ctx context.Context, editorPath string) bool {
	return isInstalled(m, editorPath)
}

// Install implements editor.Plugin
func (m *macAtom) Install(ctx context.Context, editorPath string) error {
	return install(m, editorPath)
}

// Uninstall implements editor.Plugin
func (m *macAtom) Uninstall(ctx context.Context, editorPath string) error {
	if !m.IsInstalled(ctx, editorPath) {
		return nil
	}
	return uninstall(m, editorPath)
}

// Update implements editor.Plugin
func (m *macAtom) Update(ctx context.Context, editorPath string) error {
	return update(m, editorPath)
}

func (m *macAtom) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	// If editorPath is sent by the plugin, it's suffixed with /Contents/Resources/app/apm/bin/apm
	editorPath = strings.TrimSuffix(editorPath, "Contents/Resources/app/apm/bin/apm")
	return openFile(m, editorPath, filePath, line)
}

func (m *macAtom) apmPath(editorPath string) string {
	return filepath.Join(editorPath, "Contents", "Resources", "app", "apm", "bin", "apm")
}

func (m *macAtom) atomPath(editorPath string) string {
	return filepath.Join(editorPath, "Contents", "Resources", "app", "atom.sh")
}

func (m *macAtom) run(path string, args ...string) ([]byte, error) {
	return m.process.Run(path, args...)
}
