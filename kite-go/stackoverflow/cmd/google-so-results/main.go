package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func main() {
	var input string
	var outputDir string
	var concurrency int
	flag.StringVar(&input, "input", "", "file containing list of queries (one per line)")
	flag.StringVar(&outputDir, "outputDir", "", "directory to store json.gz files containing stackoverflow.SearchResults items")
	flag.IntVar(&concurrency, "concurrency", 10, "number of fetches at a time")
	flag.Parse()

	// To ensure that HTTP_PROXY is set with the Crawlera proxy and credentials so we don't actually
	// bombard google with requests from a single IP
	if os.Getenv("HTTP_PROXY") == "" {
		log.Fatalln("Expected HTTP_PROXY environment variable. Please set it to the Crawlera proxy server with proper credentials.")
	}

	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Fatalln("could not create output directory:", outputDir)
	}

	queryChan := make(chan string, concurrency)
	resultsChan := make(chan *stackoverflow.SearchResults, concurrency)

	// Start up fetching goroutines. queryChan will provide a stream of queries, and
	// resultsChan is used to send the output back to the main goroutine.
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go fetcher(queryChan, resultsChan, &wg)
	}

	// Start the query supplier goroutine. This computes the remaining queries and
	// feeds queryChan with those queries. These are consumed by the fetchers above.
	go func() {
		remaining := remainingQueries(input, outputDir)
		for _, query := range remaining {
			queryChan <- query
		}
		close(queryChan)
	}()

	// Start a goroutine that closes the resultsChan once all fetchers have completed.
	// This allows for the for loop below to terminate when everything has completed.
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	nextOutput := nextOutputName(outputDir)
	out, err := os.Create(nextOutput)
	if err != nil {
		log.Fatalln("could not create next output file:", err)
	}
	defer out.Close()
	encoder := json.NewEncoder(out)

	for results := range resultsChan {
		err = encoder.Encode(results)
		if err != nil {
			log.Println("error encoding results:", err)
		}
		log.Println(len(results.Results), "results for", results.Query)

		// Flush anything buffered in out (via encoder)
		out.Sync()
	}

	fmt.Println("done!")
}

func fetcher(queryChan chan string, resultsChan chan *stackoverflow.SearchResults, wg *sync.WaitGroup) {
	defer wg.Done()
	for query := range queryChan {
		results, err := stackoverflow.SearchGoogle(query)
		if err != nil {
			log.Println("error fetching query:", query, "error:", err)
			continue
		}
		if len(results.Results) == 0 {
			log.Println("got no results for query:", query)
		} else {
			resultsChan <- results
		}
	}
}

func remainingQueries(input, outputDir string) []string {
	r, err := fileutil.NewReader(input)
	if err != nil {
		log.Fatalln("error opening file:", err)
	}

	queries := make(map[string]struct{})

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		queries[scanner.Text()] = struct{}{}
	}

	var fetched int
	err = filepath.Walk(outputDir, func(path string, fi os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(in)
		for {
			var results stackoverflow.SearchResults
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
		log.Fatalln("error walking outputDir:", err)
	}

	log.Println("found", len(queries), "queries")
	log.Println(fetched, "already fetched, getting the rest...")

	var queryList []string
	for q := range queries {
		queryList = append(queryList, q)
	}

	return queryList
}

func nextOutputName(outputDir string) string {
	fis, err := ioutil.ReadDir(outputDir)
	if err != nil {
		log.Fatalln("could not read outputDir:", outputDir, "error:", err)
	}

	return path.Join(outputDir, fmt.Sprintf("results-%d.json", len(fis)))
}
