package livemetrics

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/stretchr/testify/assert"
)

func TestVersions(t *testing.T) {
	v := newVersionsMetrics()
	assert.Empty(t, v.get())

	v.TrackEvent(&component.EditorEvent{
		Source:        "pycharm",
		EditorVersion: "PY-183.42.1",
		PluginVersion: "1.0.207",
	})
	v.TrackEvent(&component.EditorEvent{
		Source:        "atom",
		EditorVersion: "1.42.0",
		PluginVersion: "2.0.10",
	})

	values := v.dump()
	assert.EqualValues(t, "PY-183.42.1", values["pycharm_version"])
	assert.EqualValues(t, "1.0.207", values["pycharm_plugin_version"])

	assert.EqualValues(t, "1.42.0", values["atom_version"])
	assert.EqualValues(t, "2.0.10", values["atom_plugin_version"])

	values = v.get()
	assert.Equal(t, 0, len(values))
}
