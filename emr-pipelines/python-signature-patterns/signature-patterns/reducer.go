package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/emr-pipelines/python-signature-patterns/internal/util"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[signature-patterns-reducer] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Aggregates the data for all the call specs for a given function into a *pythoncode.MethodPatterns object
// Input: all call specs for a specific function, keyed by an anyname for the import graph node associated with the function
// Output: pythocode.MethodPatterns representing the aggregated usage information for a specific funciton.
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	var anyname string
	var specs []*util.CallSpec
	for in.Next() {
		if anyname != in.Key() {
			if len(specs) > 0 {
				patterns := util.SignaturePatterns(anyname, specs)
				emitPatterns(out, patterns, anyname)
			}
			specs = nil
			anyname = in.Key()
		}

		var spec util.CallSpec
		if err := json.Unmarshal(in.Value(), &spec); err != nil {
			log.Fatalf("error unmarshaling spec for %s: %v\n", in.Key(), err)
		}
		specs = append(specs, &spec)
	}

	if len(specs) > 0 {
		patterns := util.SignaturePatterns(anyname, specs)
		emitPatterns(out, patterns, anyname)
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading stdin: %v\n", err)
	}

	log.Printf("Done! Took %v\n", time.Since(start))
}

func emitPatterns(out *awsutil.EMRWriter, patterns *pythoncode.MethodPatterns, anyname string) {
	if patterns == nil {
		return
	}

	buf, err := json.Marshal(patterns)
	if err != nil {
		log.Fatalf("error marshaling pythoncode.MethodPatterns for %s: %v\n", anyname, err)
	}

	if err := out.Emit(anyname, buf); err != nil {
		log.Fatalf("error emitting pythoncode.MethodPatterns for %s: %v\n", anyname, err)
	}
}
