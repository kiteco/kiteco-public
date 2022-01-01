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
	pluginRoot = ".vim"
)

var (
	executableNames = []string{"vim", "gvim"}
)

// NewManager returns a new Vim plugin manager suitable for Linux.
func NewManager(process process.LinuxProcessManager) (editor.Plugin, error) {
	return &linuxVim{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *linuxVim {
	return &linuxVim{
		process: process,
	}
}

type linuxVim struct {
	process process.LinuxProcessManager
}

func (v *linuxVim) ID() string {
	return vimID
}

func (v *linuxVim) Name() string {
	return vimName
}

// InstallConfig implements editor.Plugin
func (v *linuxVim) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               v.isRunning(ctx),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (v *linuxVim) DetectEditors(ctx context.Context) ([]string, error) {
	var paths []string
	for _, executableName := range executableNames {
		paths = append(paths, v.process.FindBinary(executableName)...)
	}
	return paths, nil
}

func (v *linuxVim) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := v.process.List(ctx)
	if err != nil {
		return nil, err
	}
	return list.MatchingExeName(ctx, executableNames)
}

func (v *linuxVim) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	output, err := v.runBinary(editorPath)
	if err != nil {
		return system.Editor{}, err
	}
	return ValidateExecutable(output, editorPath, validate, parseVersion, requiredEditorVersion)
}

func (v *linuxVim) IsInstalled(ctx context.Context, editorPath string) bool {
	parentDir, err := v.pluginParentDir()
	return err == nil && fs.DirExists(filepath.Join(parentDir, vimPluginName))
}

func (v *linuxVim) Install(ctx context.Context, editorPath string) error {
	return v.installOrUpdate()
}

func (v *linuxVim) Update(ctx context.Context, editorPath string) error {
	return v.installOrUpdate()
}

func (v *linuxVim) Uninstall(ctx context.Context, editorPath string) error {
	home, err := v.process.Home()
	if err != nil {
		log.Printf("Uninstall failed with error: %s", err.Error())
		return err
	}
	// Remove $HOME/.vim/pack/kite/ and its subdirectories.
	return shared.UninstallPlugin(filepath.Join(home, pluginRoot, vimPluginDir))
}

func (v *linuxVim) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

// Runs binary on executable path, then returns the output as a string.
func (v *linuxVim) runBinary(executablePath string) (string, error) {
	// -v enabled vim mode and disables the gvim mode.
	// It has to be passed before --version
	bytes, err := v.process.Run(executablePath, "-v", args)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (v *linuxVim) isRunning(ctx context.Context) bool {
	for _, name := range executableNames {
		result, err := v.process.IsProcessRunning(ctx, name)
		if err == nil && result == true {
			return true
		}
	}
	return false
}

func (v *linuxVim) pluginParentDir() (string, error) {
	home, err := v.process.Home()
	return filepath.Join(home, pluginRoot, vimPluginDir, vimPluginPathPrefix), err
}

func (v *linuxVim) installOrUpdate() error {
	parentDir, err := v.pluginParentDir()
	if err != nil {
		return err
	}
	return shared.InstallOrUpdatePluginAssets(parentDir, vimPluginName, func(parentDir string) error {
		return vim.RestoreAssets(parentDir, vimPluginName)
	})
}
