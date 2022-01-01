package status

import (
	"math"
	"math/rand"
	"sort"
	"sync/atomic"
	"time"
)

var (
	samplePercentiles = []float64{0.25, 0.50, 0.75, 0.95, 0.99}
	defaultSampleRate = float32(0.25)
)

// --

// SampleDuration samples time.Duration values and reports percentiles
type SampleDuration struct {
	*SampleInt64
}

func newSampleDuration() *SampleDuration {
	return &SampleDuration{
		SampleInt64: newSampleInt64(),
	}
}

// RecordDuration records a time.Duration
func (s *SampleDuration) RecordDuration(d time.Duration) {
	s.SampleInt64.Record(int64(d))
}

// DeferRecord records the amount of time since the provided start time.
// Intended usage: `defer s.DeferRecord(time.Now())`
func (s *SampleDuration) DeferRecord(start time.Time) {
	s.RecordDuration(time.Since(start))
}

func (s *SampleDuration) aggregate(other *SampleDuration) {
	s.SampleInt64.aggregate(other.SampleInt64)
}

// --

// SampleBytes samples int64's representing bytes and reports percentiles
type SampleBytes struct {
	*SampleInt64
}

func newSampleBytes() *SampleBytes {
	return &SampleBytes{
		SampleInt64: newSampleInt64(),
	}
}

func (r *SampleBytes) aggregate(other *SampleBytes) {
	r.SampleInt64.aggregate(other.SampleInt64)
}

// --

type int64Sort []int64

func (s int64Sort) Len() int           { return len(s) }
func (s int64Sort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s int64Sort) Less(i, j int) bool { return s[i] < s[j] }

// SampleInt64 samples raw int64 and reports percentiles
type SampleInt64 struct {
	Samples    []int64
	samplesIdx int32
	sampleRate float32
	*settings
}

func newSampleInt64() *SampleInt64 {
	samples := make([]int64, 500)
	for idx := range samples {
		samples[idx] = math.MinInt64
	}
	return &SampleInt64{
		Samples:    samples,
		sampleRate: defaultSampleRate,
		settings:   newSettings(),
	}
}

// Record a time.SampleInt64.
func (r *SampleInt64) Record(d int64) {
	if rand.Float32() < r.sampleRate {
		idx := atomic.AddInt32(&r.samplesIdx, 1)
		r.Samples[int(idx)%len(r.Samples)] = d
	}
}

// SetSampleRate changes the percentage of records that are sampled for the metric.
func (r *SampleInt64) SetSampleRate(rate float32) {
	r.sampleRate = rate
}

// Values returns an array of int64's representing time.SampleInt64 values
// at the pre-defined percentiles in SampleInt64Percentiles.
func (r *SampleInt64) Values() []int64 {
	// copy the samples into a new slice
	var samples []int64
	for _, sample := range r.Samples {
		samples = append(samples, sample)
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

func (r *SampleInt64) aggregate(other *SampleInt64) {
	r.Samples = append(r.Samples, other.Samples...)
	r.settings.aggregate(other.settings)
}
