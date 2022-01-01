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
	"github.com/kiteco/kiteco/emr-pipelines/python-module-stats/internal/stats"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[python-module-stats-extract-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Extracts counts of python any names from python source files.
// Input: python source files
// Output: map from python any name to counts for the name, keyed by a hash of the source file.
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions.SymbolOnly())
	if err := <-errc; err != nil {
		log.Fatalf("error creating resource manager: %v", err)
	}

	client, err := typeinduction.LoadModel(rm, typeinduction.DefaultClientOptions)
	if err != nil {
		log.Fatalf("error creating typeinduction client: %v\n", err)
	}

	params := stats.Params{
		Manager: rm,
		Client:  client,
	}

	var skipped int64
	for in.Next() {
		stats, err := stats.Extract(params, in.Value())
		if err != nil {
			skipped++
			continue
		}

		buf, err := json.Marshal(stats)
		if err != nil {
			log.Fatalf("error marshaling stats for file `%s`: %v\n", in.Key(), err)
		}

		key := hash(in.Value())
		if err := out.Emit(key.String(), buf); err != nil {
			log.Fatalf("error emitting stats for file `%s`: %v\n", in.Key(), err)
		}
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading stdin: %v\n", err)
	}
	log.Println("Done! Took", time.Since(start), "Skipped", skipped)
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
