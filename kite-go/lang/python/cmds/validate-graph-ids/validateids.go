package main

import (
	"log"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

func main() {
	var args struct {
		Graph        string `arg:"required"`
		GraphStrings string `arg:"required"`
	}
	arg.MustParse(&args)

	graph, err := pythonimports.NewGraph(args.Graph)
	if err != nil {
		log.Fatal(err)
	}

	graphstrings, err := pythonimports.LoadGraphStrings(args.GraphStrings)
	if err != nil {
		log.Fatal(err)
	}

	ids := make(map[int64]struct{})
	for _, node := range graph.Nodes {
		ids[node.ID] = struct{}{}
	}

	var missingIDs []int64
	var missingReprs []string
	for id := range graphstrings {
		if _, found := ids[id]; !found {
			missingIDs = append(missingIDs, id)
			missingReprs = append(missingReprs, graphstrings[id].Repr)
		}
	}
	log.Printf("%d of %d graph strings IDs were missing from graph", len(missingIDs), len(graphstrings))
}
