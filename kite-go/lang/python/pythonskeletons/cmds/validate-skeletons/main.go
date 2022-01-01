package main

import (
	"log"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons/internal/skeleton"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// Validates a dataset of curated python skeletons
func main() {
	var args struct {
		Skeletons string `arg:"positional,required"`
	}
	arg.Parse(&args)
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatalf("error loading import graph `%s`: %v\n", pythonimports.DefaultImportGraph, err)
	}

	manager, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}

	// validate skeletons
	var index skeleton.Builder
	if err := serialization.Decode(args.Skeletons, &index); err != nil {
		log.Fatalf("error decoding skeletons `%s`: %v\n", args.Skeletons, err)
	}

	if err := index.Validate(); err != nil {
		log.Fatalf("error validating skeletons: %v\n", err)
	}

	// make sure we can UpdateGraph with no errors
	if err := pythonskeletons.UpdateGraph(graph); err != nil {
		log.Fatalf("error updating import graph: %v\n", err)
	}

	// update type tables so we can get debug messages
	client, err := typeinduction.LoadModel(manager, typeinduction.DefaultClientOptions)
	if err != nil {
		log.Fatalf("error loading type induction tables: %v\n", err)
	}
	pythonskeletons.UpdateReturnTypes(graph, client)
}
