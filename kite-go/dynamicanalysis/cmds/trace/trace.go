package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
)

func main() {
	var input, output string
	flag.StringVar(&input, "input", "", "path to python file to run tracing on")
	flag.StringVar(&output, "output", "", "path to which trace will be written")
	flag.Parse()

	if input == "" {
		log.Fatalln("You must pass --input <source.py>")
	}

	if output == "" {
		log.Fatalln("You must pass --output <trace.json>")
	}

	src, err := ioutil.ReadFile(input)
	if err != nil {
		log.Fatalf("Error reading from %s: %v", input, src)
	}

	result, err := dynamicanalysis.Trace(string(src), dynamicanalysis.DefaultTraceOptions)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Create(output)
	if err != nil {
		log.Fatalf("Error opening %s: %v", output, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(result.Tree)
	if err != nil {
		log.Fatalf("Error encoding json: %v", err)
	}
}
