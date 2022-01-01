package status

import (
	"math"
	"sort"
	"sync/atomic"
	"time"
)

// CounterDistribution is a basic counter metric
type CounterDistribution struct {
	Value  int64
	values []int64
	*settings
}

func newCounterDistribution() *CounterDistribution {
	return &CounterDistribution{
		Value:    math.MinInt64,
		settings: newSettings(),
	}
}

// Add increments the counter by delta
func (c *CounterDistribution) Add(delta int64) {
	if !atomic.CompareAndSwapInt64(&c.Value, math.MinInt64, delta) {
		atomic.AddInt64(&c.Value, delta)
	}
}

// Set sets the counter to val
func (c *CounterDistribution) Set(val int64) {
	atomic.StoreInt64(&c.Value, val)
}

func (c *CounterDistribution) aggregate(other *CounterDistribution) {
	c.values = append(c.values, atomic.LoadInt64(&other.Value))
	c.settings.aggregate(other.settings)
}

// Values returns an array of int64's representing the distribution of counter values
// at the pre-defined percentiles in SampleInt64Percentiles.
func (c *CounterDistribution) Values() []int64 {
	// copy the samples into a new slice
	var samples []int64
	for _, sample := range c.values {
		samples = append(samples, sample)
	}

	// If there are no samples, use c.Value. This allows the UI to display the
	// correct value when its only reporting on its own Status.
	if len(samples) == 0 {
		samples = []int64{atomic.LoadInt64(&c.Value)}
	}

	// sort, ascending
	sort.Sort(int64Sort(samples))

	// truncate samples to the first sample (note that Samples is
	// initialized with math.MinInt64, so since its sorted, we just have
	// to find the first non-math.MinInt64)
	var hasSamples bool
	for idx := range samples {
		if samples[idx] != math.MinInt64 {
			samples = samples[idx:]
			hasSamples = true
			break
		}
	}

	// compute percentiles
	var ret []int64
	for _, p := range samplePercentiles {
		if len(samples) == 0 || !hasSamples {
			ret = append(ret, 0)
			continue
		}

		idx := int(float64(len(samples)) * p)
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		ret = append(ret, samples[idx])
	}
	return ret
}

// --

// BoolDistribution collects single boolean values and reports the percentage of true/false.
type BoolDistribution struct {
	Value      bool
	trueCount  int
	falseCount int
	*settings
}

func newBoolDistribution() *BoolDistribution {
	return &BoolDistribution{
		settings: newSettings(),
	}
}

// Set sets the value of the bool
func (b *BoolDistribution) Set(val bool) {
	b.Value = val

	// Update trueCount/falseCount so everything works when a node is only
	// reporting its own Status.
	if b.Value {
		b.trueCount = 1
		b.falseCount = 0
	} else {
		b.trueCount = 0
		b.falseCount = 1
	}
}

// TruePercentage returns the percent of times the value is true
func (b *BoolDistribution) TruePercentage() float64 {
	if b.trueCount+b.falseCount == 0 {
		return 0.0
	}
	return 100.0 * float64(b.trueCount) / float64(b.trueCount+b.falseCount)
}

// FalsePercentage returns the percent of times the value is false
func (b *BoolDistribution) FalsePercentage() float64 {
	if b.trueCount+b.falseCount == 0 {
		return 0.0
	}
	return 100.0 * float64(b.falseCount) / float64(b.trueCount+b.falseCount)
}

func (b *BoolDistribution) aggregate(other *BoolDistribution) {
	if other.Value {
		b.trueCount++
	} else {
		b.falseCount++
	}

	b.settings.aggregate(other.settings)
}

// --

// RatioDistribution collects ratios and reports the distribution of ratio values
type RatioDistribution struct {
	Value float64

	values      []float64
	numerator   int64
	denominator int64
	*settings
}

func newRatioDistribution() *RatioDistribution {
	return &RatioDistribution{
		Value:    math.MaxFloat64,
		settings: newSettings(),
	}
}

// Hit increments the ratio and total count.
func (r *RatioDistribution) Hit() {
	atomic.AddInt64(&r.numerator, 1)
	atomic.AddInt64(&r.denominator, 1)
	r.calc()
}

// Miss increments the total count without changing the numerator.
func (r *RatioDistribution) Miss() {
	atomic.AddInt64(&r.denominator, 1)
	r.calc()
}

func (r *RatioDistribution) aggregate(other *RatioDistribution) {
	r.values = append(r.values, other.Value)
	r.settings.aggregate(other.settings)
}

func (r *RatioDistribution) calc() {
	if atomic.LoadInt64(&r.denominator) == 0 {
		r.Value = 0.0
		return
	}
	r.Value = 100.0 * float64(atomic.LoadInt64(&r.numerator)) / float64(atomic.LoadInt64(&r.denominator))
}

// Values returns an array of float64's representing the distribution of ratio values
// at the pre-defined percentiles.
func (r *RatioDistribution) Values() []float64 {
	// copy the samples into a new slice
	var samples []float64
	for _, sample := range r.values {
		samples = append(samples, sample)
	}

	// If there are no samples, use r.Value. This allows the UI to display the
	// correct value when reporting its own Status (i.e when aggragate is not called)
	if len(samples) == 0 {
		samples = []float64{r.Value}
	}

	// sort, ascending
	sort.Float64s(samples)

	// truncate samples to up to the last "real" sample (note that Value is
	// initialized with math.MaxFloat64, so since its sorted, we just have
	// to find the first math.MaxFloat64 and truncate)
	for idx := range samples {
		if samples[idx] == math.MaxFloat64 {
			samples = samples[:idx]
			break
		}
	}

	// compute percentiles
	var ret []float64
	for _, p := range samplePercentiles {
		if len(samples) == 0 {
			ret = append(ret, 0)
			continue
		}

		idx := int(float64(len(samples)) * p)
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		ret = append(ret, samples[idx])
	}
	return ret
}

// --

// DurationDistribution collects a single duration value and reports the percentiles of all values collected
type DurationDistribution struct {
	*CounterDistribution
}

func newDurationDistribution() *DurationDistribution {
	return &DurationDistribution{
		CounterDistribution: newCounterDistribution(),
	}
}

// SetDuration sets the duration value
func (d *DurationDistribution) SetDuration(v time.Duration) {
	d.CounterDistribution.Set(int64(v))
}

func (d *DurationDistribution) aggregate(other *DurationDistribution) {
	d.CounterDistribution.aggregate(other.CounterDistribution)
}
