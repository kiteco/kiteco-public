package pythonstatic

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func refineUnions(ctx kitectx.Context, capabilities map[*pythontype.Symbol][]Capability) {
	for s, caps := range capabilities {
		u, ok := s.Value.(pythontype.Union)
		if !ok {
			continue
		}

		disjunct := pythontype.Disjuncts(ctx, u)
		scores := make([]int, len(disjunct))
		for _, cap := range caps {
			match := make([]bool, len(disjunct))
			var matchCount int
			for i, v := range disjunct {
				if attr, _ := pythontype.Attr(ctx, v, cap.Attr); attr.Found() {
					match[i] = true
					matchCount++
				}
			}
			if matchCount == 0 {
				// None of the type match this attribute, we just ignore it as it might be a typo or partially typed attribute
				continue
			}
			if matchCount == len(disjunct) {
				// All types match this attribute, we can't refine anything from it
				continue
			}
			for i := range disjunct {
				if match[i] {
					scores[i]++
				}
			}
		}

		var maxScore int
		for _, s := range scores {
			if s > maxScore {
				maxScore = s
			}
		}

		if maxScore == 0 {
			// No refinements, we just leave the symbol's Value unchanged
			continue
		}

		var refined []pythontype.Value
		for i, v := range disjunct {
			if scores[i] == maxScore {
				refined = append(refined, v)
			}
		}

		if len(refined) == 1 {
			s.Value = refined[0]
		} else if len(refined) > 1 && len(refined) < len(disjunct) {
			s.Value = pythontype.Unite(ctx, refined...)
		}
	}
}
