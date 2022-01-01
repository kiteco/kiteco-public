package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
)

// Input: a list of results from the google-so-results tool,
// i.e input is a file containing []stackoverflow.SearchResults as json.
// Output: a list of search.Log structs written as json to a file,
// i.e output is a file containing one search.Log object per line.
func main() {
	var (
		inputPath string
		outPath   string
	)
	flag.StringVar(&inputPath, "in", "", "path to directory containing results from google-so-results binary")
	flag.StringVar(&outPath, "out", "", "path to write fetched SO SearchLogs")
	flag.Parse()

	if inputPath == "" {
		flag.Usage()
		log.Fatal("must specify input path")
	}
	if outPath == "" {
		flag.Usage()
		log.Fatal("must specify output path")
	}

	// for writing to out stream
	fout, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()
	encoder := json.NewEncoder(fout)

	// for fetching SO pages from DB
	client, err := stackoverflow.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	count := 0

	err = filepath.Walk(inputPath, func(path string, fi os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(in)

		for {
			var searcherResults stackoverflow.SearchResults
			err = decoder.Decode(&searcherResults)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			sl := search.Log{
				Query: searcherResults.Query,
			}

			for i, sr := range searcherResults.Results {
				if sr.ID < 1 {
					// err case, unable to parse so id from url for page
					continue
				}
				posts, err := client.PostsByID([]int{int(sr.ID)})
				if err != nil {
					log.Println(err)
				}
				if len(posts) == 0 {
					continue
				}
				if len(posts) > 1 {
					log.Fatal("multiple pages returned for the same SO url")
				}

				sl.Results = append(sl.Results, search.Document{
					ID:    sr.ID,
					URL:   sr.URL,
					Page:  posts[0],
					Score: 10 - i,
				})

				count++
				if count%500 == 0 {
					fmt.Printf("Have fetched %d SO pages \n", count)
				}
			}

			if len(sl.Results) > 0 {
				err = encoder.Encode(sl)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Fetched and wrote %d SO pages \n", count)
}
