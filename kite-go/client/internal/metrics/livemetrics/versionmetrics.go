package livemetrics

import (
	"sync"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

// versionMetrics tracks which editors we get events from
type versionMetrics struct {
	versions map[string]string
	m        sync.Mutex
}

func newVersionsMetrics() *versionMetrics {
	return &versionMetrics{
		versions: make(map[string]string),
	}
}

func (e *versionMetrics) zero() {
	e.m.Lock()
	defer e.m.Unlock()
	e.versions = make(map[string]string)
}

func (e *versionMetrics) get() map[string]string {
	e.m.Lock()
	defer e.m.Unlock()
	m := make(map[string]string)
	for k, v := range e.versions {
		m[k] = v
	}
	return m
}

func (e *versionMetrics) dump() map[string]string {
	d := e.get()
	e.zero()

	return d
}

func (e *versionMetrics) TrackEvent(evt *component.EditorEvent) {
	editor := evt.EditorVersion
	plugin := evt.PluginVersion
	if editor == "" && plugin == "" {
		return
	}

	e.m.Lock()
	defer e.m.Unlock()

	source := evt.Source
	if source == "" {
		source = "unknown"
	}

	if editor != "" {
		e.versions[source+"_version"] = editor
	}
	if plugin != "" {
		e.versions[source+"_plugin_version"] = plugin
	}
}
