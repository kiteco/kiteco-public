package main

import (
	"flag"
	"io/ioutil"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

func main() {
	var index int
	var input, output, pkg string
	flag.StringVar(&input, "input", "", "path to snippets.json.gz or python source file")
	flag.StringVar(&output, "output", "", "path to which to write events as JSON")
	flag.StringVar(&pkg, "package", "", "only trace snippets from this package")
	flag.IntVar(&index, "index", -1, "process only the snippet at this index")
	flag.Parse()

	if input == "" {
		log.Fatalln("You must specify --input")
	}

	if output == "" {
		log.Fatalln("You must specify --output")
	}

	// Load input data
	var buffers []string
	if strings.HasSuffix(input, ".py") {
		// Load from a python file
		buf, err := ioutil.ReadFile(input)
		if err != nil {
			log.Fatalln(err)
		}
		buffers = append(buffers, string(buf))
	} else {
		// Load snippets from json
		var i int
		err := serialization.Decode(input, func(example *curation.Example) error {
			if index != -1 && i != index {
				return nil
			}
			i++

			s := example.Snippet
			if pkg != "" && s.Package != pkg {
				return nil
			}

			joined := s.Prelude + "\n" + s.Code + "\n" + s.Postlude
			stencil, err := annotate.ParseExample(joined, lang.Python)
			if err != nil {
				return err
			}

			buffers = append(buffers, stencil.Runnable)
			return err
		})
		if err != nil {
			log.Fatalln(err)
		}
	}

	// Open encoder
	enc, err := serialization.NewEncoder(output)
	if err != nil {
		log.Fatalln(err)
	}
	defer enc.Close()

	// Run dynamic analysis on each buffer
	var numFailed, numUsages int
	for i, buf := range buffers {
		log.Printf("Tracing snippet %d of %d...", i+1, len(buffers))

		buf = strings.TrimPrefix(buf, "from kite import kite")

		// Get the events
		events, err := dynamicanalysis.TraceEvents(buf, dynamicanalysis.DefaultTraceOptions)
		if err != nil {
			log.Printf("Failed to trace snippet, will ignore: %v\n", err)
			numFailed++
			continue
		}

		// Compute usages
		usages, err := dynamicanalysis.UsagesFromEvents(events)
		if err != nil {
			log.Printf("Failed to compute usages, will ignore: %v\n", err)
			numFailed++
			continue
		}

		for _, usage := range usages {
			// For now, ignore objects that were not returned from a function
			if usage.ReturnedFrom == "" {
				continue
			}
			numUsages++
			err = enc.Encode(usage)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}

	numSucceeded := len(buffers) - numFailed
	log.Printf("Generated %d usages from %d snippets and ignored %d snippets that failed to execute",
		numUsages, numSucceeded, numFailed)
}
