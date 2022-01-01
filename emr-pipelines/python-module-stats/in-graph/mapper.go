package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[python-module-stats-in-graph-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Only emit counts for top-level modules that are in the resource manager
// Input: aggregated counts for a given top-level module, keyed by the module's name.
// Output: aggregated counts for a given top-level module only if those counts match a symbol in the resource manager.
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions.SymbolOnly())
	if err := <-errc; err != nil {
		log.Fatalf("error creating resource manager: %v", err)
	}

	for in.Next() {
		if dists := rm.DistsForPkg(in.Key()); len(dists) == 0 {
			continue
		}

		var counts symbolcounts.TopLevelCounts
		if err := json.Unmarshal(in.Value(), &counts); err != nil {
			log.Fatalf("error unmarshaling counts for top-level module %s: %v\n", in.Key(), err)
		}

		newCounts := symbolcounts.TopLevelCounts{
			TopLevel: in.Key(),
			Count:    counts.Count,
		}

		for _, symCounts := range counts.Symbols {
			if _, err := rm.PathSymbol(pythonimports.NewDottedPath(symCounts.Path)); err != nil {
				continue
			}
			newCounts.Symbols = append(newCounts.Symbols, symCounts)
		}

		buf, err := json.Marshal(newCounts)
		if err != nil {
			log.Fatalf("error marshalling new counts for top-level module %s: %v\n", in.Key(), err)
		}

		if err := out.Emit(in.Key(), buf); err != nil {
			log.Fatalf("error emitting new counts for top-level module %s: %v\n", in.Key(), err)
		}
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading stdin: %v\n", err)
	}

	log.Printf("Done! took %v\n", time.Since(start))
}
