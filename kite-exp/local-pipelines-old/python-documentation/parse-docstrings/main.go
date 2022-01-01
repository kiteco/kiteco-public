package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

// docstringHTML contains the id of an import graph node and it's docstring converted to HTML.
type docstringHTML struct {
	NodeID    int64  `json:"node_id"`
	Docstring string `json:"description"`
}

func parallelParse(g *pythonimports.Graph, s *pythonimports.ArgSpecs) (chan<- *docstringHTML, <-chan *pythondocs.LangEntity) {
	inputDocstrings := make(chan *docstringHTML)
	outputEntities := make(chan *pythondocs.LangEntity)
	parser := pythondocs.NewDocstringParser(g)

	numCPU := runtime.NumCPU()
	log.Println("parsing objects using %d goroutines...", numCPU)

	var wg sync.WaitGroup
	wg.Add(numCPU)
	for i := 0; i < numCPU; i++ {
		go func() {
			defer wg.Done()
			for docstring := range inputDocstrings {
				node, ok := g.FindByID(docstring.NodeID)
				if !ok {
					log.Println(fmt.Sprintf("could not find node with id %d", docstring.NodeID))
					continue
				}
				entity, err := parser.Parse(node, s.Find(node), docstring.Docstring)
				if err != nil {
					log.Fatalln(err)
				}
				outputEntities <- entity
			}
		}()
	}

	go func() {
		wg.Wait()
		close(outputEntities)
	}()

	return inputDocstrings, outputEntities
}

func main() {
	var (
		input    string
		output   string
		progress uint64
	)
	flag.StringVar(&output, "output", "pythondocstrings.gob.gz", "gob.gz file that will contain LangEntity objects")
	flag.Uint64Var(&progress, "progress", 0, "number of objects to process between progress updates")
	flag.Parse()
	input = flag.Arg(0)

	if !strings.HasSuffix(input, ".json.gz") {
		log.Fatalf("input filename %q must end with .json.gz", input)
	}
	if !strings.HasSuffix(output, ".gob.gz") {
		log.Fatalf("output filename %q must end with .gob.gz", output)
	}

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatalln("Error loading import graph:", err)
	}

	argSpecs, err := pythonimports.LoadArgSpecs(graph, pythonimports.DefaultImportGraphArgSpecs, pythonimports.DefaultTypeshedArgSpecs)
	if err != nil {
		log.Fatalln("Error loading arg specs:", err)
	}

	enc, err := serialization.NewEncoder(output)
	if err != nil {
		log.Fatalln(err)
	}
	defer enc.Close()

	inputDocstrings, outputEntities := parallelParse(graph, argSpecs)

	var readCount, writeCount uint64

	go func() {
		err = serialization.Decode(input, func(docstring *docstringHTML) error {
			readCount++
			inputDocstrings <- docstring
			return nil
		})
		if err != nil { // error during parsing
			log.Fatalln(err)
		}
		close(inputDocstrings)
	}()

	for entity := range outputEntities {
		if entity != nil {
			err = enc.Encode(entity)
			if err != nil {
				log.Fatalln(err)
			}

			writeCount++
			if progress > 0 && writeCount%progress == 0 {
				log.Printf("%d objects written", writeCount)
			}
		}
	}

	log.Printf("Done: read %d objects, wrote %d objects.", readCount, writeCount)
}
