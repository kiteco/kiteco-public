package plugins

import (
	"context"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/errors"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

// TODO this will be updated/removed when we change the response format on the frontend

// PluginResponse contains the status of all editor plugins.
type PluginResponse struct {
	Plugins []*PluginStatus `json:"plugins"`
}

type uninstallAllResponse struct {
	PluginResponse
	Errors []*errorResponse `json:"errors,omitempty"`
}

// A subset of http://jsonapi.org/format/#error-objects
type errorResponse struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func newErrorResponse(title string, err error) *errorResponse {
	resp := errorResponse{
		Title:  title,
		Detail: err.Error(),
	}
	if ui, ok := err.(errors.UI); ok {
		resp.Detail = ui.UI()
	}
	return &resp
}

// PluginStatus describes a Kite plugin for a particular editor family on the user's system.
type PluginStatus struct {
	ID                       string         `json:"id"`
	Name                     string         `json:"name"`
	RequiresRestart          bool           `json:"requires_restart"`
	MultipleInstallLocations bool           `json:"multiple_install_locations"`
	Running                  bool           `json:"running"`
	InstallWhileRunning      bool           `json:"install_while_running"`
	UpdateWhileRunning       bool           `json:"update_while_running"`
	UninstallWhileRunning    bool           `json:"uninstall_while_running"`
	ManualInstallOnly        bool           `json:"manual_install_only"`
	Encountered              bool           `json:"encountered"`
	Editors                  []EditorStatus `json:"editors"`
}

// EditorStatus describes an editor install at given path, with plugin metadata.
type EditorStatus struct {
	system.Editor
	PluginInstalled bool `json:"plugin_installed"`
}

// status queries for all information about a plugin.
func status(ctx context.Context, p editor.Plugin, editors []system.Editor) *PluginStatus {
	installs := make([]EditorStatus, 0)
	for _, editor := range editors {
		install := EditorStatus{
			Editor:          editor,
			PluginInstalled: p.IsInstalled(ctx, editor.Path),
		}
		installs = append(installs, install)
	}

	installConfig := p.InstallConfig(ctx)

	return &PluginStatus{
		ID:                       p.ID(),
		Name:                     p.Name(),
		RequiresRestart:          installConfig.RequiresRestart,
		MultipleInstallLocations: installConfig.MultipleInstallLocations,
		Running:                  installConfig.Running,
		InstallWhileRunning:      installConfig.InstallWhileRunning,
		UpdateWhileRunning:       installConfig.UpdateWhileRunning,
		UninstallWhileRunning:    installConfig.UninstallWhileRunning,
		ManualInstallOnly:        installConfig.ManualInstallOnly,
		Editors:                  installs,
	}
}
