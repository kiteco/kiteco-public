package pythontracking

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	segmentStatus             = status.NewSection("lang/python/pythontracking Segment tracking")
	calleeTrackBreakdown      = segmentStatus.Breakdown("Callee Segment track results")
	completionsTrackBreakdown = segmentStatus.Breakdown("Completions Segment track results")
)
