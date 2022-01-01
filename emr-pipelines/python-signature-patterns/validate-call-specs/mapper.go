package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/emr-pipelines/python-signature-patterns/internal/util"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[validate-call-specs-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Validates the call specs contained in a snippet and splits them into individual specs and emits them,
// the emitted calls specs also have their `Code` field cleared to reduce their size.
// Input: Snippets containing specs for resolvable calls that appeared in python source file.
// Output: Valid Call specs keyed by an AnyName for the import graph node associated with the function,
//         the emitted Call specs also have their `Code` field cleared to reduce their size.
func main() {
	start := time.Now()
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	for r.Next() {
		var snippet util.Snippet
		if err := json.Unmarshal(r.Value(), &snippet); err != nil {
			log.Fatalln("error unmarshalling snippet:", err)
		}

		// Emit each incantation, keyed by the AnyName of each util.CallSpec.
		for _, inc := range snippet.Incantations {
			proccessIncantation(inc, w)
		}

		// Emit each incantation, keyed by the AnyName of each util.CallSpec.
		for _, inc := range snippet.Decorators {
			proccessIncantation(inc, w)
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
	log.Printf("Done! Took %v\n", time.Since(start))
}

func proccessIncantation(inc *util.CallSpec, w *awsutil.EMRWriter) {
	if !inc.Valid() {
		return
	}

	// clear the code field
	inc.Code = ""

	buf, err := json.Marshal(inc)
	if err != nil {
		log.Fatalln("error marshaling CallSpec:", err)
		return
	}

	if err := w.Emit(inc.AnyName.String(), buf); err != nil {
		log.Fatalln("error emitting CallSpec:", err)
	}
}
