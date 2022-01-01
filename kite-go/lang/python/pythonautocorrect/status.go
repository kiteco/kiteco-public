package pythonautocorrect

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("lang/python/pythonautocorrect")

	funnel = section.Breakdown("request funnel")

	segmentResults = section.Breakdown("segment results")
)
