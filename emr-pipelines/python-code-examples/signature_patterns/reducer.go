package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"sort"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	var incs []*pythoncode.Incantation

	for r.Next() {
		var inc pythoncode.Incantation
		err := json.Unmarshal(r.Value(), &inc)
		if err != nil {
			log.Fatal(err)
		}

		if r.Key() != lastKey {
			if len(incs) > 0 {
				// Build pattern objects from incantations
				patterns := signaturePatterns(incs)
				buf, err := json.Marshal(patterns)
				if err != nil {
					log.Fatal(err)
				}
				err = w.Emit("signature", buf)
				if err != nil {
					log.Fatal(err)
				}
			}

			// Reset incantation slice
			incs = nil
		}

		// Collect incantations for each key
		incs = append(incs, &inc)
		lastKey = r.Key()
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}

	if len(incs) > 0 {
		// Build pattern objects from incantations
		patterns := signaturePatterns(incs)
		buf, err := json.Marshal(patterns)
		if err != nil {
			log.Fatal(err)
		}
		err = w.Emit("signature", buf)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// --

type argSpecByKey []*pythoncode.ArgSpec

func (b argSpecByKey) Len() int           { return len(b) }
func (b argSpecByKey) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b argSpecByKey) Less(i, j int) bool { return b[i].Key < b[j].Key }

func signatureHash(inc *pythoncode.Incantation) string {
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
	varNames       map[string]int
	varTypes       map[string]int
	literalsByType map[string]map[string]int
}

func newArgCounters() *argCounters {
	return &argCounters{
		varNames:       make(map[string]int),
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
		ExprStrs:       pythoncode.StringCountFromMap(counts.varNames),
		Types:          pythoncode.StringCountFromMap(counts.varTypes),
		LiteralsByType: literalsByType,
	}
}

func signaturePatterns(incantations []*pythoncode.Incantation) *pythoncode.MethodPatterns {
	identMap := make(map[string]int)
	patternMap := make(map[string]*pattern)

	var args []*argCounters
	kwargs := make(map[string]*argCounters)

	// For each incantation, we want to:
	// - Compute its "pattern" (number of positional arguments + sorted list of keywords)
	// - Accumuate statistics for each positional argument and keyword argument
	for _, inc := range incantations {
		// Keep track of the name used for this method
		identMap[inc.ExampleOf]++

		// Compute a signature hash (a hash that encapsulates our definition of a pattern)
		fp := signatureHash(inc)
		pat, ok := patternMap[fp]
		if !ok {
			var kwargs []string
			sort.Sort(argSpecByKey(inc.Kwargs))
			for _, kw := range inc.Kwargs {
				kwargs = append(kwargs, kw.Key)
			}
			pat = &pattern{len(inc.Args), kwargs, 0}
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
				stats.varNames[arg.ExprStr]++
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
				stats.varNames[arg.ExprStr]++
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

	// Compute the most common name used for this node
	var max int
	var commonIdent string
	for ident, count := range identMap {
		if count > max {
			count = max
			commonIdent = ident
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
		Method:      commonIdent,
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

	return methodPatterns
}
