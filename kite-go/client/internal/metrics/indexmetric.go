package metrics

import "sync"

// IndexSnapshot captures a snapshot of the statistics collected by this
// metric.
type IndexSnapshot struct {
	EventsWithIndex    int
	EventsWithoutIndex int
}

// IndexMetric records the number of events handled with or without an index.
type IndexMetric struct {
	mu    sync.Mutex
	stats IndexSnapshot
}

// Read returns the current count.
func (m *IndexMetric) Read() IndexSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.stats
	return out
}

// ReadAndClear returns the current count and clears it.
func (m *IndexMetric) ReadAndClear() IndexSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.stats
	m.stats = IndexSnapshot{}
	return out
}

// EventHandled is called when a response is returned due to an event being
// handled. It accepts as input whether or not an index was loaded at the
// time the event was handled.
func (m *IndexMetric) EventHandled(index bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index {
		m.stats.EventsWithIndex++
	} else {
		m.stats.EventsWithoutIndex++
	}
}
