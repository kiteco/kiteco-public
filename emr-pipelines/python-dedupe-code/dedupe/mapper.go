package main

import (
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[python-module-index-dedupe-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Extracts counts of python any names from python source files.
// Input:
//   Key: name of source file
//   Value: contents of source file (BYTES)
// Output:
//   Key: hash of source code contents
//   Value: contents of source file (BYTES)
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	for in.Next() {
		hsh := pythoncode.CodeHash(in.Value())

		if err := out.Emit(hsh, in.Value()); err != nil {
			log.Fatalf("error emitting source and name for %s: %v\n", in.Key(), err)
		}
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading stdin: %v\n", err)
	}
	log.Printf("Done! Took %v\n", time.Since(start))
}
