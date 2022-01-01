package sample

// Stats represents aggregated statistics for a series of values.
type Stats struct {
	// The number of elements seen
	Count int64
	// The sum of the values seen
	Sum float64

	Average float64
}

// Add returns a new Stats object containing the combined statistics of two Stats ojbects.
func (s Stats) Add(other Stats) Stats {
	res := Stats{
		Count: s.Count + other.Count,
		Sum:   s.Sum + other.Sum,
	}
	if res.Count > 0 {
		res.Average = res.Sum / float64(res.Count)
	}

	return res
}

// StatsMap represents aggregate stats for each label
type StatsMap map[string]Stats

// SampleTag implements Addable
func (StatsMap) SampleTag() {}

// Add implements Addable
func (s StatsMap) Add(other Addable) Addable {
	for k, v := range other.(StatsMap) {
		s[k] = s[k].Add(v)
	}

	return s
}
