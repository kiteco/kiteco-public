package livemetrics

import (
	"sync"

	"github.com/kiteco/kiteco/kite-go/event"
)

// editorMetrics tracks which editors we get events from
type editorMetrics struct {
	editors map[string]uint64
	m       sync.Mutex
}

func newEditorMetrics() *editorMetrics {
	return &editorMetrics{
		editors: make(map[string]uint64),
	}
}

func (e *editorMetrics) zero() {
	e.m.Lock()
	defer e.m.Unlock()
	e.editors = make(map[string]uint64)
}

func (e *editorMetrics) get() map[string]uint64 {
	e.m.Lock()
	defer e.m.Unlock()
	m := make(map[string]uint64)
	for k, v := range e.editors {
		m[k+"_events"] = v
	}
	return m
}

func (e *editorMetrics) dump() map[string]uint64 {
	d := e.get()
	e.editors = make(map[string]uint64)

	return d
}

func (e *editorMetrics) TrackEvent(evt *event.Event) {
	e.m.Lock()
	defer e.m.Unlock()
	e.editors[evt.GetSource()]++
}
