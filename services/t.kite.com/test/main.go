/*
This package is a binary for testing schema changes.
Use it to run sample data files against the schema. It will print
validation errors to stdout.
*/
package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

func main() {
	var schemaFile string
	flag.StringVar(&schemaFile, "schema", "", "path to schema file")

	var dataPath string
	flag.StringVar(&dataPath, "data", "", "path to data")

	var scrubMode bool
	flag.BoolVar(&scrubMode, "scrub", false, "output valid records rather than errors")

	flag.Parse()
	if schemaFile == "" {
		log.Fatal("schema argument is required.")
	}
	schemaPath, _ := filepath.Abs(schemaFile)

	if dataPath == "" {
		log.Fatal("data argument is required.")
	}
	dataPath, _ = filepath.Abs(dataPath)

	loader := gojsonschema.NewReferenceLoader(filepath.Join("file://", schemaPath))
	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		log.Fatalf("Error loading schema: %s", err)
	}

	fileInfo, err := os.Stat(dataPath)
	if err != nil {
		log.Fatalf("Error loading schema files: %s", err)
	}

	if fileInfo.IsDir() == false {
		validateFile(schema, dataPath, scrubMode)
		return
	}

	filepath.Walk(dataPath, func(root string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if info.IsDir() {
			return nil
		}

		validateFile(schema, root, scrubMode)
		return nil
	})
}

func validateFile(schema *gojsonschema.Schema, path string, scrubMode bool) int {
	count := 0
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error opening data file: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	start := time.Now()

	for scanner.Scan() {
		loader := gojsonschema.NewStringLoader(scanner.Text())
		result, err := schema.Validate(loader)
		if err != nil {
			log.Printf("Error parsing JSON: data=%s, err=%s", scanner.Text(), err)
		}
		if scrubMode && result.Valid() {
			os.Stdout.WriteString(scanner.Text())
			os.Stdout.WriteString("\n")
		}
		if !scrubMode && !result.Valid() {
			var strErrors = []string{}
			for _, e := range result.Errors() {
				strErrors = append(strErrors, e.String())
			}
			os.Stdout.WriteString(strings.Join(strErrors, "\n "))
		}
		count++
	}

	duration := time.Since(start)
	log.Printf("Finished %d records in %s, %f/sec", count, duration, float64(count)/duration.Seconds())

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return count
}
