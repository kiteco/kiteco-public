package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpatterns"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/sigstats"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/popularsignatures"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

func transformPatterns(rm pythonresource.Manager, patterns *symPatterns) []*editorapi.Signature {
	var out []*editorapi.Signature

	argSpec := rm.ArgSpec(patterns.Sym)
	sort.Slice(patterns.Patterns, func(i, j int) bool {
		return patterns.Patterns[i].Count > patterns.Patterns[j].Count
	})
	for _, pat := range patterns.Patterns {
		if pat.Frequency < minUsage || len(out) >= maxSignatures {
			break
		}

		if !filterInvalidPattern(pat, argSpec) {
			continue
		}

		outPat := transformPattern(rm, pat)

		if outPat != nil {
			if hasRareVarargName(pat, outPat, len(out)) {
				continue
			}
			if filterLowInformationPattern(outPat, argSpec, len(patterns.Patterns)) {
				out = append(out, outPat)
			}
		} else {
			fmt.Printf("unable to transform pattern %v for %v\n", pat, patterns.Sym)
		}
	}
	return out
}

func hasRareVarargName(p pattern, signature *editorapi.Signature, patternCount int) bool {
	if patternCount == 0 {
		// We keep at least 1 pattern for any method
		return false
	}
	for i, arg := range p.Positional {
		if arg.Name == "" {
			if arg.SrcStrs[signature.Args[i].Name]*25 < p.Count {
				// We test if the frequency of this name for this argument is at least 4% of the global usage count
				// of this arg

				return true
			}
		}
	}
	return false
}

func filterInvalidPattern(p pattern, argSpec *pythonimports.ArgSpec) bool {
	if argSpec == nil {
		return true
	}

	if hasIllegalArgumentCount(p, argSpec) {
		return false
	}

	if !areArgumentsNameUnique(p) {
		return false
	}

	return true
}

func filterLowInformationPattern(p *editorapi.Signature, argSpec *pythonimports.ArgSpec, patCount int) bool {
	if argSpec == nil {
		// We need the argspec to filter, so we keep every pattern we don't have the argspec for
		return true
	}
	if patCount > 1 {
		// We don't filter if there's more than 1 pattern
		return true
	}

	var mandatoryArgs, positional, keywordsArgs int

	for _, arg := range argSpec.Args {
		if arg.KeywordOnly {
			keywordsArgs++
		} else {
			positional++
			if arg.DefaultValue == "" {
				mandatoryArgs++
			}
		}
	}

	if positional+keywordsArgs > 1 {
		// We only filter function that have 0 or 1 arg
		return true
	}

	noKwArgs := p.LanguageDetails.Python == nil || len(p.LanguageDetails.Python.Kwargs) == 0

	if len(p.Args) == mandatoryArgs && noKwArgs {
		// Only mandatory args
		return false
	}

	if len(p.Args) == positional && noKwArgs {
		// All positionals
		return false
	}

	if len(p.Args) == positional && p.LanguageDetails.Python == nil && len(p.LanguageDetails.Python.Kwargs) == keywordsArgs {
		// All args
		return false
	}
	return true
}

func hasIllegalArgumentCount(pattern pattern, spec *pythonimports.ArgSpec) bool {
	if spec == nil {
		return false
	}
	return countArguments(pattern) < minArgCount(spec)
}

func countArguments(p pattern) int {
	return len(p.Positional) + len(p.Keyword)
}

func minArgCount(spec *pythonimports.ArgSpec) int {
	if spec == nil {
		return 0
	}
	var c int
	for _, arg := range spec.Args {
		if arg.DefaultValue == "" && arg.Name != "self" && arg.Name != "cls" {
			c++
		}
	}
	return c

}

// determine how often the most common source string for
// each argument is repeated
func areArgumentsNameUnique(pattern pattern) bool {
	var maxKeys []string
	pattern.ForArgs(func(arg *argument) {
		if arg.Name != "" {
			maxKeys = append(maxKeys, arg.Name)
		} else {
			maxKeys = append(maxKeys, maxKey(arg.SrcStrs))
		}
	})

	for i := 0; i < len(maxKeys); i++ {

		for j := 0; j < len(maxKeys); j++ {
			if i != j && maxKeys[i] == maxKeys[j] {
				return false
			}
		}
	}
	return true
}

func maxKey(sc pythonpatterns.StrCount) string {
	var count int
	var key string
	for k, c := range sc {
		if c > count {
			count = c
			key = k
		} else if c == count && k < key {
			count = c
			key = k
		}
	}
	return key
}

func getTotalCount(patterns *symPatterns) int {
	var result int
	for _, p := range patterns.Patterns {
		result += p.Count
	}
	return result
}

func makeStats(patterns *symPatterns) sigstats.Entity {
	positional := make([]sigstats.ArgStat, 0, len(patterns.Positional))
	argsByName := make(map[string]sigstats.ArgStat, len(patterns.Keyword))
	totalCount := getTotalCount(patterns)

	for n, arg := range patterns.Keyword {
		argsByName[n] = sigstats.ArgStat{
			Name:  n,
			Count: arg.Count,
			Types: mostFrequentTypes(arg.Types),
		}
	}

	for _, arg := range patterns.Positional {
		positional = append(positional, sigstats.ArgStat{
			Name:  arg.Name,
			Count: arg.Count,
			Types: mostFrequentTypes(arg.Types),
		})
	}

	return sigstats.Entity{
		ArgsByName: argsByName,
		Positional: positional,
		Count:      totalCount,
	}
}

func mostFrequentTypes(counts symCounts) map[pythonimports.Hash]sigstats.TypeInfo {
	type counter struct {
		hash   pythonimports.Hash
		count  int
		symbol pythonresource.Symbol
	}
	s := make([]counter, 0, len(counts))
	for h, d := range counts {
		s = append(s, counter{h, d.Count, d.Sym})
	}
	sort.Slice(s, func(i, j int) bool {
		if s[i].count == s[j].count {
			return s[i].symbol.Less(s[j].symbol)
		}
		return s[i].count > s[j].count
	})
	result := make(map[pythonimports.Hash]sigstats.TypeInfo)
	for i, c := range s {
		if i >= 5 {
			break
		}
		result[c.hash] = sigstats.TypeInfo{
			Path:  c.symbol.PathString(),
			Dist:  c.symbol.Distribution(),
			Count: c.count,
		}
	}
	return result
}

func buildResources() {
	bOpts := builder.DefaultOptions
	bOpts.ManifestPath = manifestOutputPath
	bOpts.ResourceRoot = newResourcesRoot
	b := builder.New(bOpts)

	// Load the symbol graph
	opts := pythonresource.DefaultOptions
	if manifestPath != "" {
		mF, err := os.Open(manifestPath)
		maybeQuit(err)
		opts.Manifest, err = manifest.New(mF)
		maybeQuit(err)
		mF.Close()
	}
	rm, errc := pythonresource.NewManager(opts)
	maybeQuit(<-errc)

	// Load the call patterns
	patterns := loadPatterns(rm, callPatternsDataset, true)

	// Filter call patterns and shard
	patShards := make(map[keytypes.Distribution]popularsignatures.Entities)
	statsShards := make(map[keytypes.Distribution]sigstats.Entities)

	for _, sp := range patterns {
		dist := sp.Sym.Dist()
		pathHash := sp.Sym.PathHash()

		shard := patShards[dist]
		if shard == nil {
			shard = make(popularsignatures.Entities)
			patShards[dist] = shard
		}

		sigs := transformPatterns(rm, sp)
		if len(sigs) == 0 {
			continue
		}
		shard[sp.Sym.PathHash()] = popularsignatures.CastEntity(sigs)

		// shard statistics
		statsShard := statsShards[sp.Sym.Dist()]
		if statsShard == nil {
			statsShard = make(sigstats.Entities)
			statsShards[sp.Sym.Dist()] = statsShard
		}
		statsShard[pathHash] = makeStats(sp)
	}

	for dist, rs := range patShards {
		maybeQuit(b.PutResource(dist, rs))
	}

	for dist, rs := range statsShards {
		maybeQuit(b.PutResource(dist, rs))
	}
	maybeQuit(b.Commit())
}
