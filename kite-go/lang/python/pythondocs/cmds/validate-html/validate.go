package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-golib/xmlvalidation"
)

func main() {
	var schemaFile, inputFile string
	flag.StringVar(&schemaFile, "schema", "", "schema file (.xsd)")
	flag.StringVar(&inputFile, "input", "", "input html")
	flag.Parse()

	if schemaFile == "" {
		log.Fatalln("Please specify schema file (.xsd)")
	}
	if inputFile == "" {
		log.Fatalln("Please specify input html file")
	}

	validator := xmlvalidation.NewHTMLValidator(schemaFile)
	if err := validator.ValidateFile(inputFile); err != nil {
		log.Fatalf("Error validating: %+v", err)
	}

	fmt.Println("HTML Valid as per XSD")
}
