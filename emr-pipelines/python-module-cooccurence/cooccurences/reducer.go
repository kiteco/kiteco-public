package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/emr-pipelines/python-module-cooccurence/internal/cooccurence"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[cooccurences-reducer] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Input: Cooccurence objects keyed by (top level) package/module name.
// Output: Map from Cooccuring (top level) packages/modules for a given (top level) package/module keyed by the (top level) package/module name.
func main() {
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	start := time.Now()
	var current string
	counts := make(map[string]int64)
	for in.Next() {
		if in.Key() != current {
			emit(out, current, counts)
			counts = make(map[string]int64)
			current = in.Key()
		}

		var cooccurs cooccurence.Cooccurence
		if err := json.Unmarshal(in.Value(), &cooccurs); err != nil {
			log.Fatalf("error unmarshalling co-occurences for package %s: %v\n", current, err)
		}

		for _, module := range cooccurs.Cooccuring {
			counts[module]++
		}
	}

	emit(out, current, counts)

	if err := in.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}

	log.Printf("Done! Took %v.", time.Since(start))
}

func emit(out *awsutil.EMRWriter, pkg string, counts map[string]int64) {
	buf, err := json.Marshal(counts)
	if err != nil {
		log.Fatalf("error marshaling json for package %s: %v\n", pkg, err)
	}

	if err := out.Emit(pkg, buf); err != nil {
		log.Fatalf("error emitting stats for package %s: %v\n", pkg, err)
	}
}
