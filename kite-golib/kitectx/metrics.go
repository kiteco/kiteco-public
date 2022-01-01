package kitectx

import (
	"context"
	"sync/atomic"
)

var globalMetrics *Metrics

// InitializeMetrics initializes global metrics tracking.
// It is not thread-safe, so should be called early during process initialization. It is idempotent
func InitializeMetrics() *Metrics {
	if globalMetrics == nil {
		globalMetrics = &Metrics{}
	}
	return globalMetrics
}

// Metrics tracks the number of kitectx expiries
type Metrics struct {
	deadlineExceeded uint64
	canceled         uint64
	other            uint64
}

// MetricsSnapshot is a by-value snapshot of kitectx expiry metrics
type MetricsSnapshot struct {
	DeadlineExceeded uint64
	Canceled         uint64
	Other            uint64
}

func (m *Metrics) hit(err error) {
	switch err {
	case context.DeadlineExceeded:
		atomic.AddUint64(&m.deadlineExceeded, 1)
	case context.Canceled:
		atomic.AddUint64(&m.canceled, 1)
	default:
		atomic.AddUint64(&m.other, 1)
	}
}

// Read returns the current count.
func (m *Metrics) Read() MetricsSnapshot {
	return MetricsSnapshot{
		DeadlineExceeded: atomic.LoadUint64(&m.deadlineExceeded),
		Canceled:         atomic.LoadUint64(&m.canceled),
		Other:            atomic.LoadUint64(&m.other),
	}
}

// ReadAndClear returns the current count and then clears it.
func (m *Metrics) ReadAndClear() MetricsSnapshot {
	return MetricsSnapshot{
		DeadlineExceeded: atomic.SwapUint64(&m.deadlineExceeded, 0),
		Canceled:         atomic.SwapUint64(&m.canceled, 0),
		Other:            atomic.SwapUint64(&m.other, 0),
	}
}
