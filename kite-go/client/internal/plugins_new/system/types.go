package system

import (
	"errors"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

// Options to pass to plugin managers and the main manager
type Options struct {
	DevMode     bool
	BetaChannel bool
}

// ErrProcessRunning indicates that a plugin cannot be updated because the editor is
// currently running.
var ErrProcessRunning = errors.New("process is running")

// ProcessError implements error. It wraps an error message, stdout, and stderr of the process which failed to execute
type ProcessError struct {
	msg    string
	stderr string
	stdout string
}

// Error returns the error message intended for users
func (e ProcessError) Error() string {
	return e.msg
}

// Stdout returns the content which was printed to stdout by the failed process
func (e ProcessError) Stdout() string {
	return e.stdout
}

// Stderr returns the content which was printed to stderr by the failed process
func (e ProcessError) Stderr() string {
	return e.stderr
}

// Editor records the path and version of an editor installation.
type Editor struct {
	Path          string `json:"path"`
	Version       string `json:"version"`
	Compatibility string `json:"compatibility,omitempty"`
	// property to signal that the current version is incompatible,
	// an empty means that Version is compatible
	RequiredVersion string `json:"version_required,omitempty"`
}

// EditorFinder finds all editor installations.
type EditorFinder func() []Editor

// EditorProcessChecker returns whether or not the editor is running
type EditorProcessChecker func() bool

// EditorInstallWhileRunningChecker returns whether or not a plugin can be
// installed while the editor is running
type EditorInstallWhileRunningChecker func() bool

// EditorUninstallWhileRunningChecker returns whether or not a plugin can be
// uninstalled while the editor is running
type EditorUninstallWhileRunningChecker func() bool

// PluginInstaller installs the plugin for the editor at the given path.
type PluginInstaller func(string) error

// PluginUninstaller uninstalls the plugin for the editor at the given path.
type PluginUninstaller func(string) error

// PluginInstallerLocal installs the plugin for the editor at the given path when Kite Local is enabled.
type PluginInstallerLocal func(string) error

// PluginUninstallerLocal uninstalls the plugin for the editor at the given path when Kite Local is enabled.
type PluginUninstallerLocal func(string) error

// PluginChecker returns whether the plugin for the editor at the given path is installed.
type PluginChecker func(string) bool

// PluginManager holds functions to add and remove plugins.
type PluginManager struct {
	ID                       string
	Name                     string
	Icon                     string
	RequiresRestart          bool
	MultipleInstallLocations bool

	FindEditors           EditorFinder
	EditorRunning         EditorProcessChecker
	InstallWhileRunning   EditorInstallWhileRunningChecker
	UninstallWhileRunning EditorUninstallWhileRunningChecker
	InstallPlugin         PluginInstaller
	UninstallPlugin       PluginUninstaller
	PluginInstalled       PluginChecker

	// For Kite Local
	InstallLocalPlugin   PluginInstallerLocal
	UninstallLocalPlugin PluginUninstallerLocal
}

// PluginComponent provides a shared interface for the old and new plugin manager components.
// TODO: Remove once transition to new plugin manager is complete.
type PluginComponent interface {
	component.Core
	UpdateAllInstalled()
}
