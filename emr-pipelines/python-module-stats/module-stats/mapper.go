package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/emr-pipelines/python-module-stats/internal/stats"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[python-module-stats-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Separate per file module stats into stats for a top-level module.
// Input: map from a symbol's path to its counts.
// Output: map of top level module's name to its symbolcounts.TopLevelCounts.
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	for in.Next() {
		var counts stats.PathCounts
		if err := json.Unmarshal(in.Value(), &counts); err != nil {
			log.Fatalf("error unmarshaling stats: %v\n", err)
		}

		// aggregate stats by top-level modules
		tlCounts := make(map[string]*symbolcounts.TopLevelCounts)
		for name, count := range counts {
			tlModule := name
			if pos := strings.Index(name, "."); pos > -1 {
				tlModule = name[:pos]
			}

			modCounts := tlCounts[tlModule]
			if modCounts == nil {
				modCounts = &symbolcounts.TopLevelCounts{
					TopLevel: tlModule,
				}
				tlCounts[tlModule] = modCounts
			}

			modCounts.Count = modCounts.Count.Add(*count)

			if tlModule == name {
				continue
			}

			modCounts.Symbols = append(modCounts.Symbols, &symbolcounts.SymbolCounts{
				Path:  name,
				Count: *count,
			})
		}

		for tlModule, modCounts := range tlCounts {
			buf, err := json.Marshal(modCounts)
			if err != nil {
				log.Fatalf("error marshaling counts for top-level module `%s`: %v\n", tlModule, err)
			}

			if err := out.Emit(tlModule, buf); err != nil {
				log.Fatalf("error emiting counts for top-leve module `%s`: %v\n", tlModule, err)
			}
		}
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading stdin: %v\n", err)
	}
	log.Printf("Done! took %v.\n", time.Since(start))
}
