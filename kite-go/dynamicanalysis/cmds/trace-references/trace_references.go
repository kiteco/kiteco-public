package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-golib/jsonutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

func main() {
	var index, limit int
	var input, output, pkg string
	flag.StringVar(&input, "input", "", "path to snippets.json.gz or python source file")
	flag.StringVar(&output, "output", "", "path to which to write references as JSON")
	flag.StringVar(&pkg, "package", "", "only trace snippets from this package")
	flag.IntVar(&index, "index", -1, "process only the item at this index")
	flag.IntVar(&limit, "limit", -1, "process only this number of snippets")
	flag.Parse()

	if input == "" {
		log.Fatalln("You must specify --input")
	}

	if output == "" {
		log.Fatalln("You must specify --output")
	}

	var numSucceeded, numFailed int

	if strings.HasSuffix(input, ".py") {
		// Load from a python file
		buf, err := ioutil.ReadFile(input)
		if err != nil {
			log.Fatalln(err)
		}

		// Open encoder
		w, err := os.Create(output)
		if err != nil {
			log.Fatalln(err)
		}
		defer w.Close()
		enc, err := serialization.NewEncoder(output)
		if err != nil {
			log.Fatalln(err)
		}
		defer enc.Close()

		// Trace references
		refs, _, err := dynamicanalysis.TraceReferences(string(buf), dynamicanalysis.DefaultTraceOptions)
		if err != nil {
			log.Printf("Failed to trace snippet, will ignore: %v\n", err)
			numFailed++
		}
		traced := dynamicanalysis.ResolvedSnippet{
			Code:       string(buf),
			References: refs,
		}
		err = enc.Encode(&traced)
		if err != nil {
			log.Fatalln(err)
		}
		numSucceeded++

	} else {
		// Load snippets from json
		var snippets []*curation.CuratedSnippet
		var i int
		err := jsonutil.DecodeAllFrom(input, func(example *curation.Example) error {
			i++
			if index != -1 && i != index+1 {
				return nil
			}
			s := example.Snippet
			if pkg != "" && s.Package != pkg {
				return nil
			}
			snippets = append(snippets, s)
			return nil
		})
		if err != nil {
			log.Fatalln(err)
		}

		// Open encoder
		w, err := os.Create(output)
		if err != nil {
			log.Fatalln(err)
		}
		defer w.Close()
		enc, err := serialization.NewEncoder(output)
		if err != nil {
			log.Fatalln(err)
		}
		defer enc.Close()

		// Apply limit on number of traces
		if limit > 0 && len(snippets) > limit {
			snippets = snippets[:limit]
		}

		// Trace references
		for i, snip := range snippets {
			log.Printf("Tracing snippet %d of %d...", i+1, len(snippets))
			refs, flow, err := dynamicanalysis.TraceSnippetReferences(snip, dynamicanalysis.DefaultTraceOptions)
			if err != nil {
				log.Printf("Failed to trace snippet, will ignore: %v\n", err)
				numFailed++
				continue
			}
			traced := dynamicanalysis.ResolvedSnippet{
				SnippetID:     snip.SnippetID,
				Code:          flow.Stencil.Runnable,
				PresentedCode: flow.Stencil.Presentation,
				LineMap:       flow.Stencil.LineMap,
				References:    refs,
			}
			err = enc.Encode(&traced)
			if err != nil {
				log.Fatalln(err)
			}
			numSucceeded++
		}
	}

	log.Printf("Wrote %d snippets and ignored %d that failed to execute", numSucceeded, numFailed)
}
