package api

import (
	"sync"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

// Events tracks the `cap` most recent editor events for a given user/machine
type Events struct {
	cap    int
	m      sync.Mutex
	cursor int
	evts   []*component.EditorEvent
}

// NewEvents ...
func NewEvents(cap int) *Events {
	return &Events{cap: cap}
}

// Push ...
func (e *Events) Push(evt *component.EditorEvent) {
	e.m.Lock()
	defer e.m.Unlock()
	if len(e.evts) < e.cap {
		e.evts = append(e.evts, evt)
	} else {
		e.evts[e.cursor] = evt
		e.cursor = (e.cursor + 1) % e.cap
	}
}

// Collect ...
func (e *Events) Collect() []*component.EditorEvent {
	e.m.Lock()
	defer e.m.Unlock()

	events := make([]*component.EditorEvent, len(e.evts))
	copy(events, e.evts[e.cursor:])
	copy(events[len(e.evts)-e.cursor:], e.evts[:e.cursor])

	e.evts = nil
	e.cursor = 0

	return events
}
