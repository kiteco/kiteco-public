package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/curation"
)

// nextOutputName returns the name of output file name
func nextOutputName(outputDir string) string {
	fis, err := ioutil.ReadDir(outputDir)
	if err != nil {
		log.Fatalln("could not read outputDir:", outputDir, "error:", err)
	}

	return path.Join(outputDir, fmt.Sprintf("results-%d.json", len(fis)))
}

// remainingQueries returns a list of queries that have been fetched yet.
func remainingQueries(inputDir, outputDir string) []*queryPackage {
	queries := doneQueries(inputDir)

	var fetched int
	err := filepath.Walk(outputDir, func(path string, fi os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		decoder := json.NewDecoder(in)
		for {
			var results searchResults
			err = decoder.Decode(&results)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if _, exists := queries[results.Query]; exists {
				delete(queries, results.Query)
				fetched++
			}
		}

		return nil
	})

	if err != nil && err != io.EOF {
		log.Fatal("error walking outputDir:", err)
	}

	log.Println("found", len(queries), "queries")
	log.Println(fetched, "already fetched, getting the rest...")

	var queryList []*queryPackage
	for _, query := range queries {
		queryList = append(queryList, query)
	}
	return queryList
}

func doneQueries(inputDir string) map[string]*queryPackage {
	queries := make(map[string]*queryPackage)
	err := filepath.Walk(inputDir, func(path string, fi os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		decoder := json.NewDecoder(in)
		for {
			var suggestions curation.Suggestions
			err := decoder.Decode(&suggestions)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			for _, sugg := range suggestions.Suggestions {
				queries[sugg] = &queryPackage{
					query:       sugg,
					packageName: suggestions.Package,
				}
			}
		}
		return nil
	})
	if err != nil && err != io.EOF {
		log.Fatal("error walking inputDir:", err)
	}
	return queries
}

// setProxy sets the environment variable HTTP_PROXY to be the given value.
func setProxy(proxy string) error {
	return os.Setenv("HTTP_PROXY", proxy)
}
