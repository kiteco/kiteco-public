package driver

import (
	"math"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

func higherPriority(node1, node2 *CompletionTree, selBuffer data.SelectedBuffer) bool {
	// exact prefix case match at top
	if selBuffer.Selection.Len() == 0 {
		exactMatch1 := node1.completion.meta.Completion.ExactCaseMatchPrecedingIdent(selBuffer)
		exactMatch2 := node2.completion.meta.Completion.ExactCaseMatchPrecedingIdent(selBuffer)
		if exactMatch1 && !exactMatch2 {
			return true
		} else if !exactMatch1 && exactMatch2 {
			return false
		}
	}

	if math.Abs(node1.completion.meta.Score-node2.completion.meta.Score) >= 1e-6 {
		return node1.completion.meta.Score > node2.completion.meta.Score
	}

	return node1.completion.meta.Snippet.Text < node2.completion.meta.Snippet.Text
}

func (m *Mixer) sortCompletions(completions *CompletionTree) {
	sort.Slice(completions.children, func(i, j int) bool {
		return higherPriority(completions.children[i], completions.children[j], m.selectedBuffer)
	})
}
