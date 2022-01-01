package status

import (
	"sync"
	"sync/atomic"
)

// Counter is a basic counter metric
type Counter struct {
	Value int64
	*settings
}

func newCounter() *Counter {
	return &Counter{
		settings: newSettings(),
	}
}

// Add increments the counter by delta
func (c *Counter) Add(delta int64) {
	atomic.AddInt64(&c.Value, delta)
}

// Set sets the counter to val
func (c *Counter) Set(val int64) {
	atomic.StoreInt64(&c.Value, val)
}

// GetValue ...
func (c *Counter) GetValue() int64 {
	return atomic.LoadInt64(&c.Value)
}

func (c *Counter) aggregate(other *Counter) {
	atomic.AddInt64(&c.Value, atomic.LoadInt64(&other.Value))
	c.settings.aggregate(other.settings)
}

// --

// Ratio is a basic ratio metric. The metric will report the percentage
// that Hit is called (vs Miss).
type Ratio struct {
	Numerator   int64
	Denominator int64
	*settings
}

func newRatio() *Ratio {
	return &Ratio{
		settings: newSettings(),
	}
}

// Hit increments the ratio and total count.
func (r *Ratio) Hit() {
	atomic.AddInt64(&r.Numerator, 1)
	atomic.AddInt64(&r.Denominator, 1)
}

// Miss increments the total count without changing the numerator.
func (r *Ratio) Miss() {
	atomic.AddInt64(&r.Denominator, 1)
}

// Set allows you to set a custom numerator and denominator
func (r *Ratio) Set(num, den int64) {
	atomic.StoreInt64(&r.Numerator, num)
	atomic.StoreInt64(&r.Denominator, den)
}

// Value returns the current ratio as a percentage.
func (r *Ratio) Value() float64 {
	numerator, denominator := atomic.LoadInt64(&r.Numerator), atomic.LoadInt64(&r.Denominator)
	if denominator == 0 {
		return 0
	}
	return 100.0 * float64(numerator) / float64(denominator)
}

func (r *Ratio) aggregate(other *Ratio) {
	atomic.AddInt64(&r.Numerator, atomic.LoadInt64(&other.Numerator))
	atomic.AddInt64(&r.Denominator, atomic.LoadInt64(&other.Denominator))
	r.settings.aggregate(other.settings)
}

// --

// Breakdown is a metric that can be used to show how often different categories of
// a particular kind appear. Similar to Ratio, except you can "Hit" any one of the
// categories set via AddCategories.
type Breakdown struct {
	rw          sync.RWMutex
	Categories  []string
	Numerators  []int64
	Denominator int64

	*settings
}

func newBreakdown() *Breakdown {
	return &Breakdown{
		settings: newSettings(),
	}
}

// AddCategories sets the categories to expect. If Hit encounters a name not provided
// to Breakdown via AddCategories, it will be ignored.
func (b *Breakdown) AddCategories(names ...string) {
	b.rw.Lock()
	defer b.rw.Unlock()
	for _, name := range names {
		for _, c := range b.Categories {
			if name == c {
				return
			}
		}
		b.Categories = append(b.Categories, name)
		b.Numerators = append(b.Numerators, 0)
	}
}

// Hit increments the counter for the provided categories, and increments the total.
func (b *Breakdown) Hit(names ...string) {
	b.rw.RLock()
	defer b.rw.RUnlock()
	var found bool
	for idx, c := range b.Categories {
		for _, name := range names {
			if name == c {
				atomic.AddInt64(&b.Numerators[idx], 1)
				found = true
			}
		}
	}
	if found {
		atomic.AddInt64(&b.Denominator, 1)
	}
}

// HitAndAdd increments the counter if the category exists. If it doesn't, it adds
// a new category, sets the counter to 1 and increments the total.
func (b *Breakdown) HitAndAdd(name string) {
	b.rw.RLock()
	var found bool
	for idx, c := range b.Categories {
		if name == c {
			atomic.AddInt64(&b.Numerators[idx], 1)
			found = true
			break
		}
	}
	if !found {
		b.rw.RUnlock()
		b.rw.Lock()
		defer b.rw.Unlock()
		b.Categories = append(b.Categories, name)
		b.Numerators = append(b.Numerators, 1)
		atomic.AddInt64(&b.Denominator, 1)
	} else {
		atomic.AddInt64(&b.Denominator, 1)
		b.rw.RUnlock()
	}
}

// Value returns a map of category to percentage value.
func (b *Breakdown) Value() map[string]float64 {
	values := make(map[string]float64)
	for idx, c := range b.Categories {
		if b.Denominator == 0 {
			values[c] = 0
			continue
		}
		values[c] = 100.0 * float64(b.Numerators[idx]) / float64(b.Denominator)
	}
	return values
}

func (b *Breakdown) aggregate(other *Breakdown) {
	b.AddCategories(other.Categories...)
	atomic.AddInt64(&b.Denominator, atomic.LoadInt64(&other.Denominator))

	vals := make(map[string]int64)
	for idx, value := range other.Numerators {
		vals[other.Categories[idx]] = value
	}

	for idx, cat := range b.Categories {
		atomic.AddInt64(&b.Numerators[idx], vals[cat])
	}

	b.settings.aggregate(other.settings)
}
