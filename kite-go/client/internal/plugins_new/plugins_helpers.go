package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/errors"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

func (m *Manager) installedProductIDs() []string {
	ids := make([]string, 0)
	for _, mgr := range m.pluginManagers {
		if ip, ok := mgr.(internal.InstalledProductIDs); ok {
			ids = append(ids, ip.InstalledProductIDs(context.Background())...)
		}
	}
	return ids
}

func (m *Manager) getPluginManager(r *http.Request) (editor.Plugin, bool) {
	vars := mux.Vars(r)
	id := vars["id"]
	p, found := m.pluginManagers[id]

	// accept JetBrains build ids as an alternative to Kite's own IDs
	if !found {
		for _, mgr := range m.pluginManagers {
			if ids, ok := mgr.(internal.AdditionalIPluginDs); ok && shared.StringsContain(ids.AdditionalIDs(), id) {
				return mgr, true
			}
		}
	}

	return p, found
}

func (m *Manager) detectEditors(ctx context.Context, manager editor.Plugin) ([]system.Editor, *errorResponse) {
	installs, err := manager.DetectEditors(ctx)
	if err != nil {
		errorObj := newErrorResponse(fmt.Sprintf("Failed to detect installs for  %s", manager.Name()), err)
		return []system.Editor{}, errorObj
	}

	installs = append(installs, m.editors.detected(manager.ID())...)

	return shared.MapEditors(ctx, installs, manager), nil
}

func (m *Manager) markEncountered(s *PluginStatus) *PluginStatus {
	m.m.Lock()
	defer m.m.Unlock()
	s.Encountered = m.encountered[s.ID]
	return s
}

// reportInstallError reports an error which occurred while installing or updating a plugin.
// If err is a ProcessError, then stdio will also be reported.
func (m *Manager) reportInstallError(title string, path string, manager editor.Plugin, err error) {
	props := map[string]string{
		"error":  err.Error(),
		"editor": manager.Name(),
		"path":   path,
	}
	if pErr, ok := err.(process.Error); ok {
		props["stdout"] = pErr.Stdout()
		props["stderr"] = pErr.Stderr()
	}
	if pErr, ok := err.(errors.UI); ok {
		props["ui"] = pErr.UI()
	}
	log.Printf("install error: %#v", props)
}

// BackgroundTask repeatedly updates the list of detected editors, updates plugins and automatically
// install plugins (if the setting is enabled) for all editors which were not yet encountered.
func (m *Manager) BackgroundTask(ctx context.Context, interval time.Duration, updatePlugins bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var toUpdate map[string][]string
	if updatePlugins {
		// Grab installed plugins.
		toUpdate = m.pluginsToUpdate(ctx)
	}

	for {
		// update and clean-up list of detected, running editors
		m.refreshRunning(ctx)

		if len(toUpdate) > 0 {
			// update plugins, which were not yet updated since startup
			// keep plugins which failed to update for the next iteration
			log.Println(updateMsg(toUpdate))
			toUpdate = m.update(ctx, toUpdate)
		}

		m.AutoInstallPlugins(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// continue with the loop
		}
	}
}

func getPlugin(plugin editor.Plugin, err error) editor.Plugin {
	if err != nil {
		panic(fmt.Sprintf("Error creating editor.Plugin for plugin: %s", plugin.Name()))
	}
	return plugin
}

func getPlugins(plugins []editor.Plugin, err error) []editor.Plugin {
	if err != nil {
		panic(fmt.Sprintf("Error creating plugins: %s", err.Error()))
	}
	return plugins
}

// Transforms a list of editor.Plugins into a map of Plugin.ID : editor.Plugin
func buildPluginMap(plugins []editor.Plugin) map[string]editor.Plugin {
	pluginManagers := make(map[string]editor.Plugin)
	for _, plugin := range plugins {
		pluginManagers[plugin.ID()] = plugin
	}
	return pluginManagers
}

// Transforms a list of editor.Plugins into a list of Plugin.IDs, then sorts
// them alphabetically.
func buildPluginKeys(plugins []editor.Plugin) []string {
	keys := make([]string, len(plugins))
	for i, plugin := range plugins {
		keys[i] = plugin.ID()
	}
	sort.Strings(keys)
	return keys
}

func editorPath(r *http.Request) string {
	return r.URL.Query().Get("path")
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}
