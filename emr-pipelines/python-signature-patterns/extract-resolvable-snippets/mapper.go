package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/emr-pipelines/python-signature-patterns/internal/util"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[extract-resolvable-snippets-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// Transforms python source files into snippets containing specs for the resolvable calls that occured in the source file.
// Input: python source code files.
// Output: Snippets containing specs for the resolvable calls that occured in the source code, keyed by a hash of the python source code.
func main() {
	start := time.Now()
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatalf("error loading graph %s: %v\n", pythonimports.DefaultImportGraph, err)
	}

	anynames := pythonimports.ComputeAnyPaths(graph)
	if anynames == nil {
		log.Fatalln("error computing anynames")
	}

	typeInducer, err := typeinduction.LoadModel(graph, typeinduction.DefaultClientOptions)
	if err != nil {
		log.Fatalf("error loading type induction client %s: %v\n", typeinduction.DefaultRoot, err)
	}

	argSpecs, err := pythonimports.LoadArgSpecs(graph, pythonimports.DefaultImportGraphArgSpecs, pythonimports.DefaultTypeshedArgSpecs)
	if err != nil {
		log.Fatalf("error loading arg specs %s: %v", pythonimports.DefaultImportGraphArgSpecs, err)
	}

	params := util.Params{
		Graph:       graph,
		ArgSpecs:    argSpecs,
		TypeInducer: typeInducer,
		AnyNames:    anynames,
	}

	for r.Next() {
		snippet := util.Extract(r.Value(), params)
		if snippet == nil {
			continue
		}

		out, err := json.Marshal(snippet)
		if err != nil {
			log.Fatalf("error marshaling snippet for file %s: %v\n", r.Key(), err)
		}

		if err := w.Emit(snippet.Hash.String(), out); err != nil {
			log.Fatalf("error emitting for file %s: %v\n", r.Key(), err)
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
	log.Printf("Done! Took %v\n", time.Since(start))
}
