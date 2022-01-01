package driver

import (
	"strings"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
)

func (m *Mixer) makeCompletion(ctx kitectx.Context, g lexicalproviders.Global, c completion) data.RCompletion {
	ctx.CheckAbort()
	rcompl := data.RCompletion{
		Completion: renderCompletion(c.meta.Completion),
		Display:    makeDisplayText(c.meta.Completion, m.selectedBuffer, m.options.NoUnicode),
		Provider:   c.meta.Provider,
		Smart:      data.CompletionIsSmart(c.meta.Completion, c.meta.FromSmartProvider, m.options.RenderOptions),
		Debug:      c.meta.DebugStr,
		Metrics:    c.meta.Metrics,
		IsServer:   c.meta.IsServer,
	}

	return rcompl
}

func renderCompletion(c data.Completion) data.Completion {
	insertion := c.Snippet.ForFormat()

	// Trim trailing spaces
	insertion = strings.TrimRightFunc(insertion, unicode.IsSpace)

	// If a Blank Placeholder is adjacent to a TabStopPlaceholder,
	// collapse them and only keep the BlankPlaceholder
	adjacent := data.Hole(render.BlankPlaceholder) + data.Hole("")
	for strings.Index(insertion, adjacent) != -1 {
		insertion = strings.ReplaceAll(insertion, adjacent, data.Hole(render.BlankPlaceholder))
	}

	// After collapsing, change the insertion for BlankPlaceholder to empty tab stop
	insertion = strings.ReplaceAll(
		insertion,
		data.Hole(render.BlankPlaceholder),
		data.Hole(""),
	)
	return data.Completion{
		Snippet: data.BuildSnippet(insertion),
		Replace: c.Replace,
	}
}

func makeDisplayText(c data.Completion, root data.SelectedBuffer, noUnicode bool) string {
	opts := data.DisplayOptions{
		NoUnicode:         noUnicode,
		TrimBeforeEmptyPH: true,
	}
	return c.DisplayText(root, opts)
}

func (m *Mixer) renderCompletions(ctx kitectx.Context, g lexicalproviders.Global, completions *CompletionTree) []data.NRCompletion {
	renderedComps := make([]data.NRCompletion, 0, len(completions.children))

	var prefix string
	for _, c := range completions.children {
		pRComp := m.makeCompletion(ctx, g, c.completion)
		if pRComp.Display == "" {
			continue
		}

		if m.options.PrependCompletionContext {
			prefix = getCompletionContext(m.selectedBuffer.Buffer.Text(), pRComp.Completion)
			if prefix != "" {
				pRComp.Completion = pRComp.Completion.Prepend(prefix)
			}
		}

		renderedComps = append(renderedComps, data.NRCompletion{
			RCompletion: pRComp,
		})
	}

	if m.options.AddSmartStar() {
		for i := range renderedComps {
			renderedComps[i].AddSmartStar(m.options.RenderOptions)
		}
	}

	return renderedComps
}

var longestPrefix = 150

// getCompletionContext returns a prefix based on the root buffer to prepend onto a completion
func getCompletionContext(textField string, completion data.Completion) string {
	begin := completion.Replace.Begin
	for i := begin - 1; i > max(begin-1-longestPrefix, 0)-1; i-- {
		if strings.ContainsAny(string(textField[i]), "\n\r\f") {
			// Don't include the newline or leading whitespace in the prefix
			return strings.TrimLeft(textField[i+1:begin], "\t ")
		}
	}
	return ""
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
