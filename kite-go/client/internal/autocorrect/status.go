package autocorrect

import (
	"github.com/kiteco/kiteco/kite-golib/status"
)

var (
	section = status.NewSection("/kite/autocorrect")

	responseCodes = section.Breakdown("Autocorrect response codes")
)
