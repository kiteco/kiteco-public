package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

type symCount struct {
	Sym   pythonresource.Symbol
	Count int
}

type symCounts map[pythonimports.Hash]*symCount

type srcStrsBySym map[pythonimports.Hash]pythonpatterns.StrCount

type argument struct {
	// Name of the corresponding parameter in the function signature, if
	// we were able to find an argspec for the function
	Name string

	Count int

	SrcStrs pythonpatterns.StrCount

	Types symCounts

	SrcStrsByType srcStrsBySym
}

func newArgument(name string) *argument {
	return &argument{
		Name:          name,
		SrcStrs:       make(pythonpatterns.StrCount),
		Types:         make(symCounts),
		SrcStrsByType: make(srcStrsBySym),
	}
}

type pattern struct {
	Count      int
	Frequency  float64
	Positional []*argument
	Keyword    map[string]*argument
}

func (p pattern) String() string {
	var kws []string
	for kw := range p.Keyword {
		kws = append(kws, kw)
	}
	sort.Strings(kws)
	return fmt.Sprintf("%d %s", len(p.Positional), strings.Join(kws, ","))
}

func (p pattern) ForArgs(f func(*argument)) {
	for _, arg := range p.Positional {
		f(arg)
	}
	for _, arg := range p.Keyword {
		f(arg)
	}
}

type symPatterns struct {
	Sym      pythonresource.Symbol
	Patterns []pattern

	// We run into some pretty serious data sparsity issues
	// so we have to group argument information across all patterns
	// TODO: we could do something smarter here by not neccesarily merging
	// information for all patterns, but we forgo that for now
	Positional []*argument
	Keyword    map[string]*argument
}

type patternsByHash map[pythonimports.Hash]*symPatterns

func loadPatterns(rm pythonresource.Manager, dir string, validate bool) patternsByHash {
	files, err := fileutil.ListDir(dir)
	maybeQuit(err)

	byHash := make(patternsByHash)
	for _, f := range files {
		if strings.HasSuffix(f, "DONE") {
			continue
		}
		err := serialization.Decode(f, func(calls *pythonpatterns.Calls) {
			if len(calls.Calls) == 0 {
				maybeQuit(fmt.Errorf("got empty calls from file %s", f))
			}

			sym, err := rm.NewSymbol(calls.Func.Dist, calls.Func.Path)
			if err != nil {
				log.Println(errors.Wrapf(err, "ERROR: could not load symbol for %s %s", calls.Func.Dist, calls.Func.Path))
				return
			}

			sym = sym.Canonical()
			if _, ok := byHash[sym.Hash()]; ok {
				maybeQuit(fmt.Errorf("got multiple patterns for canonical sym %s from %v", sym, calls.Func))
			}

			sp := newSymPatterns(rm, sym, calls.Calls, validate)
			if sp == nil {
				return
			}

			byHash[sym.Hash()] = sp
		})
		maybeQuit(err)
	}

	return byHash
}

func newSymPatterns(rm pythonresource.Manager, sym pythonresource.Symbol, calls []pythonpatterns.Call, validate bool) *symPatterns {
	as := rm.ArgSpec(sym)

	addToArgument := func(arg *argument, as pythonpatterns.ArgSummary) {
		for _, es := range as {
			arg.Count += es.Count
			for s, c := range es.SrcStrs {
				arg.SrcStrs[s] += c
			}

			for _, s := range es.Syms {
				// skip external return values because we currently
				// do not support them as part of the editorapi
				if s.Kind == pythonpatterns.ExternalReturnValue {
					continue
				}

				sym, err := rm.NewSymbol(s.Dist, s.Path)
				if err != nil {
					continue
				}

				// always canonicalize since we do not
				// know where these symbols are coming from
				sym = sym.Canonical()

				// ideally we would use the full symbol hash here
				// but the current way we have locating symbols
				// does not support this so we use the path hash
				sh := sym.PathHash()

				symCounts := arg.Types[sh]
				if symCounts == nil {
					symCounts = &symCount{Sym: sym}
					arg.Types[sh] = symCounts
				}
				symCounts.Count += es.Count

				srcCounts := arg.SrcStrsByType[sh]
				if srcCounts == nil {
					srcCounts = make(pythonpatterns.StrCount)
					arg.SrcStrsByType[sh] = srcCounts
				}
				for s, c := range es.SrcStrs {
					srcCounts[s] += c
				}
			}
		}
	}

	var maxNumPos int
	for _, call := range calls {
		if len(call.Positional) > maxNumPos {
			maxNumPos = len(call.Positional)
		}
	}

	sp := &symPatterns{
		Sym:        sym,
		Positional: make([]*argument, 0, maxNumPos),
		Keyword:    make(map[string]*argument),
	}

	for i := 0; i < maxNumPos; i++ {
		var name string
		if as != nil && len(as.Args) > 0 {
			idx := i
			if as.Args[0].Name == "self" || as.Args[0].Name == "cls" {
				idx++
			}
			if idx < len(as.Args) {
				arg := as.Args[idx]
				if !arg.KeywordOnly {
					name = arg.Name
				}
			}
		}

		arg := newArgument(name)
		// share information between parameters that are passed by
		// name and position
		if arg.Name != "" {
			sp.Keyword[arg.Name] = arg
		}
		sp.Positional = append(sp.Positional, arg)
	}

	var total float64
	for _, call := range calls {
		if validate && as != nil {
			if err := call.Validate(as); err != nil {
				continue
			}
		}
		total += float64(call.Count)

		pat := pattern{
			Count:     call.Count,
			Frequency: float64(call.Count),
			Keyword:   make(map[string]*argument),
		}

		for i, as := range call.Positional {
			arg := sp.Positional[i]
			addToArgument(arg, as)
			pat.Positional = append(pat.Positional, arg)
		}
		for k, as := range call.Keyword {
			arg := sp.Keyword[k]
			if arg == nil {
				arg = newArgument(k)
				sp.Keyword[k] = arg
			}
			addToArgument(arg, as)
			pat.Keyword[k] = arg
		}

		sp.Patterns = append(sp.Patterns, pat)
	}

	for i, pat := range sp.Patterns {
		sp.Patterns[i].Frequency = float64(pat.Frequency) / total
	}

	// input calls should already be sorted but just to be safe
	sort.Slice(sp.Patterns, func(i, j int) bool {
		return sp.Patterns[i].Frequency > sp.Patterns[j].Frequency
	})

	if len(sp.Patterns) == 0 {
		return nil
	}

	return sp
}
