package api

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	// Stats ...
	Stats = status.NewSection("api")
	// CompletionDuration ...
	CompletionDuration = Stats.SampleDuration("completion duration")
)
