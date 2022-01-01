package plugins

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
)

// UpdateAllInstalled updates the plugins for all editors which have a plugin installed.
// This should be invoked asynchronously.
func (m *Manager) UpdateAllInstalled() {
	ctx := context.Background()

	// Grab installed plugins.
	toUpdate := m.pluginsToUpdate(ctx)
	// Perform Update
	duration := 30 * time.Minute
	for len(toUpdate) > 0 {
		log.Println(updateMsg(toUpdate))
		toUpdate = m.update(ctx, toUpdate)
		time.Sleep(duration)
	}
}

func (m *Manager) pluginsToUpdate(ctx context.Context) map[string][]string {
	installedEditors := make(map[string][]string)
	toUpdate := make(map[string][]string)
	for id, pluginMgr := range m.pluginManagers {
		paths, err := pluginMgr.DetectEditors(ctx)
		if err != nil {
			log.Println("Failed to detect installs for: ", id)
		}

		for _, install := range shared.MapEditors(ctx, paths, pluginMgr) {
			installedEditors[id] = append(installedEditors[id], install.Path)
			if pluginMgr.IsInstalled(ctx, install.Path) {
				toUpdate[id] = append(toUpdate[id], install.Path)
			}
		}
	}
	if len(installedEditors) > 0 {
		clienttelemetry.KiteTelemetry("Found Editors", flatten(installedEditors))
	} else {
		clienttelemetry.KiteTelemetry("Found No Editors", nil)
	}
	if len(toUpdate) > 0 {
		clienttelemetry.KiteTelemetry("Found Installed Plugins", flatten(toUpdate))
	} else {
		clienttelemetry.KiteTelemetry("Found No Installed Plugins", nil)
	}
	return toUpdate
}

func (m *Manager) update(ctx context.Context, toUpdate map[string][]string) map[string][]string {
	failed := make(map[string][]string)
	succeeded := make(map[string][]string)
	for id, paths := range toUpdate {
		manager := m.pluginManagers[id]
		for _, path := range paths {
			if !manager.IsInstalled(ctx, path) {
				// plugin was uninstalled in the last interval, so we don't need to update it anymore.
				continue
			}

			if cfg := manager.InstallConfig(ctx); cfg.Running && !cfg.UpdateWhileRunning {
				log.Printf("skipping update of %s, path %s because editor is running", id, path)
				continue
			}

			if err := manager.Update(ctx, path); err != nil {
				failed[id] = append(failed[id], path)
				m.reportInstallError("failed to update editor plugin", path, manager, err)
			} else {
				succeeded[id] = append(succeeded[id], path)
			}
		}
	}
	if len(failed) > 0 {
		clienttelemetry.KiteTelemetry("Failed Plugin Updates", flatten(failed))
	}
	if len(succeeded) > 0 {
		clienttelemetry.KiteTelemetry("Successful Plugin Updates", flatten(succeeded))
	}
	return failed
}

func updateMsg(toUpdate map[string][]string) string {
	var ids []string
	for id := range toUpdate {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	msg := strings.Join(ids, ", ")
	return "attempting to update " + msg
}

func flatten(m map[string][]string) map[string]interface{} {
	props := make(map[string]interface{})
	for k, v := range m {
		props[k] = v
	}
	return props
}
