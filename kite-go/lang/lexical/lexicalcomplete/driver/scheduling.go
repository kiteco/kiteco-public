package driver

import (
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// ScheduleOptions configures scheduling
type ScheduleOptions struct {
	// DepthLimit limits how deep speculation can go. This depth is 1-indexed (1 is the root): <=0 defaults to a depth of 3.
	DepthLimit int `json:"-"`
}

func (o ScheduleOptions) depthLimit() int {
	if o.DepthLimit <= 0 {
		return 3
	}
	return o.DepthLimit
}

func (s *scheduler) rescheduleCompletion(p lexicalproviders.Provider, c completion, depth int, score float64) {
	return
}

func (s *scheduler) reschedule(sb data.SelectedBuffer, depth int, score float64) {
	s.get(sb).depth = depth
	if depth == 0 {
		for p := range allProviders {
			s.get(sb).get(p).increasePriority(schedulerHeap{s}, score)
		}
	}
	return
}
