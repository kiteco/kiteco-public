package editor

import (
	"context"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

// Plugin defines the interface to support one editor
type Plugin interface {
	ID() string
	Name() string
	InstallConfig(ctx context.Context) *InstallConfig

	// DetectEditors detects editors on disk.
	DetectEditors(ctx context.Context) ([]string, error)
	// DetectRunningEditors detects possible install locations of
	// the editor by analyzing the process list
	DetectRunningEditors(ctx context.Context) ([]string, error)
	// EditorConfig provides compatibility information for the given editor path
	EditorConfig(ctx context.Context, editorPath string) (system.Editor, error)

	IsInstalled(ctx context.Context, editorPath string) bool
	// Install installs the plugin assets for the editor at editorPath
	// Callers must validate that the plugin is allowed to be installed
	// by checking the InstallConfig() first before calling Install()
	Install(ctx context.Context, editorPath string) error
	// Uninstall uninstalls the plugin assets for the editor at editorPath
	// Callers must validate that the plugin is allowed to be uninstalled
	// by checking the InstallConfig() first before calling Uninstall()
	Uninstall(ctx context.Context, editorPath string) error
	// Update updates the plugin assets for the editor at editorPath
	// Callers must validate that the plugin is allowed to be updated
	// by checking the InstallConfig() first before calling Update()
	Update(ctx context.Context, editorPath string) error
	// OpenFile opens the specified filePath,
	// line is 1-based, a value <= 0 means that it's not set.
	// The returned error is for immediate errors or when short-lived processes were started.
	// The returned error channel is non-nil, if a background process was started.
	OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error)
}

// InstallConfig configures behavior regarding installation & uninstallation.
type InstallConfig struct {
	RequiresRestart          bool
	MultipleInstallLocations bool
	Running                  bool
	InstallWhileRunning      bool
	UpdateWhileRunning       bool
	UninstallWhileRunning    bool
	ManualInstallOnly        bool
}
