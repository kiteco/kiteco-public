package vim

import (
	"context"
	"errors"
	"log"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins/vim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

const (
	pluginRoot = "vimfiles"
	// not using gvim here because:
	// - gvim.exe --version displays a dialog window, so we're unable to use it
	// - the version metadata of the exe file isn't updated for builds of vim
	// this means that there's no data we can use to detect compatible versions of gvim
	exeName = "vim.exe"
)

var (
	pathExeNames = []string{"vim", "vim80", "vim81"}
)

// NewManager returns a new vim plugin manager suitable for Windows.
func NewManager(process process.WindowsProcessManager) (editor.Plugin, error) {
	return &windowsVim{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *windowsVim {
	return &windowsVim{
		process: process,
	}
}

type windowsVim struct {
	process process.WindowsProcessManager
}

func (v *windowsVim) ID() string {
	return vimID
}

func (v *windowsVim) Name() string {
	return vimName
}

// InstallConfig implements editor.Plugin
func (v *windowsVim) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               v.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (v *windowsVim) DetectEditors(ctx context.Context) ([]string, error) {
	var paths []string
	for _, executable := range pathExeNames {
		paths = append(paths, v.process.FindBinary(executable)...)
	}

	targets := v.process.FilterStartMenuTargets(func(t string) bool {
		return filepath.Base(t) == exeName
	})

	return system.Union(paths, targets), nil
}

func (v *windowsVim) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := v.process.List(ctx)
	if err != nil {
		return nil, err
	}
	return list.MatchingExeName(ctx, []string{exeName})
}

func (v *windowsVim) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	output, err := v.runBinary(editorPath)
	if err != nil {
		return system.Editor{}, err
	}
	return ValidateExecutable(output, editorPath, validate, parseVersion, requiredEditorVersion)
}

func (v *windowsVim) IsInstalled(ctx context.Context, editorPath string) bool {
	pluginParentDir, err := v.pluginParentDir()
	return err == nil && fs.DirExists(filepath.Join(pluginParentDir, vimPluginName))
}

func (v *windowsVim) Install(ctx context.Context, editorPath string) error {
	return v.installOrUpdate()
}

func (v *windowsVim) Update(ctx context.Context, editorPath string) error {
	return v.installOrUpdate()
}

func (v *windowsVim) Uninstall(ctx context.Context, editorPath string) error {
	home, err := v.process.Home()
	if err != nil {
		log.Printf("Uninstall failed with error: %s", err.Error())
		return err
	}
	// Remove $HOME/vimfiles/pack/kite/ and its subdirectories.
	return shared.UninstallPlugin(filepath.Join(home, pluginRoot, vimPluginDir))
}

func (v *windowsVim) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

// Runs binary on executable path, then returns the output as a string.
func (v *windowsVim) runBinary(executablePath string) (string, error) {
	bytes, err := v.process.Run(executablePath, args)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (v *windowsVim) isRunning(ctx context.Context) bool {
	result, err := v.process.IsProcessRunning(ctx, exeName)
	if err != nil {
		return false
	}
	return result
}

func (v *windowsVim) pluginParentDir() (string, error) {
	home, err := v.process.Home()
	return filepath.Join(home, pluginRoot, vimPluginDir, vimPluginPathPrefix), err
}

func (v *windowsVim) installOrUpdate() error {
	parentDir, err := v.pluginParentDir()
	if err != nil {
		return err
	}
	return shared.InstallOrUpdatePluginAssets(parentDir, vimPluginName, func(parentDir string) error {
		return vim.RestoreAssets(parentDir, vimPluginName)
	})
}
