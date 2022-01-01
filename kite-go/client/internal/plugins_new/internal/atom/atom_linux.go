package atom

import (
	"context"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

const (
	atomBinaryName = "atom"
	snapInstall    = "usr/bin/atom"
)

// NewManager returns a new atom plugin manager suitable for Linux
func NewManager(process process.LinuxProcessManager) (editor.Plugin, error) {
	return &linuxAtom{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *linuxAtom {
	return &linuxAtom{
		process: process,
	}
}

type linuxAtom struct {
	process process.LinuxProcessManager
}

// ID implements editor.Plugin
func (m *linuxAtom) ID() string {
	return atomID
}

// Name implements editor.Plugin
func (m *linuxAtom) Name() string {
	return atomName
}

// InstallConfig implements editor.Plugin
func (m *linuxAtom) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               m.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

// DetectEditors implements editor.Plugin
func (m *linuxAtom) DetectEditors(ctx context.Context) ([]string, error) {
	p, err := exec.LookPath(atomBinaryName)
	if err != nil {
		return nil, err
	}
	p, err = shared.SnapPath(p, atomBinaryName, snapInstall)
	var atomPaths []string
	atomPaths = append(atomPaths, p)
	return atomPaths, nil
}

func (m *linuxAtom) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := m.process.List(ctx)
	if err != nil {
		return nil, err
	}

	return list.Matching(ctx, func(ctx context.Context, p process.Process) string {
		cmd, err := p.CmdlineSliceWithContext(ctx)
		// traditional installation
		if err == nil && len(cmd) >= 2 && strings.Contains(cmd[0], "bash") && strings.HasSuffix(cmd[1], "/atom") {
			return cmd[1]
		}
		// snap installation
		return ""
	})
}

func (m *linuxAtom) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	v, err := m.readAtomBinaryVersion(editorPath)
	if err != nil {
		return system.Editor{}, err
	}

	return system.Editor{
		Path:    editorPath,
		Version: v,
	}, nil
}

// IsInstalled implements editor.Plugin
func (m *linuxAtom) IsInstalled(ctx context.Context, editorPath string) bool {
	return isInstalled(m, editorPath)
}

// Install implements editor.Plugin
func (m *linuxAtom) Install(ctx context.Context, editorPath string) error {
	return install(m, editorPath)
}

// Uninstall implements editor.Plugin
func (m *linuxAtom) Uninstall(ctx context.Context, editorPath string) error {
	// fixme remove this? calling apm is slow
	if !m.IsInstalled(ctx, editorPath) {
		return nil
	}
	return uninstall(m, editorPath)
}

// Update implements editor.Plugin
func (m *linuxAtom) Update(ctx context.Context, editorPath string) error {
	return update(m, editorPath)
}

func (m *linuxAtom) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	re := regexp.MustCompile(`share/atom/resources/app/apm/bin/apm`)
	editorPath = re.ReplaceAllLiteralString(editorPath, "bin/atom")
	return openFile(m, editorPath, filePath, line)
}

func (m *linuxAtom) apmPath(editorPath string) string {
	return filepath.Join(filepath.Dir(editorPath), "apm")
}

func (m *linuxAtom) atomPath(editorPath string) string {
	return filepath.Join(filepath.Dir(editorPath), "atom")
}

func (m *linuxAtom) run(path string, args ...string) ([]byte, error) {
	return m.process.Run(path, args...)
}

func (m *linuxAtom) readAtomBinaryVersion(path string) (string, error) {
	out, err := m.process.Run(path, "--version")
	if err != nil {
		return "", err
	}
	return readAtomVersion(out)
}

func (m *linuxAtom) isRunning(ctx context.Context) bool {
	result, err := m.process.IsProcessRunning(ctx, atomBinaryName)
	if err != nil {
		return false
	}
	return result
}
