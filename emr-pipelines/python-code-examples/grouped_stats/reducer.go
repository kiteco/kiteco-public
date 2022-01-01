package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

// usage represents a package that was used in a particular aggregation pool,
// which could be a file, directory, or repository.
type usage struct {
	Identifier string
	Group      string
	Aggregate  string
}

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	var counts map[string]int
	var seen map[usage]struct{}

	for r.Next() {
		var p usage
		err := json.Unmarshal(r.Value(), &p)
		if err != nil {
			log.Fatalln(err)
		}

		if r.Key() != lastKey {
			if len(counts) > 0 {
				// Key has changed, so lets summarize the counts for the last
				// key. Remember, the key is the package name.
				emitGroupedStats(lastKey, counts, w)
			}

			counts = make(map[string]int)
			seen = make(map[usage]struct{})
		}

		// Each pool only counts once towards usage
		if _, duplicate := seen[p]; !duplicate {
			counts[p.Aggregate]++
			seen[p] = struct{}{}
		}
		lastKey = r.Key()
	}

	// emit the last key
	if len(counts) > 0 {
		emitGroupedStats(lastKey, counts, w)
	}

	if err := r.Err(); err != nil {
		log.Fatal(err)
	}
}

func emitGroupedStats(key string, counts map[string]int, w *awsutil.EMRWriter) {
	parts := strings.Split(key, ".")
	groupedCounts := &pythoncode.GroupedStats{
		Package:    parts[0],
		Identifier: key,
		Counts:     counts,
	}

	buf, err := json.Marshal(groupedCounts)
	if err != nil {
		log.Fatalln(err)
	}

	err = w.Emit(key, buf)
	if err != nil {
		log.Fatalln(err)
	}
}
