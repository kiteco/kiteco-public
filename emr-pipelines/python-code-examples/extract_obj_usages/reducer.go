package main

import (
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	for r.Next() {
		// Only emit if the key has changed. This effectively dedupes
		// the obj incantations (keyed by code hash). Duplicates can
		// occur due to forked repos, copied code, etc.
		if r.Key() != lastKey {
			err := w.Emit(r.Key(), r.Value())
			if err != nil {
				log.Fatalln(err)
			}
			lastKey = r.Key()
		}
	}
	if err := r.Err(); err != nil {
		log.Fatalln(err)
	}
}
