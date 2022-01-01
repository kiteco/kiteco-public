package status

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Section represents a grouping of Counters, Ratios, Durations and Breakdowns.
type Section struct {
	Name string

	Counters   map[string]*Counter
	Ratios     map[string]*Ratio
	Breakdowns map[string]*Breakdown

	SampleInt64s    map[string]*SampleInt64
	SampleDurations map[string]*SampleDuration
	SampleBytes     map[string]*SampleBytes

	CounterDistributions  map[string]*CounterDistribution
	RatioDistributions    map[string]*RatioDistribution
	BoolDistributions     map[string]*BoolDistribution
	DurationDistributions map[string]*DurationDistribution

	m sync.Mutex
}

// NewSection builds a new Section with the provided name.
func NewSection(name string) *Section {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var section *Section
	if section, exists = s.Sections[name]; !exists {
		section = newEmptySection(name)
		s.Sections[name] = section
	}
	return section
}

func newEmptySection(name string) *Section {
	return &Section{
		Name: name,

		Counters:   make(map[string]*Counter),
		Ratios:     make(map[string]*Ratio),
		Breakdowns: make(map[string]*Breakdown),

		SampleInt64s:    make(map[string]*SampleInt64),
		SampleDurations: make(map[string]*SampleDuration),
		SampleBytes:     make(map[string]*SampleBytes),

		CounterDistributions:  make(map[string]*CounterDistribution),
		RatioDistributions:    make(map[string]*RatioDistribution),
		BoolDistributions:     make(map[string]*BoolDistribution),
		DurationDistributions: make(map[string]*DurationDistribution),
	}
}

// MarshalJSON is implemented to avoid concurrent map access. It holds the section lock,
// and avoids recursive calls into MarshalJSON.
func (s *Section) MarshalJSON() ([]byte, error) {
	s.m.Lock()
	defer s.m.Unlock()

	// to avoid recursive call into MarshalJSON (and the subsequent deadlock),
	// create a temporary type to mask the MarshalJSON method
	type tmp Section
	return json.Marshal((*tmp)(s))
}

// Counter creates a new counter with the provided name.
func (s *Section) Counter(name string) *Counter {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var counter *Counter
	if counter, exists = s.Counters[name]; !exists {
		counter = newCounter()
		s.Counters[name] = counter
	}
	return counter
}

// Ratio creates a new ratio metric with the provided name.
func (s *Section) Ratio(name string) *Ratio {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var ratio *Ratio
	if ratio, exists = s.Ratios[name]; !exists {
		ratio = newRatio()
		s.Ratios[name] = ratio
	}
	return ratio
}

// Breakdown returns a new Breakdown metric with the provided name.
func (s *Section) Breakdown(name string) *Breakdown {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var breakdown *Breakdown
	if breakdown, exists = s.Breakdowns[name]; !exists {
		breakdown = newBreakdown()
		s.Breakdowns[name] = breakdown
	}

	return breakdown
}

// SampleInt64 creates a new SampleInt64 metric with the provided name.
func (s *Section) SampleInt64(name string) *SampleInt64 {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var ad *SampleInt64
	if ad, exists = s.SampleInt64s[name]; !exists {
		ad = newSampleInt64()
		s.SampleInt64s[name] = ad
	}

	return ad
}

// SampleByte creates a new SampleByte metric with the provided name.
func (s *Section) SampleByte(name string) *SampleBytes {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var ad *SampleBytes
	if ad, exists = s.SampleBytes[name]; !exists {
		ad = newSampleBytes()
		s.SampleBytes[name] = ad
	}

	return ad
}

// SampleDuration creates a new SampleDuration metric with the provided name.
func (s *Section) SampleDuration(name string) *SampleDuration {
	s.m.Lock()
	defer s.m.Unlock()
	var exists bool
	var ad *SampleDuration
	if ad, exists = s.SampleDurations[name]; !exists {
		ad = newSampleDuration()
		s.SampleDurations[name] = ad
	}

	return ad
}

// CounterDistribution creates a new CounterDistribution metric with the provided name.
func (s *Section) CounterDistribution(name string) *CounterDistribution {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var ad *CounterDistribution
	if ad, exists = s.CounterDistributions[name]; !exists {
		ad = newCounterDistribution()
		s.CounterDistributions[name] = ad
	}

	return ad
}

// RatioDistribution creates a new RatioDistribution metric with the provided name.
func (s *Section) RatioDistribution(name string) *RatioDistribution {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var ad *RatioDistribution
	if ad, exists = s.RatioDistributions[name]; !exists {
		ad = newRatioDistribution()
		s.RatioDistributions[name] = ad
	}

	return ad
}

// BoolDistribution creates a new BoolDistribution metric with the provided name.
func (s *Section) BoolDistribution(name string) *BoolDistribution {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var ad *BoolDistribution
	if ad, exists = s.BoolDistributions[name]; !exists {
		ad = newBoolDistribution()
		s.BoolDistributions[name] = ad
	}

	return ad
}

// DurationDistribution creates a new DurationDistribution metric with the provided name.
func (s *Section) DurationDistribution(name string) *DurationDistribution {
	s.m.Lock()
	defer s.m.Unlock()

	var exists bool
	var ad *DurationDistribution
	if ad, exists = s.DurationDistributions[name]; !exists {
		ad = newDurationDistribution()
		s.DurationDistributions[name] = ad
	}

	return ad
}

// Percentiles returns the percentiles being used for Sample* metrics.
func (s *Section) Percentiles() []float64 {
	return samplePercentiles
}

func (s *Section) aggregate(other *Section) error {
	if s.Name != other.Name {
		return fmt.Errorf("aggregating different sections (%s and %s)", s.Name, other.Name)
	}

	for key, counter := range other.Counters {
		mine, exists := s.Counters[key]
		if !exists {
			mine = newCounter()
			s.Counters[key] = mine
		}
		mine.aggregate(counter)
	}

	for key, ratio := range other.Ratios {
		mine, exists := s.Ratios[key]
		if !exists {
			mine = newRatio()
			s.Ratios[key] = mine
		}
		mine.aggregate(ratio)
	}

	for key, breakdown := range other.Breakdowns {
		mine, exists := s.Breakdowns[key]
		if !exists {
			mine = newBreakdown()
			s.Breakdowns[key] = mine
		}
		mine.aggregate(breakdown)
	}

	for key, sample := range other.SampleInt64s {
		mine, exists := s.SampleInt64s[key]
		if !exists {
			mine = newSampleInt64()
			s.SampleInt64s[key] = mine
		}
		mine.aggregate(sample)
	}

	for key, sample := range other.SampleBytes {
		mine, exists := s.SampleBytes[key]
		if !exists {
			mine = newSampleBytes()
			s.SampleBytes[key] = mine
		}
		mine.aggregate(sample)
	}

	for key, sample := range other.SampleDurations {
		mine, exists := s.SampleDurations[key]
		if !exists {
			mine = newSampleDuration()
			s.SampleDurations[key] = mine
		}
		mine.aggregate(sample)
	}

	for key, counter := range other.CounterDistributions {
		mine, exists := s.CounterDistributions[key]
		if !exists {
			mine = newCounterDistribution()
			s.CounterDistributions[key] = mine
		}
		mine.aggregate(counter)
	}

	for key, counter := range other.RatioDistributions {
		mine, exists := s.RatioDistributions[key]
		if !exists {
			mine = newRatioDistribution()
			s.RatioDistributions[key] = mine
		}
		mine.aggregate(counter)
	}

	for key, counter := range other.BoolDistributions {
		mine, exists := s.BoolDistributions[key]
		if !exists {
			mine = newBoolDistribution()
			s.BoolDistributions[key] = mine
		}
		mine.aggregate(counter)
	}

	for key, counter := range other.DurationDistributions {
		mine, exists := s.DurationDistributions[key]
		if !exists {
			mine = newDurationDistribution()
			s.DurationDistributions[key] = mine
		}
		mine.aggregate(counter)
	}

	return nil
}
