package githubcorpus

import "github.com/kiteco/kiteco/kite-golib/status"

// Stats ...
var (
	Stats                 = status.NewSection("githubcorpus")
	GetContentSuccessRate = Stats.Ratio("GetContentSuccessRate")
	GetCommitSuccessRate  = Stats.Ratio("GetCommitSuccessRate")
	MergeCommitRatio      = Stats.Ratio("MergeCommitRatio")
	FullCommitSuccessRate = Stats.Ratio("FullCommitSuccessRate")
)
