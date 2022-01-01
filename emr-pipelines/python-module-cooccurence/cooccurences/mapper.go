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
	logPrefix = "[cooccurences-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Input: List of python (top level) packages/modules imported in a python file.
// Output: One Cooccurence object per (top level) packages/modules, keyed by the (top level) package/module name.
//         A Cooccurence contains a (top level) package/module name along with a slice of of the (top level) packages/modules that occured in the same file.
func main() {
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	start := time.Now()
	for in.Next() {
		var modules []string
		if err := json.Unmarshal(in.Value(), &modules); err != nil {
			log.Fatalf("error unmarshalling json: %v\n", err)
		}

		for _, cooccur := range cooccurence.Cooccurences(modules) {
			buf, err := json.Marshal(cooccur)
			if err != nil {
				log.Fatalf("error marshalling json: %v\n", err)
			}

			if err := out.Emit(cooccur.Module, buf); err != nil {
				log.Fatalf("error emitting: %v\n", err)
			}
		}
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading from stdin: %v\n", err)
	}

	log.Printf("Done! Took %v\n", time.Since(start))
}
