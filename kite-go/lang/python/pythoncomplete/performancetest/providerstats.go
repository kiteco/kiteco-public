package performancetest

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

// ProviderStatsList defines custom JSON marshalling for a slice of ProviderStats
type ProviderStatsList []*ProviderStats

// MarshalJSON implements json.Marshaler
func (l ProviderStatsList) MarshalJSON() ([]byte, error) {
	d := struct {
		Type         string           `json:"type"`
		Timestamp    time.Time        `json:"timestamp"`
		ProviderData []*ProviderStats `json:"provider_data"`
	}{
		Type:         "completion-provider",
		Timestamp:    time.Now(),
		ProviderData: l,
	}
	return json.Marshal(d)
}

// ProviderStats keeps information about one completion provider
type ProviderStats struct {
	Name   string
	Source string

	completions         []string
	completionDurations []time.Duration
	start               time.Time
	end                 time.Time
	prev                time.Time
}

// MarshalJSON implements json.Marshaler
func (p *ProviderStats) MarshalJSON() ([]byte, error) {
	durations := []int64{}
	for _, d := range p.Durations() {
		durations = append(durations, d.Nanoseconds())
	}

	// JSON supported by our AWS lambda function
	data := struct {
		Name              string    `json:"name"`
		Source            string    `json:"file"`
		Timestamp         time.Time `json:"timestamp"`
		DurationTotalNano int64     `json:"duration_total_ns"`
		DurationFirstNano int64     `json:"duration_first_ns"`
		DurationsNano     []int64   `json:"durations_ns"`
	}{
		Name:              p.Name,
		Source:            filepath.Base(p.Source),
		Timestamp:         p.end,
		DurationTotalNano: p.TotalDuration().Nanoseconds(),
		DurationFirstNano: p.First().Nanoseconds(),
		DurationsNano:     durations,
	}

	return json.Marshal(data)
}

// Start starts the data recording, Stop requires a previous call of Start
func (p *ProviderStats) Start() {
	p.start = time.Now()
	p.prev = p.start
}

// Stop sets the time the provider finished to compute completions
func (p *ProviderStats) Stop() {
	p.end = time.Now()
}

// Add adds a new entry, the elapsed time for this completion is based on the previously added item
// or on the start time if there's not item yet
func (p *ProviderStats) Add(completion string) {
	now := time.Now()
	sub := now.Sub(p.prev)
	p.completionDurations = append(p.completionDurations, sub)
	p.prev = now

	p.completions = append(p.completions, completion)
}

// Empty returns true if no data was recorded
func (p *ProviderStats) Empty() bool {
	return len(p.completionDurations) == 0
}

// Durations returns a slice of duration value, one for each computed completion
func (p *ProviderStats) Durations() []time.Duration {
	return p.completionDurations
}

// First return the first duration or 0
func (p *ProviderStats) First() time.Duration {
	if p.Empty() {
		return time.Duration(0)
	}
	return p.completionDurations[0]
}

// TotalDuration returns how stop time - start time
func (p *ProviderStats) TotalDuration() time.Duration {
	if p.Empty() {
		return time.Duration(0)
	}
	return p.end.Sub(p.start)
}

// TotalItems returns the number of recorded completions
func (p *ProviderStats) TotalItems() int {
	return len(p.completionDurations)
}

// Min returns the fastest recorded completion
func (p *ProviderStats) Min() time.Duration {
	if p.Empty() {
		return time.Duration(0)
	}

	min := p.completionDurations[0]
	for _, e := range p.completionDurations {
		if e < min {
			min = e
		}
	}
	return min
}

// Max returns the slowest recorded completion
func (p *ProviderStats) Max() time.Duration {
	if p.Empty() {
		return time.Duration(0)
	}

	max := p.completionDurations[0]
	for _, e := range p.completionDurations {
		if e > max {
			max = e
		}
	}
	return max
}

// Completions returns the recorded completion values
func (p *ProviderStats) Completions() []string {
	return p.completions
}

// String returns a string representation
func (p *ProviderStats) String() string {
	return fmt.Sprintf("min: %.3fms\tmax: %.3fms\tfirst: %.3fms\ttotal: %.3fms / %d",
		float64(p.Min().Nanoseconds())/1e6,
		float64(p.Max().Nanoseconds())/1e6,
		float64(p.First().Nanoseconds())/1e6,
		float64(p.TotalDuration().Nanoseconds())/1e6,
		len(p.completionDurations))
}
