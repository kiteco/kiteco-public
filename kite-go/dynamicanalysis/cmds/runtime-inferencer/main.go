package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	var examples, image, output string
	flag.StringVar(&examples, "examples", "", "emr file containing codeexample.CuratedSnippet structs to run runtime type inference on")
	flag.StringVar(&image, "image", "", "specify docker image name")
	flag.StringVar(&output, "output", "", "specify filename (json.gz) to output annotated ASTs of each snippet to")
	flag.Parse()

	if output == "" {
		log.Fatalln("Please specify output file name.")
	}

	if image == "" {
		log.Fatalln("Please specify the docker image name")
	}

	f, err := os.Create(output)
	if err != nil {
		log.Fatalf("Error creating output file at %s: %v\n", output, err)
	}
	defer f.Close()

	gzipper := gzip.NewWriter(f)
	defer gzipper.Close()

	opts := dynamicanalysis.TraceOptions{
		DockerImage: image,
	}

	file, err := os.Open(examples)
	if err != nil {
		log.Fatal(err)
	}
	r := awsutil.NewEMRIterator(file)
	for r.Next() {
		var cs pythoncuration.Snippet
		if err := json.Unmarshal(r.Value(), &cs); err != nil {
			log.Fatal(err)
		}

		snippet := cs.Curated.Snippet
		snippetHash := cs.Snippet.Hash().String()

		log.Printf("Running snippet %d: %s\n", snippet.SnippetID, snippet.Title)

		src := snippet.Prelude + "\n" + snippet.Code + "\n" + snippet.Postlude
		result, err := dynamicanalysis.Trace(src, opts)
		if err != nil {
			log.Printf("Failed to run dynamic analysis on snippet %d: %v", snippet.SnippetID, err)
			continue
		}

		// Additional fields to identify the AST
		dat := result.Tree
		dat["SnippetID"] = snippet.SnippetID
		dat["SnapshotID"] = snippet.SnapshotID
		dat["hash"] = snippetHash

		b, err := json.Marshal(dat)
		if err != nil {
			log.Printf("Error marshaling updated ast map to json: %v\n", err)
			continue
		}

		_, err = gzipper.Write(b)
		if err != nil {
			log.Printf("Error writing snippet %d with AST %s: %v\n", snippet.SnippetID, dat, err)
			continue
		}

		fmt.Println(string(result.Output.Stdout))
	}
	if err := r.Err(); err != nil {
		log.Fatal(err)
	}
}
