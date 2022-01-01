package main

import (
	"flag"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// Docstring contains the id of an import graph node and it's docstring.
type Docstring struct {
	NodeID    int64  `json:"node_id"`
	Docstring string `json:"docstring"`
}

func main() {
	var output string
	flag.StringVar(&output, "output", "", "path to write output to (json.gz)")
	flag.Parse()

	graphStrings, err := pythonimports.LoadGraphStrings(pythonimports.DefaultImportGraphStrings)
	if err != nil {
		log.Fatalln(err)
	}

	enc, err := serialization.NewEncoder(output)
	if err != nil {
		log.Fatalln("Error creating encoder: ", err)
	}
	defer enc.Close()

	for nodeID, s := range graphStrings {
		out := &Docstring{
			NodeID:    nodeID,
			Docstring: s.Docstring,
		}
		if err := enc.Encode(&out); err != nil {
			log.Fatalln("Error encoding output: ", err)
		}
	}
}
