package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// This binary filters a file containing dynamicanalysis.Usage objects and outputs only those that are
// functions or types and which are present in the python import graph.

const defaultUsages = "s3://kite-emr/users/tarak/python-code-examples/2015-10-21_13-13-06-PM/merge_group_obj_usages/output/part-00000"

func main() {
	var importgraph, usages, output string
	flag.StringVar(&importgraph, "importgraph", pythonimports.DefaultImportGraph, "path to import graph")
	flag.StringVar(&usages, "usages", defaultUsages, "path to usages")
	flag.StringVar(&output, "output", "", "output path")
	flag.Parse()

	// Open the usages
	rr, err := fileutil.NewCachedReader(usages)
	if err != nil {
		log.Fatal(err)
	}
	defer rr.Close()

	// Open the writer
	ww, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer ww.Close()
	w := awsutil.NewEMRWriter(ww)
	defer w.Close()

	// Load the import graph
	log.Println("Loading import graph...")
	graph, err := pythonimports.NewGraph(importgraph)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Enumerating usages...")
	var numRecords, numOtherTypes, numUnknown, numFuncs, numTypes int
	var prevPkg string
	var prevPkgKnown bool

	r := awsutil.NewEMRIterator(rr)
	for r.Next() {
		numRecords++
		if numRecords%100000 == 0 {
			log.Printf("Processed %d records (%d funcs, %d types accepted so far)",
				numRecords, numFuncs, numTypes)
		}

		pkg := r.Key()
		if pos := strings.Index(pkg, "."); pos != -1 {
			pkg = pkg[:pos]
		}

		if pkg == prevPkg && !prevPkgKnown {
			continue
		}

		prevPkg = pkg
		_, err := graph.Find(pkg)
		prevPkgKnown = err == nil

		// Look up node for this item
		node, err := graph.Find(r.Key())
		if err != nil {
			numUnknown++
			continue
		}
		if node.Classification == pythonimports.Function {
			numFuncs++
		} else if node.Classification == pythonimports.Type {
			numTypes++
		} else {
			numOtherTypes++
			continue
		}

		w.Emit(r.Key(), r.Value())
	}
	if r.Err() != nil {
		log.Fatalln(r.Err())
	}

	log.Printf("Of %d input records:\n", numRecords)
	log.Printf("  Ignored %d unknown names", numUnknown)
	log.Printf("  Ignored %d that were other types", numOtherTypes)
	log.Printf("  Wrote %d functions and %d types to %s", numFuncs, numTypes, output)
}
