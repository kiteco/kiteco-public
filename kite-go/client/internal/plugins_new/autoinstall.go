package plugins

import (
	"context"
	"log"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
)

// AutoInstallPlugins automatically installs plugins for all newly discovered editors
// Editors are detected by facilitating DetectEditors() and DetectRunningEditors().
// Plugins which are already contained in the list of encountered editors
// are not installed by this method.
// Returns the number of installed plugins
func (m *Manager) AutoInstallPlugins(ctx context.Context) int {
	// install all plugins for editors which were never encountered before
	if enabled, _ := m.settings.GetBool(settings.AutoInstallPluginsEnabledKey); !enabled {
		return 0
	}

	paths := make(map[string][]string)
	for _, mgr := range m.encounteredEditors(false) {
		id := mgr.ID()

		editorPaths := m.editors.detected(id)
		if editors, err := mgr.DetectEditors(ctx); err == nil {
			editorPaths = append(editorPaths, editors...)
		}

		paths[id] = shared.DedupePaths(editorPaths)
	}

	return m.autoInstallPlugins(ctx, paths)
}

func (m *Manager) autoInstallPlugins(ctx context.Context, paths map[string][]string) int {
	// install all plugins for editors which were never encountered before
	if enabled, _ := m.settings.GetBool(settings.AutoInstallPluginsEnabledKey); !enabled {
		return 0
	}

	installed := make(map[string]bool)
	for _, mgr := range m.encounteredEditors(false) {
		id := mgr.ID()
		for _, editorPath := range paths[id] {
			if mgr.IsInstalled(ctx, editorPath) {
				continue
			}

			if cfg := mgr.InstallConfig(ctx); cfg.Running && !cfg.InstallWhileRunning {
				log.Printf("skipping update of %s, path %s because editor is running", id, editorPath)
				continue
			}

			if err := mgr.Install(ctx, editorPath); err != nil {
				log.Printf("error during automatic installation of plugins %s at %s: %v", id, editorPath, err)
				continue
			}

			installed[id] = true
			log.Printf("successfully installed plugin %s automatically at %s", id, editorPath)
		}
	}

	if len(installed) > 0 {
		log.Printf("%d plugins automcatically installed", len(installed))

		// access setting key under lock
		m.m.Lock()
		defer m.m.Unlock()

		// we need to merge with the existing IDs of autoinstalled plugins
		// Copilot will reset it when it displays the notification
		var pluginIDs []string
		_ = m.settings.GetObj(settings.AutoInstalledPluginIDsKey, &pluginIDs)
		for id := range installed {
			pluginIDs = append(pluginIDs, id)
		}

		_ = m.settings.SetObj(settings.AutoInstalledPluginIDsKey, pluginIDs)
	} else {
		log.Print("no plugins installed automatically")
	}
	return len(installed)
}

func (m *Manager) encounteredEditors(encountered bool) []editor.Plugin {
	m.m.Lock()
	defer m.m.Unlock()

	var editors []editor.Plugin
	for id, mgr := range m.pluginManagers {
		if m.encountered[id] == encountered {
			editors = append(editors, mgr)
		}
	}
	return editors
}
