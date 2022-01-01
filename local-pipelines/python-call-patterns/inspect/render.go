package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"
	"github.com/kiteco/kiteco/local-pipelines/python-call-patterns/internal/data"
)

type strCount struct {
	Str   string `json:"str"`
	Count int    `json:"count"`
}

func renderCalls(calls data.Calls, total int, err error) string {
	if err != nil {
		return err.Error()
	}

	srcStr := func(es pythonpatterns.ExprSummary) string {
		// all call expr summaries have a single entry
		for s := range es.SrcStrs {
			return s
		}
		return ""
	}

	var cs []string
	for _, c := range calls {
		var srcStrs []string
		var symStrs []string
		for _, arg := range c.Positional {
			srcStrs = append(srcStrs, srcStr(arg))
			symStrs = append(symStrs, symsStr(arg.Syms))
		}

		var kws []string
		for k := range c.Keyword {
			kws = append(kws, k)
		}

		sort.Strings(kws)
		for _, k := range kws {
			arg := c.Keyword[k]
			srcStrs = append(srcStrs, srcStr(arg))
			symStrs = append(symStrs, symsStr(arg.Syms))
		}

		cs = append(cs, "(")
		cs = append(cs, fmt.Sprintf("  %s", strings.Join(srcStrs, " , ")))
		cs = append(cs, fmt.Sprintf("  %s", strings.Join(symStrs, " , ")))
		cs = append(cs, ")")
	}

	cs = append([]string{
		fmt.Sprintf("total calls %d", total),
	}, cs...)
	return strings.Join(cs, "\n")
}

type renderedPattern struct {
	Display string     `json:"display"`
	Hashes  []strCount `json:"hashes"`
}

func renderPatterns(patterns []pythonpatterns.Call, err error) []renderedPattern {
	if err != nil {
		return []renderedPattern{{
			Display: err.Error(),
			Hashes:  []strCount{},
		}}
	}

	exprSummaryStr := func(es pythonpatterns.ExprSummary) string {
		var scs []strCount
		for s, c := range es.SrcStrs {
			scs = append(scs, strCount{
				Str:   s,
				Count: c,
			})
		}

		sort.Slice(scs, func(i, j int) bool {
			return scs[i].Count > scs[j].Count
		})

		if len(scs) > 10 {
			scs = scs[:10]
		}

		var parts []string
		for _, sc := range scs {
			if strings.Contains(sc.Str, "\n") {
				sc.Str = "MULTILINE"
			}
			parts = append(parts, fmt.Sprintf("'%s' (%d)", sc.Str, sc.Count))
		}

		return fmt.Sprintf("%s: %s",
			symsStr(es.Syms),
			strings.Join(parts, " | "),
		)
	}

	argStr := func(as pythonpatterns.ArgSummary) []string {
		if len(as) > 5 {
			as = as[:5]
		}

		var parts []string
		for _, es := range as {
			parts = append(parts,
				fmt.Sprintf("%s%s", strings.Repeat(" ", 4), exprSummaryStr(es)),
			)
		}
		return parts
	}

	patternStr := func(pat pythonpatterns.Call) []string {
		var args []string
		for _, as := range pat.Positional {
			args = append(args, "  {")
			args = append(args, argStr(as)...)
			args = append(args, "  }")
		}

		var kws []string
		for kw := range pat.Keyword {
			kws = append(kws, kw)
		}
		sort.Strings(kws)

		for _, kw := range kws {
			args = append(args, fmt.Sprintf("  %s={", kw))
			args = append(args, argStr(pat.Keyword[kw])...)
			args = append(args, "  }")
		}
		return args
	}

	var rendered []renderedPattern
	for _, pat := range patterns {
		parts := []string{
			fmt.Sprintf("Count: %d, NumPos: %d (", pat.Count, len(pat.Positional)),
		}
		parts = append(parts, patternStr(pat)...)
		parts = append(parts, ")")

		var hashes []strCount
		for h, c := range pat.Hashes {
			hashes = append(hashes, strCount{
				Str:   h,
				Count: c,
			})
		}

		sort.Slice(hashes, func(i, j int) bool {
			if hashes[i].Count == hashes[j].Count {
				return hashes[i].Str < hashes[j].Str
			}
			return hashes[i].Count > hashes[j].Count
		})

		rendered = append(rendered, renderedPattern{
			Display: strings.Join(parts, "\n"),
			Hashes:  hashes,
		})
	}

	return rendered
}

func renderPopularSignatures(sigs []*editorapi.Signature) string {
	if len(sigs) == 0 {
		return "no signatures found"
	}

	typeSummary := func(t *editorapi.ParameterTypeExample) string {
		var parts []string
		for _, e := range t.Examples {
			parts = append(parts, fmt.Sprintf("'%s'", e))
		}
		return fmt.Sprintf("%s: %s", t.ID.LanguageSpecific(), strings.Join(parts, " | "))
	}

	argStr := func(pe *editorapi.ParameterExample) []string {
		space := strings.Repeat(" ", 4)

		name := pe.Name
		if pe.Name == "" {
			name = "NO_NAME"
		}

		parts := []string{fmt.Sprintf("%s%s", space, name)}
		for _, t := range pe.Types {
			parts = append(parts,
				fmt.Sprintf("%s%s", space, typeSummary(t)),
			)
		}
		return parts
	}

	sigStr := func(sig *editorapi.Signature) []string {
		var args []string
		for _, arg := range sig.Args {
			args = append(args, "  {")
			args = append(args, argStr(arg)...)
			args = append(args, "  }")
		}

		if sig.LanguageDetails.Python != nil {
			for _, arg := range sig.LanguageDetails.Python.Kwargs {
				args = append(args, "  KW{")
				args = append(args, argStr(arg)...)
				args = append(args, "  }")
			}
		}
		return args
	}

	var parts []string
	for _, sig := range sigs {
		parts = append(parts, "(")
		parts = append(parts, sigStr(sig)...)
		parts = append(parts, ")")
	}
	return strings.Join(parts, "\n")
}

func renderArgspec(as *pythonimports.ArgSpec) string {
	if as == nil {
		return "NIL"
	}

	var args []string
	var keywordOnly []string
	for _, arg := range as.Args {
		as := fmt.Sprintf("  %s DefaultType: '%s' DefaultValue: '%s' KeywordOnly: %v, Types: %s",
			arg.Name, arg.DefaultType, arg.DefaultValue, arg.KeywordOnly, strings.Join(arg.Types, " | "),
		)
		if arg.KeywordOnly {
			keywordOnly = append(keywordOnly, as)
		} else {
			args = append(args, as)
		}
	}

	if as.Vararg != "" {
		args = append(args, "  *"+as.Vararg)
	}

	if as.Kwarg != "" {
		args = append(args, "  **"+as.Kwarg)
	}

	args = append([]string{"("}, args...)
	args = append(args, keywordOnly...)
	args = append(args, ")")

	return strings.Join(args, "\n")
}

func symsStr(syms []pythonpatterns.Symbol) string {
	var parts []string
	for _, s := range syms {
		parts = append(parts, s.String())
	}

	if len(parts) == 0 {
		return "UNKNOWN"
	}
	return strings.Join(parts, " | ")
}
