package traindata

import (
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
)

const maxNumTypes = 1
const maxNumSubtokens = 9

// Arg in a call pattern
type Arg struct {
	Name      string
	Subtokens []string
	Types     []pythonresource.SigStatTypeInfo
}

// CallPatterns ...
type CallPatterns struct {
	Spec       *pythonimports.ArgSpec
	Positional []*Arg
	ArgsByName map[string]*Arg
	Signatures []*editorapi.Signature
}

// MinNumArgs possible in a call
func (cp *CallPatterns) MinNumArgs() int {
	min := -1
	for _, s := range cp.Signatures {
		if min == -1 || len(s.Args) < min {
			min = len(s.Args)
		}
	}
	return min
}

// MaxNumArgs possible in a call
func (cp *CallPatterns) MaxNumArgs() int {
	return len(cp.ArgsByName)
}

// ArgName ...
func (cp *CallPatterns) ArgName(i int) string {
	if i < len(cp.Positional) {
		return cp.Positional[i].Name
	}

	// TODO: we should not need this but we currently do
	// not handle varargs correctly basically anywhere in the pipeline
	if cp.Spec != nil && i < len(cp.Spec.NonReceiverArgs()) {
		return cp.Spec.NonReceiverArgs()[i].Name
	}

	return ""
}

// PositionalOK ...
func (cp *CallPatterns) PositionalOK(i int) bool {
	if i < len(cp.Positional) {
		return true
	}
	if cp.Spec == nil {
		return false
	}

	if args := cp.Spec.NonReceiverArgs(); i < len(args) {
		arg := args[i]
		if arg.KeywordOnly {
			return false
		}
		if _, ok := cp.ArgsByName[arg.Name]; ok {
			return true
		}
	}
	return false
}

// KeywordOK ...
func (cp *CallPatterns) KeywordOK(name string) bool {
	_, ok := cp.ArgsByName[name]
	return ok
}

// Matches the specified call
// TODO: this could be better
// - take into account if keywords appeared in an actual pattern vs just valid
// - look at the type and subtoken matches
func (cp *CallPatterns) Matches(call *pythonast.CallExpr) bool {
	for i, arg := range call.Args {
		if name, ok := arg.Name.(*pythonast.NameExpr); ok {
			if _, ok := cp.ArgsByName[name.Ident.Literal]; !ok {
				return false
			}
		} else {
			if i >= len(cp.Positional) {
				return false
			}
		}
	}
	return true
}

// Feed for the specified arg
// NOTE: this is only safe to call after CallPatterns.PositonalOK or CallPatterns.KeywordOK
// has been called
func (cp *CallPatterns) Feed(name string, pos int) ([]string, []string) {
	var arg *Arg
	if name != "" {
		arg = cp.ArgsByName[name]
	} else if pos < len(cp.Positional) {
		arg = cp.Positional[pos]
	} else if cp.Spec != nil && pos < len(cp.Spec.NonReceiverArgs()) {
		arg = cp.ArgsByName[cp.Spec.NonReceiverArgs()[pos].Name]
	}

	if arg == nil {
		// TODO: this is terrible. We have multiple signature datasets
		// and they are not consistent.
		return []string{UnknownType}, []string{UnknownTokenMarker}
	}

	seen := make(map[string]bool)
	var types []string
	for _, t := range arg.Types {
		ts := pythonimports.NewDottedPath(t.Path).WithTail(InstanceTail).String()
		if seen[ts] {
			continue
		}
		seen[ts] = true

		types = append(types, ts)
		if maxNumTypes > 0 && len(types) >= maxNumTypes {
			break
		}
	}

	if len(types) == 0 {
		types = append(types, UnknownType)
	}

	seen = make(map[string]bool)
	var subtokens []string
	for _, st := range arg.Subtokens {
		if seen[st] {
			continue
		}
		seen[st] = true

		subtokens = append(subtokens, st)
		if maxNumSubtokens > 0 && len(subtokens) >= maxNumSubtokens {
			break
		}
	}

	if len(subtokens) == 0 {
		subtokens = append(subtokens, UnknownTokenMarker)
	}

	return types, subtokens
}

// NewCallPatterns for the provided symbol
func NewCallPatterns(rm pythonresource.Manager, sym pythonresource.Symbol) *CallPatterns {
	ss := rm.SigStats(sym)
	sigs := rm.PopularSignatures(sym)
	if ss == nil || len(sigs) == 0 {
		// TODO
		return nil
	}

	spec := rm.ArgSpec(sym)
	var specArgs []pythonimports.Arg
	if spec != nil {
		specArgs = spec.NonReceiverArgs()
	}

	cps := &CallPatterns{
		Signatures: sigs,
		ArgsByName: make(map[string]*Arg),
		Spec:       spec,
	}

	var maxPos int
	for _, sig := range sigs {
		if len(sig.Args) > maxPos {
			maxPos = len(sig.Args)
		}
	}

	newArg := func(name string, stats pythonresource.SigStatArg) *Arg {
		var ts []pythonresource.SigStatTypeInfo
		for _, t := range stats.Types {
			ts = append(ts, t)
		}

		sort.Slice(ts, func(i, j int) bool {
			return ts[i].Count > ts[j].Count
		})

		if name == "" {
			name = stats.Name
		}

		var subtoks []string
		if name != "" {
			subtoks = SplitNameLiteral(name)
		}

		return &Arg{
			Name:      name,
			Types:     ts,
			Subtokens: subtoks,
		}
	}

	for i := 0; i < maxPos; i++ {
		var name string
		if i < len(specArgs) {
			if arg := specArgs[i]; !arg.KeywordOnly {
				name = arg.Name
			}
		}

		arg := newArg(name, ss.Positional[i])
		cps.Positional = append(cps.Positional, arg)
		if name != "" {
			cps.ArgsByName[name] = arg
		}
	}

	addExample := func(arg *Arg, pe *editorapi.ParameterExample) {
		for _, t := range pe.Types {
			for _, e := range t.Examples {
				if !pythonscanner.IsValidIdent(e) {
					continue
				}
				arg.Subtokens = append(arg.Subtokens, SplitNameLiteral(e)...)
			}
		}
	}

	for _, sig := range sigs {
		for i, pe := range sig.Args {
			arg := cps.Positional[i]
			addExample(arg, pe)
		}

		pyDeets := sig.LanguageDetails.Python
		if pyDeets == nil {
			continue
		}

		for _, pe := range pyDeets.Kwargs {
			arg := cps.ArgsByName[pe.Name]
			if arg == nil {
				arg = newArg(pe.Name, ss.ArgsByName[pe.Name])
				cps.ArgsByName[pe.Name] = arg
			}
			addExample(arg, pe)
		}
	}

	for name, arg := range ss.ArgsByName {
		if _, ok := cps.ArgsByName[name]; ok {
			continue
		}
		cps.ArgsByName[name] = newArg(name, arg)
	}

	return cps
}
