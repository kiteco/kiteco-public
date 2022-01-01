package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
)

// This binary takes a list of queries and their type, and outputs
// a json file that can be the input of cmds/searcher.

// Query represents a query and its type.
type Query struct {
	Content string
	Kind    string
}

func main() {
	var (
		input  string
		kind   string
		output string
	)

	flag.StringVar(&input, "input", "", "a text file in which each line is a query")
	flag.StringVar(&kind, "kind", "", "type of the queries")
	flag.StringVar(&output, "output", "", "output filename, e.g., queries.json")
	flag.Parse()

	if output == "" || input == "" || kind == "" {
		log.Fatal("must specify --output --input --kind")
	}

	f, err := os.Open(input)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fout, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	encoder := json.NewEncoder(fout)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		err = encoder.Encode(&Query{
			Content: scanner.Text(),
			Kind:    kind,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
