package driver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// MixOptions allows passing maximum number of completion mixing can returned
type MixOptions struct {
	data.APIOptions

	MaxReturnedCompletions int
	MaxLexicalCompletions  int

	// Intended for a/b testing conversion impact of Python lexical.
	LexicalCompletionsDisabled bool

	// If APIOptions.NoSnippets, NoEmptyCalls is forced on.
	NoEmptyCalls bool
	NoElision    bool
	NoExactMatch bool

	// NoAttributeToSubscript is provided by VS Code since we don't have control over the filtering
	NoAttributeToSubscript bool

	GGNNSubtokenEnabled bool

	// PrependCompletionContext adds part of the buffer to the beginning of a completion for ranking completions higher
	PrependCompletionContext bool

	// AllowCompletionsWithNewlines allows multi-line completions
	AllowCompletionsWithNewlines bool

	// Intended for offline analysis in local-pipelines/mixing
	DisabledLexicalFilters pythonproviders.LexicalFiltersMeta
	UseExperimentalScoring bool

	data.RenderOptions
}

// Mixer ...
type Mixer struct {
	options        MixOptions
	s              *scheduler
	root           *CompletionTree
	selectedBuffer data.SelectedBuffer
}

// NewMixer instantiate a new mixing instance
func NewMixer(opts MixOptions, s *scheduler, selectedBuffer data.SelectedBuffer) *Mixer {
	return &Mixer{
		options:        opts,
		s:              s,
		selectedBuffer: selectedBuffer,
	}
}

func (s *scheduler) Mix(ctx kitectx.Context, opts MixOptions, pyctx *python.Context, g pythonproviders.Global, root data.SelectedBuffer) []data.NRCompletion {
	ctx.CheckAbort()
	if opts.NoSnippets {
		opts.NoEmptyCalls = true
	}
	mixer := NewMixer(opts, s, root)
	return mixer.Mix(ctx, pyctx, g, root)
}

// Mix mixes completion from different providers together
func (m *Mixer) Mix(ctx kitectx.Context, pyctx *python.Context, g pythonproviders.Global, root data.SelectedBuffer) []data.NRCompletion {
	completions := m.collectCompletions(root)
	m.flattenCompletions(completions)
	m.validateCompletions(completions, root)
	m.sortCompletions(completions)
	m.filterCompletions(completions)
	rendered := m.renderCompletions(ctx, pyctx, g, completions)
	return data.DedupeTrailingSpace(rendered)
}

func (m *Mixer) flattenCompletions(completions *CompletionTree) {
	var flat []*CompletionTree
	level := completions.children
	for len(level) > 0 {
		var nextLevel []*CompletionTree
		for _, completion := range level {
			nextLevel = append(nextLevel, completion.children...)
			completion.children = nil
			flat = append(flat, completion)
		}
		level = nextLevel
	}
	completions.children = flat
}

func (m *Mixer) validateCompletions(completions *CompletionTree, buffer data.SelectedBuffer) {
	var validChildren []*CompletionTree
	for _, child := range completions.children {
		validated, valid := child.Completion.Meta.Completion.Validate(buffer)
		if !valid {
			continue
		}
		child.Completion.Meta.Completion = validated
		maybeRemovePlaceholders(child, buffer)
		validChildren = append(validChildren, child)
	}
	completions.children = validChildren
}

func maybeRemovePlaceholders(child *CompletionTree, buffer data.SelectedBuffer) {
	sel := buffer.Selection.Offset(-child.Completion.Meta.Replace.Begin)
	if sel.Len() == 0 {
		return
	}
	if len(child.Completion.Meta.Snippet.Text) < sel.Len() {
		return
	}
	if child.Completion.Meta.Snippet.Text == buffer.TextAt(buffer.Selection) {
		return
	}
	child.Completion.Meta.Snippet = child.Completion.Meta.Snippet.RemovePlaceholders(sel)
}

func (m *Mixer) sortCompletions(completions *CompletionTree) {
	for _, completion := range completions.children {
		completion.Completion.Meta.NormalizedScore = m.normalizeScore(completion.Completion.Meta)
	}
	sortKey := func(c *CompletionTree) float64 {
		return c.Completion.Meta.NormalizedScore
	}
	sort.Slice(completions.children, func(i, j int) bool {
		return sortKey(completions.children[i]) > sortKey(completions.children[j])
	})
}

func (m *Mixer) normalizeScore(mc pythonproviders.MetaCompletion) float64 {
	pName := mc.MixingMeta.Provider.Name()
	useExperimental := m.options.UseExperimentalScoring && pName == data.PythonLexicalProvider
	score := mc.Score
	if useExperimental {
		score = mc.ExperimentalScore
	}
	return m.s.global.Normalizer.Normalize(int(pName), score, useExperimental)
}

func (m *Mixer) printCompletions(completionTree *CompletionTree, title string) {
	fmt.Println(title)
	count := m.printCompletionsAux(completionTree, 0)
	fmt.Println(count, " printed completions\n ")
}

func (m *Mixer) printCompletionsAux(completionTree *CompletionTree, indentLevel int) int {
	printedComp := 1
	if !m.isRoot(completionTree) {
		compSnippet := completionTree.Completion.Meta.Snippet.Delimit("⦉", "⦊", "⦉⦊").Text
		source := string(completionTree.Completion.Meta.Source)
		fmt.Printf(
			"%s##%s##  (%s - %T) %v\n",
			strings.Repeat("  ", indentLevel),
			compSnippet,
			source,
			completionTree.Completion.Meta.MixingMeta.Provider,
			completionTree.Completion.Meta.Score,
		)
	} else {
		fmt.Println("Root")
	}
	for _, child := range completionTree.children {
		printedComp += m.printCompletionsAux(child, indentLevel+1)
	}
	return printedComp
}
