package pythontype

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	translationSection = status.NewSection("pythontype/translate")

	translateGlobalSuccesRatio = translationSection.Ratio("Translate global success ratio")

	translateGlobalFailures = translationSection.Breakdown("Translate global failures")
)
