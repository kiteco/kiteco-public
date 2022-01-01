package main

import (
	"fmt"
	"os"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kr/pretty"
)

func fail(msg interface{}, parts ...interface{}) {
	fmt.Printf(fmt.Sprintf("%v", msg)+"\n", parts...)
	os.Exit(1)
}

func main() {
	var args struct {
		Query string `arg:"positional,required"`
		Index string
	}
	arg.MustParse(&args)

	// load import graph
	graph, err := pythonimports.NewGraph(pythonimports.SmallImportGraph)
	if err != nil {
		fail(err)
	}

	// create the index
	opts := stackoverflow.DefaultOptions
	if args.Index != "" {
		opts.Path = args.Index
	}

	idx, err := stackoverflow.Load(graph, opts)
	if err != nil {
		fail(err)
	}

	// first resolve the query to a node
	node, err := graph.Find(args.Query)
	if err != nil {
		fail(err)
	}
	results, err := idx.LookupNode(node)
	if err != nil {
		fail(err)
	}

	// print the results
	pretty.Println(results)
}
