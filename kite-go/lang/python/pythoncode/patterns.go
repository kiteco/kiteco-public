package pythoncode

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

// MethodPatternsFromCallSpecs generates a MethodPatterns for a given value
// using the CallSpecs provided. A list of string parameters describing the
// positional arguments can also be used to set positional argument names
// if necessary. In addition, an optional import graph ArgSpec can also be
// used to set the names and types of positional and keyword arguments. If
// both the list of string names and ArgSpec are passed to this function, the
// ArgSpec will overwrite the string names if possible.
func MethodPatternsFromCallSpecs(
	val pythontype.Value, specs []*CallSpec,
	params []string) *MethodPatterns {
	if pythonenv.Locator(val) == "" {
		return nil
	}

	var args []*argument
	kwargs := make(map[string]*argument)
	patternMap := make(map[string]*pattern)

	// For each instance of a call, we want to:
	// - Compute its "pattern" (number of positional arguments + sorted list of keywords)
	// - Accumuate statistics for each positional argument and keyword argument
	for _, spec := range specs {
		// Compute a signature hash (a hash that encapsulates our definition of a pattern)
		hash := spec.hash()
		pat, exists := patternMap[hash]
		if !exists {
			var kwargs []string
			for _, kw := range spec.Kwargs {
				kwargs = append(kwargs, kw.Key)
			}
			sort.Sort(sort.StringSlice(kwargs))
			pat = &pattern{
				args:   len(spec.Args),
				kwargs: kwargs,
				count:  0,
			}
			patternMap[hash] = pat
		}

		// Count the number of times this pattern occurs
		pat.count++

		count := func(stats *argument, arg *ArgSpec) {
			stats.count++
			if arg.ExprStr != "" {
				stats.exprStrs[arg.ExprStr]++
			}
			if arg.Type != "" {
				stats.varTypes[arg.Type]++
				if arg.Literal != "" {
					byType, ok := stats.literalsByType[arg.Type]
					if !ok {
						byType = make(map[string]int)
						stats.literalsByType[arg.Type] = byType
					}
					byType[arg.Literal]++
				}
			}
		}

		// For each positional argument, retrieve or construct associated
		// ArgStats struct, and accumulate variable names, types and literal
		// values we see.
		for pos, arg := range spec.Args {
			if pos > len(args)-1 {
				args = append(args, newArgument())
			}
			stats := args[pos]
			count(stats, arg)
		}

		// For each keyword argument, retrieve or construct associated
		// ArgStats struct, and accumulate variable names, types and literal
		// values we see.
		for _, arg := range spec.Kwargs {
			stats, exists := kwargs[arg.Key]
			if !exists {
				stats = newKeywordArgument(arg.Key)
				kwargs[arg.Key] = stats
			}
			count(stats, arg)
		}
	}

	// Sort patterns by how often they occur
	var total int
	var patterns []*pattern
	for _, pat := range patternMap {
		total += pat.count
		patterns = append(patterns, pat)
	}
	sort.Sort(sort.Reverse(patternByCount(patterns)))

	var argStats []*ArgStats
	for pos, arg := range args {
		stats := arg.stats()
		// Fill in the argument name if necessary from the function def parameters
		if !isDottedIdent(stats.Name) && pos < len(params) {
			stats.Name = params[pos]
		}
		argStats = append(argStats, stats)
	}

	kwargStats := make(map[string]*ArgStats)
	for k, arg := range kwargs {
		kwargStats[k] = arg.stats()
	}

	methods := &MethodPatterns{
		Method:      pythonenv.Locator(val),
		MethodCount: total,
		Args:        argStats,
		Kwargs:      kwargStats,
	}

	for _, pat := range patterns {
		methods.Patterns = append(methods.Patterns, &SignaturePattern{
			PatternCount: pat.count,
			Frequency:    float64(pat.count) / float64(total),
			Args:         pat.args,
			Kwargs:       pat.kwargs,
		})
	}

	return methods
}

// --

// pattern is used to count how many times a function was called in a specific
// manner. It corresponds to a unique hashed value of a CallSpec.
type pattern struct {
	args   int
	kwargs []string
	count  int
}

type patternByCount []*pattern

func (b patternByCount) Len() int           { return len(b) }
func (b patternByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b patternByCount) Less(i, j int) bool { return b[i].count < b[j].count }

// argument is used to count number of times a given argument was used in a
// function call. It also contains counters for the names, types and literals
// associated with the argument usages.
type argument struct {
	name    string
	keyword bool
	count   int
	// Maps argument expression strings to the number of times they occurred
	exprStrs map[string]int
	// Maps variable types to the number of times they occurred
	varTypes map[string]int
	// Maps variable types to a map of string literals and the number of times
	// each literal occurred
	literalsByType map[string]map[string]int
}

func newArgument() *argument {
	return &argument{
		exprStrs:       make(map[string]int),
		varTypes:       make(map[string]int),
		literalsByType: make(map[string]map[string]int),
	}
}

func newKeywordArgument(name string) *argument {
	return &argument{
		name:           name,
		keyword:        true,
		exprStrs:       make(map[string]int),
		varTypes:       make(map[string]int),
		literalsByType: make(map[string]map[string]int),
	}
}

// stats creates an ArgStats object from an argument counter.
func (arg *argument) stats() *ArgStats {
	literalsByType := make(map[string][]*StringCount)
	for k, v := range arg.literalsByType {
		literalsByType[k] = StringCountFromMap(v)
	}

	// Set the argument name if one doesn't exist
	name := arg.name
	if name == "" {
		var names []*StringCount
		for n, i := range arg.exprStrs {
			names = append(names, &StringCount{n, i})
		}
		sort.Sort(sort.Reverse(stringByCount(names)))
		if len(names) > 0 {
			name = names[0].Value
		}
	}

	return &ArgStats{
		Name:           name,
		Keyword:        arg.keyword,
		Count:          arg.count,
		ExprStrs:       StringCountFromMap(arg.exprStrs),
		Types:          StringCountFromMap(arg.varTypes),
		LiteralsByType: literalsByType,
	}
}

// -

var identPat = `[_a-zA-Z][_a-zA-Z0-9]*`
var dottedIdentPat = fmt.Sprintf(`%s(\.%s)*`, identPat, identPat)
var dottedIdentRE = regexp.MustCompile(fmt.Sprintf(`^%s$`, dottedIdentPat))

func isDottedIdent(exprStr string) bool {
	return dottedIdentRE.Match([]byte(exprStr))
}
