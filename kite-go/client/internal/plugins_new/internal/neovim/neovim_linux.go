package neovim

import (
	"context"
	"errors"
	"log"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins/vim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	validator "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/vim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

const (
	executableName = "nvim"
	pluginRoot     = ".config/nvim"
)

// NewManager returns a new Neovim plugin manager suitable for Linux.
func NewManager(process process.LinuxProcessManager) (editor.Plugin, error) {
	return &linuxNeovim{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *linuxNeovim {
	return &linuxNeovim{
		process: process,
	}
}

type linuxNeovim struct {
	process process.LinuxProcessManager
}

func (n *linuxNeovim) ID() string {
	return neovimID
}

func (n *linuxNeovim) Name() string {
	return neovimName
}

// InstallConfig implements editor.Plugin
func (n *linuxNeovim) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               n.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (n *linuxNeovim) DetectEditors(ctx context.Context) ([]string, error) {
	return n.process.FindBinary(executableName), nil
}

func (n *linuxNeovim) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := n.process.List(ctx)
	if err != nil {
		return nil, err
	}
	return list.MatchingExeName(ctx, []string{executableName, "nvim-qt"})
}

func (n *linuxNeovim) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	bytes, err := n.process.Run(editorPath, argsVersion)
	if err != nil {
		return system.Editor{}, err
	}
	return validator.ValidateExecutable(string(bytes), editorPath, validate, parseVersion, requiredEditorVersion)
}

func (n *linuxNeovim) IsInstalled(ctx context.Context, editorPath string) bool {
	pluginParentDir, err := n.pluginParentDir()
	return err == nil && fs.DirExists(filepath.Join(pluginParentDir, neovimPluginName))
}

func (n *linuxNeovim) Install(ctx context.Context, editorPath string) error {
	return n.installOrUpdate()
}

func (n *linuxNeovim) Update(ctx context.Context, editorPath string) error {
	return n.installOrUpdate()
}

func (n *linuxNeovim) Uninstall(ctx context.Context, editorPath string) error {
	home, err := n.process.Home()
	if err != nil {
		log.Printf("Uninstall failed with error: %s", err.Error())
		return err
	}
	// Remove $HOME/.config/nvim/pack/kite/ and its subdirectories.
	return shared.UninstallPlugin(filepath.Join(home, pluginRoot, neovimPluginDir))
}

func (n *linuxNeovim) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

func (n *linuxNeovim) isRunning(ctx context.Context) bool {
	result, err := n.process.IsProcessRunning(ctx, executableName)
	if err == nil && result == true {
		return true
	}
	return false
}

func (n *linuxNeovim) pluginParentDir() (string, error) {
	home, err := n.process.Home()
	return filepath.Join(home, pluginRoot, neovimPluginDir, neovimPluginPathPrefix), err
}

func (n *linuxNeovim) installOrUpdate() error {
	parentDir, err := n.pluginParentDir()
	if err != nil {
		return err
	}
	return shared.InstallOrUpdatePluginAssets(parentDir, neovimPluginName, func(parentDir string) error {
		return vim.RestoreAssets(parentDir, neovimPluginName)
	})
}
