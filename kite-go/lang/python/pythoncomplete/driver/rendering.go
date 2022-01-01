package driver

import (
	"strings"

	"github.com/jaytaylor/html2text"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// hints can be any string (since we use types as hints), but we also have the following fixed hints
const (
	unknownHint        = ""
	keywordHint        = "keyword"
	callHint           = "call"
	partialCallHint    = "partial call"
	popularPatternHint = "snippet"

	functionKindHint   = "function"
	typeKindHint       = "type"
	moduleKindHint     = "module"
	descriptorKindHint = "descriptor"
	unionKindHint      = "union"
	ProHint            = "pro"
)

var (
	maxDocLength = 500
)

func getIDs(ctx kitectx.Context, g pythonproviders.Global, val pythontype.Value) (web string, local string) {
	if val == nil {
		return "", ""
	}

	val = pythontype.MostSpecific(ctx, pythontype.Translate(ctx, val, g.ResourceManager))
	switch val := val.(type) {
	case pythontype.External:
		web = val.Symbol().PathString()
	case pythontype.ExternalInstance:
		web = val.TypeExternal.Symbol().PathString()
	}

	// if we wanted to use pythonenv.SymbolLocator here, we'd also need a namespace;
	// but namespaces are nasty, so let's not do that unless it's really needed.
	local = python.ValidateID(ctx, g.ResourceManager, g.LocalIndex, nil, pythonenv.Locator(val)).String()

	return
}

func getDoc(ctx kitectx.Context, pyctx *python.Context, g pythonproviders.Global, val pythontype.Value) (doc string, isHTML bool) {
	if pyctx != nil && pyctx.BufferIndex != nil {
		if doc, err := pyctx.BufferIndex.Documentation(val); err == nil {
			if doc.HTML != "" {
				return doc.HTML, true
			}
			return doc.Description, false

		}
	}
	if g.LocalIndex != nil {
		if doc, err := g.LocalIndex.Documentation(ctx, val); err == nil {
			if doc.HTML != "" {
				return doc.HTML, true
			}
			return doc.Description, false
		}
	}
	for _, val := range pythontype.Disjuncts(ctx, val) {
		var sym pythonresource.Symbol
		switch val := val.(type) {
		case pythontype.External:
			sym = val.Symbol()
		case pythontype.ExternalInstance:
			sym = val.TypeExternal.Symbol()
		}
		if sym.Nil() {
			continue
		}
		docs := g.ResourceManager.Documentation(sym)
		if docs != nil {
			if docs.HTML != "" {
				return docs.HTML, true
			}
			return docs.Text, false
		}
	}
	return "", false
}

func (m *Mixer) makeCompletion(ctx kitectx.Context, pyctx *python.Context, g pythonproviders.Global, c Completion) data.RCompletion {
	ctx.CheckAbort()

	webID, localID := getIDs(ctx, g, c.Meta.RenderMeta.Referent)
	rcompl := data.RCompletion{
		Completion: c.Meta.Completion,
		Hint:       unknownHint,
		ReferentInfo: data.ReferentInfo{
			WebID:   webID,
			LocalID: localID,
		},
		Provider: c.Meta.Provider,
		Source:   c.Meta.Source,
		Smart:    data.CompletionIsSmart(c.Meta.Completion, c.Meta.FromSmartProvider, m.options.RenderOptions),
		IsServer: c.Meta.IsServer,
		Debug:    c,
	}

	opts := data.DisplayOptions{
		NoUnicode: m.options.NoUnicode,
	}
	switch c.Meta.MixingMeta.Provider.Name() {
	// TODO get rid of GGNN logic
	case data.PythonGGNNModelProvider:
		opts.NoEmptyPH = true
		opts.TrimBeforeEmptyPH = c.Meta.GGNNMeta.SpeculationPlaceholderPresent
	case data.PythonLexicalProvider:
		opts.TrimBeforeEmptyPH = true
	}
	rcompl.Display = c.Meta.Completion.DisplayText(m.selectedBuffer, opts)

	switch {
	// No reason to tag with "Pro" if all completions are pro
	case m.options.ProHint && rcompl.Smart && !m.options.AllCompletionsStarred:
		rcompl.Hint = ProHint
	case c.Meta.CallModelMeta != nil:
		rcompl.Hint = callHint
	case c.Meta.CallPatternMeta != nil:
		rcompl.Hint = popularPatternHint
	case c.Meta.GGNNMeta != nil:
		if c.Meta.GGNNMeta.Call != nil {
			if c.Meta.GGNNMeta.SpeculationPlaceholderPresent {
				rcompl.Hint = partialCallHint
			} else {
				rcompl.Hint = callHint
			}
		}
	case c.Meta.KeywordModelMeta:
		rcompl.Hint = keywordHint
	}

	val := c.Meta.RenderMeta.Referent
	if val == nil {
		return rcompl
	}

	vals := pythontype.Disjuncts(ctx, val)
	knd := vals[0].Kind()
	for _, val := range vals[1:] {
		if val.Kind() != knd {
			knd = pythontype.UnionKind
			break
		}
	}

	if rcompl.Hint == unknownHint {
		switch knd {
		case pythontype.UnknownKind:
			rcompl.Hint = unknownHint
		case pythontype.FunctionKind:
			rcompl.Hint = functionKindHint
		case pythontype.TypeKind:
			rcompl.Hint = typeKindHint
		case pythontype.ModuleKind:
			rcompl.Hint = moduleKindHint
		case pythontype.InstanceKind:
			if reprs := pythontype.Reprs(ctx, val.Type(), g.ResourceManager, true); reprs != nil {
				rcompl.Hint = strings.Join(reprs, " | ")
			}
		case pythontype.DescriptorKind:
			rcompl.Hint = descriptorKindHint
		case pythontype.UnionKind:
			rcompl.Hint = unionKindHint
		}
	}

	doc, isHTML := getDoc(ctx, pyctx, g, val)
	if isHTML {
		doc, _ = html2text.FromString(doc)
	}
	rcompl.Docs.Text = doc
	if len(rcompl.Docs.Text) > maxDocLength {
		rcompl.Docs.Text = rcompl.Docs.Text[:maxDocLength]
	}
	return rcompl
}

func (m *Mixer) renderCompletions(ctx kitectx.Context, pyctx *python.Context, g pythonproviders.Global, completions *CompletionTree) []data.NRCompletion {
	renderedComps := make([]data.NRCompletion, 0, len(completions.children))

	var prefix string
	for _, c := range completions.children {
		pRComp := m.makeCompletion(ctx, pyctx, g, c.Completion)
		if pRComp.Display == "" {
			continue
		}

		// only include a single pro CTA completion
		if pRComp.Smart && m.options.SingleTokenProCompletion && !m.options.FullProCompletion {
			// truncate to a single token
			if truncated, err := pRComp.Completion.SingleToken(); err == nil {
				pRComp.Completion = truncated
			}
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
