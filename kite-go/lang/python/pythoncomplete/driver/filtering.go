package driver

import (
	"fmt"
	"strings"

	lexDriver "github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/driver"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

type mixingSet map[data.BufferHash]struct{}

func getCompletionHash(c Completion) data.BufferHash {
	if c.Meta.Snippet.Placeholders() == nil {
		return c.Target.Hash()
	}
	return c.Target.Hash().AddHashInfo(fmt.Sprint(c.Meta.Snippet.Placeholders()))
}

// add returns True if the completion hasn't been added yet (it can be added to the returned completion)
// and False if the completion is already presents in the mixingSet
func (s mixingSet) add(c Completion) bool {
	hash := getCompletionHash(c)
	if _, exists := s[hash]; exists {
		return false
	}
	s[hash] = struct{}{}
	return true
}

func (m *Mixer) filterCompletions(completions *CompletionTree) {
	m.emptyCallsFilters(completions)
	m.dropSimilarLexical(completions)
	m.mainFilters(completions)
	if m.options.NoSnippets {
		m.noSnippetFilters(completions)
	}
	if m.options.NoAttributeToSubscript {
		m.noAttributeToSubscriptFilters(completions)
	}
	if m.options.NoExactMatch {
		m.noExactMatchFilters(completions)
	}

	// Temporary filter, multi-line completions will only be valid in the editors
	// that can show them correctly. Related: https://github.com/kiteco/kiteco/pull/10647
	if !m.options.AllowCompletionsWithNewlines {
		filterCompAux(completions, func(t *CompletionTree) bool {
			return !strings.Contains(t.Completion.Meta.Snippet.Text, "\n")
		})
	}
}

func (m *Mixer) emptyCallsFilters(completions *CompletionTree) {
	// For empty call completions, if we have `alpha` and `alpha()`, then we discard one.
	// For types, we keep `alpha` and discard `alpha()`.
	// For non-types, we keep `alpha()` and discard `alpha`.

	treeContains := make(map[string]bool)
	for _, c := range completions.children {
		treeContains[c.Completion.Meta.Snippet.Text] = true
	}
	nodeToRemove := make(map[string]bool)

	filterCompAux(completions, func(tree *CompletionTree) bool {
		if tree.Completion.Meta.MixingMeta.Provider.Name() != data.PythonEmptyCallsProvider {
			return true
		}
		if m.options.NoEmptyCalls {
			return false
		}
		if tree.Completion.Meta.EmptyCallMeta == nil {
			return true
		}
		index := strings.LastIndex(tree.Completion.Meta.Snippet.Text, "(")
		if index == -1 {
			return true
		}
		if tree.Completion.Meta.EmptyCallMeta.IsTypeKind {
			return !treeContains[tree.Completion.Meta.Snippet.Text[:index]]
		}
		nodeToRemove[tree.Completion.Meta.Snippet.Text[:index]] = true
		return true
	})

	filterCompAux(completions, func(t *CompletionTree) bool {
		return !nodeToRemove[t.Completion.Meta.Snippet.Text]
	})
}

func (m *Mixer) mainFilters(completions *CompletionTree) {
	var lexicalCount int
	distinct := make(mixingSet)
	filterCompAux(completions, func(tree *CompletionTree) bool {
		if m.options.MaxReturnedCompletions != 0 && len(distinct) >= m.options.MaxReturnedCompletions {
			return false
		}
		if tree.Completion.Meta.MixingMeta.Provider.Name() != data.PythonLexicalProvider {
			if tree.Completion.Meta.MixingMeta.HideCompletion {
				return false
			}
			return distinct.add(tree.Completion)
		}

		// This approach to limiting the number of lexical completions relies on
		// the lexical completions being traversed from highest score to lowest score.
		// This is the case because we call sortCompletions before filterCompletions.
		lexicalCount++
		if lexicalCount > m.options.MaxLexicalCompletions {
			return false
		}

		// If lexicalFilter were checked before lexicalCount, then sometimes
		// the top completions would be filtered and replaced with lower ranked completions.
		// These would usually also be semantically invalid, only in harder to detect ways.
		// Some examples: (one line scripts without imports)
		// "df = $" would complete to "df = df.groupby()"
		// "fig, ax$" would complete "fig, axes = fig.axes"
		if !m.lexicalFilter(tree.Completion.Meta.LexicalFiltersMeta) {
			return false
		}
		return distinct.add(tree.Completion)
	})
}

func (m *Mixer) dropSimilarLexical(completions *CompletionTree) {
	keywords := make(map[string]bool)
	for _, child := range completions.children {
		if child.Completion.Meta.MixingMeta.Provider.Name() != data.PythonKeywordsProvider {
			continue
		}
		keywords[strings.TrimRight(child.Completion.Meta.Completion.Snippet.Text, " ")] = true
	}

	similarSet := lexDriver.NewLexicallySimilarSet(m.selectedBuffer)
	filterCompAux(completions, func(t *CompletionTree) bool {
		if t.Completion.Meta.MixingMeta.Provider.Name() != data.PythonLexicalProvider {
			return true
		}
		if keywords[strings.TrimRight(t.Completion.Meta.Completion.Snippet.Text, " ")] {
			return false
		}
		return !similarSet.CheckExcludeAndUpdate(t.Completion.Meta.Completion)
	})
}

func (m *Mixer) lexicalFilter(filters *pythonproviders.LexicalFiltersMeta) bool {
	if filters.InvalidArgument && !m.options.DisabledLexicalFilters.InvalidArgument {
		return false
	}
	if filters.InvalidAssignment && !m.options.DisabledLexicalFilters.InvalidAssignment {
		return false
	}
	if filters.InvalidAttribute && !m.options.DisabledLexicalFilters.InvalidAttribute {
		return false
	}
	if filters.HasBadStmt && !m.options.DisabledLexicalFilters.HasBadStmt {
		return false
	}
	if filters.InvalidClassDef && !m.options.DisabledLexicalFilters.InvalidClassDef {
		return false
	}
	if filters.InvalidFunctionDef && !m.options.DisabledLexicalFilters.InvalidFunctionDef {
		return false
	}
	if filters.InvalidImport && !m.options.DisabledLexicalFilters.InvalidImport {
		return false
	}
	return true
}

func (m *Mixer) noSnippetFilters(completions *CompletionTree) {
	filterCompAux(completions, func(comp *CompletionTree) bool {
		phs := comp.Completion.Meta.Snippet.Placeholders()
		if len(phs) == 0 {
			return true
		}

		for _, p := range phs {
			if strings.Contains(comp.Completion.Meta.Snippet.Text[p.Begin:p.End], data.PlaceholderBeginMark) {
				return false
			}
		}
		// discard placeholders
		comp.Completion.Meta.Snippet = data.Snippet{
			Text: comp.Completion.Meta.Snippet.Text,
		}
		return true
	})
}

func (m *Mixer) noExactMatchFilters(completions *CompletionTree) {
	filterCompAux(completions, func(comp *CompletionTree) bool {
		provider := comp.Completion.Meta.MixingMeta.Provider.Name()
		if provider != data.PythonLexicalProvider && provider != data.PythonDictKeysProvider {
			return true
		}
		return m.selectedBuffer.Buffer.Hash() != comp.Completion.Target.Hash()
	})
}

func (m *Mixer) noAttributeToSubscriptFilters(completions *CompletionTree) {
	filterCompAux(completions, func(t *CompletionTree) bool {
		if t.Completion.Meta.DictMeta == nil {
			return true
		}
		return !t.Completion.Meta.DictMeta.AttributeToSubscript
	})
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
