package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

func main() {
	var full bool
	var importgraph string
	flag.StringVar(&importgraph, "importgraph", pythonimports.DefaultImportGraph, "path to import graph")
	flag.BoolVar(&full, "full", false, "print full node")
	flag.Parse()

	if importgraph == "" {
		fmt.Println("You must specify --importgraph")
		os.Exit(1)
	}

	query := flag.Arg(0)

	graph, err := pythonimports.NewGraph(importgraph)
	if err != nil {
		log.Fatalln(err)
	}

	if full {
		node, err := graph.Find(query)
		if err != nil {
			log.Fatalln(err)
		}
		buf, err := json.MarshalIndent(node, "", "  ")
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Print(string(buf))
	} else {
		canonical, err := graph.CanonicalName(query)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(canonical)
	}
}
