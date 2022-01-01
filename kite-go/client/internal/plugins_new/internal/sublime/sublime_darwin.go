package sublime

import (
	"context"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins/sublimetext/sublime3"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system/nsbundle"
)

const (
	bundleID = "com.sublimetext.3"
)

// NewManager returns a new sublime plugin manager suitable for macOS
func NewManager(process process.MacProcessManager) (editor.Plugin, error) {
	path, err := sublimeUserDataLocation()
	if err != nil {
		return nil, err
	}

	return &macSublime{
		process:             process,
		sublimeUserDataPath: path,
	}, nil
}

func newTestManager(process *process.MockManager) *macSublime {
	return &macSublime{
		process: process,
	}
}

type macSublime struct {
	process             process.MacProcessManager
	sublimeUserDataPath string
}

func (s *macSublime) ID() string {
	return id
}

func (s *macSublime) Name() string {
	return name
}

// InstallConfig implements editor.Plugin
func (s *macSublime) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               s.process.IsBundleRunning(bundleID),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (s *macSublime) DetectEditors(ctx context.Context) ([]string, error) {
	return s.process.BundleLocations(ctx, bundleID)
}

func (s *macSublime) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := s.process.RunningApplications()
	if err != nil {
		return nil, err
	}
	return list.Matching(func(process process.Process) string {
		if process.BundleID == bundleID && process.BundleLocation != "" {
			return process.BundleLocation
		}
		return ""
	}), nil
}

func (s *macSublime) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	// fixme handle error returned by AppVersion
	// our bundle id is for Sublime Text 3, so there's no need to validate the version
	version, _ := nsbundle.AppVersion(editorPath)
	return system.Editor{
		Version: version,
		Path:    editorPath,
	}, nil
}

func (s *macSublime) IsInstalled(ctx context.Context, path string) bool {
	// fixme validate directory structure?
	return fs.DirExists(filepath.Join(s.packagesDirectory(), pluginDirName))
}

func (s *macSublime) Install(ctx context.Context, editorPath string) error {
	return s.installOrUpdate()
}

func (s *macSublime) Update(ctx context.Context, editorPath string) error {
	return s.installOrUpdate()
}

func (s *macSublime) Uninstall(ctx context.Context, path string) error {
	return shared.UninstallPlugin(filepath.Join(s.packagesDirectory(), pluginDirName))
}

func (s *macSublime) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	// If editorPath is sent by the plugin, it's suffixed with Contents/MacOS/Sublime Text
	editorPath = strings.TrimSuffix(editorPath, "/Contents/MacOS/Sublime Text")
	return openFile(s, editorPath, filePath, line)
}

func (s *macSublime) runSublime(editorPath string, args ...string) ([]byte, error) {
	return s.process.Run(s.cliPath(editorPath), args...)
}

func (s *macSublime) cliPath(editorPath string) string {
	return filepath.Join(editorPath, "Contents", "SharedSupport", "bin", "subl")
}

// packagesDirectory retrieves the path to the user's Packages folder.
func (s *macSublime) packagesDirectory() string {
	return filepath.Join(s.sublimeUserDataPath, "Packages")
}

// installedPackagesDirectory retrieves the path to the user's Installed Packages folder.
func (s *macSublime) installedPackagesDirectory() string {
	return filepath.Join(s.sublimeUserDataPath, "Installed Packages")
}

func (s *macSublime) installOrUpdate() error {
	return shared.InstallOrUpdatePluginAssets(s.packagesDirectory(), pluginDirName, func(parentDir string) error {
		return sublime3.RestoreAssets(parentDir, pluginDirName)
	})
}

// packagesDirectory retrieves the path to the user's Packages folder.
func sublimeUserDataLocation() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, "Library", "Application Support", "Sublime Text 3"), nil
}
