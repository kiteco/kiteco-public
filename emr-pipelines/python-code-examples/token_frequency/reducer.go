package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

type wordCount struct {
	word  string
	count int
}

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	var lastKey string
	var totalCount int

	for r.Next() {
		var wc int
		err := json.Unmarshal(r.Value(), &wc)
		if err != nil {
			log.Fatalln(err)
		}

		if r.Key() != lastKey && totalCount > 0 {
			buf, err := json.Marshal(totalCount)
			if err != nil {
				log.Fatal(err)
			}
			w.Emit(lastKey, buf)
			totalCount = 0
		}
		totalCount++
		lastKey = r.Key()
	}

	// emit the last key
	if totalCount > 0 {
		buf, err := json.Marshal(totalCount)
		if err != nil {
			log.Fatal(err)
		}
		w.Emit(lastKey, buf)
	}

	if err := r.Err(); err != nil {
		log.Fatal(err)
	}
}
