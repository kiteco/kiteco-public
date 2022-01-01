package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// testDatum represents an entry of the human-labelled test data
// for method prediction
type testDatum struct {
	query   string
	methods []string
}

// loadTestData loads test data for method prediction from the given file path.
func loadTestData(file string) []testDatum {
	in, err := fileutil.NewCachedReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	decomp, err := gzip.NewReader(in)
	if err != nil {
		log.Fatal(err)
	}

	var testData []testDatum

	decoder := json.NewDecoder(decomp)
	for {
		var intermediate struct {
			Query  string
			Method string
		}
		err := decoder.Decode(&intermediate)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		var methods []string
		for _, m := range strings.Split(intermediate.Method, ",") {
			methods = append(methods, strings.TrimSpace(m))
		}
		testData = append(testData, testDatum{
			query:   intermediate.Query,
			methods: methods,
		})

	}
	return testData
}
