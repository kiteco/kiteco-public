package main

import (
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[extract-resolvable-snippets-reducer] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Dedupes python snippets based on a hash of the source file that generated the snippet.
// Input: Snippets keyed by a hash of the source file that generated the snippet.
// Output: Deduped snippets keyed by a hash of the source file that generated the snippet.
func main() {
	start := time.Now()
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	for r.Next() {
		key, value := r.Key(), r.Value()

		// Only emit if the key has changed. This effectively dedupes
		// the snippets (keyed by code hash). Duplicate code snippets can
		// occur due to forked repos, copied code, etc.
		if key != lastKey {
			if err := w.Emit(key, value); err != nil {
				log.Fatalf("error emitting for key %s: %v\n", key, err)
			}
			lastKey = key
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
	log.Printf("Done! Took %v\n", time.Since(start))
}
