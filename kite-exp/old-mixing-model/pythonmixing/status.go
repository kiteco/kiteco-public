package pythonmixing

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("lang/python/pythonmixing")

	newFeaturesDuration = section.SampleDuration("NewFeatures")

	modelInferDuration = section.SampleDuration("Infer")
)
