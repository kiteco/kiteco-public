package status

// settings contains reporting options for each metric
type settings struct {
	// Headline makes the metric prominantly visible in /debug/status
	// and the status-inspector.
	Headline bool

	// Timeseries makes the metric polled periodically and send to
	// a timeseries metric service (currently Instrumental)
	Timeseries bool
}

func newSettings() *settings {
	return &settings{}
}

func (s *settings) aggregate(other *settings) {
	if s == nil || other == nil {
		return
	}
	s.Headline = s.Headline || other.Headline
	s.Timeseries = s.Timeseries || other.Timeseries
}
