package driver

import (
	"strings"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// filterCompletions will execute the keep method on all node of the completionTree.
// All nodes returning true will be kept and the others will be discarded
// If keepChildren is set to true, valid children of an invalid node will be copied to the children list of the parent
// of the node. If keepChildren is set to false, when a node is discarded, all its subtree is also discarded
// TODO: This may be changed in favor of inserting dummy values.
func (m *Mixer) filterCompletions(completions *CompletionTree) {
	m.dropSimilarLexical(completions)
	isNotSnippet := func(comp *CompletionTree) bool {
		phs := comp.completion.meta.Snippet.Placeholders()
		if len(phs) == 0 {
			return true
		}

		for _, p := range phs {
			if strings.Contains(comp.completion.meta.Snippet.Text[p.Begin:p.End], data.PlaceholderBeginMark) {
				return false
			}
		}
		// discard placeholders
		comp.completion.meta.Snippet = data.Snippet{
			Text: comp.completion.meta.Snippet.Text,
		}
		return true
	}

	isNotExactMatch := func(comp *CompletionTree) bool {
		return m.selectedBuffer.Buffer.Hash() != comp.completion.target.Hash()
	}

	hasPrefix := func(comp *CompletionTree) bool {
		prefix := m.selectedBuffer.Buffer.TextAt(comp.completion.meta.Replace)
		text := strings.ToLower(comp.completion.meta.Snippet.Text)
		pre := strings.ToLower(prefix)
		if strings.HasPrefix(text, pre) {
			return true
		}
		return false
	}

	// Temporary filter, multi-line completions will only be valid in the editors
	// that can show them correctly. Related: https://github.com/kiteco/kiteco/pull/10647
	validForMultiLine := func(comp *CompletionTree) bool {
		return !strings.Contains(comp.completion.meta.Snippet.Text, "\n") ||
			m.options.AllowCompletionsWithNewlines
	}

	// Temporary filter until https://github.com/kiteco/kiteco/issues/11055 is resolved
	validForDollarSignDot := func(comp *CompletionTree) bool {
		if !m.options.NoDollarSignDotCompletions {
			return true
		}

		dollarSign := strings.Contains(comp.completion.meta.Snippet.Text, "$")
		dot := strings.Contains(comp.completion.meta.Snippet.Text, ".")

		return !(dot && dollarSign)
	}

	// Temporary filter until https://github.com/kiteco/kiteco/issues/11054 is resolved
	validForDollarSign := func(comp *CompletionTree) bool {
		if !m.options.NoDollarSignCompletions {
			return true
		}
		return !strings.Contains(comp.completion.meta.Snippet.Text, "$")
	}

	predicate := func(comp *CompletionTree) bool {
		return hasPrefix(comp) && validForMultiLine(comp) &&
			(!m.options.NoSnippets || isNotSnippet(comp)) &&
			(!m.options.NoExactMatches || isNotExactMatch(comp)) &&
			validForDollarSignDot(comp) && validForDollarSign(comp)
	}

	filterCompAux(completions, predicate)
}

func (m *Mixer) dropSimilarLexical(completions *CompletionTree) {
	similarSet := NewLexicallySimilarSet(m.selectedBuffer)
	filterCompAux(completions, func(t *CompletionTree) bool {
		return !similarSet.CheckExcludeAndUpdate(t.completion.meta.Completion)
	})
}

// LexicallySimilarSet filters out similar completions such as foo(), foo(, and foo
type LexicallySimilarSet struct {
	root    data.SelectedBuffer
	exclude map[string]bool
	cut     string
}

// NewLexicallySimilarSet ...
func NewLexicallySimilarSet(root data.SelectedBuffer) LexicallySimilarSet {
	return LexicallySimilarSet{
		root:    root,
		exclude: make(map[string]bool),
		cut:     "()[]{}+- ",
	}
}

// CheckExcludeAndUpdate checks if the display should be excluded and updates the set
func (s LexicallySimilarSet) CheckExcludeAndUpdate(completion data.Completion) bool {
	display := makeDisplayText(completion, s.root, false)
	key := strings.TrimRight(display, s.cut)
	exclude := s.exclude[key]
	s.exclude[key] = true
	return exclude
}

// filterCompAux keeps only the children for which keep returns true.
// It assumes the completions have already been flattened.
func filterCompAux(completions *CompletionTree, keep func(tree *CompletionTree) bool) {
	var filtered []*CompletionTree
	for _, child := range completions.children {
		if !keep(child) {
			continue
		}
		filtered = append(filtered, child)
	}
	completions.children = filtered
}
