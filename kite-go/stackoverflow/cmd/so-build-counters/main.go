package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

func main() {
	var (
		pagesPath   string
		outBasePath string
	)
	flag.StringVar(&pagesPath, "pages", "", "path to so pages dump in GOB format (REQUIRED)")
	flag.StringVar(&outBasePath, "out", "", "path to write doc counts to in GOB format (REQUIRED)")
	flag.Parse()
	if pagesPath == "" || outBasePath == "" {
		flag.Usage()
		log.Fatal("pages and out params REQUIRED")
	}
	start := time.Now()

	// counts for each corpus
	counts := make(map[string]map[string]int)
	for _, dt := range search.DocTypes {
		counts[dt] = make(map[string]int)
	}
	var docCount int

	// page decoder
	f, err := os.Open(pagesPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	decoder := gob.NewDecoder(f)
	for {
		var page stackoverflow.StackOverflowPage
		err = decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		docCount++
		doc := search.Document{Page: &page}
		for dt, dts := range search.DTSelectors {
			toks := search.DTTokenizers[dt].Tokenize(dts(doc))
			toks = search.CountProcessor.Apply(toks)
			for _, tok := range toks {
				counts[dt][tok]++
			}
		}
	}

	// train idf counters
	idfs := make(map[string]*tfidf.IDFCounter)
	for dt, count := range counts {
		idfs[dt] = tfidf.TrainIDFCounter(docCount, count)
	}

	fout, err := os.Create(outBasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()
	encoder := gob.NewEncoder(fout)
	err = encoder.Encode(idfs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Duration: ", time.Since(start))
}
