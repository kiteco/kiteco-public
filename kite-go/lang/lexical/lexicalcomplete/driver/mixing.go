package driver

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// MixOptions allows passing maximum number of completion mixing can returned
type MixOptions struct {
	data.APIOptions

	MaxReturnedCompletions int
	NestCompletions        bool

	// NoExactMatches excludes completions that result in the same buffer state before and after insertion.
	NoExactMatches bool

	// PrependCompletionContext adds text preceding a completion for ranking completions higher
	PrependCompletionContext bool

	// AllowCompletionsWithNewlines allows multi-line completions
	AllowCompletionsWithNewlines bool

	// NoDollarSignDotCompletions is a hack to disable completions that contain $ and . for editors
	// that insert the wrong thing, see: https://github.com/kiteco/kiteco/issues/11055
	NoDollarSignDotCompletions bool

	// NoDollarSignCompletions is a hack to disable completions that contain $ for editors
	// that insert the wrong thing, see: https://github.com/kiteco/kiteco/issues/11054
	NoDollarSignCompletions bool

	data.RenderOptions
}

// MixCompletion encapsulates all the information for the given completion at the time of mixing
type MixCompletion struct {
	completion
}

// Meta returns the MetaCompletion
func (mc MixCompletion) Meta() lexicalproviders.MetaCompletion {
	return mc.meta
}

type mixingSet map[data.BufferHash]struct{}

func getCompletionHash(c completion) data.BufferHash {
	if len(c.meta.Snippet.Placeholders()) == 0 {
		return c.target.Hash()
	}
	return c.target.Hash().AddHashInfo(fmt.Sprint(c.meta.Snippet.Placeholders()))
}

// add returns True if the completion hasn't been added yet (it can be added to the returned completion)
// and False if the completion is already presents in the mixingSet
func (s mixingSet) add(c completion) bool {
	hash := getCompletionHash(c)
	if _, exists := s[hash]; exists {
		return false
	}
	s[hash] = struct{}{}
	return true
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

func (s *scheduler) Mix(ctx kitectx.Context, opts MixOptions, g lexicalproviders.Global, root data.SelectedBuffer) []data.NRCompletion {
	ctx.CheckAbort()
	mixer := NewMixer(opts, s, root)
	return mixer.Mix(ctx, g)
}

// Mix mixes completion from different providers together
func (m *Mixer) Mix(ctx kitectx.Context, g lexicalproviders.Global) []data.NRCompletion {
	completions := m.collectCompletions()
	if m.options.NestCompletions {
		m.nestCompletions(completions)
	}
	m.sortCompletions(completions)
	m.filterCompletions(completions)
	if m.options.MaxReturnedCompletions > 0 {
		m.pruneTree(completions)
	}
	rendered := m.renderCompletions(ctx, g, completions)
	rendered = data.DedupeTrailingSpace(rendered)

	// do final sort to make sure prefixes always come first,
	// this operates on the rendered completion so that we can sort based
	// on the display text
	for i, ci := range rendered {
		for j := i + 1; j < len(rendered); j++ {
			cj := rendered[j]
			if strings.HasPrefix(ci.Display, cj.Display) {
				rendered[i], rendered[j] = cj, ci
				break
			}
		}
	}

	return rendered
}
