package data

import (
	"sort"
	"strings"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"
)

func symbolsHash(syms []pythonpatterns.Symbol) pythonimports.Hash {
	hs := make([]string, 0, len(syms))
	for _, s := range syms {
		hs = append(hs, s.Hash().String())
	}
	sort.Strings(hs)
	return pythonimports.Hash(spooky.Hash64([]byte(strings.Join(hs, ""))))
}

type exprByHash map[pythonimports.Hash]*pythonpatterns.ExprSummary

func (ebh exprByHash) Add(es pythonpatterns.ExprSummary) {
	h := symbolsHash(es.Syms)

	base := ebh[h]
	if base == nil {
		base = &pythonpatterns.ExprSummary{
			Syms:     append([]pythonpatterns.Symbol{}, es.Syms...),
			SrcStrs:  make(pythonpatterns.StrCount),
			ASTTypes: make(pythonpatterns.StrCount),
		}
		ebh[h] = base
	}

	base.Count += es.Count
	for s, c := range es.SrcStrs {
		base.SrcStrs[s] += c
	}
	for s, c := range es.ASTTypes {
		base.ASTTypes[s] += c
	}
}

func (ebh exprByHash) Summaries() []pythonpatterns.ExprSummary {
	es := make([]pythonpatterns.ExprSummary, 0, len(ebh))
	for _, e := range ebh {
		es = append(es, *e)
	}
	sort.Slice(es, func(i, j int) bool {
		return es[i].Count > es[j].Count
	})
	return es
}

type callPatterns struct {
	Count      int
	Positional []exprByHash
	Keyword    map[string]exprByHash
	Hashes     pythonpatterns.StrCount
}

func newCallPatterns(numPos int, numKW int) *callPatterns {
	c := &callPatterns{
		Positional: make([]exprByHash, numPos),
		Keyword:    make(map[string]exprByHash, numKW),
		Hashes:     make(pythonpatterns.StrCount),
	}
	for i := range c.Positional {
		c.Positional[i] = make(exprByHash)
	}
	return c
}

// FilterParams ...
type FilterParams struct {
	MinCallCount    int
	MinPatternCount int
	MinSourceCount  int
	MinArgCount     int
	MinTypeCount    int
}

// BuildPatterns from calls
func BuildPatterns(fps FilterParams, calls Calls) pythonpatterns.Calls {
	if len(calls) < fps.MinCallCount {
		return pythonpatterns.Calls{}
	}

	byHash := make(map[pythonimports.Hash]*callPatterns)
	for _, c := range calls {
		h := c.hash()
		pat := byHash[h]
		if pat == nil {
			pat = newCallPatterns(len(c.Positional), len(c.Keyword))
			byHash[h] = pat
		}
		pat.Count++
		pat.Hashes[c.Hash]++

		for i, arg := range c.Positional {
			pat.Positional[i].Add(arg)
		}
		for k, arg := range c.Keyword {
			if pat.Keyword[k] == nil {
				pat.Keyword[k] = make(exprByHash)
			}
			pat.Keyword[k].Add(arg)
		}
	}

	patterns := make([]pythonpatterns.Call, 0, len(byHash))
	for _, pat := range byHash {
		if pat.Count < fps.MinPatternCount {
			continue
		}

		cp := pythonpatterns.Call{
			Count:      pat.Count,
			Positional: make([]pythonpatterns.ArgSummary, 0, len(pat.Positional)),
			Keyword:    make(map[string]pythonpatterns.ArgSummary, len(pat.Keyword)),
			Hashes:     pat.Hashes,
		}

		for k, ebh := range pat.Keyword {
			cp.Keyword[k] = ebh.Summaries()
		}

		for _, ebh := range pat.Positional {
			cp.Positional = append(cp.Positional, ebh.Summaries())
		}

		cp = filterPattern(fps, cp)
		if cp.Count == 0 {
			continue
		}

		patterns = append(patterns, cp)
	}

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	if len(patterns) == 0 {
		return pythonpatterns.Calls{}
	}

	return pythonpatterns.Calls{
		Func:  calls[0].Func,
		Calls: patterns,
	}
}

func filterPattern(fps FilterParams, cp pythonpatterns.Call) pythonpatterns.Call {
	if cp.Count < fps.MinPatternCount {
		return pythonpatterns.Call{}
	}

	filterArg := func(es pythonpatterns.ArgSummary) pythonpatterns.ArgSummary {
		var total int
		nes := make(pythonpatterns.ArgSummary, 0, len(es))
		for _, e := range es {
			if e.Count < fps.MinTypeCount {
				// assumes that es are already sorted by count
				break
			}

			var totalStrs int
			for s, c := range e.SrcStrs {
				if c < fps.MinSourceCount {
					delete(e.SrcStrs, s)
				} else {
					totalStrs += c
				}
			}

			if len(e.SrcStrs) == 0 {
				continue
			}
			e.Count = totalStrs

			total += e.Count
			nes = append(nes, e)
		}

		if total < fps.MinArgCount {
			return nil
		}
		return nes
	}

	for i, arg := range cp.Positional {
		arg := filterArg(arg)
		if len(arg) == 0 {
			return pythonpatterns.Call{}
		}
		cp.Positional[i] = arg
	}

	for k, arg := range cp.Keyword {
		arg := filterArg(arg)
		if len(arg) == 0 {
			return pythonpatterns.Call{}
		}
		cp.Keyword[k] = arg
	}

	return cp
}
