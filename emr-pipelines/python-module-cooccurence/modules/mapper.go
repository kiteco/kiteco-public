package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/emr-pipelines/python-module-cooccurence/internal/cooccurence"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[modules-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Input: python source file.
// Output: list of (top level) pacakges/modules imported in the python file, keyed by a hash of the source code for the file.
func main() {
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	var errs int64
	start := time.Now()
	for in.Next() {
		modules, err := cooccurence.ExtractModules(in.Value())
		if err != nil {
			errs++
			continue
		}

		if len(modules) == 0 {
			continue
		}

		buf, err := json.Marshal(modules)
		if err != nil {
			log.Fatalf("error marshaling json for file %s: %v\n", in.Key(), err)
		}

		if err := out.Emit(hash(in.Value()).String(), buf); err != nil {
			log.Fatalf("error emitting json for file %s: %v \n", in.Key(), err)
		}
	}

	if err := in.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}

	log.Printf("Done! Took %v, %d files had errors.\n", time.Since(start), errs)
}

// hash gets a hash of the input code block.
func hash(src []byte) codeHash {
	var h codeHash
	spooky.Hash128(src, &h[0], &h[1])
	return h
}

// codeHash represents a 128-bit hash of a piece of code.
type codeHash [2]uint64

// String returns a base64-encoded string representation of the hash.
func (h codeHash) String() string {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, h[0])
	binary.Write(&buf, binary.LittleEndian, h[1])
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
