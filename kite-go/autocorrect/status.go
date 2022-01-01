package autocorrect

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("/autocorrect")

	breakdown = section.Breakdown("Outcomes")

	segmentResults = section.Breakdown("segment results")
)
