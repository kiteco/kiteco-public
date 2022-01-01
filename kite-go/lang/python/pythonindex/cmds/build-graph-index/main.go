package main

import (
	"flag"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonindex"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	"github.com/kiteco/kiteco/kite-golib/text"
)

var (
	minCount               = pythonindex.DefaultClientOptions.MinOccurrence
	defaultInspectionDepth = 2
)

func main() {
	var (
		graphPath       string
		output          string
		inspectionDepth int
	)
	flag.StringVar(&graphPath, "graph", pythonimports.DefaultImportGraph, "path to import graph")
	flag.StringVar(&output, "output", "graph-index.json.gz", "output file")
	flag.IntVar(&inspectionDepth, "depth", defaultInspectionDepth, "depth to inspect in the graph")
	flag.Parse()

	// Load import graph
	graph, err := pythonimports.NewGraph(graphPath)
	if err != nil {
		log.Fatalln(err)
	}

	invertedIndex := make(map[string][]*pythonindex.IdentCount)
	for p := range graph.PkgToNode {
		nc := &pythonindex.IdentCount{
			Ident:       p,
			ForcedCount: minCount,
		}
		part := strings.ToLower(p)
		invertedIndex[part] = append(invertedIndex[part], nc)

		err := graph.Walk(p, func(name string, node *pythonimports.Node) bool {
			if strings.Count(name, ".") > inspectionDepth {
				return false
			}
			if node.Classification != pythonimports.Module && node.Classification != pythonimports.Type {
				return false
			}
			for m, child := range node.Members {
				if strings.HasPrefix(m, "_") {
					continue
				}
				if child == nil {
					continue
				}
				identifier := name + "." + m

				nc := &pythonindex.IdentCount{
					Ident:       identifier,
					ForcedCount: minCount,
				}

				parts := text.Uniquify(strings.Split(strings.ToLower(identifier), "."))
				for _, part := range parts {
					invertedIndex[part] = append(invertedIndex[part], nc)
				}
			}
			return true
		})
		if err != nil {
			log.Fatalf("error in walking through graph for package %s: %v\n", p, err)
		}
	}
	err = serialization.Encode(output, invertedIndex)
	if err != nil {
		log.Fatalln("error serializing output:", err)
	}
}
