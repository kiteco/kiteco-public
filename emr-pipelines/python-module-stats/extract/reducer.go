package main

import (
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[python-module-stats-extract-reducer] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Dedupe python files.
// Input: map from python any name to counts for the name, keyed by a hash of the source file.
// Output: map from python any name to counts for the name, keyed by a hash of the source file.
//         Stats from duplicate files are removed.
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	var lastKey string
	for in.Next() {
		// Only emit if the key has changed. This effectively dedupes
		// the data (keyed by code hash). Duplicate code can
		// occur due to forked repos, copied code, etc.
		if in.Key() != lastKey {
			if err := out.Emit(in.Key(), in.Value()); err != nil {
				log.Fatalf("error emitting: %v\n", err)
			}
			lastKey = in.Key()
		}
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading stdin: %v\n", err)
	}

	log.Printf("Done! Took %v\n", time.Since(start))
}
