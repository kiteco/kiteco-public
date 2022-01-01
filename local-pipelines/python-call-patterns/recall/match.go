package main

import (
	"bytes"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/local-pipelines/python-call-patterns/internal/data"
)

const (
	noPatterns               = "no_patterns"
	havePatterns             = "have_patterns"
	matchedPattern           = "matched_pattern"
	noMatchedPattern         = "no_matched_pattern"
	ratioPosArgsTypeMatch    = "ratio_pos_args_type_match"
	ratioPosArgsLiteralMatch = "ratio_pos_args_literal_matched"
	ratioKWArgsTypeMatch     = "ratio_kw_args_type_match"
	ratioKWArgsLiteralMatch  = "ratio_kw_args_literal_match"
)

type strCounts map[string]float64

type distCounts struct {
	Dist         string
	Counts       strCounts
	PosTotal     int
	KeywordTotal int
}

func (*distCounts) SampleTag() {}
func (d *distCounts) Add(dd *distCounts) {
	d.PosTotal += dd.PosTotal
	d.KeywordTotal += dd.KeywordTotal

	for k, v := range dd.Counts {
		d.Counts[k] += v
	}
}

func (d *distCounts) results() []rundb.Result {
	ks := []string{
		noPatterns,
		havePatterns,
		matchedPattern,
		noMatchedPattern,
		ratioPosArgsTypeMatch,
		ratioKWArgsTypeMatch,
		ratioPosArgsLiteralMatch,
		ratioKWArgsLiteralMatch,
	}

	var buf bytes.Buffer
	tw := new(tabwriter.Writer)
	tw.Init(&buf, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "name\tvalue")
	for _, k := range ks {
		var denom float64
		var prefix string
		switch k {
		case noPatterns, havePatterns:
			denom = d.Counts[noPatterns] + d.Counts[havePatterns]
			prefix = "ratio_"
		case matchedPattern, noMatchedPattern:
			denom = d.Counts[matchedPattern] + d.Counts[noMatchedPattern]
			prefix = "ratio_"
		case ratioPosArgsLiteralMatch, ratioPosArgsTypeMatch:
			denom = float64(d.PosTotal)
			prefix = "avg_"
		case ratioKWArgsTypeMatch, ratioKWArgsLiteralMatch:
			denom = float64(d.KeywordTotal)
			prefix = "avg_"
		}

		num := d.Counts[k]
		fmt.Fprintf(tw, "%s\t%.f\n", "total_"+k, num)

		if denom == 0 {
			fmt.Fprintf(tw, "%s\tNA\n", prefix+k)
		} else {
			fmt.Fprintf(tw, "%s\t%.3f\n", prefix+k, num/denom)
		}
		fmt.Fprintf(tw, "--\t--\n")
	}

	fail(tw.Flush())

	return []rundb.Result{
		{
			Name:  fmt.Sprintf("%s: total calls", d.Dist),
			Value: d.Counts[noPatterns] + d.Counts[havePatterns],
		},
		{
			Name:  fmt.Sprintf("%s: total calls with atleast one positional", d.Dist),
			Value: d.PosTotal,
		},
		{
			Name:  fmt.Sprintf("%s: total calls with atleast one keyword", d.Dist),
			Value: d.KeywordTotal,
		},
		{
			Name:  fmt.Sprintf("%s: match values", d.Dist),
			Value: fmt.Sprintf("<pre>%s</pre>", buf.String()),
		},
	}
}

type byDist map[string]*distCounts

func (b byDist) results() []rundb.Result {
	var dists []string
	for d := range b {
		dists = append(dists, d)
	}

	sort.Strings(dists)

	total := &distCounts{
		Dist:   "ALL",
		Counts: make(strCounts),
	}

	var results []rundb.Result
	for _, d := range dists {
		if b[d].Counts[havePatterns]+b[d].Counts[noPatterns] < minCallsPerDist {
			// skip dists with only a few calls to reduce noise
			continue
		}
		total.Add(b[d])
		results = append(results, b[d].results()...)
	}

	return append(total.results(), results...)
}

func (byDist) SampleTag() {}
func (b byDist) Add(o sample.Addable) sample.Addable {
	for d, dc := range o.(byDist) {
		if _, ok := b[d]; !ok {
			b[d] = &distCounts{
				Dist:   d,
				Counts: make(strCounts),
			}
		}
		b[d].Add(dc)
	}
	return b
}

func match(rm pythonresource.Manager, patterns data.PatternsByHash, c call) *distCounts {
	dc := &distCounts{
		Dist:   c.Sym.Dist().String(),
		Counts: make(strCounts),
	}

	s := c.Sym.Canonical()
	pats, ok := patterns[s.Hash()]
	if !ok {
		dc.Counts[noPatterns] = 1
		return dc
	}
	dc.Counts[havePatterns] = 1

	var numPos int
	kws := make(map[string]*pythonast.Argument)
	for _, arg := range c.Call.Args {
		name, ok := arg.Name.(*pythonast.NameExpr)
		if !ok {
			numPos++
		} else {
			kws[name.Ident.Literal] = arg
		}
	}

	var pat pythonpatterns.Call
	for _, p := range pats.Patterns.Calls {
		if len(p.Positional) != numPos {
			continue
		}
		if !keysMatch(kws, p.Keyword) {
			continue
		}
		pat = p
		break
	}

	if pat.Count == 0 {
		dc.Counts[noMatchedPattern] = 1
		return dc
	}
	dc.Counts[matchedPattern] = 1

	matchArg := func(arg *pythonast.Argument, as pythonpatterns.ArgSummary) (float64, float64) {
		var tMatch, sMatch float64
		if matchType(rm, c.RAST.References[arg.Value], as) {
			tMatch++
		}
		if matchLiteral(c.Src[arg.Value.Begin():arg.Value.End()], as) {
			sMatch++
		}
		return tMatch, sMatch
	}

	if len(pat.Positional) > 0 {
		var tMatch, sMatch float64
		for i, pa := range pat.Positional {
			t, s := matchArg(c.Call.Args[i], pa)
			tMatch += t
			sMatch += s
		}

		dc.PosTotal = 1
		dc.Counts[ratioPosArgsTypeMatch] = tMatch / float64(len(pat.Positional))
		dc.Counts[ratioPosArgsLiteralMatch] = sMatch / float64(len(pat.Positional))
	}

	if len(pat.Keyword) > 0 {
		var tMatch, sMatch float64
		for k, pa := range pat.Keyword {
			t, s := matchArg(kws[k], pa)
			tMatch += t
			sMatch += s
		}

		dc.KeywordTotal = 1
		dc.Counts[ratioKWArgsTypeMatch] = tMatch / float64(len(pat.Keyword))
		dc.Counts[ratioKWArgsLiteralMatch] = sMatch / float64(len(pat.Keyword))
	}

	return dc
}

func matchLiteral(lit string, as pythonpatterns.ArgSummary) bool {
	for _, a := range as {
		for s := range a.SrcStrs {
			if s == lit {
				return true
			}
		}
	}
	return false
}

func matchType(rm pythonresource.Manager, val pythontype.Value, as pythonpatterns.ArgSummary) bool {
	syms := python.GetExternalSymbols(kitectx.Background(), rm, val)
	if len(syms) == 0 {
		fail(fmt.Errorf("got value that does not resolve to any symbols"))
	}

	matchSyms := func(patSyms []pythonpatterns.Symbol) bool {
		for _, s := range syms {
			s = s.Canonical()
			for _, ps := range patSyms {
				if s.Dist() != ps.Dist {
					continue
				}
				if s.PathHash() == ps.Path.Hash {
					return true
				}
			}
		}
		return false
	}

	for _, a := range as {
		if matchSyms(a.Syms) {
			return true
		}
	}

	return false
}

func keysMatch(obs map[string]*pythonast.Argument, kws map[string]pythonpatterns.ArgSummary) bool {
	if len(obs) != len(kws) {
		return false
	}
	for kw := range obs {
		if _, ok := kws[kw]; !ok {
			return false
		}
	}
	return true
}
