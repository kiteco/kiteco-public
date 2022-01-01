package sublime

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins/sublimetext/sublime3"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

var (
	// most distrubutions use "subl". "subl3" is used by Arch Linux, for example.
	executableNames = []string{"subl", "subl3"}
	// sublime_text is used by Snap packages and on most Linux distributions
	// subl3 is used by Arch Linux, for example
	runningExecutableNames = []string{"sublime_text", "subl3"}
	snapName               = "sublime-text"
	snapInstall            = "opt/sublime_text/sublime_text"
)

var buildOutput = regexp.MustCompile(`Sublime Text Build (\d+)`)

// NewManager returns a new sublime plugin manager suitable for Linux
func NewManager(process process.LinuxProcessManager) (editor.Plugin, error) {
	path, err := sublimeUserDataLocation()
	if err != nil {
		return nil, err
	}

	return &linuxSublime{
		process:             process,
		sublimeUserDataPath: path,
	}, nil
}

func newTestManager(process *process.MockManager) *linuxSublime {
	return &linuxSublime{
		process: process,
	}
}

type linuxSublime struct {
	process             process.LinuxProcessManager
	sublimeUserDataPath string
}

func (s *linuxSublime) ID() string {
	return id
}

func (s *linuxSublime) Name() string {
	return name
}

// InstallConfig implements editor.Plugin
func (s *linuxSublime) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               s.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (s *linuxSublime) DetectEditors(ctx context.Context) ([]string, error) {
	var paths []string
	for _, name := range executableNames {
		p, err := exec.LookPath(name)
		if err != nil {
			log.Println(err)
			continue
		}
		p, err = shared.SnapPath(p, snapName, snapInstall)
		if err != nil {
			log.Println(err)
			continue
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (s *linuxSublime) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := s.process.List(ctx)
	if err != nil {
		return nil, err
	}
	return list.MatchingExeName(ctx, runningExecutableNames)
}

func (s *linuxSublime) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	n, err := s.version(editorPath)
	if err != nil {
		return system.Editor{}, err
	}
	return editorConfig(editorPath, n)
}

func (s *linuxSublime) IsInstalled(ctx context.Context, path string) bool {
	// fixme validate directory structure?
	return fs.DirExists(filepath.Join(s.packagesDirectory(), pluginDirName))
}

func (s *linuxSublime) Install(ctx context.Context, editorPath string) error {
	return s.installOrUpdate()
}

func (s *linuxSublime) Update(ctx context.Context, editorPath string) error {
	return s.installOrUpdate()
}

func (s *linuxSublime) Uninstall(ctx context.Context, path string) error {
	return shared.UninstallPlugin(filepath.Join(s.packagesDirectory(), pluginDirName))
}

func (s *linuxSublime) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return openFile(s, editorPath, filePath, line)
}

func (s *linuxSublime) runSublime(editorPath string, args ...string) ([]byte, error) {
	return s.process.Run(editorPath, args...)
}

// cliPath implements the sublimeManager interface
// For Linux, the editorPath is sufficient to run the CLI.
func (s *linuxSublime) cliPath(editorPath string) string {
	return ""
}

// packagesDirectory retrieves the path to the user's Packages folder.
func (s *linuxSublime) packagesDirectory() string {
	return filepath.Join(s.sublimeUserDataPath, "Packages")
}

// Version returns the build version retrieved from the `subl` command.
func (s *linuxSublime) version(executablePath string) (int, error) {
	out, err := s.process.Run(executablePath, "--version")
	if err != nil {
		return 0, err
	}
	output := string(out)
	matches := buildOutput.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("unexpected build output: %s", output)
	}
	return strconv.Atoi(matches[1])
}

func sublimeUserDataLocation() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".config", "sublime-text-3"), nil
}

func (s *linuxSublime) isRunning(ctx context.Context) bool {
	for _, exeName := range runningExecutableNames {
		if result, err := s.process.IsProcessRunning(ctx, exeName); result && err == nil {
			return true
		}
	}
	return false
}

func (s *linuxSublime) installOrUpdate() error {
	return shared.InstallOrUpdatePluginAssets(s.packagesDirectory(), pluginDirName, func(parentDir string) error {
		return sublime3.RestoreAssets(parentDir, pluginDirName)
	})
}
