package pythonlocal

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("lang/python/pythonlocal (diskmap)")

	definitionDuration       = section.SampleDuration("Definitions")
	definitionCachedDuration = section.SampleDuration("Definitions (with cache)")
	definitionRatio          = section.Ratio("Definitions cache hit rate")
	definitionErrRatio       = section.Ratio("Definitions error rate")

	methodsDuration       = section.SampleDuration("Methods")
	methodsCachedDuration = section.SampleDuration("Methods (with cache)")
	methodsRatio          = section.Ratio("Methods cache hit rate")
	methodsErrRatio       = section.Ratio("Methods error rate")
)
