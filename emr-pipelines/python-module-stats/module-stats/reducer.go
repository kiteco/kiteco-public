package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[python-module-stats-reducer] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Aggregate stats for a given top-level module into a single `symbolcounts.TopLevelCounts` object.
// Input: map from top-level module name to counts for symbols in that module.
// Output: Aggregated `symbolcounts.TopLevelCounts` for a top-level module keyed by the module's name.
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	var current string
	symbols := make(map[string]*symbolcounts.SymbolCounts)
	var countsForTopLevel symbolcounts.Counts
	for in.Next() {
		if in.Key() != current {
			if !countsForTopLevel.Empty() {
				emitSymbolCounts(current, symbols, countsForTopLevel, out)
			}
			current = in.Key()
			symbols = make(map[string]*symbolcounts.SymbolCounts)
			countsForTopLevel = symbolcounts.NewCounts()
		}

		var tlCounts symbolcounts.TopLevelCounts
		if err := json.Unmarshal(in.Value(), &tlCounts); err != nil {
			log.Fatalf("error unmarshaling counts for top-level module `%s`: %v\n", in.Key(), err)
		}

		countsForTopLevel = countsForTopLevel.Add(tlCounts.Count)
		for _, sym := range tlCounts.Symbols {
			if symbols[sym.Path] == nil {
				symbols[sym.Path] = sym
			} else {
				symbols[sym.Path].Count = symbols[sym.Path].Count.Add(sym.Count)
			}
		}
	}

	if !countsForTopLevel.Empty() {
		emitSymbolCounts(current, symbols, countsForTopLevel, out)
	}

	log.Println("Done! Took", time.Since(start))
}

type byCount []*symbolcounts.SymbolCounts

func (b byCount) Len() int           { return len(b) }
func (b byCount) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byCount) Less(i, j int) bool { return b[i].Count.Sum() < b[j].Count.Sum() }

func emitSymbolCounts(current string, symbols map[string]*symbolcounts.SymbolCounts, countsForTopLevel symbolcounts.Counts, out *awsutil.EMRWriter) {
	counts := symbolcounts.TopLevelCounts{
		TopLevel: current,
		Count:    countsForTopLevel,
	}
	for _, sym := range symbols {
		counts.Symbols = append(counts.Symbols, sym)
	}

	sort.Sort(sort.Reverse(byCount(counts.Symbols)))

	buf, err := json.Marshal(counts)
	if err != nil {
		log.Fatalf("error marshalling counts for `%s`: %v\n", current, err)
	}

	if err := out.Emit(current, buf); err != nil {
		log.Fatalf("error emitting counts: %v\n", err)
	}
}
