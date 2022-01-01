package main

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	muxBundle = newMetricBundle(" tfserving-mux")
	goBundle  = newMetricBundle("golang")
	pyBundle  = newMetricBundle("python")
	jsBundle  = newMetricBundle("javascript")
	allBundle = newMetricBundle("all-langs")
)

var modelBreakdown *status.Breakdown

func init() {
	modelBreakdown = muxBundle.section.Breakdown("request by language")
	modelBreakdown.AddCategories("golang", "python", "javascript", "all-langs")
}

type metricBundle struct {
	section   *status.Section
	requests  *status.Counter
	durations *status.SampleDuration
	canceled  *status.Counter
	deadlined *status.Counter
	otherErrs *status.Counter
	inflight  *status.Counter
}

func newMetricBundle(lang string) metricBundle {
	section := status.NewSection(lang)
	return metricBundle{
		section:   section,
		requests:  section.Counter("requests"),
		durations: section.SampleDuration("Predict() latency"),
		canceled:  section.Counter("context canceled"),
		deadlined: section.Counter("deadline exceeded"),
		otherErrs: section.Counter("other errors"),
		inflight:  section.Counter("inflight requests"),
	}
}
