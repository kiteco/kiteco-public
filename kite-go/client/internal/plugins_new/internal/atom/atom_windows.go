package atom

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

const (
	// executable at <prefix>\
	atomBinaryName = "atom.exe"
	// executable at <prefix>\bin\
	atomBinBinaryName = "atom.cmd"
)

var (
	commonPaths = []string{
		// use atom/bin/atom. There also is atom/atom.exe which is used by the shortcuts,
		// but you can't call -version on that one.
		filepath.Join(fs.LocalAppData(), "atom", "bin", atomBinBinaryName),
	}
)

// NewManager returns a new atom plugin manager suitable for Linux
func NewManager(process process.WindowsProcessManager) (editor.Plugin, error) {
	return &windowsAtom{
		process:     process,
		commonPaths: commonPaths,
	}, nil
}

func newTestManager(process *process.MockManager) *windowsAtom {
	return &windowsAtom{
		process:     process,
		commonPaths: []string{},
	}
}

type windowsAtom struct {
	process     process.WindowsProcessManager
	commonPaths []string
}

// ID implements editor.Plugin
func (m *windowsAtom) ID() string {
	return atomID
}

// Name implements editor.Plugin
func (m *windowsAtom) Name() string {
	return atomName
}

// InstallConfig implements editor.Plugin
func (m *windowsAtom) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               m.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

// DetectEditors implements editor.Plugin
func (m *windowsAtom) DetectEditors(ctx context.Context) ([]string, error) {
	// we need atom/bin/atom, not atom/atom.exe. bin/atom is in $PATH by default
	atomPath, _ := exec.LookPath(atomBinBinaryName)
	startMenu := m.locateByStartMenu()
	all := system.Union(startMenu, m.commonPaths, []string{atomPath})
	return fs.KeepExistingFiles(all), nil
}

// DetectRunningEditors implements editor.Plugin
func (m *windowsAtom) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := m.process.List(ctx)
	if err != nil {
		return nil, err
	}

	return list.Matching(ctx, func(ctx context.Context, process process.Process) string {
		if exe, err := process.ExeWithContext(ctx); err == nil && filepath.Base(exe) == atomBinaryName {
			return exe
		}
		return ""
	})
}

// EditorConfig implements editor.Plugin
func (m *windowsAtom) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	if !fs.FileExists(editorPath) {
		return system.Editor{}, fmt.Errorf("file doesn't exist: %s", editorPath)
	}

	v, err := m.readAtomBinaryVersion(editorPath)
	if err != nil {
		return system.Editor{}, nil
	}

	return system.Editor{
		Path:    editorPath,
		Version: v,
	}, nil
}

// IsInstalled implements editor.Plugin
func (m *windowsAtom) IsInstalled(ctx context.Context, editorPath string) bool {
	return isInstalled(m, editorPath)
}

// Install implements editor.Plugin
func (m *windowsAtom) Install(ctx context.Context, editorPath string) error {
	return install(m, editorPath)
}

// Uninstall implements editor.Plugin
func (m *windowsAtom) Uninstall(ctx context.Context, editorPath string) error {
	// fixme remove this? calling apm is slow
	if !m.IsInstalled(ctx, editorPath) {
		return nil
	}
	return uninstall(m, editorPath)
}

// Update implements editor.Plugin
func (m *windowsAtom) Update(ctx context.Context, editorPath string) error {
	return update(m, editorPath)
}

func (m *windowsAtom) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	re := regexp.MustCompile(`app-\d+\.\d+\.\d+\\resources\\app\\apm\\bin\\apm\.cmd`)
	editorPath = re.ReplaceAllLiteralString(editorPath, "bin\\atom.cmd")
	return openFile(m, editorPath, filePath, line)
}

func (m *windowsAtom) apmPath(editorPath string) string {
	return filepath.Join(filepath.Dir(editorPath), "apm")
}

func (m *windowsAtom) atomPath(editorPath string) string {
	return filepath.Join(filepath.Dir(editorPath), "atom")
}

func (m *windowsAtom) run(path string, args ...string) ([]byte, error) {
	return m.process.Run(path, args...)
}

func (m *windowsAtom) readAtomBinaryVersion(path string) (string, error) {
	out, err := m.process.Run(path, "--version")
	if err != nil {
		return "", err
	}
	return readAtomVersion(out)
}

func (m *windowsAtom) locateByStartMenu() []string {
	targets := m.process.FilterStartMenuTargets(func(t string) bool {
		return filepath.Base(t) == atomBinaryName
	})
	for i, t := range targets {
		// fix the path so we get the binary in bin/atom
		dir, _ := filepath.Split(t)
		targets[i] = filepath.Join(dir, "bin", atomBinBinaryName)
	}
	return targets
}

func (m *windowsAtom) isRunning(ctx context.Context) bool {
	result, err := m.process.IsProcessRunning(ctx, atomBinaryName)
	if err != nil {
		return false
	}
	return result
}
