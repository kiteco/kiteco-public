package pythonproviders

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/rollbar"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const maxTypesForPlaceholder = 1

// KWArgs provides semantic keyword argument completions.
type KWArgs struct{}

// Name implements Provider
func (KWArgs) Name() data.ProviderName {
	return data.PythonKWArgsProvider
}

// Provide implements Provider
func (a KWArgs) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	if _, ok := SmartProviders[a.Name()]; ok && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	if in.Selection.Len() > 0 {
		// Fix for issue 8162 : https://github.com/kiteco/kiteco/issues/8162
		// Title: kwarg completion shown when positional arg completion is clearly the user's intention
		// The selection is not empty, so it's either a value either a placeholder, so we don't want to replace it
		// by a keyword argument (as it was a positional argument before or a placeholder for it)
		return data.ProviderNotApplicableError{}
	}
	cursor := in.Selection.Begin

	underPos := in.UnderSelection()
	var call *pythonast.CallExpr
	for i := len(underPos) - 1; i >= 0; i-- {
		n := underPos[i]
		if call, _ = n.(*pythonast.CallExpr); call != nil {
			break
		}
	}
	if call == nil ||
		cursor <= int(call.LeftParen.Begin) ||
		(call.RightParen != nil && cursor > int(call.RightParen.Begin)) {
		return data.ProviderNotApplicableError{}
	}

	// compute the replacement range by finding the argument under the cursor
	replace := data.Cursor(cursor)
	var arg *pythonast.Argument
	if len(call.Args) > len(call.Commas) {
		// by default, assume we're in the argument after the last comma
		arg = call.Args[len(call.Args)-1]
	}
	for i, comma := range call.Commas {
		// check if we're not in the argument after the last comma
		if cursor <= int(comma.Begin) {
			if i < len(call.Args) {
				arg = call.Args[i]
			} else {
				// TODO how can this happen? it's observed in Rollbar
				rollbar.Error(errors.New("unexpected call with more commas than args"), call)
				arg = nil
				replace.Begin = int(comma.Begin)
				replace.End = int(comma.Begin)
			}
			break
		}
	}
	if arg != nil {
		replace = data.Selection{
			Begin: int(arg.Begin()),
			End:   int(arg.End()),
		}
	}

	val := in.ResolvedAST().References[call.Func]
	if val == nil {
		return nil
	}

	// TODO: better ranking for the value constituents
	var cs []MetaCompletion
	for _, fi := range funcInfos(ctx, g.ResourceManager, g.LocalIndex, val) {
		cs = append(cs, a.keywordCompletions(fi, replace, call)...)
	}

	// TODO(naman) use frequencies for scores?
	n := float64(len(cs))
	for i, c := range cs {
		// P(i) = 2/n - (2i + 1)/n^2
		if valid, ok := c.Validate(in.SelectedBuffer); ok {
			c.Completion = valid
		} else {
			continue
		}
		c.Score = 2./n - (2.*float64(i)+1)/(n*n)
		out(ctx, in.SelectedBuffer, c)
	}

	return nil
}

// MarshalJSON implements Provider
func (a KWArgs) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: a.Name(),
	})
}

func (a KWArgs) keywordCompletions(fi funcInfo, s data.Selection, call *pythonast.CallExpr) []MetaCompletion {
	ci := newCallInfo(s, call, fi.Spec)
	if ci.FirstKeyword == -1 || ci.Selected < ci.FirstKeyword {
		// have not seen a keyword arg or the slot we are in is before the first keyword
		if fi.Spec == nil {
			// no arg spec so we have to wait until the selected argument is past the first keyword
			return nil
		}

		// NOTE: we ignore required keyword only args for now
		var required int
		for _, arg := range fi.Spec.Args {
			if receiverParam(arg.Name) {
				continue
			}

			if arg.DefaultValue == "" {
				required++
			} else {
				break
			}
		}

		if ci.Selected < required {
			return nil
		}
	}

	skip := func(p string) bool {
		return ci.Filled[p] || receiverParam(p)
	}

	_, isSmart := SmartProviders[a.Name()]
	comp := func(name, hole string, score float64) MetaCompletion {
		// NOTE: we explicitly do not set `RenderMeta` since
		// we do not want to show documentation for the function
		// when the user is inside the call.
		return MetaCompletion{
			Provider:         a.Name(),
			Source:           response.ArgSpecCompletionSource,
			KeywordModelMeta: true,
			Score:            score,
			Completion: data.Completion{
				Snippet: data.BuildSnippet(fmt.Sprintf("%s=%s", name, hole)),
				Replace: s,
			},
			ArgSpecMeta:       &ArgSpecMeta{ArgSpec: fi.Spec, ArgumentCount: getArgCount(call.Args)},
			FromSmartProvider: isSmart,
		}
	}

	var cs []MetaCompletion
	if fi.Spec != nil {
		for _, param := range fi.Spec.Args {
			if skip(param.Name) {
				continue
			}
			// just use a score of 1 since we do not have a better score
			// and the caller assumes the returned completions are already in sorted order
			cs = append(cs, comp(param.Name, holeForPlaceholderTypes(param.Types), 1.))
		}
	}

	if fi.Kwargs != nil {
		for _, kw := range fi.Kwargs.Kwargs {
			if skip(kw.Name) {
				continue
			}
			// just use a score of 1 since we assume the kwargs
			// are already sorted by score and the caller assumes
			// the returned completions are already in sorted order
			cs = append(cs, comp(kw.Name, holeForPlaceholderTypes(kw.Types), 1.))
		}
	}

	if fi.Patterns != nil {
		ts := func(scs []*pythoncode.StringCount) []string {
			var ts []string
			for i, sc := range scs {
				if i == maxTypesForPlaceholder {
					break
				}
				ts = append(ts, sc.Value)
			}
			return ts
		}

		// maintain a separate list that we will sort
		// and then append after the arg spec arguments
		var scored []MetaCompletion
		for _, a := range fi.Patterns.Args {
			if skip(a.Name) {
				continue
			}
			scored = append(scored, comp(a.Name, holeForPlaceholderTypes(ts(a.Types)), float64(a.Count)))
		}

		for _, kw := range fi.Patterns.Kwargs {
			if skip(kw.Name) {
				continue
			}
			scored = append(scored, comp(kw.Name, holeForPlaceholderTypes(ts(kw.Types)), float64(kw.Count)))
		}

		sort.Slice(scored, func(i, j int) bool {
			si, sj := scored[i].Score, scored[j].Score
			if math.Abs(si-sj) < 1e-6 {
				return scored[i].Snippet.Text < scored[j].Snippet.Text
			}
			return si > sj
		})
		cs = append(cs, scored...)
	}

	return cs
}

type callInfo struct {
	Filled       map[string]bool
	FirstKeyword int
	Selected     int
}

func newCallInfo(s data.Selection, call *pythonast.CallExpr, spec *pythonimports.ArgSpec) callInfo {
	argCount := getArgCount(call.Args)
	ci := callInfo{
		Filled:       make(map[string]bool, argCount-1),
		Selected:     -1,
		FirstKeyword: -1,
	}

	for i, comma := range call.Commas {
		if s.End <= int(comma.Begin) {
			ci.Selected = i
			break
		}
	}
	if ci.Selected == -1 {
		// must be at the final argument
		ci.Selected = len(call.Commas)
	}

	var numBoundArgs int
	if spec != nil && len(spec.Args) > 0 && receiverParam(spec.Args[0].Name) {
		numBoundArgs++
	}

	for i := 0; i < argCount; i++ {
		arg := call.Args[i]
		if name, ok := arg.Name.(*pythonast.NameExpr); ok {
			ci.Filled[name.Ident.Literal] = true
			if ci.FirstKeyword == -1 {
				ci.FirstKeyword = i
			}
		} else if i != ci.Selected && spec != nil && (i+numBoundArgs) < len(spec.Args) && ci.FirstKeyword == -1 {
			ci.Filled[spec.Args[i+numBoundArgs].Name] = true
		}
	}

	return ci
}

func receiverParam(name string) bool {
	return name == "cls" || name == "self"
}
