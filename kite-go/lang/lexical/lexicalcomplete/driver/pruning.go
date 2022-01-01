package driver

import (
	"math"
)

// round is equivalent to math.Round, but less efficient.
// It is here only for compatibility with Go 1.9
func round(x float64) float64 {
	t := math.Trunc(x)
	if math.Abs(x-t) >= 0.5 {
		return t + math.Copysign(1, x)
	}
	return t
}

func (m *Mixer) pruneTree(completions *CompletionTree) {
	count := countDescendents(completions)
	if count < m.options.MaxReturnedCompletions {
		return
	}
	pruneTreeAux(completions, m.options.MaxReturnedCompletions+1)
}

func pruneTreeAux(completions *CompletionTree, maxCount int) {
	maxCount--
	if maxCount == 1 {
		completions.children = nil
		return
	}
	if len(completions.children) > maxCount {
		completions.children = completions.children[:maxCount]
	}
	var scoreSum float64
	for _, c := range completions.children {
		scoreSum += c.completion.meta.Score
	}
	for i, c := range completions.children {
		// We display a number of child proportional to the score of this completion divided by the total score
		// We add 0.01 to the sum to be sure to not divide by 0

		// WARNING: That make a big dependence between the pruning and the score, and the score can depend on the provider
		// So if there's a duplicate for a completion, it's score might depend on which completion we keep during the dedup
		// So this part might be not deterministic if the dedup process is not (we tried to make it as much deterministic as possible
		// but some edge case can still exists)
		count := int(round(float64(maxCount) * c.completion.meta.Score / (scoreSum + 0.01)))
		if count == 0 && maxCount > 0 {
			// We want at least to display c so we set count to be at least 1 to make sure it will be kept
			count = 1
		}
		if count == 1 {
			c.children = nil
		} else {
			grandChild := countDescendents(c)
			if count > 1+grandChild {
				// c doesn't contain enough child to make it to the count
				count = 1 + grandChild
			}
			pruneTreeAux(c, count)
			maxCount -= count
			scoreSum -= c.completion.meta.Score
		}
		if maxCount <= 0 {
			completions.children = completions.children[:i+1]
			break
		}
	}
}

func countDescendents(tree *CompletionTree) int {
	var result int
	for _, c := range tree.children {
		result += 1 + countDescendents(c)
	}
	return result
}
