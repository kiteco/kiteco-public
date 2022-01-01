package diff

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("diff")

	resendRatio = section.Ratio("Requesting client resend text (buffer mismatch)")
)
