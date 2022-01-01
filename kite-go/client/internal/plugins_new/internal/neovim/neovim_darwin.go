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

// NewManager returns a new Neovim plugin manager suitable for Darwin.
func NewManager(process process.MacProcessManager) (editor.Plugin, error) {
	return &macNeovim{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *macNeovim {
	return &macNeovim{
		process: process,
	}
}

// InstallConfig implements editor.Plugin
func (n *macNeovim) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               n.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (n *macNeovim) ID() string {
	return neovimID
}

func (n *macNeovim) Name() string {
	return neovimName
}

type macNeovim struct {
	process process.MacProcessManager
}

func (n *macNeovim) DetectEditors(ctx context.Context) ([]string, error) {
	paths := n.process.FindBinary(executableName)
	return paths, nil
}

func (n *macNeovim) DetectRunningEditors(ctx context.Context) ([]string, error) {
	// NVim is not available on macOS as an application bundle
	return []string{}, nil
}

func (n *macNeovim) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	bytes, err := n.process.Run(editorPath, argsVersion)
	if err != nil {
		return system.Editor{}, err
	}
	return validator.ValidateExecutable(string(bytes), editorPath, validate, parseVersion, requiredEditorVersion)
}

func (n *macNeovim) IsInstalled(ctx context.Context, editorPath string) bool {
	pluginParentDir, err := n.pluginParentDir()
	return err == nil && fs.DirExists(filepath.Join(pluginParentDir, neovimPluginName))
}

func (n *macNeovim) Install(ctx context.Context, editorPath string) error {
	return n.installOrUpdate()
}

func (n *macNeovim) Update(ctx context.Context, editorPath string) error {
	return n.installOrUpdate()
}

func (n *macNeovim) Uninstall(ctx context.Context, editorPath string) error {
	home, err := n.process.Home()
	if err != nil {
		log.Printf("Uninstall failed with error: %s", err.Error())
		return err
	}
	// Remove $HOME/.config/nvim/pack/kite/ and its subdirectories.
	return shared.UninstallPlugin(filepath.Join(home, pluginRoot, neovimPluginDir))
}

func (n *macNeovim) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

func (n *macNeovim) isRunning(ctx context.Context) bool {
	result, err := n.process.IsProcessRunning(executableName)
	if err != nil {
		log.Printf("isRunning failed with error: %s", err.Error())
	}
	return result
}

func (n *macNeovim) pluginParentDir() (string, error) {
	home, err := n.process.Home()
	return filepath.Join(home, pluginRoot, neovimPluginDir, neovimPluginPathPrefix), err
}

func (n *macNeovim) installOrUpdate() error {
	parentDir, err := n.pluginParentDir()
	if err != nil {
		return err
	}
	return shared.InstallOrUpdatePluginAssets(parentDir, neovimPluginName, func(parentDir string) error {
		return vim.RestoreAssets(parentDir, neovimPluginName)
	})
}
