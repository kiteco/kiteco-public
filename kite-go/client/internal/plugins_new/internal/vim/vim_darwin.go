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
	bundleID       = "org.vim.MacVim"
	executableName = "vim"
	macVimPath     = "Contents/MacOS/Vim"
	pluginRoot     = ".vim"
)

// NewManager returns a new vim plugin manager suitable for Mac.
func NewManager(process process.MacProcessManager) (editor.Plugin, error) {
	return &macVim{
		process: process,
	}, nil
}

func newTestManager(process *process.MockManager) *macVim {
	return &macVim{
		process: process,
	}
}

type macVim struct {
	process process.MacProcessManager
}

func (v *macVim) ID() string {
	return vimID
}

func (v *macVim) Name() string {
	return vimName
}

// InstallConfig implements editor.Plugin
func (v *macVim) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:       true,
		Running:               v.isEditorRunning(),
		InstallWhileRunning:   true,
		UpdateWhileRunning:    true,
		UninstallWhileRunning: true,
	}
}

func (v *macVim) DetectEditors(ctx context.Context) ([]string, error) {
	var macBundleLocations []string
	if bundleLocations, err := v.process.BundleLocations(ctx, bundleID); err == nil {
		macBundleLocations = shared.MapStrings(bundleLocations, func(path string) string {
			return filepath.Join(path, macVimPath)
		})
	}

	binary := v.process.FindBinary(executableName)
	return system.Union(binary, macBundleLocations), nil
}

func (v *macVim) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := v.process.RunningApplications()
	if err != nil {
		return nil, err
	}
	return list.Matching(func(process process.Process) string {
		if process.BundleID == bundleID && process.BundleLocation != "" {
			return filepath.Join(process.BundleLocation, macVimPath)
		}
		return ""
	}), nil
}

func (v *macVim) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	output, err := v.runBinary(editorPath)
	if err != nil {
		return system.Editor{}, err
	}

	return ValidateExecutable(output, editorPath, validate, parseVersion, requiredEditorVersion)
}

func (v *macVim) IsInstalled(ctx context.Context, editorPath string) bool {
	pluginParentDir, err := v.pluginParentDir()
	return err == nil && fs.DirExists(filepath.Join(pluginParentDir, vimPluginName))

}

func (v *macVim) Install(ctx context.Context, editorPath string) error {
	return v.installOrUpdate()
}

func (v *macVim) Update(ctx context.Context, editorPath string) error {
	return v.installOrUpdate()
}

func (v *macVim) Uninstall(ctx context.Context, editorPath string) error {
	home, err := v.process.Home()
	if err != nil {
		log.Printf("Uninstall failed with error: %s", err.Error())
		return err
	}
	// Remove $HOME/.vim/pack/kite/ and its subdirectories.
	return shared.UninstallPlugin(filepath.Join(home, pluginRoot, vimPluginDir))
}

func (v *macVim) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

// Runs binary on executable path, then returns the output as a string.
func (v *macVim) runBinary(executablePath string) (string, error) {
	bytes, err := v.process.Run(executablePath, args)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Creates Editor for MacVim installation if it exists.
func (v *macVim) findGraphicalEditors(ctx context.Context) []system.Editor {
	log.Println("Looking for MacVim bundle")
	var installs []system.Editor
	locations, err := v.process.BundleLocations(ctx, bundleID)
	if err != nil {
		log.Println("MacVim bundle not found")
		return installs
	}
	for _, path := range locations {
		mVimPath := filepath.Join(path, macVimPath)
		output, err := v.runBinary(mVimPath)
		if err != nil {
			log.Printf("got error running binary of %s: %v", path, err)
			continue
		}
		validEditor, err := ValidateExecutable(output, mVimPath, validate,
			parseVersion, requiredEditorVersion)
		if err != nil {
			log.Printf("got error looking up version for %s: %v", path, err)
			continue
		}
		installs = append(installs, validEditor)
	}
	return installs
}

func (v *macVim) isEditorRunning() bool {
	result, err := v.process.IsProcessRunning(executableName)
	if err != nil {
		log.Printf("isRunning failed with error: %s", err.Error())
	}
	// Returns true if MacVim or Vim is active.
	return v.process.IsBundleRunning(bundleID) || result
}

func (v *macVim) pluginParentDir() (string, error) {
	home, err := v.process.Home()
	return filepath.Join(home, pluginRoot, vimPluginDir, vimPluginPathPrefix), err
}

func (v *macVim) installOrUpdate() error {
	parentDir, err := v.pluginParentDir()
	if err != nil {
		return err
	}
	return shared.InstallOrUpdatePluginAssets(parentDir, vimPluginName, func(parentDir string) error {
		return vim.RestoreAssets(parentDir, vimPluginName)
	})
}
