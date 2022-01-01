package util

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"sort"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

type argSpecByKey []*pythoncode.ArgSpec

func (b argSpecByKey) Len() int           { return len(b) }
func (b argSpecByKey) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b argSpecByKey) Less(i, j int) bool { return b[i].Key < b[j].Key }

type pattern struct {
	args   int
	kwargs []string
	count  int
}

type patternByCount []*pattern

func (b patternByCount) Len() int           { return len(b) }
func (b patternByCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b patternByCount) Less(i, j int) bool { return b[i].count < b[j].count }

type argCounters struct {
	name           string
	keyword        bool
	count          int
	exprStrs       map[string]int
	varTypes       map[string]int
	literalsByType map[string]map[string]int
}

func newArgCounters() *argCounters {
	return &argCounters{
		exprStrs:       make(map[string]int),
		varTypes:       make(map[string]int),
		literalsByType: make(map[string]map[string]int),
	}
}

func countsToStats(counts *argCounters) *pythoncode.ArgStats {
	literalsByType := make(map[string][]*pythoncode.StringCount)
	for k, v := range counts.literalsByType {
		literalsByType[k] = pythoncode.StringCountFromMap(v)
	}
	return &pythoncode.ArgStats{
		Name:           counts.name,
		Keyword:        counts.keyword,
		Count:          counts.count,
		ExprStrs:       pythoncode.StringCountFromMap(counts.exprStrs),
		Types:          pythoncode.StringCountFromMap(counts.varTypes),
		LiteralsByType: literalsByType,
	}
}

// SignaturePatterns uses the provided incantations to calculate the `*pythoncoede.MethodPatterns`
// for the call, anyname is used to key the returned `*pythoncode.MethodPatterns`.
func SignaturePatterns(anyname string, incantations []*CallSpec) *pythoncode.MethodPatterns {
	patternMap := make(map[string]*pattern)

	var args []*argCounters
	kwargs := make(map[string]*argCounters)

	var argSpec *pythonimports.ArgSpec

	// For each incantation, we want to:
	// - Compute its "pattern" (number of positional arguments + sorted list of keywords)
	// - Accumuate statistics for each positional argument and keyword argument
	for _, inc := range incantations {
		argSpec = inc.NodeArgSpec
		// Compute a signature hash (a hash that encapsulates our definition of a pattern)
		fp := signatureHash(inc)
		pat, ok := patternMap[fp]
		if !ok {
			var kwargs []string
			sort.Sort(argSpecByKey(inc.Kwargs))
			for _, kw := range inc.Kwargs {
				kwargs = append(kwargs, kw.Key)
			}
			pat = &pattern{
				args:   len(inc.Args),
				kwargs: kwargs,
				count:  0,
			}
			patternMap[fp] = pat
		}

		// Count the number of times this pattern occurs
		pat.count++

		// For each positional argument, retrieve or construct associated
		// ArgStats struct, and accumulate variable names, types and literal
		// values we see.
		for pos, arg := range inc.Args {
			if pos > len(args)-1 {
				args = append(args, newArgCounters())
			}
			stats := args[pos]
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

		// For each keyword argument, retrieve or construct associated
		// ArgStats struct, and accumulate variable names, types and literal
		// values we see.
		for _, arg := range inc.Kwargs {
			stats, exists := kwargs[arg.Key]
			if !exists {
				stats = newArgCounters()
				stats.name = arg.Key
				stats.keyword = true
				kwargs[arg.Key] = stats
			}
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
	}

	// Sort paterns by how often they occur
	var total int
	var patterns []*pattern
	for _, pattern := range patternMap {
		total += pattern.count
		patterns = append(patterns, pattern)
	}
	sort.Sort(sort.Reverse(patternByCount(patterns)))

	var argStats []*pythoncode.ArgStats
	for _, arg := range args {
		argStats = append(argStats, countsToStats(arg))
	}

	kwargStats := make(map[string]*pythoncode.ArgStats)
	for k, arg := range kwargs {
		kwargStats[k] = countsToStats(arg)
	}

	methodPatterns := &pythoncode.MethodPatterns{
		Method:      anyname,
		MethodCount: total,
		Args:        argStats,
		Kwargs:      kwargStats,
	}

	// Construct a SignaturePattern struct for each pattern.
	for _, pattern := range patterns {
		methodPatterns.Patterns = append(methodPatterns.Patterns, &pythoncode.SignaturePattern{
			PatternCount: pattern.count,
			Frequency:    float64(pattern.count) / float64(total),
			Args:         pattern.args,
			Kwargs:       pattern.kwargs,
		})
	}

	// Set argument names/types from the arg spec
	if argSpec != nil {
		for idx, arg := range argSpec.Args {
			// Update names for positional arguments
			if idx < len(methodPatterns.Args) && arg.DefaultType == "" && arg.Name != "self" {
				methodPatterns.Args[idx].Name = arg.Name
			}

			if arg.DefaultType != "" {
				if kwarg, exists := methodPatterns.Kwargs[arg.Name]; exists {
					// Update types for keyword arguments based on their default. Note that "None" type
					// usually does not mean that the type is None - just a placeholder
					kwarg.Type = arg.DefaultType
				}
			}
		}
	}

	return methodPatterns
}

// --

func signatureHash(inc *CallSpec) string {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.LittleEndian, int32(len(inc.Args)))
	sort.Sort(argSpecByKey(inc.Kwargs))
	for _, arg := range inc.Kwargs {
		buf.Write([]byte(arg.Key))
	}

	var fp [2]uint64
	spooky.Hash128(buf.Bytes(), &fp[0], &fp[1])
	buf.Reset()

	binary.Write(buf, binary.LittleEndian, fp[0])
	binary.Write(buf, binary.LittleEndian, fp[1])
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
