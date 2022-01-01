package kitestatus

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	metrics sync.Map
)

// Metric is an interface that when implemented allows a
// metric to be registered with this package
type Metric interface {
	// Value returns a map of metric name -> value/object to be added to kite_status
	Value() map[string]interface{}

	// Reset will reset the metrics, determined by the cadence of kite_status metrics updates
	Reset()
}

// GetMetric returns a handle to the Metric object registered with kitestatus. This method
// ensures you are always using the correct object (that kitestatus is aware of) when manipulating a metric.
func GetMetric(name string, m Metric) Metric {
	obj, loaded := metrics.LoadOrStore(name, m)
	if loaded {
		panic(fmt.Sprintf("called kitestatus.GetMetric on '%s' multiple times", name))
	}
	return obj.(Metric)
}

// Counter ...
type Counter struct {
	name  string
	value int64
}

// Incr will increment the counter
func (c *Counter) Incr(delta int64) {
	atomic.AddInt64(&c.value, delta)
}

// Set will set the counter value
func (c *Counter) Set(value int64) {
	atomic.StoreInt64(&c.value, value)
}

// Value implements Metric
func (c *Counter) Value() map[string]interface{} {
	return map[string]interface{}{
		c.name: atomic.LoadInt64(&c.value),
	}
}

// Reset implements Metric
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// GetCounter ...
func GetCounter(name string) *Counter {
	obj, loaded := metrics.LoadOrStore(name, &Counter{
		name: name,
	})
	if loaded {
		panic(fmt.Sprintf("called kitestatus.GetCounter on '%s' multiple times", name))
	}
	return obj.(*Counter)
}

// --

// Boolean ...
type Boolean struct {
	m            sync.Mutex
	name         string
	value        bool
	defaultValue bool
}

// GetBooleanDefault returns a boolean with provided default value
func GetBooleanDefault(name string, defaultValue bool) *Boolean {
	obj, loaded := metrics.LoadOrStore(name, &Boolean{
		name:         name,
		value:        defaultValue,
		defaultValue: defaultValue,
	})
	if loaded {
		panic(fmt.Sprintf("called kitestatus.GetBooleanDefault on '%s' multiple times", name))
	}
	return obj.(*Boolean)
}

// SetBool can change the value of the Boolean
func (b *Boolean) SetBool(val bool) {
	b.m.Lock()
	defer b.m.Unlock()
	b.value = val
}

// Value implements Metric
func (b *Boolean) Value() map[string]interface{} {
	b.m.Lock()
	defer b.m.Unlock()
	return map[string]interface{}{
		b.name: b.value,
	}
}

// Reset implements Metric
func (b *Boolean) Reset() {
	b.m.Lock()
	defer b.m.Unlock()
	b.value = b.defaultValue
}
