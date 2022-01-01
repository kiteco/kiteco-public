package metrics

import "sync"

// SignaturesSnapshot captures a snapshot of the statistics collected by
// this metric.
type SignaturesSnapshot struct {
	Triggered int // Number of distinct times signatures were triggered
	Shown     int // Number of signatures that were shown (for first-time triggers only)
}

// SignaturesMetric records the number of signatures suggested to the user.
type SignaturesMetric struct {
	mu    sync.Mutex
	stats SignaturesSnapshot
}

// Read returns the current count.
func (m *SignaturesMetric) Read() SignaturesSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.stats
	return out
}

// ReadAndClear returns the current count and then clears it.
func (m *SignaturesMetric) ReadAndClear() SignaturesSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.stats
	m.stats = SignaturesSnapshot{}
	return out
}

// SignatureRequested is called when a signature is triggered for the first
// time. It accepts as input whether or not a signature was successfully
// retrieved.
func (m *SignaturesMetric) SignatureRequested(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.Triggered++
	if success {
		m.stats.Shown++
	}
}
