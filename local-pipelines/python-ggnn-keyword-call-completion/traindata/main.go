package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

// struct that represents the source code part of the response
type sourcesResp struct {
	Sources []string `json:"sources"`
}

// struct that packages the request to be sent to the endpoint.
type sourcesRequest struct {
	Symbol string `json:"symbol"`
	Limit  int    `json:"limit"`
}

// Function to make a request to the graph server.
func makeRequest(ch chan<- nameAndSource, symbolName string, endpoint string, limit int) error {
	var sourcesByte [][]byte
	message := sourcesRequest{
		Symbol: symbolName,
		Limit:  limit,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(message); err != nil {
		return err
	}

	resp, err := http.Post(endpoint, "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var sr sourcesResp
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return err
	}
	for _, source := range sr.Sources {
		sourcesByte = append(sourcesByte, []byte(source))
	}
	fmt.Printf("Processed the response from %v\n", symbolName)
	ch <- nameAndSource{Name: symbolName, Source: sourcesByte}
	return nil
}

func loadPackageList(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("unable to open package list %s: %v", file, err)
	}
	defer f.Close()

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("unable to read package list %s: %v", file, err)
	}

	var packages []string
	for _, line := range strings.Split(string(contents), "\n") {
		if pkg := strings.TrimSpace(line); pkg != "" {
			packages = append(packages, pkg)
		}
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("no packages found in %s", file)
	}

	return packages, nil
}

func main() {
	args := struct {
		Out             string
		SourcesEndpoint string
		ScoresEndpoint  string
		Packages        string
		NumSamples      int
		NumGo           int
	}{
		Out:             "kwcntdata.json",
		SourcesEndpoint: "http://ml-training-2.kite.com:3039/symbol/sources",
		ScoresEndpoint:  "http://ml-training-2.kite.com:3039/symbol/scores",
		Packages:        "./packagelist.txt",
		NumSamples:      10,
		NumGo:           8,
	}

	arg.MustParse(&args)

	// table for symbols and their corresponding kws.
	symbolTable := make(map[string]keywordsCount)

	// logic to walk a tree of packages and get a list of functions. The input will be just a file with the
	pkgs, err := loadPackageList(args.Packages)
	if err != nil {
		log.Fatal(err)
	}

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions.SymbolOnly())
	if err := <-errc; err != nil {
		log.Fatalf("error loading rm: %v\n", err)
	}

	ti, err := typeinduction.LoadModel(rm, typeinduction.DefaultClientOptions)
	if err != nil {
		log.Fatalf("error loading typeinduction: %v\n", err)
	}

	walker := &walker{
		rm:   rm,
		seen: make(map[pythonimports.Hash]bool),
	}
	for _, pkg := range pkgs {
		if err := walker.Walk(pkg); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("done walking %s\n", pkg)
	}

	scores, err := getScores(walker.funcs, args.ScoresEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	// list of symbol names.
	var nameList []string
	for symbol := range scores {
		nameList = append(nameList, symbol)
	}
	sort.Strings(nameList)

	// Get source code and extract
	var jobs []workerpool.Job
	res := make(chan nameAndSource)
	for _, symbol := range nameList {
		symbolClose := symbol
		jobs = append(jobs, func() error { return makeRequest(res, symbolClose, args.SourcesEndpoint, args.NumSamples) })

	}

	pool := workerpool.New(args.NumGo)
	pool.Add(jobs)
	go func() {
		pool.Wait()
		pool.Stop()
		close(res)
	}()

	// make a channel to hold the results.
	symbolFeed := make(chan pair)
	var wg sync.WaitGroup
	for i := 0; i < args.NumGo; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Do work
			for nameSource := range res {
				name := nameSource.Name
				for _, source := range nameSource.Source {
					symbolName, kws, err := extract(name, rm, ti, source)
					if err != nil {
						log.Printf("Problem extracting for symbol %v: %v\n", symbolName, err)
					}
					symbolFeed <- pair{Name: symbolName, KW: kws}
				}
			}

		}()
	}

	go func() {
		wg.Wait()
		close(symbolFeed)
	}()

	for pair := range symbolFeed {
		merge(pair.Name, pair.KW, symbolTable)
	}

	for symbol, kwct := range symbolTable {
		if len(kwct) == 0 {
			delete(symbolTable, symbol)
		}
	}

	outf, err := os.Create(args.Out)
	if err != nil {
		log.Fatal(err)
	}
	defer outf.Close()

	if err := json.NewEncoder(outf).Encode(symbolTable); err != nil {
		log.Fatal(err)
	}
}

type pair struct {
	Name string
	KW   keywordsCount
}

type nameAndSource struct {
	Name   string
	Source [][]byte
}
