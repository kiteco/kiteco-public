package main

import (
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[python-module-index-reducer-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Dedupe files by content
// Input
//   Key: hash of source code contents
//   Value: source code contents (BYTES)
// Output
//   Key: hash of source code contents
//   Value: source code contents (BYTES)
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
