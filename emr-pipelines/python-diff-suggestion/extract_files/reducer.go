package main

import (
	"io"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRReader(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	for {
		key, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// Only emit if the key has changed. This effectively dedupes
		// the snippets (keyed by code hash). Duplicate code snippets can
		// occur due to forked repos, copied code, etc.
		if key != lastKey {
			err = w.Emit(key, value)
			if err != nil {
				log.Fatal(err)
			}
			lastKey = key
		}
	}
}
