package metrics

import "sync/atomic"

// WatcherMetric records the number of active watches
type WatcherMetric struct {
	watchCount int64
}

// Read returns the current count. It reset the count to 0 if the clear parameter is set to true.
func (m *WatcherMetric) Read(clear bool) int64 {
	if clear {
		return atomic.SwapInt64(&m.watchCount, 0)
	}
	return atomic.LoadInt64(&m.watchCount)
}

// Set is called to update the number of active watches
func (m *WatcherMetric) Set(count int64) {
	atomic.StoreInt64(&m.watchCount, count)
}
