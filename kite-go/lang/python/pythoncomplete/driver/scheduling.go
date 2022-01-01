package driver

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// ScheduleOptions configures scheduling
type ScheduleOptions struct {
	// DepthLimit limits how deep speculation can go. This depth is 1-indexed (1 is the root): <=0 defaults to a depth of 3.
	DepthLimit          int `json:"-"`
	GGNNSubtokenEnabled bool
}

func (o ScheduleOptions) depthLimit() int {
	if o.DepthLimit <= 0 {
		return 3
	}
	return o.DepthLimit
}

func (s *scheduler) rescheduleCompletion(p pythonproviders.Provider, c Completion, depth int, score float64) {
	if _, ok := zeroDepthProviders[p]; !ok {
		depth++
	}

	score *= c.Meta.Score

	// We only speculate after the selected providers
	switch p.(type) {
	case pythonproviders.Names, pythonproviders.Attributes, pythonproviders.EmptyCalls:
	case pythonproviders.GGNNModel:
		s.rescheduleGGNNCompletion(c, depth, score)
		return
	default:
		return
	}

	for _, sb := range c.speculate() {
		s.reschedule(sb, depth, score, false)
	}
}

func (s *scheduler) rescheduleGGNNCompletion(c Completion, depth int, score float64) {
	if c.Meta.GGNNMeta == nil || c.Meta.GGNNMeta.Predictor == nil {
		return
	}
	var endCursor data.Selection
	if c.Meta.GGNNMeta.SpeculationPlaceholderPresent {
		placeholders := c.Meta.Snippet.Placeholders()
		// This happens in partial call situation, we add a `[...])` at the end of the partial call
		// So we want to use this last placeholder as the speculation placeholder for the next round of EgUpdate
		endCursor = data.Cursor(c.Meta.Replace.Begin + placeholders[len(placeholders)-1].Begin)
	} else {
		endCursor = data.Cursor(c.Meta.Replace.Begin + len(c.Meta.Snippet.Text))
	}
	selTarget := c.Target.Select(endCursor)
	st := s.get(selTarget)

	workItem := st.get(pythonproviders.GGNNModel{})
	st.score = score
	st.depth = depth
	workItem.GGNNPredictor = c.Meta.GGNNMeta.Predictor
	// TODO: first is to try a simple heuristic by adding 10 to the score of the completion. Will probably have to refine it
	workItem.increasePriority(schedulerHeap{s}, score+10)
	if depth > s.opts.depthLimit() {
		return
	}
	for p, ps := range st.provisions {
		for _, cc := range ps.completions {
			for _, c := range cc {
				s.rescheduleCompletion(p, c, depth, score)
			}
		}
	}
}

func (s *scheduler) reschedule(sb data.SelectedBuffer, depth int, score float64, initialCall bool) {
	if depth < 0 {
		panic("expected non-negative depth")
	}
	// This prevents us from recursing excessively:
	// In rescheduleCompletion, either depth increases or score does not decrease (for zero-depth providers).
	// The former case is checked here via not recursing when depth becomes too great.
	// The latter case is checked below inspecting the score assigned to the speculation state.
	if depth > s.opts.depthLimit() {
		return
	}

	st := s.get(sb)

	// we base the score solely on the shortest path to the given buffer
	if 0 <= st.depth && st.depth < depth {
		return
	}
	// if there are multiple shortest paths, the greatest score is chosen
	if st.depth == depth && score <= st.score+1e-6 {
		// this is the check for the second case described above that should prevent non-termination.
		return
	}
	// TODO(naman) do something smarter for the above?

	st.score = score
	st.depth = depth

	// check feature flag
	if s.opts.GGNNSubtokenEnabled {
		ap := make(map[pythonproviders.Provider]struct{})
		for k, v := range allProviders {
			ap[k] = v
		}

		sp := make(map[pythonproviders.Provider]struct{})
		for k, v := range speculationProviders {
			sp[k] = v
		}
		delete(ap, pythonproviders.CallModel{})
		delete(ap, pythonproviders.AttributeModel{})
		ap[pythonproviders.GGNNModel{}] = struct{}{}
		delete(sp, pythonproviders.CallModel{})

		if initialCall {
			for p := range ap {
				st.get(p).increasePriority(schedulerHeap{s}, score)
			}
		} else if depth < s.opts.depthLimit() {
			for p := range sp {
				st.get(p).increasePriority(schedulerHeap{s}, score)
			}
		}
	} else {
		if depth == 0 {
			for p := range allProviders {
				st.get(p).increasePriority(schedulerHeap{s}, score)
			}
		} else if depth < s.opts.depthLimit() {
			for p := range speculationProviders {
				st.get(p).increasePriority(schedulerHeap{s}, score)
			}
		}
	}

	for p, ps := range st.provisions {
		for _, cc := range ps.completions {
			for _, c := range cc {
				s.rescheduleCompletion(p, c, depth, score)
			}
		}
	}
}
