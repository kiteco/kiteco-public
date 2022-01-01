package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang"
)

func main() {
	var path, dockerimage, langname string
	flag.StringVar(&langname, "language", lang.Python.Name(), "language to use")
	flag.StringVar(&path, "path", "", "path to example source")
	flag.StringVar(&dockerimage, "dockerimage", "kiteco/pythonsandbox", "docker image in which to run examples")
	flag.Parse()

	language := lang.FromName(langname)
	if language == lang.Unknown {
		log.Fatal("unknown language:", langname)
	}

	if path == "" {
		log.Fatal("missing required argument --path")
	}

	// Read the source
	src, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	// Run the snippet
	regions := []annotate.Region{
		annotate.Region{Name: "main", Code: string(src)},
	}
	flow, err := annotate.RunWithRegions(regions, string(src), annotate.Options{
		Language:    language,
		DockerImage: dockerimage,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Print raw output
	fmt.Printf("STDOUT:\n%s\n", string(flow.Raw.Stdout))
	fmt.Printf("STDERR:\n%s\n", string(flow.Raw.Stderr))

	if !flow.Raw.Succeeded {
		fmt.Println("Execution failed")
		return
	}

	// Build json representation for sidebar (this normally happens on user-node)
	segments := curation.SegmentsFromAnnotations(flow.Segments)
	out, err := json.MarshalIndent(segments, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("JSON:")
	os.Stdout.Write(out)
}
