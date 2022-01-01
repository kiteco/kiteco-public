package python

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("client/internal/kitelocal/internal/signatures/python")

	parseDuration = section.SampleDuration("Parse")

	parseErrs = section.Breakdown("Error types")

	parseErrRatio = section.Ratio("Parse errors")

	errTypes = section.Breakdown("Error types")

	failFindArgsStartRatio = section.Ratio("Unable to find args start")

	fetchBreakdown = section.Breakdown("Check fetch outcomes")

	invalidCursorCount = section.Counter("Invalid cursor position")

	invalidParseDelimiters = section.Counter("Invalid parse delimiters")

	invalidParseDelimitersBreakdown = section.Breakdown("Invalid parse delimiters")
)

var (
	signatureStatus = status.NewSection("client/internal/kitelocal/internal/signatures/python Signature Tracking")

	segmentResults = signatureStatus.Breakdown("Segment track results")
)

func init() {
	parseDuration.SetSampleRate(1.)
}
