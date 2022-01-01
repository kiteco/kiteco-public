package sublime

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins/sublimetext/sublime3"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

const (
	exeName = "sublime_text.exe"
)

var commonInstallLocations = []string{
	filepath.Join(fs.ProgramFiles(), "Sublime Text 3"),
	filepath.Join(fs.ProgramFilesX86(), "Sublime Text 3"),
}

// NewManager returns a new sublime plugin manager suitable for Windows
func NewManager(process process.WindowsProcessManager) (editor.Plugin, error) {
	return &winSublime{
		process:                process,
		commonInstallLocations: commonInstallLocations,
	}, nil
}

func newTestManager(process *process.MockManager) *winSublime {
	return &winSublime{
		process: process,
	}
}

type winSublime struct {
	process                process.WindowsProcessManager
	commonInstallLocations []string
}

func (s *winSublime) ID() string {
	return id
}

func (s *winSublime) Name() string {
	return name
}

// InstallConfig implements editor.Plugin
func (s *winSublime) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               s.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (s *winSublime) DetectEditors(ctx context.Context) ([]string, error) {
	var editors []string
	for _, installDir := range s.commonInstallLocations {
		editors = append(editors, filepath.Join(installDir, exeName))
	}
	return fs.KeepExistingFiles(editors), nil
}

func (s *winSublime) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := s.process.List(ctx)
	if err != nil {
		return nil, err
	}
	return list.MatchingExeName(ctx, []string{exeName})
}

func (s *winSublime) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	if !fs.FileExists(editorPath) {
		return system.Editor{}, fmt.Errorf("executable not found")
	}

	build, err := strconv.ParseInt(s.process.ExeProductVersion(editorPath), 10, 32)
	if err != nil {
		log.Printf("failed to parse product version as int for %s", editorPath)
		return editorConfig(editorPath, 0)
	}
	return editorConfig(editorPath, int(build))
}

func (s *winSublime) IsInstalled(ctx context.Context, path string) bool {
	// fixme validate directory structure?
	return fs.DirExists(filepath.Join(s.packagesDirectory(), pluginDirName))
}

func (s *winSublime) Install(ctx context.Context, editorPath string) error {
	return s.installOrUpdate()
}

func (s *winSublime) Update(ctx context.Context, editorPath string) error {
	return s.installOrUpdate()
}

func (s *winSublime) Uninstall(ctx context.Context, path string) error {
	return shared.UninstallPlugin(filepath.Join(s.packagesDirectory(), pluginDirName))
}

func (s *winSublime) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return openFile(s, editorPath, filePath, line)
}

func (s *winSublime) runSublime(editorPath string, args ...string) ([]byte, error) {
	return s.process.Run(editorPath, args...)
}

// cliPath implements the sublimeManager interface
// For Windows, the editorPath is sufficient to run the CLI.
func (s *winSublime) cliPath(editorPath string) string {
	return ""
}

// packagesDirectory retrieves the path to the user's Packages folder.
func (s *winSublime) packagesDirectory() string {
	return filepath.Join(fs.RoamingAppData(), "Sublime Text 3", "Packages")
}

// installedPackagesDirectory retrieves the path to the user's Installed Packages folder.
func (s *winSublime) installedPackagesDirectory() string {
	return filepath.Join(fs.RoamingAppData(), "Sublime Text 3", "Installed Packages")
}

func (s *winSublime) isRunning(ctx context.Context) bool {
	result, err := s.process.IsProcessRunning(ctx, exeName)
	if err != nil {
		return false
	}
	return result
}

func (s *winSublime) installOrUpdate() error {
	return shared.InstallOrUpdatePluginAssets(s.packagesDirectory(), pluginDirName, func(parentDir string) error {
		return sublime3.RestoreAssets(parentDir, pluginDirName)
	})
}
