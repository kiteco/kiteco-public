package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
)

// This binary takes in a list of curation.Suggestion objects,
// and sends these suggestions as search queries to google
// with the constraint that the return pages must be from
// stackoverflow.

// queryPackage records which package this query refers to.
type queryPackage struct {
	query       string
	packageName string
}

type searchResults struct {
	Query   string
	Source  string
	Package string
	Results []stackoverflow.SearchResult
}

func main() {
	var (
		inputDir    string
		outputDir   string
		proxy       string
		concurrency int
	)

	flag.StringVar(&inputDir, "inputDir", "", "directory that contains the output of proxy-cralwer")
	flag.StringVar(&outputDir, "outputDir", "", "directory to save the output")
	flag.StringVar(&proxy, "proxy", "", "proxy to use for crawling the results")
	flag.IntVar(&concurrency, "concurrency", 10, "number of concurrent crawling routines")
	flag.Parse()

	if inputDir == "" || outputDir == "" || proxy == "" {
		flag.Usage()
		log.Fatal("must specify --inputDir, --outputDir, --proxy, --concurrency")
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

	// set up input/output channels
	queryChan := make(chan *queryPackage, concurrency)
	resultsChan := make(chan *searchResults, concurrency)

	// start a thread for each concurrent process and use a wait group
	// to control when to exist the main func
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go fetcher(queryChan, resultsChan, &wg)
	}

	// start a go routine to get the queries ready
	go func() {
		remaining := remainingQueries(inputDir, outputDir)
		for _, query := range remaining {
			queryChan <- query
		}
		close(queryChan)
	}()

	// set up a go routine to close the results channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	nextOutput := nextOutputName(outputDir)
	out, err := os.Create(nextOutput)
	if err != nil {
		log.Fatal("could not create next output file:", err)
	}
	defer out.Close()

	encoder := json.NewEncoder(out)

	for results := range resultsChan {
		err = encoder.Encode(results)
		if err != nil {
			log.Println("error encoding results:", err)
		}
		log.Println(len(results.Results), "results for", results.Query)

		out.Sync()
	}
	fmt.Println("done!")
}

// fetcher fetches the google search results for the given query.
func fetcher(queryChan chan *queryPackage, resultsChan chan *searchResults, wg *sync.WaitGroup) {
	defer wg.Done()
	for qp := range queryChan {
		results, err := stackoverflow.SearchGoogle(qp.query)
		if err != nil {
			log.Println("error fetching query:", qp.query, "error:", err)
			continue
		}
		if len(results.Results) == 0 {
			log.Println("got no results for query:", qp.query)
		} else {
			resultsChan <- &searchResults{
				Query:   results.Query,
				Source:  results.Source,
				Package: qp.packageName,
				Results: results.Results,
			}
		}
	}
}
