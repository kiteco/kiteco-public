package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/codeexample"
	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

type attrsByType map[string][]string
type attrsBySnippet map[string]attrsByType

func main() {
	var snippets, attrs, output string
	flag.StringVar(&snippets, "snippets", "", "emr file containing codeexample.CuratedSnippet structs to populate with attribute info")
	flag.StringVar(&attrs, "attributes", "", "json.gz file containing map of snippet hash to attribute info")
	flag.StringVar(&output, "output", "", "updated emr file with attribute info per snippet, in struct codeexample.AnalyzedCuratedSnippet")
	flag.Parse()

	if snippets == "" {
		log.Fatalln("Please specify snippets file name.")
	}
	if attrs == "" {
		log.Fatalln("Please specify attributes file name.")
	}
	if output == "" {
		log.Fatalln("Please specify output file name.")
	}

	// Read in attributes into a map
	decoder := dynamicanalysis.NewDecoder(attrs)
	attributes := make(attrsBySnippet)
	err := decoder.Decode(&attributes)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			log.Fatalln("Encountered unexpected EOF")
		} else if err == io.EOF {
			log.Fatalln("Reached EOF. Exit gracefully.")
		} else {
			log.Fatal(err)
		}
	}

	out, err := os.Create(output)
	if err != nil {
		log.Fatalf("Error creating output file at %s: %v\n", output, err)
	}
	defer out.Close()

	w := awsutil.NewEMRWriter(out)
	defer w.Close()

	// Populate snippets with attributes
	file, err := os.Open(snippets)
	if err != nil {
		log.Fatal(err)
	}
	r := awsutil.NewEMRIterator(file)
	for r.Next() {
		var cs codeexample.CuratedSnippet
		if err := json.Unmarshal(r.Value(), &cs); err != nil {
			log.Fatal(err)
		}

		// Look up attributes
		snippetHash := cs.Snippet.Hash().String()
		attrsForSnippet, ok := attributes[snippetHash]
		if !ok {
			log.Printf("No attribute information for snippet %d\n", cs.Curated.Snippet.SnippetID)
		}

		analyzed := &codeexample.AnalyzedCuratedSnippet{
			Snippet: &cs,
		}

		// Populate snippet
		for typ, attrs := range attrsForSnippet {
			for _, attr := range attrs {
				analyzed.Attributes = append(analyzed.Attributes, codeexample.Attribute{
					Type:      typ,
					Attribute: attr,
				})
			}
		}

		// Output modified snippet
		buf, err := json.Marshal(analyzed)
		if err != nil {
			log.Fatalf("Error marshaling codeexample.AnalyzedCuratedSnippet to json: %v\n", err)
		}

		w.Emit(r.Key(), buf)
	}
	if err := r.Err(); err != nil {
		log.Fatalf("Error iterating over snippets: %v\n", err)
	}
}
