package main

import (
	"log"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

var defaultPackages = []string{"builtins", "__future__", "types", "exceptions"}

func main() {
	var args struct {
		Input    string
		Output   string `args:"required"`
		Packages []string
	}
	args.Input = pythonimports.DefaultImportGraph
	arg.MustParse(&args)

	args.Packages = append(args.Packages, defaultPackages...)

	// open output encoder early to surface errors quickly
	enc, err := serialization.NewEncoder(args.Output)
	if err != nil {
		log.Fatal(err)
	}
	defer enc.Close()

	// load flat nodes
	in, err := pythonimports.LoadFlatGraph(args.Input)
	if err != nil {
		log.Fatal(err)
	}

	// construct graph
	graph := pythonimports.NewGraphFromNodes(in)

	// breadth first search
	sel := make(map[int64]bool)
	seen := make(map[*pythonimports.Node]bool)
	var queue []*pythonimports.Node

	push := func(n *pythonimports.Node) {
		if n != nil && !seen[n] {
			queue = append(queue, n)
			seen[n] = true
			sel[n.ID] = true
		}
	}

	for _, pkg := range args.Packages {
		n, found := graph.PkgToNode[pkg]
		if !found {
			log.Fatalf("%s not found in graph", pkg)
		}
		push(n)
	}

	for i := 0; i < len(queue); i++ {
		cur := queue[i]
		push(cur.Type)
		for _, base := range cur.Bases {
			push(base)
		}
		for _, child := range cur.Members {
			push(child)
		}
	}

	// construct output graph
	var out []*pythonimports.FlatNode
	for _, n := range in {
		if sel[n.ID] {
			out = append(out, n)
		}
	}

	log.Printf("selected %d of %d nodes", len(out), len(in))

	// write graph
	for _, flatnode := range out {
		enc.Encode(flatnode)
	}
}
