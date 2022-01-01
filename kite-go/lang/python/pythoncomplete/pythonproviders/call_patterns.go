package pythonproviders

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// CallPatterns is a Provider for call pattern snippet completions;
// it completes situations such as foo(‸) with snippet-containing call patterns.
type CallPatterns struct{}

func getArgCount(args []*pythonast.Argument) int {
	var result int
	for _, a := range args {
		if a.Begin() != a.End() {
			result++
		}
	}
	return result
}

// Name implements Provider
func (CallPatterns) Name() data.ProviderName {
	return data.PythonCallPatternsProvider
}

// Provide implements Provider
func (c CallPatterns) Provide(ctx kitectx.Context, g Global, in Inputs, out OutputFunc) error {
	ctx.CheckAbort()

	_, isSmart := SmartProviders[c.Name()]
	if isSmart && g.Product.GetProduct() != licensing.Pro {
		return data.ProviderNotApplicableError{}
	}

	resolved := in.ResolvedAST()

	callExpr, _ := deepestNotContained(in.UnderSelection(), in.Selection).(*pythonast.CallExpr)
	if callExpr == nil {
		return data.ProviderNotApplicableError{}
	}

	// if in.Selection.Begin == callExpr.LeftParen.Begin then the selection
	// includes the left paren which we do not want
	if in.Selection.Begin <= int(callExpr.LeftParen.Begin) {
		return data.ProviderNotApplicableError{}
	}

	// the end of selection is "exclusive" so in.Selection.End == callExpr.Righparen.Begin is ok
	// since the right paren is not included in the selection
	if callExpr.RightParen != nil && in.Selection.End > int(callExpr.RightParen.Begin) {
		return data.ProviderNotApplicableError{}
	}

	argCount := getArgCount(callExpr.Args)
	// deal with cases when the user has already typed part of the call
	if argCount > 0 {
		// make sure the user has typed the comma
		if argCount != len(callExpr.Commas) {
			return data.ProviderNotApplicableError{}
		}

		// make sure the cursor is after the last comma
		// TODO: can the selection span multiple commas?
		lastComma := callExpr.Commas[len(callExpr.Commas)-1]
		if in.Selection.Begin <= int(lastComma.Begin) {
			return data.ProviderNotApplicableError{}
		}
	}

	funcVal := resolved.References[callExpr.Func]
	if funcVal == nil {
		return nil
	}

	begin := int(callExpr.LeftParen.Begin)
	if argCount > 0 {
		// Use the selection instead of the end of the last comma to avoid weird
		// issues with whitespace between the last comma and the cursor
		begin = in.Selection.Begin
	}
	cs := make(map[string]MetaCompletion)
	for _, fi := range funcInfos(ctx, g.ResourceManager, g.LocalIndex, funcVal) {
		if fi.Spec == nil {
			continue
		}
		for _, sig := range fi.Sigs {
			sig = matchSignature(callExpr, sig, fi.Spec)
			if sig == nil {
				continue
			}

			// setting a RenderMeta here will cause these completions to be thought of
			// as function completions, even though they're not. We manually fix rendering
			// for the nested case during mixing/rendering.
			key := getSignatureKey(sig)
			if v, ok := cs[key]; !ok {
				score := sig.Frequency
				if score == 0 {
					// That allows to keep the order of patterns (as they are correctly sorted in the RM)
					// And it also promotes entries that actually have a frequency
					score = float64(-len(cs))
				}
				cs[key] = MetaCompletion{
					Completion: data.Completion{
						Replace: data.Selection{
							Begin: begin,
							End:   int(callExpr.End()),
						},
						Snippet: snippetFromSig(sig, argCount == 0, g.ResourceManager, fi.Symbol),
					},
					Score:    score,
					Provider: c.Name(),
					Source:   fi.Source,
					CallPatternMeta: &CallPatternMeta{
						ArgSpec:       fi.Spec,
						ArgumentCount: argCount,
					},
					FromSmartProvider: isSmart,
				}
			} else {
				if v.Score < sig.Frequency && sig.Frequency > 0 {
					v.Score = sig.Frequency
					cs[key] = v
				}
			}
		}
	}

	var patterns []MetaCompletion
	for _, v := range cs {
		patterns = append(patterns, v)
	}
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Score > patterns[j].Score
	})

	// TODO(naman) use frequencies for scores?
	n := float64(len(cs))
	for i, c := range patterns {
		// P(i) = 2/n - (2i + 1)/n^2
		c.Score = 2./n - (2.*float64(i)+1)/(n*n)
		out(ctx, in.SelectedBuffer, c)
	}

	return nil
}

// MarshalJSON implements Provider
func (c CallPatterns) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type data.ProviderName `json:"type"`
	}{
		Type: c.Name(),
	})
}

func getSignatureKey(sig *editorapi.Signature) string {
	result := fmt.Sprintf("%d_", len(sig.Args))
	if sig.LanguageDetails.Python != nil {
		var kw []string
		for _, k := range sig.LanguageDetails.Python.Kwargs {
			kw = append(kw, k.Name)
		}
		sort.Strings(kw)
		result = result + strings.Join(kw, "_")
	}
	return result
}

func matchSignature(call *pythonast.CallExpr, sig *editorapi.Signature, spec *pythonimports.ArgSpec) *editorapi.Signature {
	argCount := getArgCount(call.Args)
	if len(sig.Args) > argCount {
		// we still have positional arguments left in the signature
		// so we remove positional arguments that have already been typed

		// use the argspec to compute the "names"/placeholder text below
		specArgs := spec.Args
		var selfOffset int
		if argCount >= len(specArgs) {
			specArgs = nil
		} else if receiverParam(specArgs[0].Name) {
			specArgs = specArgs[1+argCount:]
			selfOffset = 1
		} else {
			specArgs = specArgs[argCount:]
		}

		sigArgs := sig.Args[argCount:]
		args := make([]*editorapi.ParameterExample, 0, len(sig.Args)-argCount)
		var varargIdx int
		posArgCount := countPositionalArgsOnly(spec)
		if argCount > posArgCount-selfOffset {
			// We are filling the varargs, let's count how many of them got already fill
			varargIdx = argCount - (posArgCount - selfOffset) // We count how many varArgs have already been written
		}

		for i, arg := range sigArgs {
			copy := *arg // shouldn't modify pointer into resource manager
			if i < len(specArgs) && !specArgs[i].KeywordOnly {
				copy.Name = specArgs[i].Name
			} else if spec.Vararg != "" {
				if arg.Name != "" {
					copy.Name = arg.Name
				} else {
					copy.Name = fmt.Sprintf("%s_%d", spec.Vararg, varargIdx)
				}
				varargIdx++
			} else {
				return nil
			}
			args = append(args, &copy)
		}
		return &editorapi.Signature{
			Args:            args,
			LanguageDetails: sig.LanguageDetails,
		}
	}
	// NOTE: we explicitly stop suggesting signatures once all of the positional
	// parameters are filled, this avoids weird duplicates with the
	// argspec Provider and also reduces the visual noise.
	// TODO: should we do something different from a UX perspective here?
	return nil
}

func countPositionalArgsOnly(spec *pythonimports.ArgSpec) int {
	var result int
	for _, a := range spec.Args {
		if !a.KeywordOnly {
			result++
		}
	}
	return result
}

func snippetFromSig(sig *editorapi.Signature, includeLeftParen bool, rm pythonresource.Manager, symbol pythonresource.Symbol) data.Snippet {
	snipArgs := make([]string, 0, len(sig.Args))
	for _, arg := range sig.Args {
		if receiverParam(arg.Name) {
			continue
		}
		snipArgs = append(snipArgs, data.HoleWithPlaceholderMarks(arg.Name))
	}

	if pyDeets := sig.LanguageDetails.Python; pyDeets != nil {
		kwSnipArgs := make([]string, 0, len(pyDeets.Kwargs))
		for _, arg := range pyDeets.Kwargs {
			var ts []string
			for _, t := range arg.Types {
				if len(ts) == maxTypesForPlaceholder {
					break
				}
				if t.ID.LanguageSpecific() != "" {
					ts = append(ts, t.ID.LanguageSpecific())
				}
			}
			kwSnipArgs = append(kwSnipArgs, fmt.Sprintf("%s=%s", arg.Name, holeForPlaceholderTypes(ts)))
		}
		sort.Slice(kwSnipArgs, func(i, j int) bool {
			argI := kwSnipArgs[i]
			argJ := kwSnipArgs[j]
			if !symbol.Nil() {
				countI, okI := rm.KeywordArgFrequency(symbol, argI)
				countJ, okJ := rm.KeywordArgFrequency(symbol, argJ)
				if okI && !okJ {
					return true
				}
				if !okI && okJ {
					return false
				}
				if okI && okJ {
					return countI > countJ
				}
			}
			return argI < argJ
		})
		snipArgs = append(snipArgs, kwSnipArgs...)
	}
	fstr := "(%s)"
	if !includeLeftParen {
		fstr = "%s)"
	}
	return data.BuildSnippet(fmt.Sprintf(fstr, strings.Join(snipArgs, ", ")))
}
