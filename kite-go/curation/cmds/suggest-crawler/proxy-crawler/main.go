package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/curation"
)

const (
	prefix          = "python"
	expansionCutoff = 0.3
	lang            = "python"
	source          = "google"
	endpoint        = "http://suggestqueries.google.com/complete/search?output=toolbar&hl=en&q="
)

var (
	offset = (*unicode.Lower).R16[0].Lo
)

// queryPackage records which package this query refers to.
type queryPackage struct {
	query       string
	packageName string
}

func main() {
	var (
		input       string
		outputDir   string
		concurrency int
		proxy       string
	)
	flag.StringVar(&input, "input", "", "file containing list of pythoncode.PackageStats (.json)")
	flag.StringVar(&outputDir, "outputDir", "", "directory to store json.gz files containing google suggestion results")
	flag.IntVar(&concurrency, "concurrency", 10, "number of fetches at a time")
	flag.StringVar(&proxy, "proxy", "", "proxy used for this crawling task")
	flag.Parse()

	if proxy == "" || outputDir == "" || input == "" {
		flag.Usage()
		log.Fatal("must specify --proxy, --outputDir, --input")
	}

	// To ensure that HTTP_PROXY is set with the Crawlera proxy.
	// The value of HTTP_PROXY is reset afterwards.
	err := setProxy(proxy)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Fatalln("could not create output directory:", outputDir)
	}

	queryChan := make(chan *queryPackage, concurrency)
	resultsChan := make(chan *curation.Suggestions, concurrency)

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
		log.Println(len(results.Suggestions), "results for", results.Ident)

		// Flush anything buffered in out (via encoder)
		out.Sync()
	}

	fmt.Println("done!")
}

func fetcher(queryChan chan *queryPackage, resultsChan chan *curation.Suggestions, wg *sync.WaitGroup) {
	defer wg.Done()
	for query := range queryChan {
		result, err := fetchGoogleSuggestions(query)
		if err != nil {
			log.Println("error fetching query:", query, "error:", err)
			continue
		}
		resultsChan <- result
	}
}

// setProxy sets the environment variable HTTP_PROXY to be the given value.
func setProxy(proxy string) error {
	return os.Setenv("HTTP_PROXY", proxy)
}
