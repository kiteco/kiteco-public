package neovim

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	validator "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/vim"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins/vim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

var (
	executableNames = []string{
		"nvim.exe",
		"nvim-qt.exe",
	}
	pluginRoot        = "nvim"
	versionMatcherExe = regexp.MustCompile(`^([^\s\r\n]+)`)
)

// NewManager returns a new Neovim plugin manager suitable for Windows.
func NewManager(process process.WindowsProcessManager) (editor.Plugin, error) {
	return &windowsNeovim{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *windowsNeovim {
	return &windowsNeovim{
		process: process,
	}
}

// InstallConfig implements editor.Plugin
func (n *windowsNeovim) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               n.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (n *windowsNeovim) ID() string {
	return neovimID
}

func (n *windowsNeovim) Name() string {
	return neovimName
}

type windowsNeovim struct {
	process process.WindowsProcessManager
}

func (n *windowsNeovim) DetectEditors(ctx context.Context) ([]string, error) {
	var paths []string
	for _, executableName := range executableNames {
		paths = append(paths, n.process.FindBinary(executableName)...)
	}
	return paths, nil
}

func (n *windowsNeovim) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := n.process.List(ctx)
	if err != nil {
		return nil, err
	}
	return list.MatchingExeName(ctx, executableNames)
}

func (n *windowsNeovim) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	var version string

	// nvim-qt.exe isn't supporting --version
	// instead, we look at the version info in the exe properties
	if filepath.Base(editorPath) == "nvim-qt.exe" {
		version = n.process.ExeProductVersion(editorPath)
		if version == "" {
			return system.Editor{}, fmt.Errorf("unable to retrieve version for neovim at %s", editorPath)
		}
	} else {
		bytes, err := n.process.Run(editorPath, argsVersion)
		if err != nil {
			return system.Editor{}, err
		}
		version = string(bytes)
	}

	return validator.ValidateExecutable(version, editorPath, validate, parseVersion, requiredEditorVersion)
}

func (n *windowsNeovim) IsInstalled(ctx context.Context, editorPath string) bool {
	pluginParentDir, err := n.pluginParentDir()
	return err == nil && fs.DirExists(filepath.Join(pluginParentDir, neovimPluginName))
}

func (n *windowsNeovim) Install(ctx context.Context, editorPath string) error {
	return n.installOrUpdate()
}

func (n *windowsNeovim) Update(ctx context.Context, editorPath string) error {
	return n.installOrUpdate()
}

func (n *windowsNeovim) Uninstall(ctx context.Context, editorPath string) error {
	localAppData, err := n.process.LocalAppData()
	if err != nil {
		log.Printf("Uninstall failed with error: %s", err.Error())
		return err
	}
	// Remove %LOCALAPPDATA%/nvim/pack/kite/ and its subdirectories.
	return shared.UninstallPlugin(filepath.Join(localAppData, pluginRoot, neovimPluginDir))
}

func (n *windowsNeovim) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

func (n *windowsNeovim) isRunning(ctx context.Context) bool {
	for _, exeName := range executableNames {
		result, _ := n.process.IsProcessRunning(ctx, exeName)
		if result {
			return true
		}
	}
	return false
}

func (n *windowsNeovim) pluginParentDir() (string, error) {
	localAppData, err := n.process.LocalAppData()
	return filepath.Join(localAppData, pluginRoot, neovimPluginDir, neovimPluginPathPrefix), err
}

func (n *windowsNeovim) installOrUpdate() error {
	parentDir, err := n.pluginParentDir()
	if err != nil {
		return err
	}
	return shared.InstallOrUpdatePluginAssets(parentDir, neovimPluginName, func(parentDir string) error {
		return vim.RestoreAssets(parentDir, neovimPluginName)
	})
}
