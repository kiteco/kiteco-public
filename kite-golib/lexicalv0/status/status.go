package status

import "github.com/kiteco/kiteco/kite-golib/status"

// Metrics
var (
	Stats                       = status.NewSection("lexicalv0")
	NewPredictStateDuration     = Stats.SampleDuration("NewPredictState")
	NewPartialRunModelDuration  = Stats.SampleDuration("NewPartialRunModel")
	EmbedInitialContextDuration = Stats.SampleDuration("EmbedInitialContext")
	PartialRunQueryDuration     = Stats.SampleDuration("PartialRunQuery")
	PartialRunOverlapDist       = Stats.SampleInt64("PartialRunOverlap")
	ClosePartialRunDuration     = Stats.SampleDuration("ClosePartialRun")
	SearchDuration              = Stats.SampleDuration("Search")
	PrettifyDuration            = Stats.SampleDuration("Prettify")
	FormatCompletionDuration    = Stats.SampleDuration("FormatCompletion")
	FormatBytes                 = Stats.SampleByte("FormatBytes")
	PartialRunReuseRate         = Stats.Ratio("PartialRun Reuse")
	PredictionReuseRate         = Stats.Ratio("PredictionReuse")
)
