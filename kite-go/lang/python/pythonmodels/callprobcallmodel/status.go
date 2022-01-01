package callprobcallmodel

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("lang/python/pythonmodels/callprob")

	newFeaturesDuration = section.SampleDuration("NewFeatures")

	modelInferDuration = section.SampleDuration("Infer")
)
